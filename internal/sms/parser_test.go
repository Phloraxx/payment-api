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
