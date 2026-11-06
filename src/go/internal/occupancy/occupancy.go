// Package occupancy jämför bokningarna i det kommunala bokningssystemet med
// bokningarna i MATCHi.
//
// Paketet är strikt skrivläsande: det bokar, flyttar och avbokar aldrig
// något i MATCHi. Syftet är att kunna visa vilka tider i det kommunala
// systemet som även är bokade i MATCHi.
package occupancy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/mapping"
	"github.com/simon/repo-template/internal/matchi"
	"github.com/simon/repo-template/internal/store"
)

// cacheTTL är hur länge ett schema från MATCHi återanvänds.
//
// Schemat är dyrt att läsa (en stor HTML-sida per dag), samtidigt som det
// inte ändras snabbt. En kort cache skyddar MATCHi från upprepade anrop
// när flera besökare läser samma period.
const cacheTTL = 90 * time.Second

// KommunClient hämtar bokningar från det kommunala bokningssystemet.
type KommunClient interface {
	GetAppointments(ctx context.Context, from, to time.Time) ([]actorsmartbook.Appointment, error)
}

// MatchiClient läser scheman från MATCHi.
type MatchiClient interface {
	ScheduleForDays(ctx context.Context, days []time.Time) []matchi.DailySchedule
}

// MappingStore läser kopplingen mellan kommunresurser och MATCHi-banor.
type MappingStore interface {
	ListCourtMappings(ctx context.Context) ([]store.CourtMapping, error)
}

// Service jämför det kommunala bokningssystemet med MATCHi.
type Service struct {
	kommun   KommunClient
	matchi   MatchiClient
	mappings MappingStore
	loc      *time.Location

	mu    sync.Mutex
	cache map[string]cachedSchedule
}

// cachedSchedule är ett schema som är läst från MATCHi.
type cachedSchedule struct {
	schedule  *matchi.Schedule
	fetchedAt time.Time
}

// New skapar en tjänst som jämför det kommunala systemet med MATCHi.
func New(kommun KommunClient, matchiClient MatchiClient, mappings MappingStore, loc *time.Location) *Service {
	if loc == nil {
		loc = time.UTC
	}

	return &Service{
		kommun:   kommun,
		matchi:   matchiClient,
		mappings: mappings,
		loc:      loc,
		cache:    map[string]cachedSchedule{},
	}
}

// Booking är en bokning i det kommunala systemet som även är bokad i MATCHi.
type Booking struct {
	// @Description	Id:t för bokningen i det kommunala bokningssystemet
	// @Example		337667
	AppointmentID int64 `json:"appointment_id"`

	// @Description	Starttid i lokal tid (Europe/Stockholm)
	// @Example		2026-11-02T18:00:00+01:00
	Start time.Time `json:"start"`

	// @Description	Sluttid i lokal tid (Europe/Stockholm)
	// @Example		2026-11-02T19:30:00+01:00
	End time.Time `json:"end"`

	// @Description	Resursen i det kommunala bokningssystemet
	// @Example		Baldershallen 1
	Location string `json:"location,omitempty"`

	// @Description	MATCHi-banan eller salen som bokningen gäller
	// @Example		1. Idrottshall
	CourtName string `json:"court_name"`

	// @Description	Starttid för bokningen i MATCHi
	// @Example		2026-11-02T18:00:00+01:00
	MatchiStart time.Time `json:"matchi_start"`

	// @Description	Sluttid för bokningen i MATCHi
	// @Example		2026-11-02T19:30:00+01:00
	MatchiEnd time.Time `json:"matchi_end"`

	// @Description	Information om bokningen i MATCHi, t.ex. kundnamn
	// @Example		STK Verksamhet
	MatchiTitle string `json:"matchi_title,omitempty"`

	// @Description	MATCHis boknings-id
	// @Example		112291396
	MatchiBookingID string `json:"matchi_booking_id,omitempty"`
}

// Result är resultatet av en jämförelse.
type Result struct {
	// @Description	Periodens början i lokal tid
	// @Example		2026-11-02T00:00:00+01:00
	From time.Time `json:"from"`

	// @Description	Periodens slut i lokal tid
	// @Example		2026-11-09T00:00:00+01:00
	To time.Time `json:"to"`

	// @Description	Antal bokningar som även är bokade i MATCHi
	// @Example		3
	Count int `json:"count"`

	// @Description	Bokningarna som även är bokade i MATCHi
	Booked []Booking `json:"booked"`

	// @Description	Meddelanden om dagar som inte kunde läsas från MATCHi
	// @Example		["2026-11-03: kunde inte läsa schemat"]
	Errors []string `json:"errors,omitempty"`
}

// Booked returnerar de bokningar i det kommunala systemet som även är
// bokade i MATCHi.
//
// Endast dagar som har bokningar läses från MATCHi, vilket håller antalet
// anrop nere.
func (s *Service) Booked(ctx context.Context, from, to time.Time) (Result, error) {
	result := Result{From: from, To: to, Booked: []Booking{}}

	mappings, err := s.mappings.ListCourtMappings(ctx)
	if err != nil {
		return result, err
	}

	appointments, err := s.kommun.GetAppointments(ctx, from, to)
	if err != nil {
		return result, err
	}

	days := bookingDays(appointments, s.loc)
	if len(days) == 0 {
		return result, nil
	}

	schedules, errs := s.schedules(ctx, days)
	result.Errors = errs

	for _, appointment := range appointments {
		courts := mapping.Courts(appointment, mappings)

		for _, court := range courts {
			booking, ok := s.findBooking(appointment, court, schedules)
			if !ok {
				continue
			}

			result.Booked = append(result.Booked, booking)
		}
	}

	result.Count = len(result.Booked)

	return result, nil
}

// findBooking letar upp en MATCHi-bokning som överlappar en kommunbokning.
func (s *Service) findBooking(appointment actorsmartbook.Appointment, court store.CourtMapping, schedules map[string]*matchi.Schedule) (Booking, bool) {
	schedule, ok := schedules[s.dayKey(appointment.Start)]
	if !ok || schedule == nil {
		return Booking{}, false
	}

	var (
		matched    bool
		matchStart time.Time
		matchEnd   time.Time
		title      string
		bookingID  string
	)

	for _, slot := range schedule.Slots {
		if slot.Free || !strings.EqualFold(slot.CourtName, court.MatchiCourtName) {
			continue
		}

		if !overlaps(slot.Start, slot.End, appointment.Start, appointment.End) {
			continue
		}

		if !matched {
			matchStart = slot.Start
			matchEnd = slot.End
			title = slot.Title
			bookingID = slot.BookingID
			matched = true
			continue
		}

		// En bokning som spänner flera halvtimmeslopper ska redovisas
		// som en enda träff.
		if slot.Start.Before(matchStart) {
			matchStart = slot.Start
		}
		if slot.End.After(matchEnd) {
			matchEnd = slot.End
		}
		if title == "" {
			title = slot.Title
		}
		if bookingID == "" {
			bookingID = slot.BookingID
		}
	}

	if !matched {
		return Booking{}, false
	}

	return Booking{
		AppointmentID:   appointment.ID,
		Start:           appointment.Start,
		End:             appointment.End,
		Location:        appointment.Location,
		CourtName:       court.MatchiCourtName,
		MatchiStart:     matchStart,
		MatchiEnd:       matchEnd,
		MatchiTitle:     title,
		MatchiBookingID: bookingID,
	}, true
}

// schedules läser schemat för de angivna dagarna, med cache.
func (s *Service) schedules(ctx context.Context, days []time.Time) (map[string]*matchi.Schedule, []string) {
	schedules := map[string]*matchi.Schedule{}
	var errs []string

	missing := make([]time.Time, 0, len(days))
	for _, day := range days {
		key := s.dayKey(day)

		if schedule, ok := s.cached(key); ok {
			schedules[key] = schedule
			continue
		}

		missing = append(missing, day)
	}

	if len(missing) == 0 {
		return schedules, errs
	}

	for _, daily := range s.matchi.ScheduleForDays(ctx, missing) {
		key := s.dayKey(daily.Date)

		if daily.Err != nil {
			errs = append(errs, fmt.Sprintf("%s: kunde inte läsa schemat i MATCHi: %v", key, daily.Err))
			continue
		}

		s.cacheSchedule(key, daily.Schedule)
		schedules[key] = daily.Schedule
	}

	return schedules, errs
}

// cached returnerar ett cachat schema om det fortfarande är färskt.
func (s *Service) cached(key string) (*matchi.Schedule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.cache[key]
	if !ok {
		return nil, false
	}

	if time.Since(entry.fetchedAt) > cacheTTL {
		return nil, false
	}

	return entry.schedule, true
}

// cacheSchedule sparar ett schema i cachen.
func (s *Service) cacheSchedule(key string, schedule *matchi.Schedule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[key] = cachedSchedule{schedule: schedule, fetchedAt: time.Now()}
}

// dayKey ger en dag en nyckel i lokal tid.
func (s *Service) dayKey(t time.Time) string {
	return t.In(s.loc).Format("2006-01-02")
}

// bookingDays returnerar de unika dagar som bokningarna täcker.
func bookingDays(appointments []actorsmartbook.Appointment, loc *time.Location) []time.Time {
	seen := map[string]bool{}
	var days []time.Time

	for _, appointment := range appointments {
		local := appointment.Start.In(loc)
		key := local.Format("2006-01-02")

		if seen[key] {
			continue
		}

		seen[key] = true
		days = append(days, time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc))
	}

	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })

	return days
}

// overlaps avgör om två tidsintervall överlappar varandra.
func overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}
