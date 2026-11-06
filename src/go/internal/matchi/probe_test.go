package matchi

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestProbeFacilities(t *testing.T) {
	username := os.Getenv("MATCHI_USERNAME")
	password := os.Getenv("MATCHI_PASSWORD")
	if username == "" || password == "" {
		t.Skip("no creds")
	}

	client := NewClient("", username, password, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()
	if err := client.ensureLogin(ctx); err != nil {
		t.Fatalf("login: %v", err)
	}
	t.Logf("login took %s", time.Since(start))

	loc := client.Location()
	for i := 0; i < 3; i++ {
		day := time.Now().In(loc).AddDate(0, 0, i+1)
		s := time.Now()
		schedule, err := client.Schedule(ctx, day)
		if err != nil {
			t.Fatalf("schedule: %v", err)
		}
		booked := 0
		for _, slot := range schedule.Slots {
			if !slot.Free {
				booked++
			}
		}
		t.Logf("%s: %d slotter (%d bokade) på %s", day.Format("2006-01-02"), len(schedule.Slots), booked, time.Since(s))
	}
}
