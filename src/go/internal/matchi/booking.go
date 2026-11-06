package matchi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrUnauthenticated betyder att sessionen har gått ut och att klienten
// måste logga in igen.
var ErrUnauthenticated = errors.New("matchi: sessionen har gått ut")

// Book bokar en tid i MATCHi.
//
// Tiden avrundas till hela och halva timmar enligt NormalizeRange.
func (c *Client) Book(ctx context.Context, req BookRequest) (*BookingResult, error) {
	start, end := NormalizeRange(req.Start, req.End)
	if err := ValidateRange(start, end); err != nil {
		return nil, err
	}

	if req.CourtName == "" {
		return nil, errors.New("matchi: banan/sal måste anges")
	}

	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}

	day := start.In(c.location)
	schedule, err := c.Schedule(ctx, day)
	if err != nil {
		return nil, err
	}

	slots, err := slotsInRange(schedule, req.CourtName, start, end)
	if err != nil {
		return nil, err
	}

	slotIDs := make([]string, 0, len(slots))
	for _, slot := range slots {
		slotIDs = append(slotIDs, slot.SlotID)
	}

	customerID := req.CustomerID
	bookingType := req.Type
	if bookingType == "" {
		bookingType = "DEFAULT"
	}

	returnURL := fmt.Sprintf("%s/facility/booking/index?date=%s", c.baseURL, day.Format("2006-01-02"))

	// Steg 1: Hämta bokningsformuläret för de valda slotterna.
	formURL := url.Values{
		"slotId":         {strings.Join(slotIDs, ",")},
		"date":           {day.Format("2006-01-02")},
		"customerId":     {formatInt(customerID)},
		"customerSearch": {formatInt(customerID)},
		"type":           {bookingType},
		"returnUrl":      {returnURL},
	}

	_, formBody, err := c.postForm(ctx, "/facility/booking/bookingForm", formURL)
	if err != nil {
		return nil, err
	}

	fields, err := parseFormFields(formBody)
	if err != nil {
		return nil, err
	}

	// Steg 2: Fyll i kund, kommentar och betalning.
	fields.Set("slotId", strings.Join(slotIDs, ","))
	fields.Set("date", day.Format("2006-01-02"))
	fields.Set("type", bookingType)
	fields.Set("returnUrl", returnURL)
	fields.Set("customerSearch", formatInt(customerID))
	fields.Set("playerCustomerId", formatInt(customerID))
	fields.Set("customerId", formatInt(customerID))
	fields.Set("comments", req.Comment)
	fields.Set("useRecurrence", "false")
	fields.Set("showComment", "false")
	fields.Set("activateMpc", "false")
	fields.Set("paid", "false")
	fields.Set("sendNotification", "false")
	fields.Set("sendPaymentLink", "false")
	fields.Set("useGiftCard", "false")
	fields.Set("useCoupon", "false")
	fields.Set("selectPaymentOption", "UNPAID")

	// Steg 3: Skicka bokningen.
	_, bookBody, err := c.postForm(ctx, "/facility/booking/book", fields)
	if err != nil {
		return nil, err
	}

	if message := errorMessage(bookBody); message != "" {
		return nil, fmt.Errorf("matchi: bokningen misslyckades: %s", message)
	}

	// Steg 4: Läs av det nya boknings-id:t ur schemat.
	result := &BookingResult{
		SlotID:    slotIDs[0],
		SlotIDs:   slotIDs,
		Start:     start,
		End:       end,
		CourtName: req.CourtName,
	}

	// MATCHi skapar ett nytt slot-id för bokningen, så det går inte att
	// använda slot-id:t från det gamla schemat. Den nya bokningen hittas
	// i stället genom att matcha bana och tid.
	if updated, err := c.Schedule(ctx, day); err == nil {
		if booked, ok := findBookedSlot(updated, req.CourtName, start, end); ok {
			result.SlotID = booked.SlotID
			result.BookingID = booked.BookingID
			result.GroupID = booked.GroupID
		}
	}

	return result, nil
}

// Move flyttar en befintlig bokning till en ny tid.
//
// SlotID är slotten som bokningen ligger i. Boken flyttas till den slotten
// som börjar vid den nya starttiden.
func (c *Client) Move(ctx context.Context, req MoveRequest) (*BookingResult, error) {
	start, end := NormalizeRange(req.Start, req.End)
	if err := ValidateRange(start, end); err != nil {
		return nil, err
	}

	if req.SlotID == "" {
		return nil, errors.New("matchi: slot-id för bokningen måste anges")
	}

	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}

	day := start.In(c.location)
	schedule, err := c.Schedule(ctx, day)
	if err != nil {
		return nil, err
	}

	current, ok := findSlot(schedule, req.SlotID)
	if !ok {
		return nil, fmt.Errorf("matchi: hittade inte bokningen %s i schemat", req.SlotID)
	}

	court := req.CourtName
	if court == "" {
		court = current.CourtName
	}

	target, ok := findSlotAt(schedule, court, start)
	if !ok {
		return nil, fmt.Errorf("matchi: ingen slot på %s klockan %s", court, start.Format("15:04"))
	}

	if !target.Free {
		return nil, fmt.Errorf("matchi: tiden %s på %s är redan bokad", start.Format("15:04"), court)
	}

	returnURL := fmt.Sprintf("%s/facility/booking/index?date=%s", c.baseURL, day.Format("2006-01-02"))

	// Steg 1: Hämta flyttformuläret.
	moveForm := url.Values{
		"slotId":        {target.SlotID},
		"bookingSlotId": {req.SlotID},
		"date":          {day.Format("2006-01-02")},
		"returnUrl":     {returnURL},
	}

	_, moveBody, err := c.postForm(ctx, "/facility/booking/moveBookingForm", moveForm)
	if err != nil {
		return nil, err
	}

	fields, err := parseFormFields(moveBody)
	if err != nil {
		return nil, err
	}

	if message := errorMessage(moveBody); message != "" {
		return nil, fmt.Errorf("matchi: flytten misslyckades: %s", message)
	}

	fields.Set("bookingSlotId", req.SlotID)
	fields.Set("slotId", target.SlotID)
	fields.Set("date", day.Format("2006-01-02"))
	fields.Set("returnUrl", returnURL)

	// Steg 2: Skicka flytten.
	if _, _, err := c.postForm(ctx, "/facility/booking/move", fields); err != nil {
		return nil, err
	}

	result := &BookingResult{
		SlotID:    target.SlotID,
		SlotIDs:   []string{target.SlotID},
		Start:     start,
		End:       end,
		CourtName: court,
		BookingID: current.BookingID,
		GroupID:   current.GroupID,
	}

	// Läs av den faktiska tiden efter flytten, eftersom MATCHi kan behålla
	// bokningens ursprungliga längd.
	if updated, err := c.Schedule(ctx, day); err == nil {
		if moved, ok := findBookedSlotStartingAt(updated, court, result.Start); ok {
			result.SlotID = moved.SlotID
			result.SlotIDs = []string{moved.SlotID}
			result.Start = moved.Start
			result.End = moved.End
			if moved.BookingID != "" {
				result.BookingID = moved.BookingID
			}
			if moved.GroupID != "" {
				result.GroupID = moved.GroupID
			}
		}
	}

	return result, nil
}

// slotsInRange returnerar de fria slotterna i ett tidsintervall.
//
// En bokad cell i MATCHis schema täcker hela bokningen, vilket innebär att
// den kan överlappa det sökta intervallet utan att vara lika stor som det.
func slotsInRange(schedule *Schedule, court string, start, end time.Time) ([]Slot, error) {
	var slots []Slot

	for _, slot := range schedule.Slots {
		if !strings.EqualFold(slot.CourtName, court) {
			continue
		}
		if !overlaps(slot, start, end) {
			continue
		}
		if !slot.Free {
			return nil, fmt.Errorf("matchi: tiden %s-%s på %s är redan bokad", slot.Start.Format("15:04"), slot.End.Format("15:04"), court)
		}
		if !slot.Bookable {
			return nil, fmt.Errorf("matchi: tiden %s-%s på %s är inte bokningsbar", slot.Start.Format("15:04"), slot.End.Format("15:04"), court)
		}
		slots = append(slots, slot)
	}

	if len(slots) == 0 {
		return nil, fmt.Errorf("matchi: hittade ingen slot på %s mellan %s och %s", court, start.Format("15:04"), end.Format("15:04"))
	}

	expected := int(end.Sub(start) / slotDuration)
	if len(slots) != expected {
		return nil, fmt.Errorf("matchi: schemat har %d fria slotter mellan %s och %s, förväntade %d", len(slots), start.Format("15:04"), end.Format("15:04"), expected)
	}

	return slots, nil
}

// overlaps avgör om en slot överlappar ett tidsintervall.
func overlaps(slot Slot, start, end time.Time) bool {
	return slot.Start.Before(end) && slot.End.After(start)
}

// findSlot hittar en slot med givet slot-id.
func findSlot(schedule *Schedule, slotID string) (Slot, bool) {
	for _, slot := range schedule.Slots {
		if slot.SlotID == slotID {
			return slot, true
		}
	}

	return Slot{}, false
}

// findSlotAt hittar sloten som börjar vid en viss tid på en bana.
func findSlotAt(schedule *Schedule, court string, start time.Time) (Slot, bool) {
	for _, slot := range schedule.Slots {
		if !strings.EqualFold(slot.CourtName, court) {
			continue
		}
		if slot.Start.Equal(start) {
			return slot, true
		}
	}

	return Slot{}, false
}

// findBookedSlot hittar en bokad slot på en bana med exakt start och slut.
//
// En bokad cell i MATCHis schema täcker hela bokningen, vilket gör att
// tidsintervallet matchar både för enstaka slotter och för längre bokningar.
func findBookedSlot(schedule *Schedule, court string, start, end time.Time) (Slot, bool) {
	for _, slot := range schedule.Slots {
		if !strings.EqualFold(slot.CourtName, court) {
			continue
		}
		if slot.Free {
			continue
		}
		if slot.Start.Equal(start) && slot.End.Equal(end) {
			return slot, true
		}
	}

	return Slot{}, false
}

// findBookedSlotStartingAt hittar en bokad slot på en bana som startar vid en
// viss tid.
func findBookedSlotStartingAt(schedule *Schedule, court string, start time.Time) (Slot, bool) {
	for _, slot := range schedule.Slots {
		if !strings.EqualFold(slot.CourtName, court) {
			continue
		}
		if slot.Free {
			continue
		}
		if slot.Start.Equal(start) {
			return slot, true
		}
	}

	return Slot{}, false
}

// formatInt formaterar ett tal som sträng.
func formatInt(value int64) string {
	if value == 0 {
		return ""
	}

	return strconv.FormatInt(value, 10)
}
