package sms

import "testing"

func FuzzParseNeverPanicsOrAcceptsObviousDebits(f *testing.F) {
	seeds := []string{
		"Received Rs.100.01 from payer UPI Ref 123456789012",
		"Your A/c is credited with INR 5.25 RRN 999988887777",
		"Refund received Rs.100.01 UPI Ref 123456789012",
		"Rs.100.01 debited from your account via UPI Ref 123456789012",
		"",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		parsed, err := Parse(body)
		if err == nil && parsed.AmountPaise <= 0 {
			t.Fatalf("successful parse has nonpositive amount: %+v", parsed)
		}
		if debitPattern.MatchString(body) && err == nil {
			t.Fatalf("obvious debit was accepted: %q", body)
		}
	})
}
