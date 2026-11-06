package matchi

import (
	"fmt"
	"time"
)

// NormalizeRange avrundar en tidsintervall till hela och halva timmar.
//
// Starttid avrundas nedåt till närmaste halvtimme, sluttid avrundas uppåt
// till närmaste halvtimme. Ett exempel: 11:05-12:50 blir 11:00-13:00.
func NormalizeRange(start, end time.Time) (time.Time, time.Time) {
	return NormalizeStart(start), NormalizeEnd(end)
}

// NormalizeStart avrundar en starttid nedåt till närmaste halvtimme.
func NormalizeStart(start time.Time) time.Time {
	return start.Truncate(slotDuration)
}

// NormalizeEnd avrundar en sluttid uppåt till närmaste halvtimme.
//
// En sluttid som redan ligger på en halvtimme lämnas orörd.
func NormalizeEnd(end time.Time) time.Time {
	truncated := end.Truncate(slotDuration)
	if truncated.Equal(end) {
		return truncated
	}

	return truncated.Add(slotDuration)
}

// ValidateRange kontrollerar att ett intervall är giltigt efter avrundning.
func ValidateRange(start, end time.Time) error {
	if !end.After(start) {
		return fmt.Errorf("sluttiden måste vara efter starttiden")
	}

	return nil
}
