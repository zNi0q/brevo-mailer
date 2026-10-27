package mailer

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

type call struct {
	Method string
	Path   string
	Key    string
	Body   map[string]any
}

// fakeBrevo records every request and answers with the status and body given.
func fakeBrevo(t *testing.T, status int, reply string) (*Client, *[]call) {
	t.Helper()
	var calls []call
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		c := call{Method: r.Method, Path: r.URL.Path, Key: r.Header.Get("api-key")}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &c.Body); err != nil {
				t.Errorf("request body is not JSON: %s", raw)
			}
		}
		calls = append(calls, c)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	cl, err := New(Config{APIKey: "xkeysib-test", FromEmail: "hello@example.com", FromName: "Example",
		ReplyTo: "support@example.com", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	return cl, &calls
}

func TestNewNeedsKeyAndSender(t *testing.T) {
	for _, cfg := range []Config{
		{FromEmail: "a@b.c", FromName: "A"},
		{APIKey: "k", FromName: "A"},
		{APIKey: "k", FromEmail: "a@b.c"},
	} {
		if _, err := New(cfg); err == nil {
			t.Errorf("New(%+v) accepted an incomplete config", cfg)
		}
	}
}

func TestSend(t *testing.T) {
	cl, calls := fakeBrevo(t, http.StatusCreated, `{"messageId":"<abc@smtp-relay.mailin.fr>"}`)
	id, err := cl.Send(context.Background(), Message{
		To:      []Address{{Email: "ada@example.com", Name: "Ada"}},
		Subject: "Welcome",
		HTML:    `<p>Hello <a href="https://example.com">there</a></p>`,
		Tags:    []string{"welcome"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "<abc@smtp-relay.mailin.fr>" {
		t.Errorf("message id = %q", id)
	}
	c := (*calls)[0]
	if c.Method != "POST" || c.Path != "/smtp/email" || c.Key != "xkeysib-test" {
		t.Errorf("sent %s %s with key %q", c.Method, c.Path, c.Key)
	}
	if c.Body["sender"].(map[string]any)["email"] != "hello@example.com" ||
		c.Body["replyTo"].(map[string]any)["email"] != "support@example.com" {
		t.Errorf("sender or reply-to wrong: %v", c.Body)
	}
	if c.Body["textContent"] != "Hello there (https://example.com)" {
		t.Errorf("text version = %q", c.Body["textContent"])
	}
}

func TestSendWithABrevoTemplate(t *testing.T) {
	cl, calls := fakeBrevo(t, http.StatusCreated, `{"messageId":"m"}`)
	_, err := cl.Send(context.Background(), Message{
		To: []Address{{Email: "ada@example.com"}}, TemplateID: 7,
		Params: map[string]any{"first_name": "Ada"},
	})
	if err != nil {
		t.Fatal(err)
	}
	b := (*calls)[0].Body
	if b["templateId"] != float64(7) || b["params"].(map[string]any)["first_name"] != "Ada" {
		t.Errorf("template send body = %v", b)
	}
	if _, ok := b["htmlContent"]; ok {
		t.Error("a template send should not carry HTML")
	}
}

func TestSendRefusesBadMessagesBeforeCallingBrevo(t *testing.T) {
	cl, calls := fakeBrevo(t, http.StatusCreated, `{}`)
	ctx := context.Background()
	bad := []Message{
		{Subject: "s", HTML: "h"},
		{To: []Address{{Email: "not-an-email"}}, Subject: "s", HTML: "h"},
		{To: []Address{{Email: "a@b.c"}}, Subject: "s"},
	}
	for _, m := range bad {
		if _, err := cl.Send(ctx, m); err == nil {
			t.Errorf("sent %+v", m)
		}
	}
	if len(*calls) != 0 {
		t.Errorf("called Brevo %d times for messages that could not be sent", len(*calls))
	}
}

func TestAPIErrorsCarryBrevosReason(t *testing.T) {
	cl, _ := fakeBrevo(t, http.StatusUnauthorized, `{"code":"unauthorized","message":"Key not found"}`)
	_, err := cl.Send(context.Background(), Message{To: []Address{{Email: "a@b.c"}}, Subject: "s", HTML: "h"})
	var e *APIError
	if !asAPIError(err, &e) || e.Status != 401 || e.Code != "unauthorized" || e.Message != "Key not found" {
		t.Fatalf("err = %v", err)
	}
}

func TestUpsertContact(t *testing.T) {
	cl, calls := fakeBrevo(t, http.StatusNoContent, ``)
	err := cl.UpsertContact(context.Background(), Contact{
		Email: "ada@example.com", Attributes: map[string]any{"FIRSTNAME": "Ada", "PLAN": "starter"},
		ListIDs: []int64{3},
	})
	if err != nil {
		t.Fatal(err)
	}
	b := (*calls)[0].Body
	if (*calls)[0].Path != "/contacts" || b["updateEnabled"] != true ||
		b["attributes"].(map[string]any)["PLAN"] != "starter" || b["listIds"].([]any)[0] != float64(3) {
		t.Errorf("upsert body = %v", b)
	}
}

func TestDeletingAnUnknownContactIsFine(t *testing.T) {
	cl, calls := fakeBrevo(t, http.StatusNotFound, `{"code":"document_not_found","message":"Contact does not exist"}`)
	if err := cl.DeleteContact(context.Background(), "gone@example.com"); err != nil {
		t.Fatal(err)
	}
	if (*calls)[0].Method != "DELETE" || (*calls)[0].Path != "/contacts/gone@example.com" {
		t.Errorf("called %s %s", (*calls)[0].Method, (*calls)[0].Path)
	}
}

func TestCampaign(t *testing.T) {
	cl, calls := fakeBrevo(t, http.StatusCreated, `{"id":42}`)
	ctx := context.Background()
	when := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	id, err := cl.CreateCampaign(ctx, Campaign{
		Name: "November launch", Subject: "Something new", HTML: "<p>Hi</p>",
		PreviewText: "Look inside", ListIDs: []int64{3, 4}, ScheduledAt: when,
	})
	if err != nil || id != 42 {
		t.Fatalf("id %d, err %v", id, err)
	}
	b := (*calls)[0].Body
	if b["scheduledAt"] != "2026-11-02T09:00:00Z" || b["replyTo"] != "support@example.com" ||
		len(b["recipients"].(map[string]any)["listIds"].([]any)) != 2 {
		t.Errorf("campaign body = %v", b)
	}
	if err := cl.SendCampaignNow(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if (*calls)[1].Path != "/emailCampaigns/42/sendNow" {
		t.Errorf("sendNow path = %s", (*calls)[1].Path)
	}
	if err := cl.SendCampaignTest(ctx, 42, "me@example.com"); err != nil {
		t.Fatal(err)
	}
	if (*calls)[2].Path != "/emailCampaigns/42/sendTest" ||
		(*calls)[2].Body["emailTo"].([]any)[0] != "me@example.com" {
		t.Errorf("sendTest = %+v", (*calls)[2])
	}
	if _, err := cl.CreateCampaign(ctx, Campaign{Name: "x", Subject: "y", HTML: "z"}); err == nil {
		t.Error("created a campaign with no list to send to")
	}
}

func asAPIError(err error, target **APIError) bool {
	e, ok := err.(*APIError)
	if ok {
		*target = e
	}
	return ok
}

func TestIsDuplicate(t *testing.T) {
	if !IsDuplicate(&APIError{Status: 400, Code: "duplicate_parameter"}) || IsDuplicate(nil) ||
		IsDuplicate(&APIError{Status: 400, Code: "invalid_parameter"}) {
		t.Error("IsDuplicate misread an error")
	}
	_ = strings.TrimSpace
}
