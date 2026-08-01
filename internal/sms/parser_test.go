package sms

import (
	"strings"
	"testing"
)

func TestParseKotakCreditVariants(t *testing.T) {
	cases := []struct {
		name string
		body string
		amt  int64
		rrn  string
	}{
		{"received rs", "Kotak: Received Rs. 1,250.50 from Maya UPI Ref No. 123456789012", 125050, "123456789012"},
		{"payment for received inr", "payment for Received INR 75 from a@upi UPI Ref: 987654321", 7500, "987654321"},
		{"rupee symbol", "Payment for Received ₹99.25 from payer ref 12345678", 9925, "12345678"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := Parse(tc.body)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if parsed.AmountPaise != tc.amt || parsed.RRN != tc.rrn {
				t.Fatalf("Parse() = amount %d rrn %q; want amount %d rrn %q", parsed.AmountPaise, parsed.RRN, tc.amt, tc.rrn)
			}
		})
	}
}

func TestParseIgnoresUnrelatedMessages(t *testing.T) {
	if LooksLikeBankCredit("Your OTP is 123456 for a Rs. 100 transaction") {
		t.Fatal("OTP message looked like a bank credit")
	}
	if _, err := Parse("Your OTP is 123456"); err != ErrUnrecognized {
		t.Fatalf("Parse() error = %v; want ErrUnrecognized", err)
	}
}

func TestParseBoundsDerivedIdentityFields(t *testing.T) {
	local := "a" + strings.Repeat("b", 127)
	host := "c" + strings.Repeat("d", 127)
	upi := local + "@" + host
	body := "Received Rs.100.01 from " + strings.Repeat("P", 300) + " UPI Ref:123456789012 payer " + upi
	parsed, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(parsed.UPIId)) > 255 || len([]rune(parsed.PayerName)) > 255 {
		t.Fatalf("derived fields exceed storage bounds: upi=%d payer=%d", len([]rune(parsed.UPIId)), len([]rune(parsed.PayerName)))
	}
}

func TestParseCommonUPICreditWording(t *testing.T) {
	cases := []struct {
		name string
		body string
		amt  int64
		rrn  string
	}{
		{"credited amount first", "INR 2,345.67 has been credited to your A/c XX2526 through UPI. UPI Transaction ID 123456789012", 234567, "123456789012"},
		{"account credited", "Your A/c XX2526 is credited with Rs.450.09 by ARUN via UPI. RRN:987654321098", 45009, "987654321098"},
		{"deposited", "UPI alert: Deposited INR 10.01 in account XX2526 by payer@okaxis UTR 6A7B8C9D0E12", 1001, "6A7B8C9D0E12"},
		{"amount then credited", "Rs 75.50 credited to A/c XX2526 via UPI Ref No 606703736479", 7550, "606703736479"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := Parse(tc.body)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.AmountPaise != tc.amt || parsed.RRN != tc.rrn {
				t.Fatalf("parsed=%+v want amount=%d rrn=%s", parsed, tc.amt, tc.rrn)
			}
		})
	}
}

func TestParseRejectsNonCheckoutCreditsAndDebits(t *testing.T) {
	messages := []string{
		"Refund received Rs.100.01 via UPI Ref 123456789012",
		"Cashback of INR 100.01 credited to your account UPI Ref 123456789012",
		"Rs.100.01 debited from your account via UPI Ref 123456789012",
		"You paid to STORE Rs.100.01 UPI Ref 123456789012",
	}
	for _, message := range messages {
		if LooksLikeBankCredit(message) {
			t.Errorf("non-checkout message looked like credit: %s", message)
		}
		if _, err := Parse(message); err != ErrUnrecognized {
			t.Errorf("Parse(%q) error=%v", message, err)
		}
	}
}
