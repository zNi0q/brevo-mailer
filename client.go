// Package mailer sends email through Brevo (formerly Sendinblue): one-off
// transactional messages, contacts and lists, and marketing campaigns, all wrapped
// in a single branded HTML layout so every email a product sends looks like it came
// from the same place.
//
// It talks to Brevo's REST API (https://developers.brevo.com/reference) with the
// standard library only.
package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is Brevo's v3 API.
const DefaultBaseURL = "https://api.brevo.com/v3"

// Config is everything a Client needs. APIKey, FromEmail and FromName are required.
type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
	// ReplyTo, when set, is where replies go instead of FromEmail.
	ReplyTo string

	// BaseURL overrides the API endpoint; tests point it at an httptest server.
	BaseURL string
	// HTTPClient overrides the client used for requests. Defaults to a 20s timeout.
	HTTPClient *http.Client
}

// Client sends through one Brevo account.
type Client struct {
	cfg  Config
	base string
	http *http.Client
}

// New checks the config and returns a Client.
func New(cfg Config) (*Client, error) {
	switch {
	case strings.TrimSpace(cfg.APIKey) == "":
		return nil, errors.New("mailer: Config.APIKey is required")
	case strings.TrimSpace(cfg.FromEmail) == "":
		return nil, errors.New("mailer: Config.FromEmail is required")
	case strings.TrimSpace(cfg.FromName) == "":
		return nil, errors.New("mailer: Config.FromName is required")
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{cfg: cfg, base: base, http: hc}, nil
}

// APIError is a non-2xx answer from Brevo.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("brevo: %d %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("brevo: %d: %s", e.Status, e.Message)
}

// IsDuplicate reports whether err is Brevo refusing to create something that
// already exists, such as a contact.
func IsDuplicate(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.Code == "duplicate_parameter"
}

// do sends one request and decodes a JSON answer into out when out is not nil.
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("mailer: encoding request: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("api-key", c.cfg.APIKey)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("mailer: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		e := &APIError{Status: resp.StatusCode}
		var wire struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &wire) == nil {
			e.Code, e.Message = wire.Code, wire.Message
		}
		if e.Message == "" {
			e.Message = strings.TrimSpace(string(raw))
		}
		if e.Message == "" {
			e.Message = http.StatusText(resp.StatusCode)
		}
		return e
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("mailer: decoding %s %s: %w", method, path, err)
		}
	}
	return nil
}
