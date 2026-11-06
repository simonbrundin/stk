package sync

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/simon/repo-template/internal/store"
)

// RunSync godoc
// @Summary		Synka kommunens bokningar till MATCHi
// @Description	Hämtar bokningarna från det kommunala bokningssystemet för de kommande dagarna och speglar dem i MATCHi. Bokningar som saknas i MATCHi bokas upp och bokningar med ändrad tid flyttas.
// @Tags			sync
// @Accept			json
// @Produce		json
// @Success		200	{object}	Report
// @Failure		500	{object}	ErrorResponse
// @Router			/sync [post]
func RunSync(syncer *Syncer) gin.HandlerFunc {
	return func(c *gin.Context) {
		report, err := syncer.Run(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, report)
	}
}

// ListCourtMappings godoc
// @Summary		Lista kopplingar mellan kommunresurser och MATCHi-banor
// @Description	Returnerar alla kopplingar som används vid synkroniseringen.
// @Tags			sync
// @Accept			json
// @Produce		json
// @Success		200	{array}	store.CourtMapping
// @Failure		500	{object}	ErrorResponse
// @Router			/court-mappings [get]
func ListCourtMappings(db Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		mappings, err := db.ListCourtMappings(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, mappings)
	}
}

// UpsertCourtMapping godoc
// @Summary		Skapa eller uppdatera en koppling
// @Description	Kopplar en resurs i det kommunala bokningssystemet till en bana eller sal i MATCHi.
// @Tags			sync
// @Accept			json
// @Produce		json
// @Param			body	body		store.CourtMapping	true	"Koppling"
// @Success		200		{object}	store.CourtMapping
// @Failure		400		{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router			/court-mappings [post]
func UpsertCourtMapping(db Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var mapping store.CourtMapping
		if err := c.ShouldBindJSON(&mapping); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		if mapping.KommunResourceID == 0 || mapping.MatchiCourtID == 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "kommun_resource_id och matchi_court_id krävs"})
			return
		}

		if err := db.UpsertCourtMapping(c.Request.Context(), mapping); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, mapping)
	}
}

// ListBookings godoc
// @Summary		Lista synkade bokningar
// @Description	Returnerar de bokningar som speglats mellan det kommunala bokningssystemet och MATCHi.
// @Tags			sync
// @Accept			json
// @Produce		json
// @Param			from	query		string	false	"Periodens början (RFC3339 eller YYYY-MM-DD)"	default(now)
// @Param			to	query		string	false	"Periodens slut (RFC3339 eller YYYY-MM-DD)"	default(now+21d)
// @Success		200	{array}	store.Booking
// @Failure		400	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router			/bookings [get]
func ListBookings(db Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := time.Now().Location()
		from, err := parseBoundary(c.Query("from"), loc, time.Now())
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid from: %v", err)})
			return
		}

		to, err := parseBoundary(c.Query("to"), loc, from.AddDate(0, 0, 21))
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid to: %v", err)})
			return
		}

		bookings, err := db.ListBookingsInRange(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, bookings)
	}
}

// ErrorResponse är standard-felsvar.
type ErrorResponse struct {
	// @Description	Felmeddelande
	// @Example		invalid time range
	Error string `json:"error"`
}

// parseBoundary tolkar en periodgräns.
func parseBoundary(value string, loc *time.Location, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.In(loc), nil
	}

	if parsed, err := time.ParseInLocation("2006-01-02", value, loc); err == nil {
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DD, got %q", value)
}
