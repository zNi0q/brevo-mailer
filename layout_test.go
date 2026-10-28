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

func TestRenderMarksUpBoldAndEmailsOnly(t *testing.T) {
	out := brand.Render(Content{
		Title: "Reset", Greeting: "Hi **Ada**,",
		Notice: "Not you? Write to help@reeplot.com <b>now</b>",
	})
	if !strings.Contains(out, `Hi <strong style="font-weight:700">Ada</strong>,`) {
		t.Error("the greeting lost its bold name")
	}
	if !strings.Contains(out, `href="mailto:help@reeplot.com"`) {
		t.Error("the email address in the notice was not linked")
	}
	if strings.Contains(out, "<b>") {
		t.Error("markup in the text was not escaped")
	}
}

func TestRenderFooterAndSignOff(t *testing.T) {
	b := brand
	b.Links = []Link{{Label: "Terms of Use", URL: "https://reeplot.com/legal/terms"}}
	b.Socials = []Social{{Label: "TikTok", URL: "https://tiktok.com/@reeplot", IconURL: "https://reeplot.com/t.png"}}
	out := b.Render(Content{Title: "t"})
	for _, want := range []string{
		"Thank you,<br>The Reeplot team",
		"© Reeplot | Freetown, Sierra Leone",
		`href="https://reeplot.com/legal/terms"`,
		`alt="TikTok"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render is missing %q", want)
		}
	}
	b.SignOff = "Cheers,\nAda"
	if !strings.Contains(b.Render(Content{Title: "t"}), "Cheers,<br>Ada") {
		t.Error("a custom sign-off was ignored")
	}
}
