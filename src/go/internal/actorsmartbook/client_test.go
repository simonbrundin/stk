package actorsmartbook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetAppointments(t *testing.T) {
	var received appointmentsRequest
	var rawBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != appointmentsPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Requested-With"); got != "XMLHttpRequest" {
			t.Errorf("expected X-Requested-With XMLHttpRequest, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json; charset=UTF-8" {
			t.Errorf("unexpected content type: %q", got)
		}
		rawBody, _ = io.ReadAll(r.Body)
		if err := json.Unmarshal(rawBody, &received); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":[
			{"__type":"Telerik.Web.UI.AppointmentData","ID":337667,"EncodedID":"abc",
			 "Start":"\/Date(1793642400000)\/","End":"\/Date(1793647800000)\/",
			 "Subject":"18:00 - 19:30 Baldershallen 1","Description":"Bergsåkers Badmintonklubb  ",
			 "RecurrenceState":0,"Visible":true,"Resources":[],"Attributes":{"cssClass":"actorcolor5"},"Reminders":[]}
		]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "", 0, nil)
	from := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 7)

	appointments, err := client.GetAppointments(context.Background(), from, to)
	if err != nil {
		t.Fatalf("GetAppointments: %v", err)
	}

	if len(appointments) != 1 {
		t.Fatalf("expected 1 appointment, got %d", len(appointments))
	}

	appointment := appointments[0]
	if appointment.ID != 337667 {
		t.Errorf("expected id 337667, got %d", appointment.ID)
	}
	if appointment.Location != "Baldershallen 1" {
		t.Errorf("expected location Baldershallen 1, got %q", appointment.Location)
	}
	if appointment.Description != "Bergsåkers Badmintonklubb" {
		t.Errorf("unexpected description: %q", appointment.Description)
	}
	if appointment.DurationMinutes != 90 {
		t.Errorf("expected 90 minutes, got %d", appointment.DurationMinutes)
	}
	if appointment.CSSClass != "actorcolor5" {
		t.Errorf("expected cssClass actorcolor5, got %q", appointment.CSSClass)
	}

	// 1793642400000 är 18:00 i det kommunala systemets väggklocka.
	wantStart := time.Date(2026, 11, 2, 18, 0, 0, 0, client.Location())
	if !appointment.Start.Equal(wantStart) {
		t.Errorf("expected start %s, got %s", wantStart, appointment.Start)
	}
	wantEnd := time.Date(2026, 11, 2, 19, 30, 0, 0, client.Location())
	if !appointment.End.Equal(wantEnd) {
		t.Errorf("expected end %s, got %s", wantEnd, appointment.End)
	}

	// Anropet ska använda samma väggklockkodning: 00:00 UTC är 01:00 i
	// lokal tid (CET), vilket skickas som 01:00.
	if received.SchedulerInfo.ViewStart.Millis != 1793581200000 {
		t.Errorf("unexpected ViewStart: %d", received.SchedulerInfo.ViewStart.Millis)
	}
	if !strings.Contains(string(rawBody), `"ViewStart":"\/Date(1793581200000)\/"`) {
		t.Errorf("expected escaped date format in body, got: %s", rawBody)
	}
	if received.SchedulerInfo.ResourceFilter != DefaultResourceFilter {
		t.Errorf("unexpected ResourceFilter: %s", received.SchedulerInfo.ResourceFilter)
	}
}

func TestGetAppointmentsInvalidRange(t *testing.T) {
	client := NewClient("http://example.invalid", "", 0, nil)
	now := time.Now()

	if _, err := client.GetAppointments(context.Background(), now, now); err == nil {
		t.Fatal("expected error for invalid time range")
	}
}

func TestGetAppointmentsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "", 0, nil)
	if _, err := client.GetAppointments(context.Background(), time.Now(), time.Now().Add(time.Hour)); err == nil {
		t.Fatal("expected error when booking system fails")
	}
}

func TestParseTelerikDate(t *testing.T) {
	loc, err := time.LoadLocation(DefaultLocation)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	parsed, err := ParseTelerikDate("/Date(1793642400000)/", loc)
	if err != nil {
		t.Fatalf("ParseTelerikDate: %v", err)
	}

	want := time.Date(2026, 11, 2, 18, 0, 0, 0, loc)
	if !parsed.Equal(want) {
		t.Errorf("expected %s, got %s", want, parsed)
	}

	if _, err := ParseTelerikDate("2026-11-02T18:00:00Z", loc); err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestExtractLocation(t *testing.T) {
	tests := map[string]string{
		"18:00 - 19:30 Baldershallen 1":          "Baldershallen 1",
		"18:00 - 20:00 Baldershallen 3":          "Baldershallen 3",
		"Baldershallen Squashbana 1 ":            "Baldershallen Squashbana 1",
		"18:00-19:30 Baldershallen Squashbana 2": "Baldershallen Squashbana 2",
	}

	for subject, expected := range tests {
		if got := extractLocation(subject); got != expected {
			t.Errorf("extractLocation(%q) = %q, expected %q", subject, got, expected)
		}
	}
}
