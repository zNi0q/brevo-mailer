package mailer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Contact is one person in Brevo's audience. Attributes are Brevo contact
// attributes, upper-case by Brevo's convention: FIRSTNAME, PLAN, SIGNED_UP_AT.
type Contact struct {
	Email      string
	Attributes map[string]any
	// ListIDs adds the contact to these lists.
	ListIDs []int64
	// Unsubscribe removes the contact from these lists.
	UnlinkListIDs []int64
}

// UpsertContact creates the contact, or updates it if the email is already known,
// and puts it on the given lists. Call it when someone signs up, upgrades or
// changes something a campaign segments on.
func (c *Client) UpsertContact(ctx context.Context, ct Contact) error {
	if !strings.Contains(ct.Email, "@") {
		return errors.New("mailer: " + ct.Email + " is not an email address")
	}
	body := map[string]any{
		"email":         ct.Email,
		"updateEnabled": true,
	}
	if len(ct.Attributes) > 0 {
		body["attributes"] = ct.Attributes
	}
	if len(ct.ListIDs) > 0 {
		body["listIds"] = ct.ListIDs
	}
	if len(ct.UnlinkListIDs) > 0 {
		body["unlinkListIds"] = ct.UnlinkListIDs
	}
	return c.do(ctx, http.MethodPost, "/contacts", body, nil)
}

// DeleteContact removes a contact entirely, for example when an account is deleted.
// A contact Brevo never knew about is not an error.
func (c *Client) DeleteContact(ctx context.Context, email string) error {
	err := c.do(ctx, http.MethodDelete, "/contacts/"+url.PathEscape(email), nil, nil)
	var e *APIError
	if errors.As(err, &e) && e.Status == http.StatusNotFound {
		return nil
	}
	return err
}

// Campaign is one marketing email sent to whole lists at once: a newsletter, a
// launch, an offer. HTML is usually Layout.Render of a Template filled with
// Brevo's own contact placeholders, e.g. {{ contact.FIRSTNAME }}.
type Campaign struct {
	// Name is internal, shown only in Brevo's dashboard.
	Name    string
	Subject string
	// PreviewText is the line inbox apps show after the subject.
	PreviewText string
	HTML        string
	ListIDs     []int64
	// ScheduledAt sends it later; the zero time leaves it as a draft for SendCampaignNow.
	ScheduledAt time.Time
	Tag         string
}

// CreateCampaign saves the campaign in Brevo and returns its id. It is not sent
// until it is scheduled or SendCampaignNow is called.
func (c *Client) CreateCampaign(ctx context.Context, cp Campaign) (int64, error) {
	switch {
	case cp.Name == "" || cp.Subject == "" || cp.HTML == "":
		return 0, errors.New("mailer: a campaign needs a Name, a Subject and HTML")
	case len(cp.ListIDs) == 0:
		return 0, errors.New("mailer: a campaign needs at least one list to go to")
	}
	body := map[string]any{
		"name":        cp.Name,
		"subject":     cp.Subject,
		"sender":      Address{Email: c.cfg.FromEmail, Name: c.cfg.FromName},
		"htmlContent": cp.HTML,
		"recipients":  map[string]any{"listIds": cp.ListIDs},
	}
	if cp.PreviewText != "" {
		body["previewText"] = cp.PreviewText
	}
	if c.cfg.ReplyTo != "" {
		body["replyTo"] = c.cfg.ReplyTo
	}
	if !cp.ScheduledAt.IsZero() {
		body["scheduledAt"] = cp.ScheduledAt.UTC().Format(time.RFC3339)
	}
	if cp.Tag != "" {
		body["tag"] = cp.Tag
	}
	var out struct {
		ID int64 `json:"id"`
	}
	if err := c.do(ctx, http.MethodPost, "/emailCampaigns", body, &out); err != nil {
		return 0, err
	}
	return out.ID, nil
}

// SendCampaignNow sends a saved campaign to its lists straight away.
func (c *Client) SendCampaignNow(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/emailCampaigns/%d/sendNow", id), nil, nil)
}

// SendCampaignTest sends a saved campaign to a few addresses first, to check it.
// Brevo only delivers tests to addresses already in the account's contacts.
func (c *Client) SendCampaignTest(ctx context.Context, id int64, emails ...string) error {
	if len(emails) == 0 {
		return errors.New("mailer: say who the test goes to")
	}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/emailCampaigns/%d/sendTest", id),
		map[string]any{"emailTo": emails}, nil)
}
