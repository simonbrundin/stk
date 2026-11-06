package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	openapiui "github.com/PeterTakahashi/gin-openapi/openapiui"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/simon/repo-template/internal/actorsmartbook"
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
