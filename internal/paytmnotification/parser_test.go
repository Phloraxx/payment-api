package paytmnotification

import (
	"errors"
	"testing"
	"time"
)

func TestParsePaytmCustomerPaymentNotifications(t *testing.T) {
	cases := []struct {
		text  string
		paise int64
		payer string
	}{
		{"Payment received ₹1.01 from Rahul", 101, "Rahul"},
		{"₹150.07 paid by Anu P", 15007, "Anu P"},
		{"Payment Received | INR 2,500.99 from USER NAME", 250099, "USER NAME"},
	}
	for _, tc := range cases {
		parsed, err := Parse(tc.text)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.text, err)
		}
		if parsed.AmountPaise != tc.paise || parsed.PayerName != tc.payer {
			t.Fatalf("Parse(%q) = %+v; want %d %q", tc.text, parsed, tc.paise, tc.payer)
		}
	}
}

func TestParseRejectsNonCustomerCredits(t *testing.T) {
	for _, text := range []string{"Cashback received ₹5.00", "Settlement received INR 100.01", "Refunded ₹100.01", "Your Paytm for Business account is ready"} {
		if _, err := Parse(text); !errors.Is(err, ErrUnrecognized) {
			t.Fatalf("Parse(%q) error = %v; want ErrUnrecognized", text, err)
		}
	}
}

func TestParsePaytmRemoteViewTransactionTimestamp(t *testing.T) {
	parsed, err := Parse("₹1.02 Received from Sourav P Bijoy\nReceived on 27 Aug 2026 07:38 PM")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 27, 14, 8, 0, 0, time.UTC)
	if parsed.AmountPaise != 102 || parsed.PayerName != "Sourav P Bijoy" || !parsed.OccurredAt.Equal(want) {
		t.Fatalf("parsed = %+v; want amount=102 payer and occurredAt=%s", parsed, want)
	}
}
