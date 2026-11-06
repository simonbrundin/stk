// Package matchi är en klient för MATCHis adminapplikation.
//
// MATCHi exponerar ingen officiell API för anläggningsbokning, så klienten
// loggar in med cookie-baserad inloggning (Keycloak) och använder sedan
// adminapplikationens egna formulär för att läsa och ändra bokningar.
//
// Flödena som används:
//
//	GET  /login                       -> Keycloak-inloggning (steg 1)
//	POST /realms/<realm>/login-actions/authenticate
//	                                  -> session-cookies för www.matchi.se
//	GET  /facility/booking/index      -> dagens schema med alla tidsslotter
//	POST /facility/booking/bookingForm-> bokningsformuläret för valda slotter
//	POST /facility/booking/book       -> själva bokningen
//	POST /facility/booking/moveBookingForm -> flyttformuläret
//	POST /facility/booking/move       -> flytt av bokningen
package matchi

import "time"

const (
	// DefaultBaseURL är MATCHis webbapplikation.
	DefaultBaseURL = "https://www.matchi.se"

	// DefaultAuthRealmPath är Keycloak-realm som MATCHis webbapp använder.
	DefaultAuthRealmPath = "/realms/matchi"

	// DefaultTimeout är timeout för anrop mot MATCHi.
	DefaultTimeout = 30 * time.Second

	// DefaultLocation är tidszonen anläggningen ligger i.
	DefaultLocation = "Europe/Stockholm"

	// slotDuration är längden på en tidsslot i MATCHis schema.
	slotDuration = 30 * time.Minute
)

// Slot är en tidsslot i MATCHis bokningsschema.
//
// MATCHi delar upp dagen i halvtimmeslopser. En bokning kan täcka flera
// slotter, då hör de ihop via samma grupp-id.
type Slot struct {
	// @Description	Namnet på banan/salen som slotten tillhör
	// @Example		1. Tennis - Vasa Bryggeri
	CourtName string `json:"court_name"`

	// @Description	MATCHis interna id för slotten
	// @Example		4a66b2f0-3177-4c9c-a7a5-3a54e44a9737
	SlotID string `json:"slot_id"`

	// @Description	Starttid i lokal tid (Europe/Stockholm)
	// @Example		2026-11-06T07:00:00+01:00
	Start time.Time `json:"start"`

	// @Description	Sluttid i lokal tid (Europe/Stockholm)
	// @Example		2026-11-06T07:30:00+01:00
	End time.Time `json:"end"`

	// @Description	Om slotten är obokad
	// @Example		true
	Free bool `json:"free"`

	// @Description	Om slotten går att boka (tidsavvikelse eller stängd tid gör den obokbar)
	// @Example		true
	Bookable bool `json:"bookable"`

	// @Description	Bokningsstatus, t.ex. OK, PENDING eller PARTLY
	// @Example		OK
	Status string `json:"status,omitempty"`

	// @Description	MATCHis numeriska boknings-id
	// @Example		112291396
	BookingID string `json:"booking_id,omitempty"`

	// @Description	Grupp-id som sammanlänkar slotter till samma bokning
	// @Example		14003203
	GroupID string `json:"group_id,omitempty"`

	// @Description	Övrig information om slotten, t.ex. kundnamn
	// @Example		STK Verksamhet
	Title string `json:"title,omitempty"`
}

// Schedule är ett dygns schema för en anläggning.
type Schedule struct {
	// @Description	De bokade eller fria slotterna i schemaordning
	Slots []Slot `json:"slots"`
}

// BookRequest är en begäran om att boka en tid i MATCHi.
type BookRequest struct {
	// @Description	Namnet på banan/salen, t.ex. "1. Tennis - Vasa Bryggeri"
	// @Example		1. Tennis - Vasa Bryggeri
	CourtName string `json:"court_name" binding:"required"`

	// @Description	Starttid, avrundas nedåt till närmaste halvtimme
	// @Example		2026-11-06T11:00:00+01:00
	Start time.Time `json:"start" binding:"required"`

	// @Description	Sluttid, avrundas uppåt till närmaste halvtimme
	// @Example		2026-11-06T13:00:00+01:00
	End time.Time `json:"end" binding:"required"`

	// @Description	MATCHis kund-id som bokningen görs för
	// @Example		1660773
	CustomerID int64 `json:"customer_id"`

	// @Description	Kommentar som läggs till bokningen
	// @Example		Tennislaget
	Comment string `json:"comment,omitempty"`

	// @Description	Typ av bokning, t.ex. DEFAULT eller TRAINING
	// @Example		DEFAULT
	Type string `json:"type,omitempty"`
}

// MoveRequest är en begäran om att flytta en befintlig bokning.
type MoveRequest struct {
	// @Description	MATCHis slot-id för den bokning som ska flyttas
	// @Example		2a432de2-551d-4f05-8111-23c23b2b04bf
	SlotID string `json:"slot_id" binding:"required"`

	// @Description	Ny starttid, avrundas nedåt till närmaste halvtimme
	// @Example		2026-11-06T11:00:00+01:00
	Start time.Time `json:"start" binding:"required"`

	// @Description	Ny sluttid, avrundas uppåt till närmaste halvtimme
	// @Example		2026-11-06T13:00:00+01:00
	End time.Time `json:"end" binding:"required"`

	// @Description	Namnet på banan/salen, om bokningen ska flyttas till en annan bana
	// @Example		1. Tennis - Vasa Bryggeri
	CourtName string `json:"court_name,omitempty"`
}

// BookingResult är resultatet av en bokning.
type BookingResult struct {
	// @Description	MATCHis slot-id för den första slotten i bokningen
	// @Example		4a66b2f0-3177-4c9c-a7a5-3a54e44a9737
	SlotID string `json:"slot_id"`

	// @Description	MATCHis numeriska boknings-id
	// @Example		112291396
	BookingID string `json:"booking_id,omitempty"`

	// @Description	Grupp-id för bokningen, om den spänner flera slotter
	// @Example		14003203
	GroupID string `json:"group_id,omitempty"`

	// @Description	De slotter som bokades
	// @Example		["4a66b2f0-3177-4c9c-a7a5-3a54e44a9737"]
	SlotIDs []string `json:"slot_ids"`

	// @Description	Starttid i lokal tid
	// @Example		2026-11-06T11:00:00+01:00
	Start time.Time `json:"start"`

	// @Description	Sluttid i lokal tid
	// @Example		2026-11-06T13:00:00+01:00
	End time.Time `json:"end"`

	// @Description	Namnet på banan/salen
	// @Example		1. Tennis - Vasa Bryggeri
	CourtName string `json:"court_name"`
}

// ErrorResponse är standard-felsvar.
type ErrorResponse struct {
	// @Description	Felmeddelande
	// @Example		invalid time range
	Error string `json:"error"`
}
