package mailer

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Brand is how every email looks: one layout, your name, logo and colour.
type Brand struct {
	Name string
	// LogoURL is a hosted image, ideally a PNG about 120px wide. Optional.
	LogoURL    string
	WebsiteURL string
	// Accent colours the button and the eyebrow line, as a hex colour.
	Accent string
	// Address is the postal address marketing email must carry in most countries.
	Address string
	// UnsubscribeURL is shown in the footer of marketing email. Brevo campaigns can
	// use "{{ unsubscribe }}", which Brevo fills in per recipient.
	UnsubscribeURL string
}

// Action is the one button in an email.
type Action struct {
	Label string
	URL   string
}

// Content is what one email says. Every field but Title is optional.
type Content struct {
	// Preheader is the grey line inbox apps show after the subject.
	Preheader string
	// Eyebrow is a short label above the title: "Welcome", "Security notice".
	Eyebrow  string
	Title    string
	Greeting string
	// Paragraphs are plain text; line breaks inside one are kept.
	Paragraphs []string
	Action     *Action
	// Notice is a quieter boxed line under the button: "If this wasn't you…".
	Notice string
	// Footer is a last line above the brand footer: why they got this email.
	Footer string
}

// Render turns content into a complete HTML email: table-based and inline-styled
// so it holds up in Gmail, Outlook and Apple Mail, and every text field escaped.
func (b Brand) Render(c Content) string {
	accent := b.Accent
	if !hexColour.MatchString(accent) {
		accent = "#111111"
	}
	esc := html.EscapeString

	var body strings.Builder
	if c.Eyebrow != "" {
		fmt.Fprintf(&body, `<p style="margin:0 0 10px;font-size:12px;line-height:16px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:%s">%s</p>`, accent, esc(c.Eyebrow))
	}
	fmt.Fprintf(&body, `<h1 style="margin:0 0 18px;font-size:24px;line-height:30px;font-weight:700;color:#111111">%s</h1>`, esc(c.Title))
	if c.Greeting != "" {
		fmt.Fprintf(&body, `<p style="margin:0 0 14px;font-size:15px;line-height:24px;color:#2b2b2b">%s</p>`, esc(c.Greeting))
	}
	for _, p := range c.Paragraphs {
		if strings.TrimSpace(p) == "" {
			continue
		}
		fmt.Fprintf(&body, `<p style="margin:0 0 14px;font-size:15px;line-height:24px;color:#2b2b2b">%s</p>`,
			strings.ReplaceAll(esc(strings.TrimSpace(p)), "\n", "<br>"))
	}
	if c.Action != nil && c.Action.Label != "" && c.Action.URL != "" {
		fmt.Fprintf(&body, `<table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin:24px 0 8px"><tr><td style="border-radius:8px;background:%s"><a href="%s" style="display:inline-block;padding:13px 22px;font-size:15px;line-height:18px;font-weight:700;color:#ffffff;text-decoration:none;border-radius:8px">%s</a></td></tr></table>`,
			accent, esc(c.Action.URL), esc(c.Action.Label))
	}
	if c.Notice != "" {
		fmt.Fprintf(&body, `<p style="margin:22px 0 0;padding:12px 14px;border-radius:8px;background:#f4f4f2;font-size:13px;line-height:20px;color:#555555">%s</p>`, esc(c.Notice))
	}
	if c.Footer != "" {
		fmt.Fprintf(&body, `<p style="margin:22px 0 0;font-size:13px;line-height:20px;color:#777777">%s</p>`, esc(c.Footer))
	}

	head := esc(b.Name)
	if b.LogoURL != "" {
		head = fmt.Sprintf(`<img src="%s" alt="%s" width="120" style="display:block;border:0;max-width:120px;height:auto">`, esc(b.LogoURL), esc(b.Name))
	}
	if b.WebsiteURL != "" {
		head = fmt.Sprintf(`<a href="%s" style="color:#111111;text-decoration:none;font-size:18px;font-weight:700">%s</a>`, esc(b.WebsiteURL), head)
	}

	var foot strings.Builder
	fmt.Fprintf(&foot, `<p style="margin:0;font-size:12px;line-height:18px;color:#8a8a8a">%s`, esc(b.Name))
	if b.Address != "" {
		fmt.Fprintf(&foot, ` · %s`, esc(b.Address))
	}
	foot.WriteString(`</p>`)
	if b.UnsubscribeURL != "" {
		fmt.Fprintf(&foot, `<p style="margin:6px 0 0;font-size:12px;line-height:18px"><a href="%s" style="color:#8a8a8a;text-decoration:underline">Unsubscribe</a></p>`, unsubscribeHref(b.UnsubscribeURL))
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><title>%s</title></head>
<body style="margin:0;padding:0;background:#f4f4f2;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent">%s</div>
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background:#f4f4f2"><tr><td align="center" style="padding:32px 16px">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="max-width:560px">
<tr><td style="padding:0 4px 18px">%s</td></tr>
<tr><td style="padding:32px 32px 28px;background:#ffffff;border-radius:14px">%s</td></tr>
<tr><td style="padding:18px 4px 0">%s</td></tr>
</table></td></tr></table></body></html>`,
		esc(c.Title), esc(c.Preheader), head, body.String(), foot.String())
}

var hexColour = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}){1,2}$`)

// unsubscribeHref keeps Brevo's own per-recipient placeholder as it is, and
// escapes anything else.
func unsubscribeHref(u string) string {
	if strings.TrimSpace(u) == "{{ unsubscribe }}" {
		return "{{ unsubscribe }}"
	}
	return html.EscapeString(u)
}

var (
	reBreak  = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</h[1-6]>|</tr>|</div>`)
	reAnchor = regexp.MustCompile(`(?is)<a\s[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	reTag    = regexp.MustCompile(`(?s)<[^>]+>`)
	reHidden = regexp.MustCompile(`(?is)<(head|style|script)[^>]*>.*?</(head|style|script)>|<div style="display:none[^"]*">.*?</div>`)
	reSpaces = regexp.MustCompile(`[ \t]+`)
	reLines  = regexp.MustCompile(`\n\s*\n+`)
)

// PlainText derives a readable text version of an HTML email, for the inboxes and
// spam filters that look for one. Links become "label (url)".
func PlainText(h string) string {
	s := reHidden.ReplaceAllString(h, "")
	s = reAnchor.ReplaceAllString(s, "$2 ($1)")
	s = reBreak.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = reSpaces.ReplaceAllString(s, " ")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	s = strings.Join(lines, "\n")
	return strings.TrimSpace(reLines.ReplaceAllString(s, "\n\n"))
}
