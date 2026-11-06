package actorsmartbook

import (
	"testing"
	"time"
)

func TestParseTimeParam(t *testing.T) {
	loc, err := time.LoadLocation(DefaultLocation)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	fallback := time.Date(2026, 11, 2, 12, 0, 0, 0, loc)

	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// Exakta tider.
	exact := []struct {
		name     string
		value    string
		expected time.Time
	}{
		{"empty uses fallback", "", fallback},
		{"date", "2026-11-02", time.Date(2026, 11, 2, 0, 0, 0, 0, loc)},
		{"local date time", "2026-11-02T18:00:00", time.Date(2026, 11, 2, 18, 0, 0, 0, loc)},
		{"rfc3339", "2026-11-02T17:00:00Z", time.Date(2026, 11, 2, 18, 0, 0, 0, loc)},
		{"today minus week", "today-1w", today.AddDate(0, 0, -7)},
	}

	for _, test := range exact {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := ParseTimeParam(test.value, loc, fallback)
			if err != nil {
				t.Fatalf("ParseTimeParam(%q): %v", test.value, err)
			}
			if !parsed.Equal(test.expected) {
				t.Errorf("expected %s, got %s", test.expected, parsed)
			}
		})
	}

	// Relativa uttryck jämförs mot "nu" med en marginal.
	relative := []struct {
		name     string
		value    string
		expected time.Time
	}{
		{"now", "now", time.Now().In(loc)},
		{"today", "today", today},
		{"now plus days", "now+7d", time.Now().In(loc).AddDate(0, 0, 7)},
		{"now plus days url decoded", "now 7d", time.Now().In(loc).AddDate(0, 0, 7)},
		{"now plus days with spaces", "now + 7 days", time.Now().In(loc).AddDate(0, 0, 7)},
		{"now minus hours", "now-3h", time.Now().In(loc).Add(-3 * time.Hour)},
		{"now plus minutes", "now + 30 minutes", time.Now().In(loc).Add(30 * time.Minute)},
	}

	for _, test := range relative {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := ParseTimeParam(test.value, loc, fallback)
			if err != nil {
				t.Fatalf("ParseTimeParam(%q): %v", test.value, err)
			}
			if diff := parsed.Sub(test.expected); diff > time.Minute || diff < -time.Minute {
				t.Errorf("expected %s, got %s", test.expected, parsed)
			}
		})
	}
}

func TestParseTimeParamInvalid(t *testing.T) {
	loc, err := time.LoadLocation(DefaultLocation)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	for _, value := range []string{"hello", "2026-13-45", "now+7x"} {
		if _, err := ParseTimeParam(value, loc, time.Now()); err == nil {
			t.Errorf("expected error for invalid value %q", value)
		}
	}
}
