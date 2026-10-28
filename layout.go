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
	// LogoURL is a hosted square PNG, shown at 40px beside the name, centred above the
	// card. Optional.
	LogoURL    string
	WebsiteURL string
	// Accent colours the button, as a hex colour. Defaults to near-black.
	Accent string
	// Address is the postal address marketing email must carry in most countries,
	// shown in the footer after the name.
	Address string
	// SignOff closes every email, under the card's text: "Thank you,\nThe Acme team".
	// Defaults to that, with Name.
	SignOff string
	// Links sit in the footer: Terms of Use, Privacy Policy.
	Links []Link
	// Socials sit in the footer as small icons. IconURL is a hosted PNG.
	Socials []Social
	// UnsubscribeURL is shown in the footer of marketing email. Brevo campaigns can
	// use "{{ unsubscribe }}", which Brevo fills in per recipient.
	UnsubscribeURL string
}

type Link struct {
	Label string
	URL   string
}

type Social struct {
	Label   string
	URL     string
	IconURL string
}

// Action is the one button in an email.
type Action struct {
	Label string
	URL   string
}

// Content is what one email says. Every field but Title is optional.
//
// Text fields are plain text: they are escaped, line breaks are kept, **double
// stars** make bold, and email addresses become mailto links.
type Content struct {
	// Preheader is the grey line inbox apps show after the subject.
	Preheader string
	// Eyebrow is a short label above the greeting: "Security notice".
	Eyebrow string
	// Title names the email (the page title) and, under a greeting, is its bold
	// opening line. With no greeting it is the heading.
	Title string
	// Greeting is the heading: "Hi **Ada**,".
	Greeting   string
	Paragraphs []string
	Action     *Action
	// Notice is a quieter line under the button: "If this wasn't you, write to …".
	Notice string
	// Footer is a last line above the brand footer: why they got this email.
	Footer string
}

const (
	panelBG  = "#ebeef1"
	cardEdge = "#e1e5ea"
	ink      = "#111111"
	body     = "#222222"
	quiet    = "#6b7280"
	fontList = "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif"
)

// Render turns content into a complete HTML email: a grey panel with the logo on
// top, a white card for the message, the brand footer beneath. Table-based and
// inline-styled so it holds up in Gmail, Outlook and Apple Mail.
func (b Brand) Render(c Content) string {
	accent := b.Accent
	if !hexColour.MatchString(accent) {
		accent = ink
	}
	esc := html.EscapeString

	var card strings.Builder
	if c.Eyebrow != "" {
		fmt.Fprintf(&card, `<p style="margin:0 0 12px;font-size:12px;line-height:16px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:%s">%s</p>`, quiet, esc(c.Eyebrow))
	}
	heading, lead := c.Greeting, c.Title
	if strings.TrimSpace(heading) == "" {
		heading, lead = c.Title, ""
	}
	fmt.Fprintf(&card, `<h1 style="margin:0 0 22px;font-size:30px;line-height:38px;font-weight:400;color:%s">%s</h1>`, ink, rich(heading))
	if lead != "" {
		fmt.Fprintf(&card, `<p style="margin:0 0 14px;font-size:17px;line-height:26px;font-weight:600;color:%s">%s</p>`, ink, rich(lead))
	}
	for _, p := range c.Paragraphs {
		if strings.TrimSpace(p) == "" {
			continue
		}
		fmt.Fprintf(&card, `<p style="margin:0 0 16px;font-size:16px;line-height:26px;color:%s">%s</p>`, body, rich(p))
	}
	if c.Action != nil && c.Action.Label != "" && c.Action.URL != "" {
		fmt.Fprintf(&card, `<table role="presentation" cellspacing="0" cellpadding="0" border="0" align="center" style="margin:30px auto 32px"><tr><td align="center" style="border-radius:6px;background:%s"><a href="%s" style="display:inline-block;padding:15px 34px;font-size:15px;line-height:18px;font-weight:700;letter-spacing:.04em;text-transform:uppercase;color:#ffffff;text-decoration:none;border-radius:6px">%s</a></td></tr></table>`,
			accent, esc(c.Action.URL), esc(c.Action.Label))
	}
	if c.Notice != "" {
		fmt.Fprintf(&card, `<p style="margin:0 0 16px;font-size:16px;line-height:26px;color:%s">%s</p>`, quiet, rich(c.Notice))
	}
	signOff := b.SignOff
	if signOff == "" {
		signOff = "Thank you,\nThe " + b.Name + " team"
	}
	fmt.Fprintf(&card, `<p style="margin:24px 0 0;font-size:16px;line-height:26px;color:%s">%s</p>`, quiet, rich(signOff))

	// The header is the mark and the name side by side, centred. The name is text, so
	// the header still reads when an inbox blocks images; the mark then has no alt text
	// to show in its place.
	link := func(inner string) string {
		if b.WebsiteURL == "" {
			return inner
		}
		return fmt.Sprintf(`<a href="%s" style="color:%s;text-decoration:none">%s</a>`, esc(b.WebsiteURL), ink, inner)
	}
	name := fmt.Sprintf(`<span style="font-size:22px;line-height:40px;font-weight:700;letter-spacing:-.02em;color:%s">%s</span>`, ink, esc(b.Name))
	head := link(name)
	if b.LogoURL != "" {
		mark := fmt.Sprintf(`<img src="%s" alt="" width="40" height="40" style="display:block;border:0;width:40px;height:40px;border-radius:50%%">`, esc(b.LogoURL))
		head = fmt.Sprintf(`<table role="presentation" cellspacing="0" cellpadding="0" border="0" align="center"><tr><td style="padding-right:10px;vertical-align:middle">%s</td><td style="vertical-align:middle">%s</td></tr></table>`,
			link(mark), head)
	}

	var foot strings.Builder
	if c.Footer != "" {
		fmt.Fprintf(&foot, `<p style="margin:0 0 14px;font-size:13px;line-height:20px;color:%s">%s</p>`, quiet, rich(c.Footer))
	}
	line := "© " + esc(b.Name)
	if b.Address != "" {
		line += " | " + esc(b.Address)
	}
	fmt.Fprintf(&foot, `<p style="margin:0 0 12px;font-size:14px;line-height:20px;color:%s">%s</p>`, quiet, line)
	links := b.Links
	if b.UnsubscribeURL != "" {
		links = append(append([]Link{}, links...), Link{Label: "Unsubscribe", URL: b.UnsubscribeURL})
	}
	if len(links) > 0 {
		parts := make([]string, 0, len(links))
		for _, l := range links {
			href := esc(l.URL)
			if strings.TrimSpace(l.URL) == "{{ unsubscribe }}" {
				href = "{{ unsubscribe }}"
			}
			parts = append(parts, fmt.Sprintf(`<a href="%s" style="color:%s;text-decoration:underline">%s</a>`, href, quiet, esc(l.Label)))
		}
		fmt.Fprintf(&foot, `<p style="margin:0 0 12px;font-size:14px;line-height:20px;color:%s">%s</p>`, quiet,
			strings.Join(parts, `<span style="display:inline-block;width:22px"></span>`))
	}
	if len(b.Socials) > 0 {
		foot.WriteString(`<p style="margin:0">`)
		for _, s := range b.Socials {
			fmt.Fprintf(&foot, `<a href="%s" style="display:inline-block;margin:0 6px;text-decoration:none"><img src="%s" alt="%s" width="22" height="22" style="display:inline-block;border:0;width:22px;height:22px"></a>`,
				esc(s.URL), esc(s.IconURL), esc(s.Label))
		}
		foot.WriteString(`</p>`)
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><title>%s</title></head>
<body style="margin:0;padding:0;background:#ffffff;font-family:%s">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent">%s</div>
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background:#ffffff"><tr><td align="center" style="padding:32px 12px">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="max-width:640px;background:%s;border-radius:8px">
<tr><td align="center" style="padding:32px 20px 28px">%s</td></tr>
<tr><td align="center" style="padding:0 20px">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="max-width:480px;background:#ffffff;border:1px solid %s;border-radius:6px">
<tr><td style="padding:40px 40px 36px;text-align:left">%s</td></tr>
</table></td></tr>
<tr><td align="center" style="padding:28px 24px 40px;text-align:center">%s</td></tr>
</table></td></tr></table></body></html>`,
		esc(c.Title), fontList, esc(c.Preheader), panelBG, head, cardEdge, card.String(), foot.String())
}

var (
	hexColour = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}){1,2}$`)
	boldRun   = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	emailAddr = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
)

// rich escapes plain text, keeps its line breaks, turns **runs** bold and email
// addresses into mailto links. Nothing else becomes markup.
func rich(s string) string {
	out := html.EscapeString(strings.TrimSpace(s))
	out = boldRun.ReplaceAllString(out, `<strong style="font-weight:700">$1</strong>`)
	out = emailAddr.ReplaceAllStringFunc(out, func(addr string) string {
		return fmt.Sprintf(`<a href="mailto:%s" style="color:inherit;text-decoration:underline">%s</a>`, addr, addr)
	})
	return strings.ReplaceAll(out, "\n", "<br>")
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
	s = reAnchor.ReplaceAllStringFunc(s, func(a string) string {
		m := reAnchor.FindStringSubmatch(a)
		label := strings.TrimSpace(reTag.ReplaceAllString(m[2], ""))
		href := html.UnescapeString(m[1])
		if label == "" || strings.TrimPrefix(href, "mailto:") == html.UnescapeString(label) || href == label {
			return label
		}
		return label + " (" + m[1] + ")"
	})
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
