package sync

import (
	"context"
	"testing"
	"time"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/matchi"
	"github.com/simon/repo-template/internal/store"
)

// fakeKommun är en stub för det kommunala bokningssystemet.
type fakeKommun struct {
	appointments []actorsmartbook.Appointment
}

func (f *fakeKommun) GetAppointments(context.Context, time.Time, time.Time) ([]actorsmartbook.Appointment, error) {
	return f.appointments, nil
}

// fakeMatchi är en stub för MATCHi.
type fakeMatchi struct {
	booked []matchi.BookRequest
	moved  []matchi.MoveRequest
	cancel []string
	slots  []matchi.Slot
}

func (f *fakeMatchi) Schedule(context.Context, time.Time) (*matchi.Schedule, error) {
	return &matchi.Schedule{Slots: f.slots}, nil
}

func (f *fakeMatchi) Book(_ context.Context, req matchi.BookRequest) (*matchi.BookingResult, error) {
	f.booked = append(f.booked, req)

	slot := matchi.Slot{
		CourtName: req.CourtName,
		SlotID:    "slot-1",
		BookingID: "999",
		Start:     req.Start,
		End:       req.End,
	}
	f.slots = append(f.slots, slot)

	return &matchi.BookingResult{
		SlotID:    slot.SlotID,
		SlotIDs:   []string{slot.SlotID},
		BookingID: slot.BookingID,
		Start:     req.Start,
		End:       req.End,
		CourtName: req.CourtName,
	}, nil
}

func (f *fakeMatchi) Move(_ context.Context, req matchi.MoveRequest) (*matchi.BookingResult, error) {
	f.moved = append(f.moved, req)

	for i := range f.slots {
		if f.slots[i].SlotID != req.SlotID {
			continue
		}
		f.slots[i].SlotID = "slot-2"
		f.slots[i].Start = req.Start
		f.slots[i].End = req.End
	}

	return &matchi.BookingResult{
		SlotID:    "slot-2",
		SlotIDs:   []string{"slot-2"},
		BookingID: "999",
		Start:     req.Start,
		End:       req.End,
	}, nil
}

func (f *fakeMatchi) Cancel(_ context.Context, slotIDs []string) error {
	f.cancel = append(f.cancel, slotIDs...)

	return nil
}

// fakeStore är en minnesbaserad store.
type fakeStore struct {
	mappings []store.CourtMapping
	bookings map[[2]int64]store.Booking
}

func newFakeStore(mappings []store.CourtMapping) *fakeStore {
	return &fakeStore{mappings: mappings, bookings: map[[2]int64]store.Booking{}}
}

func (f *fakeStore) ListCourtMappings(context.Context) ([]store.CourtMapping, error) {
	return f.mappings, nil
}

func (f *fakeStore) UpsertCourtMapping(_ context.Context, mapping store.CourtMapping) error {
	f.mappings = append(f.mappings, mapping)

	return nil
}

func (f *fakeStore) FindBooking(_ context.Context, kommunAppointmentID, matchiCourtID int64) (*store.Booking, error) {
	booking, ok := f.bookings[[2]int64{kommunAppointmentID, matchiCourtID}]
	if !ok {
		return nil, nil
	}

	return &booking, nil
}

func (f *fakeStore) UpsertBooking(_ context.Context, booking store.Booking) error {
	f.bookings[[2]int64{booking.KommunAppointmentID, booking.MatchiCourtID}] = booking

	return nil
}

func (f *fakeStore) DeleteBooking(_ context.Context, kommunAppointmentID, matchiCourtID int64) error {
	delete(f.bookings, [2]int64{kommunAppointmentID, matchiCourtID})

	return nil
}

func (f *fakeStore) ListBookingsInRange(_ context.Context, from, to time.Time) ([]store.Booking, error) {
	bookings := []store.Booking{}
	for _, booking := range f.bookings {
		if booking.StartTime.Before(to) && booking.EndTime.After(from) {
			bookings = append(bookings, booking)
		}
	}

	return bookings, nil
}

func testMapping() store.CourtMapping {
	return store.CourtMapping{
		KommunResourceID:   628,
		KommunResourceName: "Baldershallen 1",
		MatchiCourtID:      11344,
		MatchiCourtName:    "1. Idrottshall",
	}
}

func testAppointment(id int64, resourceID int64, start, end time.Time, description string) actorsmartbook.Appointment {
	// Det kommunala API:t levererar resursen i ämnet, inte i Resources.
	name := "Baldershallen 1"
	if resourceID != 628 {
		name = "Baldershallen 5"
	}

	return actorsmartbook.Appointment{
		ID:          id,
		Description: description,
		Subject:     "11:00 - 13:00 " + name,
		Location:    name,
		Start:       start,
		End:         end,
	}
}

func TestSyncCreatesBookings(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, start, end, "Sundsvalls Tennisklubb"),
	}}
	matchiClient := &fakeMatchi{}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	report, err := syncer.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if report.Created != 1 {
		t.Fatalf("created = %d, vill ha 1", report.Created)
	}

	if len(matchiClient.booked) != 1 {
		t.Fatalf("bokat %d gånger, vill ha 1", len(matchiClient.booked))
	}

	if matchiClient.booked[0].CourtName != "1. Idrottshall" {
		t.Errorf("court = %q", matchiClient.booked[0].CourtName)
	}

	booking, err := db.FindBooking(context.Background(), 1, 11344)
	if err != nil || booking == nil {
		t.Fatalf("bookingen sparades inte: %v", err)
	}

	if !booking.StartTime.Equal(start) {
		t.Errorf("start = %s, vill ha %s", booking.StartTime, start)
	}
}

func TestSyncMovesChangedBookings(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)
	newStart := time.Date(2026, 11, 6, 12, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, newStart, end, "Sundsvalls Tennisklubb"),
	}}
	matchiClient := &fakeMatchi{slots: []matchi.Slot{
		{
			CourtName: "1. Idrottshall",
			SlotID:    "slot-1",
			BookingID: "999",
			Start:     start,
			End:       end,
		},
	}}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	if err := db.UpsertBooking(context.Background(), store.Booking{
		KommunAppointmentID: 1,
		KommunResourceID:    628,
		MatchiCourtID:       11344,
		MatchiSlotID:        "slot-1",
		MatchiBookingID:     "999",
		StartTime:           start,
		EndTime:             end,
	}); err != nil {
		t.Fatalf("UpsertBooking: %v", err)
	}

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	report, err := syncer.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if report.Moved != 1 {
		t.Fatalf("moved = %d, vill ha 1", report.Moved)
	}

	if len(matchiClient.moved) != 1 {
		t.Fatalf("flyttade %d gånger, vill ha 1", len(matchiClient.moved))
	}

	if !matchiClient.moved[0].Start.Equal(newStart) {
		t.Errorf("start = %s, vill ha %s", matchiClient.moved[0].Start, newStart)
	}
}

func TestSyncLeavesUnchangedBookings(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, start, end, "Sundsvalls Tennisklubb"),
	}}
	matchiClient := &fakeMatchi{slots: []matchi.Slot{
		{
			CourtName: "1. Idrottshall",
			SlotID:    "slot-1",
			BookingID: "999",
			Start:     start,
			End:       end,
		},
	}}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	if err := db.UpsertBooking(context.Background(), store.Booking{
		KommunAppointmentID: 1,
		KommunResourceID:    628,
		MatchiCourtID:       11344,
		MatchiSlotID:        "slot-1",
		MatchiBookingID:     "999",
		StartTime:           start,
		EndTime:             end,
	}); err != nil {
		t.Fatalf("UpsertBooking: %v", err)
	}

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	report, err := syncer.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if report.Unchanged != 1 {
		t.Fatalf("unchanged = %d, vill ha 1", report.Unchanged)
	}

	if len(matchiClient.booked) != 0 || len(matchiClient.moved) != 0 {
		t.Fatal("oförändrade bokningar ska inte ändras i MATCHi")
	}
}

func TestSyncMovesBackManuallyMovedBookings(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)
	movedStart := time.Date(2026, 11, 6, 14, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, start, end, "Sundsvalls Tennisklubb"),
	}}

	// Bokningen har flyttats direkt i MATCHi och ligger därför inte på den
	// tid som det kommunala systemet anger.
	matchiClient := &fakeMatchi{slots: []matchi.Slot{
		{
			CourtName: "1. Idrottshall",
			SlotID:    "slot-9",
			BookingID: "999",
			Start:     movedStart,
			End:       movedStart.Add(2 * time.Hour),
		},
	}}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	if err := db.UpsertBooking(context.Background(), store.Booking{
		KommunAppointmentID: 1,
		KommunResourceID:    628,
		MatchiCourtID:       11344,
		MatchiSlotID:        "slot-9",
		MatchiBookingID:     "999",
		StartTime:           start,
		EndTime:             end,
	}); err != nil {
		t.Fatalf("UpsertBooking: %v", err)
	}

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	report, err := syncer.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if report.Moved != 1 {
		t.Fatalf("moved = %d, vill ha 1", report.Moved)
	}

	if len(matchiClient.moved) != 1 {
		t.Fatalf("flyttade %d gånger, vill ha 1", len(matchiClient.moved))
	}

	if !matchiClient.moved[0].Start.Equal(start) {
		t.Errorf("start = %s, vill ha %s", matchiClient.moved[0].Start, start)
	}
}

func TestSyncSkipsUnmappedResources(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 999, start, end, "Någon annan"),
	}}
	matchiClient := &fakeMatchi{}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	report, err := syncer.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if report.Skipped != 1 {
		t.Fatalf("skipped = %d, vill ha 1", report.Skipped)
	}

	if len(matchiClient.booked) != 0 {
		t.Fatal("ej mappade resurser ska inte bokas i MATCHi")
	}
}

func TestSyncAppliesOrganizationFilter(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, start, end, "Ortvikens Pingisklubb"),
	}}
	matchiClient := &fakeMatchi{}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	syncer := New(kommun, matchiClient, db, Config{
		Days:               21,
		CustomerID:         1660773,
		OrganizationFilter: "Tennisklubb",
	}, loc)

	report, err := syncer.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if report.Skipped != 1 {
		t.Fatalf("skipped = %d, vill ha 1", report.Skipped)
	}

	if len(matchiClient.booked) != 0 {
		t.Fatal("bokningar som filtreras bort ska inte bokas i MATCHi")
	}
}

func TestSyncNormalizesTimes(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 5, 0, 0, loc)
	end := time.Date(2026, 11, 6, 12, 50, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, start, end, "Sundsvalls Tennisklubb"),
	}}
	matchiClient := &fakeMatchi{}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	if _, err := syncer.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(matchiClient.booked) != 1 {
		t.Fatalf("bokat %d gånger, vill ha 1", len(matchiClient.booked))
	}

	booked := matchiClient.booked[0]

	if want := time.Date(2026, 11, 6, 11, 0, 0, 0, loc); !booked.Start.Equal(want) {
		t.Errorf("start = %s, vill ha %s", booked.Start, want)
	}

	if want := time.Date(2026, 11, 6, 13, 0, 0, 0, loc); !booked.End.Equal(want) {
		t.Errorf("slut = %s, vill ha %s", booked.End, want)
	}
}
