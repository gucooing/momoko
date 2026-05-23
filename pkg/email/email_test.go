package email

import (
	"strings"
	"testing"
)

func TestBuildMessage(t *testing.T) {
	client, err := NewClient(Config{
		Host:     "smtp.example.com",
		From:     "noreply@example.com",
		FromName: "Momoko",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := client.buildMessage(Message{
		Subject:  "hello {{.Name}}",
		Template: "<b>hello {{.Name}}</b>",
		Data: map[string]string{
			"Name": "Momoko",
		},
		Recipient: "user@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw := string(data)
	for _, want := range []string{
		"From: \"Momoko\" <noreply@example.com>",
		"To: user@example.com",
		"Subject: hello Momoko",
		"Content-Type: text/html; charset=UTF-8",
		"\r\n\r\n<b>hello Momoko</b>",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("message does not contain %q:\n%s", want, raw)
		}
	}
}

func TestBuildMessageRejectsSubjectInjection(t *testing.T) {
	client, err := NewClient(Config{
		Host: "smtp.example.com",
		From: "noreply@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.buildMessage(Message{
		Subject:  "hello {{.Name}}",
		Template: "body",
		Data: map[string]string{
			"Name": "\r\nBcc: attacker@example.com",
		},
		Recipient: "user@example.com",
	})
	if err == nil {
		t.Fatal("expected subject injection to fail")
	}
}

func TestBuildMessageEscapesHTMLData(t *testing.T) {
	client, err := NewClient(Config{
		Host: "smtp.example.com",
		From: "noreply@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := client.buildMessage(Message{
		Subject:  "hello",
		Template: "<p>{{.Name}}</p>",
		Data: map[string]string{
			"Name": "<script>alert(1)</script>",
		},
		Recipient: "user@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw := string(data)
	if strings.Contains(raw, "<script>") {
		t.Fatalf("template data was not escaped:\n%s", raw)
	}
	if !strings.Contains(raw, "&lt;script&gt;") {
		t.Fatalf("escaped template data was not found:\n%s", raw)
	}
}
