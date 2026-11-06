package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	// Tidszonsdatabasen bäddas in i binären så att Europe/Stockholm alltid
	// finns, oberoende av vad basimageen innehåller.
	_ "time/tzdata"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/matchi"
	"github.com/simon/repo-template/internal/occupancy"
	"github.com/simon/repo-template/internal/store"
	"github.com/simon/repo-template/internal/sync"
)

// @title			Repo Template API
// @version			1.0
// @description		API for repo-template project
// @BasePath		/

func getDocsPath() string {
	if path := os.Getenv("DOCS_PATH"); path != "" {
		return path
	}
	return "./cmd/api/docs/swagger.json"
}

// readSwaggerSpec reads and returns the swagger spec
func readSwaggerSpec() ([]byte, error) {
	return os.ReadFile(getDocsPath())
}

// envInt läser ett heltal ur en miljövariabel, med fallback.
func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("miljövariabelen %s har ogiltigt värde %q, använder %d", name, value, fallback)
		return fallback
	}

	return parsed
}

// withHost modifies the swagger spec to use the correct host
func withHost(specJSON []byte, host string) ([]byte, error) {
	var spec map[string]interface{}
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, err
	}
	// If host is empty, remove the host field (uses relative URLs)
	if host == "" {
		delete(spec, "host")
	} else {
		spec["host"] = host
	}
	return json.Marshal(spec)
}

func main() {
	r := gin.Default()

	// CORS middleware - only enable if CORS_ORIGINS is set
	// In production with same-origin (nginx proxy), CORS_ORIGINS can be empty
	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins != "" {
		corsConfig := cors.Config{
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
		}
		if corsOrigins == "*" {
			// Development: allow all origins
			corsConfig.AllowAllOrigins = true
		} else {
			// Specific origins
			corsConfig.AllowOrigins = strings.Split(corsOrigins, ",")
		}
		r.Use(cors.New(corsConfig))
	}

	// API documentation with Scalar UI
	r.GET("/docs/*any", openapiui.WrapHandler(openapiui.Config{
		SpecURL: "/docs/openapi.json",
		SpecProvider: func() ([]byte, error) {
			host := os.Getenv("API_HOST")
			spec, err := readSwaggerSpec()
			if err != nil {
				return nil, err
			}
			return withHost(spec, host)
		},
		Title: "Repo Template API",
		Theme: "dark",
	}))

	// @Summary		Health check
	// @Description	Returns the health status of the API
	// @Tags			health
	// @Produce		json
	// @Success		200	{object}	map[string]interface{}
	// @Router		/health [get]
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"version":   os.Getenv("APP_VERSION"),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// @Summary		Hello world
	// @Description	Returns a hello message
	// @Tags			greeting
	// @Produce		json
	// @Success		200	{object}	map[string]string
	// @Router		/hello [get]
	r.GET("/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})

	// Actor Smartbook - kommunens bokningssystem
	actorsmartbookClient := actorsmartbook.NewClient(
		os.Getenv("ACTORSMARTBOOK_BASE_URL"),
		os.Getenv("ACTORSMARTBOOK_RESOURCE_FILTER"),
		actorsmartbook.DefaultTimeout,
		nil,
	)
	r.GET("/actorsmartbook/appointments", actorsmartbook.ListAppointments(actorsmartbookClient))

	// MATCHi - anläggningens bokningssystem
	matchiClient := matchi.NewClient(
		os.Getenv("MATCHI_BASE_URL"),
		os.Getenv("MATCHI_USERNAME"),
		os.Getenv("MATCHI_PASSWORD"),
		matchi.DefaultTimeout,
		nil,
	)
	r.GET("/matchi/slots", matchi.ListSlots(matchiClient))
	r.POST("/matchi/bookings", matchi.CreateBooking(matchiClient))
	r.POST("/matchi/bookings/move", matchi.MoveBooking(matchiClient))
	r.POST("/matchi/bookings/cancel", matchi.CancelBooking(matchiClient))

	// Databas - Neon/PostgreSQL
	var bookingStore *store.Store
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		db, err := store.Open(dsn)
		if err != nil {
			log.Fatalf("kunde inte ansluta till databasen: %v", err)
		}
		defer db.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := db.EnsureSchema(ctx); err != nil {
			cancel()
			log.Fatalf("kunde inte skapa databasschemat: %v", err)
		}
		cancel()

		bookingStore = db

		r.GET("/bookings", sync.ListBookings(db))
		r.GET("/court-mappings", sync.ListCourtMappings(db))
		r.POST("/court-mappings", sync.UpsertCourtMapping(db))

		// Skrivläsande jämförelse mellan kommunens bokningar och MATCHi.
		// Endpointen bokar aldrig något i MATCHi.
		occupancyService := occupancy.New(
			actorsmartbookClient,
			matchiClient,
			db,
			actorsmartbookClient.Location(),
		)
		r.GET("/matchi/bookings", occupancy.ListBooked(occupancyService))

		// Synkronisering av kommunens bokningar till MATCHi
		syncer := sync.New(
			actorsmartbookClient,
			matchiClient,
			bookingStore,
			sync.Config{
				Days:               envInt("SYNC_DAYS", sync.DefaultDays),
				CustomerID:         int64(envInt("MATCHI_CUSTOMER_ID", 0)),
				OrganizationFilter: os.Getenv("SYNC_ORGANIZATION_FILTER"),
				RemoveOrphans:      os.Getenv("SYNC_REMOVE_ORPHANS") == "true",
			},
			actorsmartbookClient.Location(),
		)
		r.POST("/sync", sync.RunSync(syncer))

		// Bokning av en enskild kommunbokning i MATCHi. Cachen töms så att
		// den nya bokningen syns direkt i jämförelsen.
		r.POST("/matchi/bookings/link", sync.LinkBooking(syncer, occupancyService.Invalidate))
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" || !strings.HasPrefix(port, ":") {
		if port == "" {
			port = "8080"
		}
		port = ":" + port
	}

	r.Run(port)
}
