package mailer

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Template is an email someone writes once — in an ops dashboard, a database row,
// a config file — and the product fills in per send. Every text field may hold
// placeholders:
//
//	{{first_name}}          replaced from the values given to Fill
//	{{first_name|there}}    the same, with a fallback when the value is empty
//
// Body is plain text; a blank line starts a new paragraph. Brevo's own
// placeholders, such as {{ contact.FIRSTNAME }} or {{ unsubscribe }}, are left
// untouched, so the same template works for a campaign Brevo personalises itself.
type Template struct {
	Key         string `json:"key"`
	Subject     string `json:"subject"`
	Preheader   string `json:"preheader"`
	Eyebrow     string `json:"eyebrow"`
	Title       string `json:"title"`
	Greeting    string `json:"greeting"`
	Body        string `json:"body"`
	ActionLabel string `json:"action_label"`
	ActionURL   string `json:"action_url"`
	Notice      string `json:"notice"`
	Footer      string `json:"footer"`
}

var placeholder = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)(?:\|([^{}]*))?\}\}`)

// Placeholders lists the names a template uses, sorted, so a dashboard can show
// what each send must provide.
func (t Template) Placeholders() []string {
	seen := map[string]bool{}
	for _, f := range t.fields() {
		for _, m := range placeholder.FindAllStringSubmatch(*f, -1) {
			seen[m[1]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Fill replaces every placeholder and returns the subject and content to render.
// A placeholder with no value and no fallback is an error naming it, so an email
// never goes out saying "Hi {{first_name}}".
func (t Template) Fill(values map[string]string) (subject string, c Content, err error) {
	var missing []string
	fill := func(s string) string {
		return placeholder.ReplaceAllStringFunc(s, func(m string) string {
			parts := placeholder.FindStringSubmatch(m)
			if v := strings.TrimSpace(values[parts[1]]); v != "" {
				return v
			}
			if strings.Contains(m, "|") {
				return parts[2]
			}
			missing = append(missing, parts[1])
			return m
		})
	}

	subject = fill(t.Subject)
	c = Content{
		Preheader: fill(t.Preheader),
		Eyebrow:   fill(t.Eyebrow),
		Title:     fill(t.Title),
		Greeting:  fill(t.Greeting),
		Notice:    fill(t.Notice),
		Footer:    fill(t.Footer),
	}
	for _, p := range strings.Split(strings.ReplaceAll(t.Body, "\r\n", "\n"), "\n\n") {
		if p = strings.TrimSpace(fill(p)); p != "" {
			c.Paragraphs = append(c.Paragraphs, p)
		}
	}
	if label, url := fill(t.ActionLabel), fill(t.ActionURL); label != "" && url != "" {
		c.Action = &Action{Label: label, URL: url}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return "", Content{}, fmt.Errorf("mailer: template %q is missing values for %s",
			t.Key, strings.Join(dedupe(missing), ", "))
	}
	if subject == "" || c.Title == "" {
		return "", Content{}, fmt.Errorf("mailer: template %q needs a subject and a title", t.Key)
	}
	return subject, c, nil
}

func (t *Template) fields() []*string {
	return []*string{&t.Subject, &t.Preheader, &t.Eyebrow, &t.Title, &t.Greeting,
		&t.Body, &t.ActionLabel, &t.ActionURL, &t.Notice, &t.Footer}
}

func dedupe(s []string) []string {
	out := s[:0]
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}
