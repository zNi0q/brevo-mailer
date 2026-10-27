package mailer

import (
	"strings"
	"testing"
)

var brand = Brand{Name: "Reeplot", WebsiteURL: "https://reeplot.com", Accent: "#E5484D",
	Address: "Freetown, Sierra Leone", UnsubscribeURL: "{{ unsubscribe }}"}

func TestRenderEscapesEveryField(t *testing.T) {
	out := brand.Render(Content{
		Title:      `Hi <script>alert(1)</script>`,
		Paragraphs: []string{"Tom & Jerry\nsecond line"},
		Action:     &Action{Label: "Open", URL: `https://x.test/?a=1&b="2"`},
	})
	if strings.Contains(out, "<script>") {
		t.Error("the title was not escaped")
	}
	if !strings.Contains(out, "Tom &amp; Jerry<br>second line") {
		t.Error("a paragraph lost its escaping or its line break")
	}
	if !strings.Contains(out, `href="https://x.test/?a=1&amp;b=&#34;2&#34;"`) {
		t.Error("the button link was not escaped")
	}
}

func TestRenderKeepsBrevosUnsubscribeTag(t *testing.T) {
	out := brand.Render(Content{Title: "News"})
	if !strings.Contains(out, `href="{{ unsubscribe }}"`) {
		t.Error("Brevo's unsubscribe placeholder was mangled")
	}
	if !strings.Contains(out, "Freetown, Sierra Leone") {
		t.Error("the postal address is missing from the footer")
	}
}

func TestRenderFallsBackToABlackButtonOnABadColour(t *testing.T) {
	out := Brand{Name: "X", Accent: "red;}body{display:none"}.Render(Content{Title: "t", Action: &Action{Label: "Go", URL: "https://x.test"}})
	if strings.Contains(out, "display:none;}") || !strings.Contains(out, "background:#111111") {
		t.Error("an unsafe accent colour reached the styles")
	}
}

func TestPlainText(t *testing.T) {
	out := PlainText(brand.Render(Content{
		Preheader: "hidden preview", Title: "Welcome", Paragraphs: []string{"First.", "Second."},
		Action: &Action{Label: "Start", URL: "https://reeplot.com/start"},
	}))
	for _, want := range []string{"Welcome", "First.", "Second.", "Start (https://reeplot.com/start)"} {
		if !strings.Contains(out, want) {
			t.Errorf("text version is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hidden preview") || strings.Contains(out, "<") {
		t.Errorf("text version kept the preheader or markup:\n%s", out)
	}
}
