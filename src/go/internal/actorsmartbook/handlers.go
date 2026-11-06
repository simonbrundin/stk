package actorsmartbook

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// defaultWindowDays är hur många dagar som hämtas om ingen period anges.
	defaultWindowDays = 7

	// maxWindowDays begränsar hur lång period som kan hämtas i ett anrop.
	maxWindowDays = 31
)

// ListAppointments godoc
// @Summary		Hämta bokningar från det kommunala bokningssystemet
// @Description	Returnerar alla bokningar i det kommunala bokningssystemet (Actor Smartbook) för en given period. Tiderna returneras i lokal tid (Europe/Stockholm).
// @Tags			actorsmartbook
// @Accept			json
// @Produce		json
// @Param			from	query		string	false	"Periodens början (RFC3339, YYYY-MM-DD eller 'now')"	default(now)
// @Param			to	query		string	false	"Periodens slut (RFC3339, YYYY-MM-DD eller 'now+7d')"	default(now+7d)
// @Success		200	{object}	AppointmentListResponse
// @Failure		400	{object}	ErrorResponse
// @Failure		502	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router			/actorsmartbook/appointments [get]
func ListAppointments(client *Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := client.Location()

		now := time.Now().In(loc)
		from, err := ParseTimeParam(c.Query("from"), loc, now)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid from: %v", err)})
			return
		}

		to, err := ParseTimeParam(c.Query("to"), loc, from.AddDate(0, 0, defaultWindowDays))
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid to: %v", err)})
			return
		}

		if !to.After(from) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid time range: to must be after from"})
			return
		}

		if to.Sub(from) > maxWindowDays*24*time.Hour {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: fmt.Sprintf("time range too large: maximum is %d days", maxWindowDays),
			})
			return
		}

		appointments, err := client.GetAppointments(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, AppointmentListResponse{
			From:         from,
			To:           to,
			Count:        len(appointments),
			Appointments: appointments,
		})
	}
}

// ParseTimeParam tolkar en tidsparameter. Tom sträng ger fallback.
// Utan tidszon tolkas värdet som lokal tid.
//
// Accepterade format:
//   - RFC3339, t.ex. "2026-11-02T18:00:00+01:00"
//   - "2026-11-02T18:00:00" (lokal tid)
//   - "2026-11-02" (lokal midnatt)
//   - "now", "today"
//   - relativa uttryck, t.ex. "now+7d", "today-1w", "now + 2h"
func ParseTimeParam(value string, loc *time.Location, fallback time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	if parsed, ok, err := parseRelativeTime(value, loc); ok {
		return parsed, err
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.In(loc), nil
	}

	if parsed, err := time.ParseInLocation("2006-01-02T15:04:05", value, loc); err == nil {
		return parsed, nil
	}

	if parsed, err := time.ParseInLocation("2006-01-02", value, loc); err == nil {
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("expected RFC3339, YYYY-MM-DD or a relative expression, got %q", value)
}

// relativeTimePattern matchar "now+7d", "today - 1w", "now + 2 hours" och liknande.
//
// Tecknet +/- är valfritt eftersom "+" i en query-sträng URL-avkodas till ett
// mellanslag ("now+7d" anländer som "now 7d"). Utan tecken räknas värdet som
// ett plus.
var relativeTimePattern = regexp.MustCompile(`^(now|today)\s*([+-])?\s*(\d+)\s*(minutes|minute|mins|min|m|hours|hour|h|days|day|d|weeks|week|w)$`)

// parseRelativeTime tolkar relativa tidsuttryck. Andra värdet betyder att
// uttrycket inte är relativt (ok=false).
func parseRelativeTime(value string, loc *time.Location) (time.Time, bool, error) {
	normalized := strings.ToLower(strings.ReplaceAll(value, " ", ""))
	matches := relativeTimePattern.FindStringSubmatch(normalized)
	if matches == nil {
		switch normalized {
		case "now":
			return time.Now().In(loc), true, nil
		case "today":
			now := time.Now().In(loc)
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), true, nil
		}
		return time.Time{}, false, nil
	}

	base := time.Now().In(loc)
	if matches[1] == "today" {
		base = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, loc)
	}

	amount, err := strconv.Atoi(matches[3])
	if err != nil {
		return time.Time{}, true, fmt.Errorf("invalid relative time %q: %w", value, err)
	}
	if matches[2] == "-" {
		amount = -amount
	}
	var unit time.Duration
	switch matches[4] {
	case "m", "min", "mins", "minute", "minutes":
		unit = time.Minute
	case "h", "hour", "hours":
		unit = time.Hour
	case "d", "day", "days":
		unit = 24 * time.Hour
	case "w", "week", "weeks":
		unit = 7 * 24 * time.Hour
	}

	// Dagar och veckor läggs till med AddDate så att väggklockan behålls
	// över DST-övergångar.
	if unit == 24*time.Hour {
		return base.AddDate(0, 0, amount), true, nil
	}
	if unit == 7*24*time.Hour {
		return base.AddDate(0, 0, amount*7), true, nil
	}

	return base.Add(time.Duration(amount) * unit), true, nil
}
