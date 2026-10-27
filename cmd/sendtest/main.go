// Command sendtest sends one branded email through Brevo, to check a key and a
// sender before wiring the package into an app.
//
//	BREVO_API_KEY=xkeysib-… BREVO_FROM_EMAIL=hello@example.com BREVO_FROM_NAME=Example \
//	  go run ./cmd/sendtest you@example.com
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	mailer "github.com/zNi0q/brevo-mailer"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: sendtest you@example.com")
		os.Exit(2)
	}
	client, err := mailer.New(mailer.Config{
		APIKey:    os.Getenv("BREVO_API_KEY"),
		FromEmail: os.Getenv("BREVO_FROM_EMAIL"),
		FromName:  os.Getenv("BREVO_FROM_NAME"),
		ReplyTo:   os.Getenv("BREVO_REPLY_TO"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	brand := mailer.Brand{
		Name:       os.Getenv("BREVO_FROM_NAME"),
		WebsiteURL: os.Getenv("BRAND_URL"),
		Accent:     "#111111",
	}
	welcome := mailer.Template{
		Key:         "welcome",
		Subject:     "It works, {{first_name|friend}}",
		Eyebrow:     "Test",
		Title:       "Your Brevo setup works",
		Greeting:    "Hi {{first_name|there}},",
		Body:        "This email came through the Brevo API with your key and sender.\n\nSent at {{sent_at}}.",
		ActionLabel: "Open Brevo",
		ActionURL:   "https://app.brevo.com",
	}
	subject, content, err := welcome.Fill(map[string]string{"sent_at": time.Now().Format(time.RFC1123)})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	id, err := client.Send(ctx, mailer.Message{
		To:      []mailer.Address{{Email: os.Args[1]}},
		Subject: subject,
		HTML:    brand.Render(content),
		Tags:    []string{"sendtest"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("sent:", id)
}
