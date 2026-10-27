package mailer

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// Address is one sender or recipient.
type Address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Message is one transactional email: a receipt, a password reset, a welcome.
//
// Give either HTML (usually from Layout.Render) or a TemplateID of a template kept
// in Brevo, whose {{ params.x }} placeholders are filled from Params.
type Message struct {
	To      []Address
	Subject string
	HTML    string
	// Text is the plain-text version. When empty and HTML is set, one is derived.
	Text string

	TemplateID int64
	Params     map[string]any

	// Tags label the message in Brevo's logs and webhooks, e.g. "welcome".
	Tags []string
	// Headers are extra email headers, e.g. "List-Unsubscribe".
	Headers map[string]string
}

type wireMessage struct {
	Sender      Address           `json:"sender"`
	To          []Address         `json:"to"`
	ReplyTo     *Address          `json:"replyTo,omitempty"`
	Subject     string            `json:"subject,omitempty"`
	HTMLContent string            `json:"htmlContent,omitempty"`
	TextContent string            `json:"textContent,omitempty"`
	TemplateID  int64             `json:"templateId,omitempty"`
	Params      map[string]any    `json:"params,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// Send delivers one transactional email and returns Brevo's message id.
func (c *Client) Send(ctx context.Context, m Message) (string, error) {
	if len(m.To) == 0 {
		return "", errors.New("mailer: a message needs at least one recipient")
	}
	for _, to := range m.To {
		if !strings.Contains(to.Email, "@") {
			return "", errors.New("mailer: " + to.Email + " is not an email address")
		}
	}
	if m.TemplateID == 0 && (m.Subject == "" || m.HTML == "") {
		return "", errors.New("mailer: a message needs a Subject and HTML, or a TemplateID")
	}

	w := wireMessage{
		Sender:      Address{Email: c.cfg.FromEmail, Name: c.cfg.FromName},
		To:          m.To,
		Subject:     m.Subject,
		HTMLContent: m.HTML,
		TextContent: m.Text,
		TemplateID:  m.TemplateID,
		Params:      m.Params,
		Tags:        m.Tags,
		Headers:     m.Headers,
	}
	if w.TextContent == "" && w.HTMLContent != "" {
		w.TextContent = PlainText(w.HTMLContent)
	}
	if c.cfg.ReplyTo != "" {
		w.ReplyTo = &Address{Email: c.cfg.ReplyTo}
	}

	var out struct {
		MessageID string `json:"messageId"`
	}
	if err := c.do(ctx, http.MethodPost, "/smtp/email", w, &out); err != nil {
		return "", err
	}
	return out.MessageID, nil
}
