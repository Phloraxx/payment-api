package paymentemail

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

const testRawEmail = "From: slice <noreply@slice.bank.in>\r\n" +
	"To: payments@example.org\r\n" +
	"Message-ID: <slice-event-1@example>\r\n" +
	"Date: Sat, 25 Jul 2026 17:31:00 +0530\r\n" +
	"Subject: Received =?UTF-8?Q?=E2=82=B9100=2E01?= in your slice account\r\n" +
	"Authentication-Results: mx.cloudflare.net; dkim=pass header.d=slice.bank.in; dmarc=pass header.from=slice.bank.in\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/alternative; boundary=paygate\r\n\r\n" +
	"--paygate\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n" +
	"You have received =E2=82=B9100.01 via UPI from Alice alice@oksbi. UPI Ref: 606703736479\r\n" +
	"--paygate\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<p>fallback</p>\r\n" +
	"--paygate--\r\n"

func TestParseRawAndExtractSliceCredit(t *testing.T) {
	message, err := ParseRaw([]byte(testRawEmail))
	if err != nil {
		t.Fatal(err)
	}
	if message.MessageID != "slice-event-1@example" || message.From != "noreply@slice.bank.in" {
		t.Fatalf("headers = %+v", message)
	}
	if message.Subject != "Received ₹100.01 in your slice account" || !strings.Contains(message.Body, "606703736479") {
		t.Fatalf("decoded content: subject=%q body=%q", message.Subject, message.Body)
	}
	if !message.Date.Equal(time.Date(2026, 7, 25, 12, 1, 0, 0, time.UTC)) {
		t.Fatalf("date = %s", message.Date)
	}
	parsed, err := Parse(message)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AmountPaise != 10001 || parsed.RRN != "606703736479" || parsed.UPIId != "alice@oksbi" {
		t.Fatalf("parsed = %+v", parsed)
	}
}

func TestAuthenticatedSenderRequiresTrustedAlignedResult(t *testing.T) {
	valid := "mx.cloudflare.net; dkim=pass header.d=slice.bank.in; spf=pass smtp.mailfrom=slice.bank.in"
	if !AuthenticatedSender([]string{valid}, "mx.cloudflare.net", "slice.bank.in") {
		t.Fatal("valid aligned DKIM was rejected")
	}
	validWithComment := "mx.cloudflare.net; dkim=pass (2048-bit key; unprotected) header.d=slice.bank.in header.i=@slice.bank.in; spf=pass smtp.mailfrom=slice.bank.in"
	if !AuthenticatedSender([]string{validWithComment}, "mx.cloudflare.net", "slice.bank.in") {
		t.Fatal("valid aligned DKIM with a semicolon in its comment was rejected")
	}
	validDMARC := "mx.cloudflare.net; dmarc=pass header.from=slice.bank.in"
	if !AuthenticatedSender([]string{validDMARC}, "mx.cloudflare.net", "slice.bank.in") {
		t.Fatal("valid aligned DMARC was rejected")
	}
	for _, result := range []string{
		"attacker.example; dkim=pass header.d=slice.bank.in",
		"mx.cloudflare.net; dkim=fail header.d=slice.bank.in",
		"mx.cloudflare.net; dkim=pass header.d=evil.example reason=header.d=slice.bank.in",
		"mx.cloudflare.net; dkim=pass (header.d=slice.bank.in) header.d=evil.example",
		"mx.cloudflare.net; dmarc=pass header.from=evil.example",
	} {
		if AuthenticatedSender([]string{result}, "mx.cloudflare.net", "slice.bank.in") {
			t.Fatalf("untrusted result accepted: %q", result)
		}
	}
	if AuthenticatedSender([]string{valid, "mx.cloudflare.net; dkim=fail header.d=slice.bank.in"}, "mx.cloudflare.net", "slice.bank.in") {
		t.Fatal("duplicate trusted Authentication-Results headers were accepted")
	}
}

func TestParseHistoricalSliceHTMLLayout(t *testing.T) {
	raw := "From: noreply@slice.bank.in\r\n" +
		"Message-ID: <slice-html@example>\r\n" +
		"Subject: Received =?UTF-8?Q?=E2=82=B91=2E02?= in your slice account\r\n" +
		"Authentication-Results: mx.cloudflare.net; dmarc=pass header.from=slice.bank.in\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		"<p>You have received &#8377;1.02 via UPI in your slice bank account</p><table><tr><td>From</td><td>ALICE USER</td></tr><tr><td>RRN</td><td>123456789012</td></tr></table>"
	message, err := ParseRaw([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(message)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AmountPaise != 102 || parsed.RRN != "123456789012" {
		t.Fatalf("parsed = %+v", parsed)
	}
}

func TestParseCurrentSliceEmailLayout(t *testing.T) {
	raw, err := os.ReadFile("testdata/slice_received_real_layout.eml")
	if err != nil {
		t.Fatal(err)
	}
	message, err := ParseRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(message)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AmountPaise != 323 || parsed.RRN != "123456789012" {
		t.Fatalf("parsed = %+v", parsed)
	}
}

func TestParsePrivateAcceptanceFixture(t *testing.T) {
	path := os.Getenv("PAYGATE_REAL_EMAIL_FIXTURE")
	if path == "" {
		t.Skip("set PAYGATE_REAL_EMAIL_FIXTURE to validate a private original message")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	message, err := ParseRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(message)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AmountPaise != 323 || len(parsed.RRN) < 10 {
		t.Fatalf("private fixture did not produce the expected amount and usable RRN")
	}
}

func TestParseRejectsSubjectBodyAmountMismatch(t *testing.T) {
	_, err := Parse(Message{
		Subject: "Received ₹3.23 in your slice account",
		Body:    "You have received ₹4.23 via UPI in your slice bank account. RRN 123456789012",
	})
	if !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("mismatch error = %v", err)
	}
}

func TestParseRawIgnoresTextAttachments(t *testing.T) {
	raw := "From: noreply@slice.bank.in\r\n" +
		"Subject: Received INR 1.02 in your slice account\r\n" +
		"Content-Type: multipart/mixed; boundary=mixed\r\n\r\n" +
		"--mixed\r\nContent-Type: text/plain\r\n\r\nYou have received INR 1.02 via UPI in your slice bank account\r\n" +
		"--mixed\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename=fake.txt\r\n\r\nRRN:999988887777\r\n" +
		"--mixed--\r\n"
	message, err := ParseRaw([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(message)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.RRN != "" {
		t.Fatalf("attachment supplied RRN %q", parsed.RRN)
	}
}

func TestParseRejectsNonPaymentAndOversizedRawEmail(t *testing.T) {
	if _, err := Parse(Message{Subject: "Your monthly UPI statement", Body: "No payment here"}); !errors.Is(err, ErrUnrecognized) {
		t.Fatalf("non-payment error = %v", err)
	}
	if _, err := ParseRaw(make([]byte, MaxRawBytes+1)); err == nil {
		t.Fatal("oversized raw email was accepted")
	}
}

func FuzzParseRaw(f *testing.F) {
	f.Add([]byte(testRawEmail))
	f.Add([]byte("From: nobody@example.org\r\n\r\nhello"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > MaxRawBytes {
			t.Skip()
		}
		message, err := ParseRaw(raw)
		if err == nil {
			_, _ = Parse(message)
		}
	})
}
