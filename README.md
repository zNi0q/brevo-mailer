# brevo-mailer

Send email from a Go service through [Brevo](https://www.brevo.com): one-off
transactional messages, contacts and lists, and marketing campaigns — all in one
branded HTML layout, from templates you can keep in a database and edit in an ops
dashboard.

- Standard library only. It calls Brevo's REST API directly, no generated SDK.
- One layout for every email: table-based and inline-styled, so it holds up in Gmail,
  Outlook and Apple Mail. Every text field is escaped.
- Templates with `{{placeholders}}` and fallbacks. A missing value is an error that
  names it, so nobody receives "Hi {{first_name}}".
- A plain-text version is derived from the HTML for you.

```sh
go get github.com/zNi0q/brevo-mailer
```

## Send a transactional email

```go
import mailer "github.com/zNi0q/brevo-mailer"

client, err := mailer.New(mailer.Config{
	APIKey:    os.Getenv("BREVO_API_KEY"),    // xkeysib-…
	FromEmail: os.Getenv("BREVO_FROM_EMAIL"), // a sender verified in Brevo
	FromName:  "Reeplot",
	ReplyTo:   "support@reeplot.com",         // optional
})

brand := mailer.Brand{
	Name:       "Reeplot",
	WebsiteURL: "https://reeplot.com",
	LogoURL:    "https://reeplot.com/email-logo.png", // optional
	Accent:     "#E5484D",
	Address:    "Freetown, Sierra Leone",              // shown in the footer
}

html := brand.Render(mailer.Content{
	Preheader:  "Your first video is two minutes away",
	Eyebrow:    "Welcome",
	Title:      "Let's make your first video",
	Greeting:   "Hi Ada,",
	Paragraphs: []string{"Drop a clip on the home page and say what you want."},
	Action:     &mailer.Action{Label: "Make a video", URL: "https://app.reeplot.com"},
	Notice:     "Didn't sign up? You can ignore this email.",
})

id, err := client.Send(ctx, mailer.Message{
	To:      []mailer.Address{{Email: "ada@example.com", Name: "Ada"}},
	Subject: "Welcome to Reeplot",
	HTML:    html,
	Tags:    []string{"welcome"},
})
```

To send a template you designed in Brevo's own editor instead, give its id:
`mailer.Message{To: …, TemplateID: 12, Params: map[string]any{"first_name": "Ada"}}`.

## Templates you edit in an ops dashboard

Store a `mailer.Template` per email — a database row, a JSON file — and let your
team edit the words without a deploy. Every text field may hold placeholders:

| Placeholder | Becomes |
| --- | --- |
| `{{first_name}}` | the value you pass; an error if it is missing |
| `{{first_name\|there}}` | the value, or `there` when it is empty |
| `{{ contact.FIRSTNAME }}` | left alone for Brevo to fill (campaigns) |

`Body` is plain text; a blank line starts a new paragraph.

```go
welcome := mailer.Template{
	Key:         "welcome",
	Subject:     "Welcome to Reeplot, {{first_name|there}}",
	Title:       "Your first video is two minutes away",
	Greeting:    "Hi {{first_name|there}},",
	Body:        "Drop a clip and say what you want.\n\nWe cut it and show you a preview.",
	ActionLabel: "Make a video",
	ActionURL:   "{{app_url}}/create",
}

welcome.Placeholders() // ["app_url" "first_name"] — show these in the dashboard

subject, content, err := welcome.Fill(map[string]string{
	"first_name": user.FirstName,
	"app_url":    "https://app.reeplot.com",
})
if err != nil { /* "template "welcome" is missing values for app_url" */ }

_, err = client.Send(ctx, mailer.Message{
	To: []mailer.Address{{Email: user.Email}}, Subject: subject, HTML: brand.Render(content),
})
```

A dashboard only needs a form with the Template fields, the list from
`Placeholders()`, and a preview pane showing `brand.Render(content)` filled with
sample values.

## Contacts and lists

Keep Brevo's audience in step with your users so campaigns can target them:

```go
err := client.UpsertContact(ctx, mailer.Contact{
	Email:      user.Email,
	Attributes: map[string]any{"FIRSTNAME": user.FirstName, "PLAN": "starter"},
	ListIDs:    []int64{3}, // e.g. "All customers"
})

err = client.DeleteContact(ctx, user.Email) // on account deletion; unknown is fine
```

Call `UpsertContact` on sign-up and whenever an attribute you segment on changes
(plan, country, last active).

## Marketing campaigns

```go
html := brand.Render(content) // Brand.UnsubscribeURL: "{{ unsubscribe }}" for campaigns

id, err := client.CreateCampaign(ctx, mailer.Campaign{
	Name:        "November launch",       // internal
	Subject:     "Something new in Reeplot",
	PreviewText: "Drop a video, get an ad",
	HTML:        html,
	ListIDs:     []int64{3},
	// ScheduledAt: time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC), // or leave as a draft
})

err = client.SendCampaignTest(ctx, id, "you@example.com") // check it first
err = client.SendCampaignNow(ctx, id)
```

Brevo requires marketing email to carry an unsubscribe link and a postal address:
set `Brand.UnsubscribeURL` to `{{ unsubscribe }}` (Brevo fills it per recipient) and
`Brand.Address`.

## Errors

Brevo's refusals come back as `*mailer.APIError` with the HTTP status, Brevo's code
and its message. `mailer.IsDuplicate(err)` spots "already exists". Bad input — no
recipient, no subject, no list — is refused before any request is made.

## Try it

```sh
BREVO_API_KEY=xkeysib-… BREVO_FROM_EMAIL=hello@example.com BREVO_FROM_NAME="Example" \
  go run ./cmd/sendtest you@example.com
```

The sender address must be verified in Brevo (Senders, Domains & Dedicated IPs).

## Tests

```sh
go test ./...
```

They run against a fake Brevo server, so no key is needed.

## License

MIT
