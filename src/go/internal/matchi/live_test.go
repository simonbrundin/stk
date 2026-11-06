package matchi

import (
	"context"
	"os"
	"testing"
	"time"
)

// liveCredentials läser inloggningsuppgifterna ur miljövariablerna.
// Testet hoppas över om de inte är satta.
func liveCredentials(t *testing.T) (string, string) {
	t.Helper()

	username := os.Getenv("MATCHI_USERNAME")
	password := os.Getenv("MATCHI_PASSWORD")

	if username == "" || password == "" {
		t.Skip("MATCHI_USERNAME/MATCHI_PASSWORD är inte satta - hoppar över live-testet")
	}

	return username, password
}

// TestLiveSchedule läser ett dygns schema från MATCHi.
//
//	cd src/go && MATCHI_USERNAME=... MATCHI_PASSWORD=... go test ./internal/matchi -run TestLive
func TestLiveSchedule(t *testing.T) {
	username, password := liveCredentials(t)

	client := NewClient("", username, password, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	schedule, err := client.Schedule(ctx, time.Now())
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	courts := map[string]int{}
	for _, slot := range schedule.Slots {
		courts[slot.CourtName]++
	}

	t.Logf("hittade %d slotter på %d banor", len(schedule.Slots), len(courts))
	for court, count := range courts {
		t.Logf("  %s: %d slotter", court, count)
	}

	if len(schedule.Slots) == 0 {
		t.Fatal("inga slotter i schemat")
	}
}
