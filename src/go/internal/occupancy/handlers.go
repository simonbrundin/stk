package occupancy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/simon/repo-template/internal/actorsmartbook"
)

const (
	// defaultWindowDays är hur många dagar som jämförs om ingen period anges.
	defaultWindowDays = 7

	// maxWindowDays begränsar hur lång period som kan jämföras i ett anrop.
	maxWindowDays = 31
)

// ListBooked godoc
// @Summary		Markera tider som är bokade i MATCHi
// @Description	Returnerar de bokningar i det kommunala bokningssystemet som även är bokade i MATCHi. Anropet är skrivläsande - det bokar aldrig något i MATCHi.
// @Tags		matchi
// @Accept		json
// @Produce	json
// @Param		from	query		string	false	"Periodens början (RFC3339, YYYY-MM-DD eller 'now')"	default(now)
// @Param		to		query		string	false	"Periodens slut (RFC3339, YYYY-MM-DD eller 'now+7d')"	default(now+7d)
// @Success		200	{object}	Result
// @Failure		400	{object}	actorsmartbook.ErrorResponse
// @Failure		502	{object}	actorsmartbook.ErrorResponse
// @Router		/matchi/bookings [get]
func ListBooked(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := service.loc

		now := time.Now().In(loc)
		from, err := actorsmartbook.ParseTimeParam(c.Query("from"), loc, now)
		if err != nil {
			c.JSON(http.StatusBadRequest, actorsmartbook.ErrorResponse{Error: fmt.Sprintf("invalid from: %v", err)})
			return
		}

		to, err := actorsmartbook.ParseTimeParam(c.Query("to"), loc, from.AddDate(0, 0, defaultWindowDays))
		if err != nil {
			c.JSON(http.StatusBadRequest, actorsmartbook.ErrorResponse{Error: fmt.Sprintf("invalid to: %v", err)})
			return
		}

		if !to.After(from) {
			c.JSON(http.StatusBadRequest, actorsmartbook.ErrorResponse{Error: "invalid time range: to must be after from"})
			return
		}

		if to.Sub(from) > maxWindowDays*24*time.Hour {
			c.JSON(http.StatusBadRequest, actorsmartbook.ErrorResponse{
				Error: fmt.Sprintf("time range too large: maximum is %d days", maxWindowDays),
			})
			return
		}

		result, err := service.Booked(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusBadGateway, actorsmartbook.ErrorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
