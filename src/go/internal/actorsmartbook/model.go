// Package actorsmartbook is a klient för kommunens bokningssystem
// (Actor Smartbook), som Sundsvalls kommun använder för att boka
// idrottsanläggningar.
//
// API:t svarar med Telerik-format där tider är serialiserade som
// "/Date(1793642400000)/". Värdena är millisekunder sedan epoch där
// UTC-komponenterna utgör den lokala väggklockan i anläggningens
// tidszon (TimeZoneOffset: 0 i anropet). Se timeutil.go.
package actorsmartbook

import "time"

// Appointment är en normaliserad bokning från det kommunala bokningssystemet.
type Appointment struct {
	// @Description	Unik id för bokningen i det kommunala systemet
	// @Example		337667
	ID int64 `json:"id"`

	// @Description	Kodad id som används för att anropa boknings-API:t med
	// @Example		eyJWYWx1ZSI6MzM3NjY3LCJUeXBlTmFtZSI6IlN5c3RlbS5JbnQzMiJ9
	EncodedID string `json:"encoded_id"`

	// @Description	Ämne enligt det kommunala systemet, t.ex. "18:00 - 19:30 Baldershallen 1"
	// @Example		18:00 - 19:30 Baldershallen 1
	Subject string `json:"subject"`

	// @Description	Sal eller resurs som bokningen avser, utvuxen ur ämnet
	// @Example		Baldershallen 1
	Location string `json:"location"`

	// @Description	Beskrivning, används av kommunen för att ange bokande organisation
	// @Example		Bergsåkers Badmintonklubb
	Description string `json:"description"`

	// @Description	Starttid i lokal tid (Europe/Stockholm)
	// @Example		2026-11-02T18:00:00+01:00
	Start time.Time `json:"start"`

	// @Description	Sluttid i lokal tid (Europe/Stockholm)
	// @Example		2026-11-02T19:30:00+01:00
	End time.Time `json:"end"`

	// @Description	Bookad tid i minuter
	// @Example		90
	DurationMinutes int `json:"duration_minutes"`

	// @Description	Om bokningen är synlig i det kommunala bokningssystemet
	// @Example		true
	Visible bool `json:"visible"`

	// @Description	Om bokningen är en del av en återkommande serie
	// @Example		false
	Recurring bool `json:"recurring"`

	// @Description	Färgklass i det kommunala systemet, används för att skilja kategorier
	// @Example		actorcolor5
	CSSClass string `json:"css_class,omitempty"`

	// @Description	Resurser som bokningen är knuten till
	Resources []Resource `json:"resources"`
}

// Resource är en resurs (sal, bana) som kan bokas.
type Resource struct {
	// @Description	Resursens id i det kommunala systemet
	// @Example		628
	ID int64 `json:"id"`

	// @Description	Resursens namn
	// @Example		Baldershallen 1
	Name string `json:"name"`
}

// AppointmentListResponse är svaret från endpointen som listar bokningar.
type AppointmentListResponse struct {
	// @Description	Periodens början i lokal tid
	// @Example		2026-11-02T00:00:00+01:00
	From time.Time `json:"from"`

	// @Description	Periodens slut i lokal tid
	// @Example		2026-11-09T00:00:00+01:00
	To time.Time `json:"to"`

	// @Description	Antal bokningar i perioden
	// @Example		151
	Count int `json:"count"`

	// @Description	Alla bokningar i perioden
	Appointments []Appointment `json:"appointments"`
}

// ErrorResponse är standard-felsvar.
type ErrorResponse struct {
	// @Description	Felmeddelande
	// @Example		invalid time range
	Error string `json:"error"`
}
