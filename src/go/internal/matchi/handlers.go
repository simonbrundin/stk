package matchi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ListSlots godoc
// @Summary		Hämta tidslotter från MATCHi
// @Description	Returnerar dagens bokade och fria tidslotter för anläggningen i MATCHi. Tiden avrundas till hela och halva timmar.
// @Tags			matchi
// @Accept			json
// @Produce		json
// @Param			date	query		string	false	"Datum (YYYY-MM-DD)"	default(today)
// @Param			court	query		string	false	"Filtrera på bana/sal"
// @Success		200		{object}	Schedule
// @Failure		400		{object}	ErrorResponse
// @Failure		502		{object}	ErrorResponse
// @Router			/matchi/slots [get]
func ListSlots(client *Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := client.Location()

		day, err := parseDate(c.Query("date"), loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid date: %v", err)})
			return
		}

		schedule, err := client.Schedule(c.Request.Context(), day)
		if err != nil {
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
			return
		}

		court := strings.TrimSpace(c.Query("court"))
		if court != "" {
			filtered := make([]Slot, 0, len(schedule.Slots))
			for _, slot := range schedule.Slots {
				if strings.EqualFold(slot.CourtName, court) {
					filtered = append(filtered, slot)
				}
			}
			schedule.Slots = filtered
		}

		c.JSON(http.StatusOK, schedule)
	}
}

// CreateBooking godoc
// @Summary		Boka en tid i MATCHi
// @Description	Bookar en tid på en bana eller sal. Starttid avrundas nedåt till närmaste halvtimme och sluttid avrundas uppåt till närmaste halvtimme.
// @Tags			matchi
// @Accept			json
// @Produce		json
// @Param			body	body		BookRequest	true	"Bokning"
// @Success		201		{object}	BookingResult
// @Failure		400		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Failure		502		{object}	ErrorResponse
// @Router			/matchi/bookings [post]
func CreateBooking(client *Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		result, err := client.Book(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusCreated, result)
	}
}

// MoveBooking godoc
// @Summary		Flytta en bokning i MATCHi
// @Description	Flyttar en befintlig bokning till en ny tid. Tiden avrundas till hela och halva timmar.
// @Tags			matchi
// @Accept			json
// @Produce		json
// @Param			body	body		MoveRequest	true	"Flytt"
// @Success		200		{object}	BookingResult
// @Failure		400		{object}	ErrorResponse
// @Failure		502		{object}	ErrorResponse
// @Router			/matchi/bookings/move [post]
func MoveBooking(client *Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MoveRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		result, err := client.Move(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// CancelBooking godoc
// @Summary		Avboka en bokning i MATCHi
// @Description	Avbokar en bokning utan återbetalning och utan notis till kunden.
// @Tags			matchi
// @Accept			json
// @Produce		json
// @Param			body	body		CancelRequest	true	"Avbokning"
// @Success		204		"No Content"
// @Failure		400		{object}	ErrorResponse
// @Failure		502		{object}	ErrorResponse
// @Router			/matchi/bookings/cancel [post]
func CancelBooking(client *Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CancelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		if err := client.Cancel(c.Request.Context(), req.SlotIDs); err != nil {
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// CancelRequest är en begäran om att avboka bokningar.
type CancelRequest struct {
	// @Description	MATCHis slot-id:n för de bokningar som ska avbokas
	// @Example		["4a66b2f0-3177-4c9c-a7a5-3a54e44a9737"]
	SlotIDs []string `json:"slot_ids" binding:"required"`
}

// parseDate tolkar ett datum i formatet YYYY-MM-DD.
func parseDate(value string, loc *time.Location) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		now := time.Now().In(loc)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	}

	parsed, err := time.ParseInLocation("2006-01-02", value, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected YYYY-MM-DD, got %q", value)
	}

	return parsed, nil
}
