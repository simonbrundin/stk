package sync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/mapping"
	"github.com/simon/repo-template/internal/matchi"
	"github.com/simon/repo-template/internal/store"
)

// lookupWindow är hur stor marginal runt bokningens tid som läses från det
// kommunala bokningssystemet när en bokning ska kopplas ihop.
//
// Fönstret gör att API:t bara behöver hämta en liten del av perioden, samtidigt
// som det fångar bokningar som startar strax före eller slutar strax efter den
// angivna tiden.
const lookupWindow = 2 * time.Hour

// LinkRequest är en begäran om att boka en tid i MATCHi för en bokning i det
// kommunala bokningssystemet.
type LinkRequest struct {
	// @Description	Id:t för bokningen i det kommunala bokningssystemet
	// @Example		337667
	AppointmentID int64 `json:"appointment_id" binding:"required"`

	// @Description	Starttid för bokningen i det kommunala systemet
	// @Example		2026-11-02T18:00:00+01:00
	Start time.Time `json:"start" binding:"required"`

	// @Description	Sluttid för bokningen i det kommunala systemet
	// @Example		2026-11-02T19:30:00+01:00
	End time.Time `json:"end" binding:"required"`

	// @Description	MATCHis id för banan eller salen. Kan utelämnas om resursen
	// bara är kopplad till en enda bana.
	// @Example		11344
	MatchiCourtID int64 `json:"matchi_court_id,omitempty"`
}

// LinkResult beskriver den bokning som skapades i MATCHi.
type LinkResult struct {
	// @Description	Id:t för bokningen i det kommunala bokningssystemet
	// @Example		337667
	AppointmentID int64 `json:"appointment_id"`

	// @Description	Resursen i det kommunala bokningssystemet
	// @Example		Baldershallen 1
	Location string `json:"location,omitempty"`

	// @Description	Starttid i lokal tid (Europe/Stockholm)
	// @Example		2026-11-02T18:00:00+01:00
	Start time.Time `json:"start"`

	// @Description	Sluttid i lokal tid (Europe/Stockholm)
	// @Example		2026-11-02T19:30:00+01:00
	End time.Time `json:"end"`

	// @Description	MATCHi-banan eller salen som bokades
	// @Example		1. Idrottshall
	CourtName string `json:"court_name"`

	// @Description	MATCHis boknings-id
	// @Example		112291396
	MatchiBookingID string `json:"matchi_booking_id,omitempty"`

	// @Description	MATCHis slot-id för bokningen
	// @Example		4a66b2f0-3177-4c9c-a7a5-3a54e44a9737
	MatchiSlotID string `json:"matchi_slot_id,omitempty"`
}

// Link bokar en tid i MATCHi för en bokning i det kommunala systemet och
// sparar kopplingen mellan dem.
//
// Tiden avrundas till hela och halva timmar, precis som i den vanliga
// synkroniseringen, så att bokningen aldrig blir kortare än den kommunala
// bokningen.
func (s *Syncer) Link(ctx context.Context, req LinkRequest) (*LinkResult, error) {
	if req.AppointmentID == 0 {
		return nil, errors.New("sync: appointment_id krävs")
	}

	if req.Start.IsZero() || req.End.IsZero() {
		return nil, errors.New("sync: start och end krävs")
	}

	appointment, err := s.findAppointment(ctx, req)
	if err != nil {
		return nil, err
	}

	mappings, err := s.store.ListCourtMappings(ctx)
	if err != nil {
		return nil, err
	}

	court, err := pickCourt(appointment, mappings, req.MatchiCourtID)
	if err != nil {
		return nil, err
	}

	start, end := matchi.NormalizeRange(appointment.Start, appointment.End)
	if !end.After(start) {
		return nil, errors.New("sync: bokningen har ingen giltig tidslängd")
	}

	result, err := s.matchiClient.Book(ctx, matchi.BookRequest{
		CourtName:  court.MatchiCourtName,
		Start:      start,
		End:        end,
		CustomerID: s.config.CustomerID,
		Comment:    commentFor(appointment),
	})
	if err != nil {
		return nil, err
	}

	if err := s.store.UpsertBooking(ctx, store.Booking{
		KommunAppointmentID: appointment.ID,
		KommunResourceID:    court.KommunResourceID,
		MatchiCourtID:       court.MatchiCourtID,
		MatchiBookingID:     result.BookingID,
		MatchiSlotID:        result.SlotID,
		StartTime:           start,
		EndTime:             end,
		Status:              "synced",
	}); err != nil {
		return nil, err
	}

	return &LinkResult{
		AppointmentID:   appointment.ID,
		Location:        appointment.Location,
		Start:           start,
		End:             end,
		CourtName:       court.MatchiCourtName,
		MatchiBookingID: result.BookingID,
		MatchiSlotID:    result.SlotID,
	}, nil
}

// findAppointment läser upp bokningen i det kommunala systemet.
//
// Tiden kontrolleras mot det kommunala systemet, så att en klient inte kan
// boka en tid som inte finns i kommunens kalender.
func (s *Syncer) findAppointment(ctx context.Context, req LinkRequest) (actorsmartbook.Appointment, error) {
	from := req.Start.Add(-lookupWindow)
	to := req.End.Add(lookupWindow)

	appointments, err := s.kommun.GetAppointments(ctx, from, to)
	if err != nil {
		return actorsmartbook.Appointment{}, fmt.Errorf("sync: hämta bokningar: %w", err)
	}

	for _, appointment := range appointments {
		if appointment.ID == req.AppointmentID {
			return appointment, nil
		}
	}

	return actorsmartbook.Appointment{}, fmt.Errorf("sync: hittade inte bokningen %d i det kommunala bokningssystemet", req.AppointmentID)
}

// pickCourt väljer den MATCHi-bana som bokningen ska läggas på.
//
// Om resursen är kopplad till flera banor måste banan anges, annars vore det
// oklart vilken som bokas.
func pickCourt(appointment actorsmartbook.Appointment, mappings []store.CourtMapping, courtID int64) (store.CourtMapping, error) {
	courts := mapping.Courts(appointment, mappings)
	if len(courts) == 0 {
		return store.CourtMapping{}, fmt.Errorf("sync: resursen %q är inte kopplad till någon MATCHi-bana", appointment.Location)
	}

	if courtID == 0 {
		if len(courts) == 1 {
			return courts[0], nil
		}

		names := make([]string, 0, len(courts))
		for _, court := range courts {
			names = append(names, fmt.Sprintf("%s (id %d)", court.MatchiCourtName, court.MatchiCourtID))
		}

		return store.CourtMapping{}, fmt.Errorf("sync: resursen är kopplad till flera MATCHi-banor (%s) - ange vilken med matchi_court_id", joinNames(names))
	}

	for _, court := range courts {
		if court.MatchiCourtID == courtID {
			return court, nil
		}
	}

	return store.CourtMapping{}, fmt.Errorf("sync: banan %d är inte kopplad till resursen %q", courtID, appointment.Location)
}

// joinNames sätter ihop namn med komma.
func joinNames(names []string) string {
	result := ""

	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += name
	}

	return result
}

// LinkBooking godoc
// @Summary		Boka en tid i MATCHi för en kommunbokning
// @Description	Bokar en tid i MATCHi för en bokning i det kommunala bokningssystemet och sparar kopplingen mellan dem. Tiden avrundas till hela och halva timmar.
// @Tags		matchi
// @Accept		json
// @Produce	json
// @Param		body	body		LinkRequest	true	"Bokning"
// @Success		201		{object}	LinkResult
// @Failure		400		{object}	ErrorResponse
// @Failure		502		{object}	ErrorResponse
// @Router		/matchi/bookings/link [post]
func LinkBooking(syncer *Syncer, invalidateCache func()) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LinkRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		result, err := syncer.Link(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadGateway, ErrorResponse{Error: err.Error()})
			return
		}

		// Schemat från MATCHi är cachat, så cachen måste tömmas för att
		// den nya bokningen ska synas direkt.
		if invalidateCache != nil {
			invalidateCache()
		}

		c.JSON(http.StatusCreated, result)
	}
}
