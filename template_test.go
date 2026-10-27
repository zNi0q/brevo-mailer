package mailer

import (
	"reflect"
	"strings"
	"testing"
)

var welcome = Template{
	Key:         "welcome",
	Subject:     "Welcome to Reeplot, {{first_name|there}}",
	Title:       "Your first video is two minutes away",
	Greeting:    "Hi {{first_name|there}},",
	Body:        "Drop a clip on the home page and say what you want.\n\nWe cut it, caption it and show you a preview.",
	ActionLabel: "Make a video",
	ActionURL:   "{{app_url}}/create",
	Footer:      "You're getting this because you signed up with {{email}}. Hi {{ contact.FIRSTNAME }}.",
}

func TestFill(t *testing.T) {
	subject, c, err := welcome.Fill(map[string]string{
		"first_name": "Ada", "app_url": "https://app.reeplot.com", "email": "ada@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Welcome to Reeplot, Ada" || c.Greeting != "Hi Ada," {
		t.Errorf("subject %q, greeting %q", subject, c.Greeting)
	}
	if len(c.Paragraphs) != 2 || c.Action == nil || c.Action.URL != "https://app.reeplot.com/create" {
		t.Errorf("content = %+v", c)
	}
	if !strings.Contains(c.Footer, "{{ contact.FIRSTNAME }}") {
		t.Error("Brevo's own placeholder should be left for Brevo")
	}
}

func TestFillUsesTheFallback(t *testing.T) {
	subject, _, err := welcome.Fill(map[string]string{"app_url": "https://a.test", "email": "x@y.z", "first_name": "  "})
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Welcome to Reeplot, there" {
		t.Errorf("subject = %q", subject)
	}
}

func TestFillNamesWhatIsMissing(t *testing.T) {
	_, _, err := welcome.Fill(map[string]string{"first_name": "Ada"})
	if err == nil || !strings.Contains(err.Error(), "app_url, email") {
		t.Fatalf("err = %v, want it to name app_url and email", err)
	}
}

func TestPlaceholders(t *testing.T) {
	got := welcome.Placeholders()
	if want := []string{"app_url", "email", "first_name"}; !reflect.DeepEqual(got, want) {
		t.Errorf("placeholders = %v, want %v", got, want)
	}
}

func TestFillNeedsASubjectAndTitle(t *testing.T) {
	if _, _, err := (Template{Key: "empty", Body: "x"}).Fill(nil); err == nil {
		t.Error("filled a template with no subject or title")
	}
}
