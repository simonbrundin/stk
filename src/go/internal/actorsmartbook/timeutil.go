package actorsmartbook

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// telerikDatePattern matchar Teleriks datumserialisering "/Date(1793642400000)/".
// Bakstreck accepteras också, eftersom kommunens JSON-parser inte hanterar
// omvänt snedstreck utan escaping.
var telerikDatePattern = regexp.MustCompile(`^\\?/Date\((-?\d+)\)\\?/?$`)

// TelerikDate är en tid i det kommunala bokningssystemets datumformat.
type TelerikDate struct {
	// Millis är millisekunder sedan epoch.
	Millis int64
}

// MarshalJSON serialiserar datumet som kommunens API förväntar sig:
// en JSON-sträng med eskapade snedstreck, t.ex. "\/Date(1793642400000)\/".
func (d TelerikDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + FormatTelerikDate(d.Millis) + `"`), nil
}

// UnmarshalJSON tolkar ett datum i det kommunala bokningssystemets format.
func (d *TelerikDate) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	matches := telerikDatePattern.FindStringSubmatch(value)
	if matches == nil {
		return fmt.Errorf("invalid date format: %q", value)
	}

	millis, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid date value %q: %w", value, err)
	}

	d.Millis = millis
	return nil
}

// FormatTelerikDate formaterar millisekunder som Teleriks datum.
func FormatTelerikDate(millis int64) string {
	return fmt.Sprintf(`\/Date(%d)\/`, millis)
}

// ParseTelerikDate tolkar ett "/Date(1793642400000)/"-värde.
//
// Det kommunala systemet skickar lokal väggklocka kodad som millisekunder
// sedan epoch (TimeZoneOffset: 0). UTC-komponenterna är därför den tid
// användaren ser i sin kalender, inte den korresponderande absoluta
// tidpunkten. Funktionen återger därför tiden som väggklocka i loc, vilket
// gör att DST-övergångar hanteras korrekt.
func ParseTelerikDate(value string, loc *time.Location) (time.Time, error) {
	matches := telerikDatePattern.FindStringSubmatch(value)
	if matches == nil {
		return time.Time{}, fmt.Errorf("invalid date format: %q", value)
	}

	millis, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date value %q: %w", value, err)
	}

	return fromWallClockMillis(millis, loc), nil
}

// TelerikMillis konverterar en tid till det millisekundsvärde som det
// kommunala systemet förstår, där lokal väggklocka kodas som UTC.
func TelerikMillis(t time.Time, loc *time.Location) int64 {
	local := t.In(loc)
	return time.Date(
		local.Year(), local.Month(), local.Day(),
		local.Hour(), local.Minute(), local.Second(), local.Nanosecond(),
		time.UTC,
	).UnixMilli()
}

// fromWallClockMillis är inversen av TelerikMillis.
func fromWallClockMillis(millis int64, loc *time.Location) time.Time {
	utc := time.UnixMilli(millis).UTC()
	return time.Date(
		utc.Year(), utc.Month(), utc.Day(),
		utc.Hour(), utc.Minute(), utc.Second(), utc.Nanosecond(),
		loc,
	)
}
