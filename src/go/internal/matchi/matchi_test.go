package matchi

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNormalizeRange(t *testing.T) {
	loc, err := time.LoadLocation(DefaultLocation)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	tests := []struct {
		name      string
		start     string
		end       string
		wantStart string
		wantEnd   string
	}{
		{
			name:      "start avrundas nedåt och slut uppåt",
			start:     "2026-11-06T11:05:00",
			end:       "2026-11-06T12:50:00",
			wantStart: "2026-11-06T11:00:00",
			wantEnd:   "2026-11-06T13:00:00",
		},
		{
			name:      "halvtimme lämnas orörd",
			start:     "2026-11-06T11:30:00",
			end:       "2026-11-06T13:00:00",
			wantStart: "2026-11-06T11:30:00",
			wantEnd:   "2026-11-06T13:00:00",
		},
		{
			name:      "slut på hel timme avrundas uppåt",
			start:     "2026-11-06T11:00:00",
			end:       "2026-11-06T13:01:00",
			wantStart: "2026-11-06T11:00:00",
			wantEnd:   "2026-11-06T13:30:00",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, err := time.ParseInLocation("2006-01-02T15:04:05", test.start, loc)
			if err != nil {
				t.Fatalf("parse start: %v", err)
			}
			end, err := time.ParseInLocation("2006-01-02T15:04:05", test.end, loc)
			if err != nil {
				t.Fatalf("parse end: %v", err)
			}

			gotStart, gotEnd := NormalizeRange(start, end)

			if !gotStart.Equal(start) && gotStart.Format("2006-01-02T15:04:05") != test.wantStart {
				t.Fatalf("start = %s, want %s", gotStart.Format("2006-01-02T15:04:05"), test.wantStart)
			}

			if gotEnd.Format("2006-01-02T15:04:05") != test.wantEnd {
				t.Fatalf("end = %s, want %s", gotEnd.Format("2006-01-02T15:04:05"), test.wantEnd)
			}
		})
	}
}

func TestValidateRange(t *testing.T) {
	base := time.Date(2026, 11, 6, 11, 0, 0, 0, time.UTC)

	if err := ValidateRange(base, base.Add(time.Hour)); err != nil {
		t.Fatalf("giltigt intervall avvisades: %v", err)
	}

	if err := ValidateRange(base, base); err == nil {
		t.Fatal("ogiltigt intervall accepterades")
	}
}

// scheduleFixture är en förenklad schemasida med en fri slot och en bokad.
const scheduleFixture = `<!DOCTYPE html>
<html><body>
<table class="facility-schedule">
<tr>
  <td class="court"><div class="name"><span class="label" rel="tooltip" title="Endast bokningsbar av klubben">1. Tennis - Vasa Bryggeri</span></div></td>
  <td><table><tr>
    <td onclick="showForm('11111111-1111-1111-1111-111111111111');return false;" slotid="11111111-1111-1111-1111-111111111111" class="slot free " rel="tooltip" title="1. Tennis - Vasa Bryggeri<br> 07:00 - 07:30"></td>
    <td onclick="showForm('22222222-2222-2222-2222-222222222222');return false;" slotid="22222222-2222-2222-2222-222222222222" bookingid="112291396" class="slot PENDING yellow group_14003203 " rel="tooltip" title=" BAB ströbokning<br /> 1. Tennis - Vasa Bryggeri<br> 07:30 - 08:30 <hr/> Bokat av Patrik<br> "></td>
    <td onclick="showForm('33333333-3333-3333-3333-333333333333');return false;" slotid="33333333-3333-3333-3333-333333333333" class="slot free " rel="tooltip" title="1. Tennis - Vasa Bryggeri<br> 08:30 - 09:00"></td>
  </tr></table></td>
</tr>
</table>
</body></html>`

func TestParseSchedule(t *testing.T) {
	loc, err := time.LoadLocation(DefaultLocation)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	day := time.Date(2026, 11, 6, 0, 0, 0, 0, loc)

	slots, err := parseSchedule(scheduleFixture, day, loc)
	if err != nil {
		t.Fatalf("parseSchedule: %v", err)
	}

	if len(slots) != 3 {
		t.Fatalf("fick %d slotter, vill ha 3", len(slots))
	}

	if slots[0].CourtName != "1. Tennis - Vasa Bryggeri" {
		t.Errorf("court = %q", slots[0].CourtName)
	}

	if !slots[0].Free {
		t.Error("första slotten ska vara fri")
	}

	if slots[1].Free {
		t.Error("andra slotten ska vara bokad")
	}

	if slots[1].BookingID != "112291396" {
		t.Errorf("booking id = %q", slots[1].BookingID)
	}

	if slots[1].GroupID != "14003203" {
		t.Errorf("grupp-id = %q", slots[1].GroupID)
	}

	if slots[1].Status != "PENDING" {
		t.Errorf("status = %q", slots[1].Status)
	}

	if want := "2026-11-06T07:30:00+01:00"; slots[1].Start.Format(time.RFC3339) != want {
		t.Errorf("start = %s, vill ha %s", slots[1].Start.Format(time.RFC3339), want)
	}

	if want := "2026-11-06T08:30:00+01:00"; slots[1].End.Format(time.RFC3339) != want {
		t.Errorf("slut = %s, vill ha %s", slots[1].End.Format(time.RFC3339), want)
	}
}

func TestSlotsInRange(t *testing.T) {
	loc, err := time.LoadLocation(DefaultLocation)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	day := time.Date(2026, 11, 6, 0, 0, 0, 0, loc)

	slots, err := parseSchedule(scheduleFixture, day, loc)
	if err != nil {
		t.Fatalf("parseSchedule: %v", err)
	}

	schedule := &Schedule{Slots: slots}

	// Ett intervall som överlappar den bokade tiden ska avvisas.
	_, err = slotsInRange(schedule, "1. Tennis - Vasa Bryggeri",
		time.Date(2026, 11, 6, 8, 0, 0, 0, loc),
		time.Date(2026, 11, 6, 9, 0, 0, 0, loc))
	if err == nil {
		t.Fatal("överlappande bokning accepteras inte")
	}

	// Ett intervall som täcker en fri slot ska accepteras.
	free, err := slotsInRange(schedule, "1. Tennis - Vasa Bryggeri",
		time.Date(2026, 11, 6, 8, 30, 0, 0, loc),
		time.Date(2026, 11, 6, 9, 0, 0, 0, loc))
	if err != nil {
		t.Fatalf("fria slotter avvisades: %v", err)
	}

	if len(free) != 1 {
		t.Fatalf("fick %d fria slotter, vill ha 1", len(free))
	}
}

func TestLoginAction(t *testing.T) {
	// Keycloaks sessions-id kan innehålla tecken som "_" och ".", vilket
	// tidigare kapade värdet och gjorde inloggningen intermittent felaktig.
	body := `<form action="https://auth.matchi.com/realms/matchi/login-actions/authenticate?session_code=U6_ab.cd&amp;execution=eebecb49-89a1-467a-90ad-475d98cb44d7&amp;client_id=webapp&amp;tab_id=xvB5bZS1ng8"></form>`

	action, err := loginAction("https://auth.matchi.com/realms/matchi", body)
	if err != nil {
		t.Fatalf("loginAction: %v", err)
	}

	parsed, err := url.Parse(action)
	if err != nil {
		t.Fatalf("parse action: %v", err)
	}

	query := parsed.Query()

	if got := query.Get("session_code"); got != "U6_ab.cd" {
		t.Errorf("session_code = %q, vill ha %q", got, "U6_ab.cd")
	}

	if got := query.Get("execution"); got != "eebecb49-89a1-467a-90ad-475d98cb44d7" {
		t.Errorf("execution = %q", got)
	}

	if got := query.Get("tab_id"); got != "xvB5bZS1ng8" {
		t.Errorf("tab_id = %q", got)
	}

	if got := query.Get("client_id"); got != "webapp" {
		t.Errorf("client_id = %q", got)
	}
}

func TestParseFormFields(t *testing.T) {
	body := `<div><form>
		<input type="hidden" name="slotId" value="abc" />
		<input type="hidden" name="date" value="2026-11-06" />
		<input type="checkbox" name="showComment" value="true" />
		<select name="type"><option value="DEFAULT">Standard</option><option value="TRAINING" selected>Träning</option></select>
		<textarea name="comments">Hej</textarea>
	</form></div>`

	fields, err := parseFormFields(body)
	if err != nil {
		t.Fatalf("parseFormFields: %v", err)
	}

	if fields.Get("slotId") != "abc" {
		t.Errorf("slotId = %q", fields.Get("slotId"))
	}

	if fields.Get("date") != "2026-11-06" {
		t.Errorf("date = %q", fields.Get("date"))
	}

	if fields.Get("type") != "TRAINING" {
		t.Errorf("type = %q", fields.Get("type"))
	}

	if fields.Get("comments") != "Hej" {
		t.Errorf("comments = %q", fields.Get("comments"))
	}

	if _, ok := fields["showComment"]; ok {
		t.Error("kryssrutor ska inte tas med")
	}
}

func TestLoginRejectsEncryptedSecrets(t *testing.T) {
	client := NewClient("", "user@example.com", "ENC[AES256_GCM,data:abc]", time.Second, nil)

	err := client.Login(context.Background())
	if err == nil {
		t.Fatal("Login med krypterat lösenord ska misslyckas")
	}

	if !strings.Contains(err.Error(), "krypterat") {
		t.Errorf("felmeddelandet ska nämna att lösenordet är krypterat, fick %q", err)
	}
}
