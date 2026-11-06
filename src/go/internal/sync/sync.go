// Package sync speglar bokningar från det kommunala bokningssystemet till
// MATCHi.
//
// Synkroniseringen hämtar ett antal dagar framåt i det kommunala
// bokningssystemet, jämför varje bokning mot det som redan finns i
// databasen och bokar, flyttar eller låter bli i MATCHi beroende på vad som
// har ändrats.
package sync

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/matchi"
	"github.com/simon/repo-template/internal/store"
)

// DefaultDays är hur många dagar som synkroniseras som standard.
const DefaultDays = 21

// DefaultLocationName är tidszonen anläggningarna ligger i.
const DefaultLocationName = "Europe/Stockholm"

// KommunClient hämtar bokningar från det kommunala bokningssystemet.
type KommunClient interface {
	GetAppointments(ctx context.Context, from, to time.Time) ([]actorsmartbook.Appointment, error)
}

// MatchiClient gör bokningar och flyttningar i MATCHi.
type MatchiClient interface {
	Schedule(ctx context.Context, date time.Time) (*matchi.Schedule, error)
	Book(ctx context.Context, req matchi.BookRequest) (*matchi.BookingResult, error)
	Move(ctx context.Context, req matchi.MoveRequest) (*matchi.BookingResult, error)
	Cancel(ctx context.Context, slotIDs []string) error
}

// Store lagrar kopplingen mellan de båda bokningssystemen.
type Store interface {
	ListCourtMappings(ctx context.Context) ([]store.CourtMapping, error)
	UpsertCourtMapping(ctx context.Context, mapping store.CourtMapping) error
	FindBooking(ctx context.Context, kommunAppointmentID, matchiCourtID int64) (*store.Booking, error)
	UpsertBooking(ctx context.Context, booking store.Booking) error
	DeleteBooking(ctx context.Context, kommunAppointmentID, matchiCourtID int64) error
	ListBookingsInRange(ctx context.Context, from, to time.Time) ([]store.Booking, error)
}

// Config styr hur synkroniseringen ska göras.
type Config struct {
	// Days är hur många dagar framåt som synkroniseras.
	Days int

	// CustomerID är MATCHis kund-id som bokningarna görs för.
	CustomerID int64

	// OrganizationFilter är ett reguljärt uttryck som filtrerar vilka
	// bokningar i det kommunala systemet som ska speglas. Tomt filter
	// innebär att alla bokningar synkroniseras.
	OrganizationFilter string

	// RemoveOrphans avgör om bokningar som försvunnit från det kommunala
	// systemet ska avbokas i MATCHi. Standard är att lämna dem ifred,
	// eftersom en bokning kan döljas i det kommunala systemet.
	RemoveOrphans bool
}

// Syncer speglar bokningar från det kommunala systemet till MATCHi.
type Syncer struct {
	kommun       KommunClient
	matchiClient MatchiClient
	store        Store
	config       Config
	loc          *time.Location
}

// DefaultLocation är tidszonen anläggningarna ligger i.
func DefaultLocation() *time.Location {
	loc, err := time.LoadLocation(DefaultLocationName)
	if err != nil {
		return time.UTC
	}

	return loc
}

// New skapar en synkroniserare.
func New(kommun KommunClient, matchiClient MatchiClient, store Store, config Config, loc *time.Location) *Syncer {
	if config.Days <= 0 {
		config.Days = DefaultDays
	}
	if loc == nil {
		loc = DefaultLocation()
	}

	return &Syncer{
		kommun:       kommun,
		matchiClient: matchiClient,
		store:        store,
		config:       config,
		loc:          loc,
	}
}

// Report sammanfattar en synkronisering.
type Report struct {
	// @Description	Periodens början i lokal tid
	// @Example		2026-11-06T00:00:00+01:00
	From time.Time `json:"from"`

	// @Description	Periodens slut i lokal tid
	// @Example		2026-11-27T00:00:00+01:00
	To time.Time `json:"to"`

	// @Description	Antal bokningar som skapades i MATCHi
	// @Example		3
	Created int `json:"created"`

	// @Description	Antal bokningar som flyttades i MATCHi
	// @Example		1
	Moved int `json:"moved"`

	// @Description	Antal bokningar som var oförändrade
	// @Example		42
	Unchanged int `json:"unchanged"`

	// @Description	Antal bokningar i det kommunala systemet som hoppades över
	// @Example		5
	Skipped int `json:"skipped"`

	// @Description	Antal bokningar i databasen som inte längre finns i det kommunala systemet
	// @Example		0
	Orphans int `json:"orphans"`

	// @Description	Fel som uppstod under synkroniseringen
	Errors []string `json:"errors,omitempty"`
}

// Run synkroniserar det kommunala bokningssystemet mot MATCHi.
func (s *Syncer) Run(ctx context.Context) (*Report, error) {
	mappings, err := s.store.ListCourtMappings(ctx)
	if err != nil {
		return nil, err
	}

	if len(mappings) == 0 {
		return nil, fmt.Errorf("sync: ingen koppling mellan kommunresurser och MATCHi-banor finns")
	}

	now := time.Now().In(s.loc)
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.loc)
	to := from.AddDate(0, 0, s.config.Days)

	appointments, err := s.kommun.GetAppointments(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("sync: hämta bokningar: %w", err)
	}

	report := &Report{From: from, To: to}
	filter, err := compileFilter(s.config.OrganizationFilter)
	if err != nil {
		return nil, err
	}

	seen := map[int64]bool{}

	for _, appointment := range appointments {
		courts := mappedCourts(appointment, mappings)
		if len(courts) == 0 {
			report.Skipped++
			continue
		}

		if !filter.Matches(appointment) {
			report.Skipped++
			continue
		}

		seen[appointment.ID] = true

		start, end := matchi.NormalizeRange(appointment.Start, appointment.End)
		if !end.After(start) {
			report.Skipped++
			continue
		}

		for _, court := range courts {
			s.syncBooking(ctx, appointment, court, start, end, report)
		}
	}

	s.handleOrphans(ctx, seen, from, to, report)

	return report, nil
}

// syncBooking synkroniserar en enskild bokning mot MATCHi.
//
// Synkroniseringen är självåterställande: om bokningen har flyttats eller
// tagits bort direkt i MATCHi flyttas eller bokas den om, så att MATCHi
// alltid speglar det kommunala bokningssystemet.
func (s *Syncer) syncBooking(ctx context.Context, appointment actorsmartbook.Appointment, court store.CourtMapping, start, end time.Time, report *Report) {
	existing, err := s.store.FindBooking(ctx, appointment.ID, court.MatchiCourtID)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("kunde inte läsa bokning %d: %v", appointment.ID, err))
		return
	}

	if existing == nil {
		s.createBooking(ctx, appointment, court, start, end, report)
		return
	}

	schedule, err := s.matchiClient.Schedule(ctx, start)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("kunde inte läsa schemat i MATCHi: %v", err))
		return
	}

	booked := bookedSlots(schedule, court.MatchiCourtName)

	// Bokningen ligger redan rätt i MATCHi.
	for _, slot := range booked {
		if slot.Start.Equal(start) && slot.End.Equal(end) {
			if slot.SlotID != existing.MatchiSlotID || slot.BookingID != existing.MatchiBookingID {
				s.saveBooking(ctx, appointment, court, slot, start, end, report)
			}
			report.Unchanged++
			return
		}
	}

	// Bokningen finns i MATCHi men på en annan tid, och flyttas tillbaka.
	for _, slot := range booked {
		if existing.MatchiBookingID != "" && slot.BookingID == existing.MatchiBookingID {
			s.moveBooking(ctx, appointment, court, slot.SlotID, start, end, report)
			return
		}
	}

	// Bokningen finns inte kvar i MATCHi, så den bokas om.
	s.createBooking(ctx, appointment, court, start, end, report)
}

// createBooking bokar en tid i MATCHi och sparar bokningen i databasen.
func (s *Syncer) createBooking(ctx context.Context, appointment actorsmartbook.Appointment, court store.CourtMapping, start, end time.Time, report *Report) {
	result, err := s.matchiClient.Book(ctx, matchi.BookRequest{
		CourtName:  court.MatchiCourtName,
		Start:      start,
		End:        end,
		CustomerID: s.config.CustomerID,
		Comment:    commentFor(appointment),
	})
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("kunde inte boka %s: %v", court.MatchiCourtName, err))
		return
	}

	s.saveBooking(ctx, appointment, court, matchi.Slot{SlotID: result.SlotID, BookingID: result.BookingID}, start, end, report)

	report.Created++
}

// moveBooking flyttar en befintlig bokning i MATCHi.
func (s *Syncer) moveBooking(ctx context.Context, appointment actorsmartbook.Appointment, court store.CourtMapping, slotID string, start, end time.Time, report *Report) {
	result, err := s.matchiClient.Move(ctx, matchi.MoveRequest{
		SlotID:    slotID,
		Start:     start,
		End:       end,
		CourtName: court.MatchiCourtName,
	})
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("kunde inte flytta %s: %v", court.MatchiCourtName, err))
		return
	}

	s.saveBooking(ctx, appointment, court, matchi.Slot{SlotID: result.SlotID, BookingID: result.BookingID}, start, end, report)

	report.Moved++
}

// saveBooking sparar en bokning i databasen.
func (s *Syncer) saveBooking(ctx context.Context, appointment actorsmartbook.Appointment, court store.CourtMapping, slot matchi.Slot, start, end time.Time, report *Report) {
	if err := s.store.UpsertBooking(ctx, store.Booking{
		KommunAppointmentID: appointment.ID,
		KommunResourceID:    court.KommunResourceID,
		MatchiCourtID:       court.MatchiCourtID,
		MatchiBookingID:     slot.BookingID,
		MatchiSlotID:        slot.SlotID,
		StartTime:           start,
		EndTime:             end,
		Status:              "synced",
	}); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("kunde inte spara bokning %d: %v", appointment.ID, err))
	}
}

// bookedSlots returnerar de bokade slotterna på en bana.
func bookedSlots(schedule *matchi.Schedule, court string) []matchi.Slot {
	var booked []matchi.Slot

	for _, slot := range schedule.Slots {
		if !strings.EqualFold(slot.CourtName, court) {
			continue
		}
		if slot.Free {
			continue
		}
		booked = append(booked, slot)
	}

	return booked
}

// handleOrphans rapporterar bokningar i databasen som inte längre finns i
// det kommunala systemet, och avbokar dem om det är aktiverat.
func (s *Syncer) handleOrphans(ctx context.Context, seen map[int64]bool, from, to time.Time, report *Report) {
	bookings, err := s.store.ListBookingsInRange(ctx, from, to)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("kunde inte läsa befintliga bokningar: %v", err))
		return
	}

	for _, booking := range bookings {
		if seen[booking.KommunAppointmentID] {
			continue
		}

		report.Orphans++

		if !s.config.RemoveOrphans {
			continue
		}

		if booking.MatchiSlotID != "" {
			if err := s.matchiClient.Cancel(ctx, []string{booking.MatchiSlotID}); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("kunde inte avboka %s: %v", booking.MatchiSlotID, err))
				continue
			}
		}

		if err := s.store.DeleteBooking(ctx, booking.KommunAppointmentID, booking.MatchiCourtID); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("kunde inte ta bort bokning %d: %v", booking.KommunAppointmentID, err))
		}
	}
}

// mappedCourts returnerar de MATCHi-banor som en bokning i det kommunala
// systemet gäller.
//
// Det kommunala boknings-API:t levererar inte alltid resurs-id:t, utan bara
// resursens namn i bokningens ämne ("11:00 - 13:00 Baldershallen 1"). Därför
// matchas bokningen på namn när det finns, och på id som backup.
func mappedCourts(appointment actorsmartbook.Appointment, mappings []store.CourtMapping) []store.CourtMapping {
	var courts []store.CourtMapping

	for _, mapping := range mappings {
		if matchesResource(appointment, mapping) {
			courts = append(courts, mapping)
		}
	}

	return courts
}

// matchesResource avgör om en bokning gäller en given kommunresurs.
func matchesResource(appointment actorsmartbook.Appointment, mapping store.CourtMapping) bool {
	if name := strings.TrimSpace(appointment.Location); name != "" && mapping.KommunResourceName != "" {
		if strings.EqualFold(name, mapping.KommunResourceName) {
			return true
		}
	}

	for _, resource := range appointment.Resources {
		if resource.ID == mapping.KommunResourceID {
			return true
		}
	}

	return false
}

// commentFor bygger en kommentar till bokningen i MATCHi.
func commentFor(appointment actorsmartbook.Appointment) string {
	parts := []string{"Synkad från kommunens bokningssystem"}

	if appointment.Description != "" {
		parts = append(parts, appointment.Description)
	}

	if appointment.Location != "" {
		parts = append(parts, appointment.Location)
	}

	return strings.Join(parts, " - ")
}

// organizationFilter filtrerar bokningar på organisation.
type organizationFilter struct {
	pattern *regexp.Regexp
}

// compileFilter bygger ett filter av ett reguljärt uttryck.
func compileFilter(expression string) (*organizationFilter, error) {
	if strings.TrimSpace(expression) == "" {
		return &organizationFilter{}, nil
	}

	pattern, err := regexp.Compile(expression)
	if err != nil {
		return nil, fmt.Errorf("sync: ogiltigt organisationsfilter: %w", err)
	}

	return &organizationFilter{pattern: pattern}, nil
}

// Matches avgör om en bokning ska synkroniseras.
func (f *organizationFilter) Matches(appointment actorsmartbook.Appointment) bool {
	if f.pattern == nil {
		return true
	}

	haystack := strings.Join([]string{appointment.Description, appointment.Subject, appointment.Location}, " ")

	return f.pattern.MatchString(haystack)
}
