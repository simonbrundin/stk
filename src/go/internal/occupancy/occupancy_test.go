package occupancy

import (
	"context"
	"testing"
	"time"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/matchi"
	"github.com/simon/repo-template/internal/store"
)

func stockholm(t *testing.T) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	return loc
}

// fakeKommun returnerar bokningar ur en lista.
type fakeKommun struct {
	appointments []actorsmartbook.Appointment
	err          error
}

func (f *fakeKommun) GetAppointments(_ context.Context, _, _ time.Time) ([]actorsmartbook.Appointment, error) {
	return f.appointments, f.err
}

// fakeMatchi returnerar scheman ur en lista.
type fakeMatchi struct {
	schedules []matchi.DailySchedule
	calls     int
}

func (f *fakeMatchi) ScheduleForDays(_ context.Context, days []time.Time) []matchi.DailySchedule {
	f.calls++

	result := make([]matchi.DailySchedule, 0, len(days))
	for _, day := range days {
		for _, schedule := range f.schedules {
			if schedule.Date.Format("2006-01-02") == day.Format("2006-01-02") {
				result = append(result, schedule)
			}
		}
	}

	return result
}

// fakeMappings returnerar kopplingar mellan kommunresurser och MATCHi-banor.
type fakeMappings struct {
	mappings []store.CourtMapping
}

func (f *fakeMappings) ListCourtMappings(_ context.Context) ([]store.CourtMapping, error) {
	return f.mappings, nil
}

func TestBookedFindsOverlappingMatchiBooking(t *testing.T) {
	loc := stockholm(t)

	start := time.Date(2026, 11, 2, 18, 0, 0, 0, loc)
	end := time.Date(2026, 11, 2, 19, 30, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{{
		ID:       337667,
		Location: "Baldershallen 1",
		Start:    start,
		End:      end,
	}}}

	matchiClient := &fakeMatchi{schedules: []matchi.DailySchedule{{
		Date: start,
		Schedule: &matchi.Schedule{Slots: []matchi.Slot{
			{CourtName: "1. Idrottshall", Start: start, End: start.Add(90 * time.Minute), Free: false, Title: "STK"},
		}},
	}}}

	mappings := &fakeMappings{mappings: []store.CourtMapping{{
		KommunResourceID:   628,
		KommunResourceName: "Baldershallen 1",
		MatchiCourtID:      11344,
		MatchiCourtName:    "1. Idrottshall",
	}}}

	service := New(kommun, matchiClient, mappings, loc)

	result, err := service.Booked(context.Background(), start, end)
	if err != nil {
		t.Fatalf("Booked: %v", err)
	}

	if result.Count != 1 {
		t.Fatalf("Count = %d, vill ha 1", result.Count)
	}

	booking := result.Booked[0]
	if booking.AppointmentID != 337667 {
		t.Errorf("AppointmentID = %d, vill ha 337667", booking.AppointmentID)
	}
	if booking.CourtName != "1. Idrottshall" {
		t.Errorf("CourtName = %q, vill ha %q", booking.CourtName, "1. Idrottshall")
	}
	if !booking.MatchiStart.Equal(start) || !booking.MatchiEnd.Equal(end) {
		t.Errorf("MATCHi-intervallet = %s-%s, vill ha %s-%s", booking.MatchiStart, booking.MatchiEnd, start, end)
	}
}

func TestBookedIgnoresFreeSlots(t *testing.T) {
	loc := stockholm(t)

	start := time.Date(2026, 11, 2, 18, 0, 0, 0, loc)
	end := time.Date(2026, 11, 2, 19, 30, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{{
		ID:       1,
		Location: "Baldershallen 1",
		Start:    start,
		End:      end,
	}}}

	matchiClient := &fakeMatchi{schedules: []matchi.DailySchedule{{
		Date: start,
		Schedule: &matchi.Schedule{Slots: []matchi.Slot{
			{CourtName: "1. Idrottshall", Start: start, End: end, Free: true},
		}},
	}}}

	mappings := &fakeMappings{mappings: []store.CourtMapping{{
		KommunResourceName: "Baldershallen 1",
		MatchiCourtName:    "1. Idrottshall",
	}}}

	service := New(kommun, matchiClient, mappings, loc)

	result, err := service.Booked(context.Background(), start, end)
	if err != nil {
		t.Fatalf("Booked: %v", err)
	}

	if result.Count != 0 {
		t.Fatalf("Count = %d, vill ha 0", result.Count)
	}
}

func TestBookedIgnoresUnmappedResources(t *testing.T) {
	loc := stockholm(t)

	start := time.Date(2026, 11, 2, 18, 0, 0, 0, loc)
	end := time.Date(2026, 11, 2, 19, 30, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{{
		ID:       2,
		Location: "Baldershallen 1",
		Start:    start,
		End:      end,
	}}}

	matchiClient := &fakeMatchi{schedules: []matchi.DailySchedule{{
		Date: start,
		Schedule: &matchi.Schedule{Slots: []matchi.Slot{
			{CourtName: "3. Idrottshall", Start: start, End: end, Free: false},
		}},
	}}}

	mappings := &fakeMappings{mappings: []store.CourtMapping{{
		KommunResourceName: "Baldershallen 1",
		MatchiCourtName:    "1. Idrottshall",
	}}}

	service := New(kommun, matchiClient, mappings, loc)

	result, err := service.Booked(context.Background(), start, end)
	if err != nil {
		t.Fatalf("Booked: %v", err)
	}

	if result.Count != 0 {
		t.Fatalf("Count = %d, vill ha 0", result.Count)
	}
}

func TestBookedGroupsSlotsIntoOneBooking(t *testing.T) {
	loc := stockholm(t)

	start := time.Date(2026, 11, 2, 18, 0, 0, 0, loc)
	end := time.Date(2026, 11, 2, 19, 30, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{{
		ID:       3,
		Location: "Baldershallen 1",
		Start:    start,
		End:      end,
	}}}

	matchiClient := &fakeMatchi{schedules: []matchi.DailySchedule{{
		Date: start,
		Schedule: &matchi.Schedule{Slots: []matchi.Slot{
			{CourtName: "1. Idrottshall", Start: start, End: start.Add(30 * time.Minute), Free: false, Title: "STK"},
			{CourtName: "1. Idrottshall", Start: start.Add(30 * time.Minute), End: start.Add(60 * time.Minute), Free: false, Title: "STK"},
			{CourtName: "1. Idrottshall", Start: start.Add(60 * time.Minute), End: end, Free: false, Title: "STK"},
		}},
	}}}

	mappings := &fakeMappings{mappings: []store.CourtMapping{{
		KommunResourceName: "Baldershallen 1",
		MatchiCourtName:    "1. Idrottshall",
	}}}

	service := New(kommun, matchiClient, mappings, loc)

	result, err := service.Booked(context.Background(), start, end)
	if err != nil {
		t.Fatalf("Booked: %v", err)
	}

	if result.Count != 1 {
		t.Fatalf("Count = %d, vill ha 1", result.Count)
	}

	if !result.Booked[0].MatchiEnd.Equal(end) {
		t.Errorf("MatchiEnd = %s, vill ha %s", result.Booked[0].MatchiEnd, end)
	}
}

func TestBookedReportsDaysThatCouldNotBeRead(t *testing.T) {
	loc := stockholm(t)

	start := time.Date(2026, 11, 2, 18, 0, 0, 0, loc)
	end := time.Date(2026, 11, 2, 19, 30, 0, 0, loc)

	kommun := &fakeKommun{appointments: []actorsmartbook.Appointment{{
		ID:       4,
		Location: "Baldershallen 1",
		Start:    start,
		End:      end,
	}}}

	matchiClient := &fakeMatchi{schedules: []matchi.DailySchedule{{
		Date: start,
		Err:  context.DeadlineExceeded,
	}}}

	mappings := &fakeMappings{mappings: []store.CourtMapping{{
		KommunResourceName: "Baldershallen 1",
		MatchiCourtName:    "1. Idrottshall",
	}}}

	service := New(kommun, matchiClient, mappings, loc)

	result, err := service.Booked(context.Background(), start, end)
	if err != nil {
		t.Fatalf("Booked: %v", err)
	}

	if len(result.Errors) != 1 {
		t.Fatalf("Errors = %v, vill ha ett felmeddelande", result.Errors)
	}
}
