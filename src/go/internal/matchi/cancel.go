package matchi

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Cancel avbokar en eller flera bokningar i MATCHi.
//
// slotIDs är slot-id:n för de bokningar som ska avbokas. MATCHi avbokar
// bokningen utan återbetalning och utan att skicka notis till kunden.
func (c *Client) Cancel(ctx context.Context, slotIDs []string) error {
	if len(slotIDs) == 0 {
		return nil
	}

	if err := c.ensureLogin(ctx); err != nil {
		return err
	}

	returnURL := c.baseURL + "/facility/booking/list"

	// Steg 1: Hämta avbokningsformuläret.
	form := url.Values{
		"cancelSlotsData": {strings.Join(slotIDs, ",")},
		"returnUrl":       {returnURL},
	}

	_, body, err := c.postForm(ctx, "/facility/booking/cancelForm", form)
	if err != nil {
		return err
	}

	if message := errorMessage(body); message != "" {
		return fmt.Errorf("matchi: avbokningen misslyckades: %s", message)
	}

	fields, err := parseFormFields(body)
	if err != nil {
		return err
	}

	fields.Set("cancelSlotsData", strings.Join(slotIDs, ","))
	fields.Set("returnUrl", returnURL)
	fields.Set("refundOption", "NO_REFUND")
	fields.Set("sendNotification", "false")

	// Steg 2: Skicka avbokningen.
	_, cancelBody, err := c.postForm(ctx, "/facility/booking/cancel", fields)
	if err != nil {
		return err
	}

	if message := errorMessage(cancelBody); message != "" {
		return fmt.Errorf("matchi: avbokningen misslyckades: %s", message)
	}

	return nil
}
