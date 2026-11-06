package actorsmartbook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	// DefaultBaseURL är kommunens bokningssystem.
	DefaultBaseURL = "https://sundsvall.actorsmartbook.se"

	// DefaultResourceFilter är de resurser (hallar och squashbanor) som
	// tennisklubbens bokningar sker i.
	DefaultResourceFilter = "628,629,630,631,632,751,633,634,635,636,637"

	// DefaultTimeout är timeout för anrop mot det kommunala bokningssystemet.
	DefaultTimeout = 15 * time.Second

	// appointmentsPath är endpointen som returnerar bokningarna.
	appointmentsPath = "/SectionAdmin.aspx/GetAppointments"

	// DefaultLocation är tidszonen anläggningarna ligger i.
	DefaultLocation = "Europe/Stockholm"
)

// Client hämtar bokningar från det kommunala bokningssystemet.
type Client struct {
	baseURL        string
	resourceFilter string
	location       *time.Location
	httpClient     *http.Client
}

// NewClient skapar en klient mot det kommunala bokningssystemet.
//
// Tomma argument faller tillbaka på defaultvärdena: bas-URL, resurfilter,
// tidszon och timeout.
func NewClient(baseURL, resourceFilter string, timeout time.Duration, location *time.Location) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if resourceFilter == "" {
		resourceFilter = DefaultResourceFilter
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if location == nil {
		location, _ = time.LoadLocation(DefaultLocation)
	}

	return &Client{
		baseURL:        strings.TrimRight(baseURL, "/"),
		resourceFilter: resourceFilter,
		location:       location,
		httpClient:     &http.Client{Timeout: timeout},
	}
}

// Location returnerar tidszonen klienten tolkar tiderna i.
func (c *Client) Location() *time.Location {
	return c.location
}

// ResourceFilter returnerar resurfiltret som skickas till bokningssystemet.
func (c *Client) ResourceFilter() string {
	return c.resourceFilter
}

// schedulerInfo är payloaden som styrs till GetAppointments.
type schedulerInfo struct {
	ViewStart                 TelerikDate `json:"ViewStart"`
	ViewEnd                   TelerikDate `json:"ViewEnd"`
	EnableDescriptionField    bool        `json:"EnableDescriptionField"`
	MinutesPerRow             int         `json:"MinutesPerRow"`
	TimeZoneOffset            int         `json:"TimeZoneOffset"`
	VisibleAppointmentsPerDay int         `json:"VisibleAppointmentsPerDay"`
	UpdateMode                int         `json:"UpdateMode"`
	EventFilter               string      `json:"EventFilter"`
	ResourceFilter            string      `json:"ResourceFilter"`
	CustomerID                int         `json:"CustomerID"`
	OnlyOwnBookings           bool        `json:"OnlyOwnBookings"`
}

type appointmentsRequest struct {
	SchedulerInfo schedulerInfo `json:"schedulerInfo"`
}

// rawAppointment motsvarar Teleriks AppointmentData.
type rawAppointment struct {
	Type               string            `json:"__type"`
	ID                 int64             `json:"ID"`
	EncodedID          string            `json:"EncodedID"`
	Start              string            `json:"Start"`
	End                string            `json:"End"`
	Subject            string            `json:"Subject"`
	Description        string            `json:"Description"`
	RecurrenceState    int               `json:"RecurrenceState"`
	RecurrenceParentID *int64            `json:"RecurrenceParentID"`
	RecurrenceRule     *string           `json:"RecurrenceRule"`
	Visible            bool              `json:"Visible"`
	TimeZoneID         *string           `json:"TimeZoneID"`
	Resources          []rawResource     `json:"Resources"`
	Attributes         map[string]string `json:"Attributes"`
	Reminders          []any             `json:"Reminders"`
}

// rawResource motsvarar en resurs i Teleriks AppointmentData.
type rawResource struct {
	ID   int64  `json:"ID"`
	Name string `json:"Name"`
}

// appointmentsResponse är svaret från GetAppointments.
type appointmentsResponse struct {
	Appointments []rawAppointment `json:"d"`
}

// GetAppointments hämtar alla bokningar mellan from och to (inklusive).
//
// Tider skickas till bokningssystemet som lokal väggklocka, i enlighet med
// hur systemet självt kodar dem.
func (c *Client) GetAppointments(ctx context.Context, from, to time.Time) ([]Appointment, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("invalid time range: to must be after from")
	}

	body, err := json.Marshal(appointmentsRequest{
		SchedulerInfo: schedulerInfo{
			ViewStart:                 TelerikDate{Millis: TelerikMillis(from, c.location)},
			ViewEnd:                   TelerikDate{Millis: TelerikMillis(to, c.location)},
			EnableDescriptionField:    true,
			MinutesPerRow:             30,
			TimeZoneOffset:            0,
			VisibleAppointmentsPerDay: 0,
			UpdateMode:                0,
			EventFilter:               "",
			ResourceFilter:            c.resourceFilter,
			CustomerID:                0,
			OnlyOwnBookings:           false,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+appointmentsPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", c.baseURL+"/SectionAdmin.aspx")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request appointments: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("booking system returned %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var decoded appointmentsResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	appointments := make([]Appointment, 0, len(decoded.Appointments))
	for _, item := range decoded.Appointments {
		appointment, err := c.toAppointment(item)
		if err != nil {
			return nil, err
		}
		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

// toAppointment normaliserar en rå bokning från bokningssystemet.
func (c *Client) toAppointment(raw rawAppointment) (Appointment, error) {
	start, err := ParseTelerikDate(raw.Start, c.location)
	if err != nil {
		return Appointment{}, fmt.Errorf("appointment %d start: %w", raw.ID, err)
	}

	end, err := ParseTelerikDate(raw.End, c.location)
	if err != nil {
		return Appointment{}, fmt.Errorf("appointment %d end: %w", raw.ID, err)
	}

	resources := make([]Resource, 0, len(raw.Resources))
	for _, resource := range raw.Resources {
		resources = append(resources, Resource{ID: resource.ID, Name: resource.Name})
	}

	return Appointment{
		ID:              raw.ID,
		EncodedID:       raw.EncodedID,
		Subject:         raw.Subject,
		Location:        extractLocation(raw.Subject),
		Description:     strings.TrimSpace(raw.Description),
		Start:           start,
		End:             end,
		DurationMinutes: int(end.Sub(start).Minutes()),
		Visible:         raw.Visible,
		Recurring:       raw.RecurrenceRule != nil || raw.RecurrenceParentID != nil,
		CSSClass:        raw.Attributes["cssClass"],
		Resources:       resources,
	}, nil
}

// subjectLocationPattern matchar ämnet "18:00 - 19:30 Baldershallen 1".
var subjectLocationPattern = regexp.MustCompile(`^\d{1,2}:\d{2}\s*-\s*\d{1,2}:\d{2}\s+(?P<location>.+)$`)

// extractLocation plockar ut salen ur ämnet, t.ex. "Baldershallen 1".
func extractLocation(subject string) string {
	matches := subjectLocationPattern.FindStringSubmatch(subject)
	if matches == nil {
		return strings.TrimSpace(subject)
	}
	return strings.TrimSpace(matches[1])
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
