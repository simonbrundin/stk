package sync

import (
	"context"
	"testing"
	"time"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/store"
)

func TestLinkBooksAndStoresConnection(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, start, end, "Sundsvalls Tennisklubb"),
	}}
	matchiClient := &fakeMatchi{}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	result, err := syncer.Link(context.Background(), LinkRequest{
		AppointmentID: 1,
		Start:         start,
		End:           end,
	})
	if err != nil {
		t.Fatalf("Link: %v", err)
	}

	if len(matchiClient.booked) != 1 {
		t.Fatalf("bokat %d gånger, vill ha 1", len(matchiClient.booked))
	}

	if matchiClient.booked[0].CourtName != "1. Idrottshall" {
		t.Errorf("court = %q, vill ha %q", matchiClient.booked[0].CourtName, "1. Idrottshall")
	}

	if result.MatchiBookingID == "" {
		t.Error("boknings-id saknas i svaret")
	}

	booking, err := db.FindBooking(context.Background(), 1, 11344)
	if err != nil || booking == nil {
		t.Fatalf("kopplingen sparades inte: %v", err)
	}

	if booking.MatchiSlotID != "slot-1" {
		t.Errorf("slot-id = %q, vill ha %q", booking.MatchiSlotID, "slot-1")
	}
}

func TestLinkNormalizesTimes(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 5, 0, 0, loc)
	end := time.Date(2026, 11, 6, 12, 50, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, start, end, "Sundsvalls Tennisklubb"),
	}}
	matchiClient := &fakeMatchi{}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	if _, err := syncer.Link(context.Background(), LinkRequest{
		AppointmentID: 1,
		Start:         start,
		End:           end,
	}); err != nil {
		t.Fatalf("Link: %v", err)
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

func TestLinkRejectsUnknownAppointment(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 628, start, end, "Sundsvalls Tennisklubb"),
	}}
	matchiClient := &fakeMatchi{}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	if _, err := syncer.Link(context.Background(), LinkRequest{
		AppointmentID: 99,
		Start:         start,
		End:           end,
	}); err == nil {
		t.Fatal("okänd bokning ska ge ett fel")
	}

	if len(matchiClient.booked) != 0 {
		t.Fatal("okända bokningar ska inte bokas i MATCHi")
	}
}

func TestLinkRejectsUnmappedResource(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 11, 6, 11, 0, 0, 0, loc)
	end := time.Date(2026, 11, 6, 13, 0, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{
		testAppointment(1, 999, start, end, "Baldershallen Coworking-rum"),
	}}
	matchiClient := &fakeMatchi{}
	db := newFakeStore([]store.CourtMapping{testMapping()})

	syncer := New(kommun, matchiClient, db, Config{Days: 21, CustomerID: 1660773}, loc)

	if _, err := syncer.Link(context.Background(), LinkRequest{
		AppointmentID: 1,
		Start:         start,
		End:           end,
	}); err == nil {
		t.Fatal("resurser utan koppling ska ge ett fel")
	}

	if len(matchiClient.booked) != 0 {
		t.Fatal("resurser utan koppling ska inte bokas i MATCHi")
	}
}
