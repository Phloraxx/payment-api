package observations

import (
	"errors"
	"testing"
	"time"
)

func TestParsePaytmCapturedNotification(t *testing.T) {
	posted := time.Date(2026, 8, 27, 19, 39, 0, 0, time.FixedZone("IST", 5*60*60+30*60))
	got, err := Parse(Snapshot{
		PackageName: PaytmBusinessPackage,
		PostedAt:    posted,
		Title:       "Payment Received on Paytm",
		BigText:     "₹1.01 Received from Test User\nReceived on 27 Aug 2026 07:38 PM",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "paytm_notification" || got.CollectionProfileID != "paytm" || got.AmountPaise != 101 {
		t.Fatalf("observation = %+v", got)
	}
	if got.PayerName != "Test User" || got.PayerUPIID != "" {
		t.Fatalf("payer = %q / %q", got.PayerName, got.PayerUPIID)
	}
	want := time.Date(2026, 8, 27, 19, 38, 0, 0, posted.Location()).UTC()
	if !got.OccurredAt.Equal(want) || got.OccurredAtSource != "notification_text" {
		t.Fatalf("occurred = %s source=%s", got.OccurredAt, got.OccurredAtSource)
	}
}
func TestPaytmRejectsNonPayGateAndNonPaymentNotifications(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	cases := []struct {
		text string
		want error
	}{
		{"₹100.00 Received from Test User", ErrNonPayGateAmount},
		{"Refund received ₹100.37", ErrUnrecognized},
		{"You paid to STORE ₹100.37", ErrUnrecognized},
		{"Settlement of ₹100.37 completed", ErrUnrecognized},
	}
	for _, tc := range cases {
		_, err := Parse(Snapshot{PackageName: PaytmBusinessPackage, PostedAt: posted, Text: tc.text})
		if !errors.Is(err, tc.want) {
			t.Errorf("Parse(%q) error=%v want=%v", tc.text, err, tc.want)
		}
	}
}

func TestPaytmKeepsMinuteTextWhenPostTimeIsTooFarAway(t *testing.T) {
	loc := time.FixedZone("IST", 5*60*60+30*60)
	posted := time.Date(2026, 9, 3, 0, 28, 5, 0, loc)
	got, err := Parse(Snapshot{
		PackageName: PaytmBusinessPackage,
		PostedAt:    posted,
		Title:       "Payment Received on Paytm",
		BigText:     "₹2.24 Received from Test User\nReceived on 3 Sep 2026 12:27 AM",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 3, 0, 27, 0, 0, loc).UTC()
	if !got.OccurredAt.Equal(want) || got.OccurredAtSource != "notification_text" {
		t.Fatalf("occurred = %s source=%s", got.OccurredAt, got.OccurredAtSource)
	}
}

func TestPaytmMinuteTimestampUsesHigherResolutionPostTime(t *testing.T) {
	loc := time.FixedZone("IST", 5*60*60+30*60)
	posted := time.Date(2026, 9, 3, 0, 27, 55, 267_000_000, loc)
	got, err := Parse(Snapshot{
		PackageName: PaytmBusinessPackage,
		PostedAt:    posted,
		Title:       "Payment Received on Paytm",
		BigText:     "₹2.24 Received from Test User\nReceived on 3 Sep 2026 12:27 AM",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AmountPaise != 224 || !got.OccurredAt.Equal(posted.UTC()) || got.OccurredAtSource != "notification_posted_at" {
		t.Fatalf("observation = %+v", got)
	}
}

func TestPaytmFallsBackToNotificationPostTime(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	got, err := Parse(Snapshot{PackageName: PaytmBusinessPackage, PostedAt: posted, Text: "₹100.37 Received from Rahul"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.OccurredAt.Equal(posted) || got.OccurredAtSource != "notification_posted_at" {
		t.Fatalf("occurred = %s source=%s", got.OccurredAt, got.OccurredAtSource)
	}
}
func TestParseBlocksRetiredMessageAndEmailPackages(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	for _, packageName := range []string{GoogleMessagesPackage, GmailPackage} {
		if _, err := Parse(Snapshot{
			PackageName: packageName,
			PostedAt:    posted,
			Title:       "Payment received",
			Text:        "Received Rs. 1,250.50 from Maya",
		}); !errors.Is(err, ErrUnrecognized) {
			t.Errorf("Parse(%q) error=%v, want %v", packageName, err, ErrUnrecognized)
		}
	}
}

func TestParseRejectsUntrustedGenericPackages(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	for _, packageName := range []string{"com.example.wallet", "com.android.shell"} {
		if _, err := Parse(Snapshot{
			PackageName: packageName,
			PostedAt:    posted,
			Title:       "received",
			Text:        "₹100.37 received from Rahul",
		}); !errors.Is(err, ErrUnrecognized) {
			t.Errorf("Parse(%q) error=%v, want %v", packageName, err, ErrUnrecognized)
		}
	}
}

func TestParseRejectsMalformedAmountTokens(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	for _, text := range []string{
		"₹12.345 received from Rahul",
		"₹12.34.56 received from Rahul",
		"₹12.34foo received from Rahul",
	} {
		if _, err := Parse(Snapshot{PackageName: BHIMPackage, PostedAt: posted, Text: text}); err == nil {
			t.Errorf("malformed amount %q was accepted", text)
		}
	}
}

func TestParseGenericIncomingPaymentApplications(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	cases := []struct{ pkg, text, source string }{
		{"in.amazon.mShop.android.shopping", "Payment received: ₹499.37 from Rahul", GenericNotificationSource},
		{"com.phonepe.app", "You received INR 250.41 from Maya via UPI", GenericNotificationSource},
		{"com.phonepe.app", "You received INR 12.3 from Maya via UPI", GenericNotificationSource},
		{"com.google.android.apps.nbu.paisa.user", "₹99.23 received from user@okaxis", GenericNotificationSource},
	}
	for _, tc := range cases {
		got, err := Parse(Snapshot{PackageName: tc.pkg, PostedAt: posted, Text: tc.text})
		if err != nil {
			t.Errorf("%s: %v", tc.pkg, err)
			continue
		}
		if got.Source != tc.source || got.AmountPaise%100 == 0 || got.CollectionProfileID != "" {
			t.Errorf("%s: %+v", tc.pkg, got)
		}
	}
}

func TestParseGenericBHIMIncomingNotifications(t *testing.T) {
	posted := time.Date(2026, 9, 4, 10, 11, 12, 345_000_000, time.FixedZone("IST", 5*60*60+30*60))
	cases := []struct {
		name      string
		text      string
		amount    int64
		payerName string
		payerUPI  string
	}{
		{
			name:      "superyes",
			text:      "Received INR 1.25 in your State Bank Of India account(XX3492) from SOURAV P BIJOY (sourav.bijoy@superyes). For further details, please check the transaction history on your BHIM app.",
			amount:    125,
			payerName: "SOURAV P BIJOY",
			payerUPI:  "sourav.bijoy@superyes",
		},
		{
			name:      "okaxis",
			text:      "Received INR 2.06 in your State Bank Of India account(XX3492) from SOURAV P BIJOY (souravpbijoy-2@okaxis). For further details, please check the transaction history on your BHIM app.",
			amount:    206,
			payerName: "SOURAV P BIJOY",
			payerUPI:  "souravpbijoy-2@okaxis",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(Snapshot{
				PackageName: "in.org.npci.upiapp",
				PostedAt:    posted,
				Title:       "Bharat Interface for Money",
				Text:        tc.text,
			})
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.Source != GenericNotificationSource || got.AmountPaise != tc.amount ||
				got.PayerName != tc.payerName || got.PayerUPIID != tc.payerUPI {
				t.Fatalf("observation = %+v", got)
			}
			if !got.OccurredAt.Equal(posted.UTC()) || got.OccurredAtSource != "notification_posted_at" {
				t.Fatalf("occurred = %s source=%s", got.OccurredAt, got.OccurredAtSource)
			}
		})
	}
}

func TestGenericPayerCleanupCompatibility(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	cases := []struct {
		name      string
		text      string
		payerName string
		payerUPI  string
	}{
		{
			name:      "dotted UPI before sentence-ending period",
			text:      "Received INR 1.25 from Alice (alice.name@upi).",
			payerName: "Alice",
			payerUPI:  "alice.name@upi",
		},
		{
			name:      "unrelated parenthesized name text survives",
			text:      "Received INR 1.25 from Alice (VIP) (alice@upi)",
			payerName: "Alice (VIP)",
			payerUPI:  "alice@upi",
		},
		{
			name:      "non-parenthesized UPI uses fallback removal",
			text:      "Received INR 1.25 from Alice alice@upi",
			payerName: "Alice",
			payerUPI:  "alice@upi",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(Snapshot{PackageName: BHIMPackage, PostedAt: posted, Text: tc.text})
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.Source != GenericNotificationSource || got.PayerName != tc.payerName || got.PayerUPIID != tc.payerUPI {
				t.Fatalf("observation = %+v", got)
			}
		})
	}
}

func TestGenericParserRejectsMoneyThatIsNotIncomingPaymentEvidence(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	texts := []string{
		"Amazon order total ₹499.37",
		"You paid to STORE ₹499.37",
		"₹499.37 debited from your account",
		"Refund received ₹499.37",
		"EMI due ₹499.37 tomorrow",
		"Cashback of ₹499.37 received",
		"Payment received ₹499.00",
	}
	for _, text := range texts {
		if _, err := Parse(Snapshot{PackageName: "example.app", PostedAt: posted, Text: text}); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
}

func TestParseGooglePayBusinessReceivedFromNotification(t *testing.T) {
	posted := time.UnixMilli(1_788_523_862_958).UTC()
	got, err := Parse(Snapshot{
		PackageName: "com.google.android.apps.nbu.paisa.merchant",
		PostedAt:    posted,
		Title:       "₹4.47 received from Sourav P B at 5:41 pm",
		Text:        "See live notifications here when you receive customer payments",
		BigText:     "See live notifications here when you receive customer payments",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Source != GenericNotificationSource || got.AmountPaise != 447 || got.PayerName != "Sourav P B" {
		t.Fatalf("observation = %+v", got)
	}
}

func TestParseGooglePayPaidYouNotification(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	got, err := Parse(Snapshot{
		PackageName: "com.google.android.apps.nbu.paisa.user",
		PostedAt:    posted,
		Text:        "Sourav P Bijoy paid you ₹3.05",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Source != GenericNotificationSource || got.AmountPaise != 305 || got.PayerName != "Sourav P Bijoy" {
		t.Fatalf("observation = %+v", got)
	}

	if _, err := Parse(Snapshot{
		PackageName: "com.google.android.apps.nbu.paisa.user",
		PostedAt:    posted,
		Text:        "You paid Rahul ₹3.05",
	}); !errors.Is(err, ErrUnrecognized) {
		t.Fatalf("outgoing GPay notification error = %v, want %v", err, ErrUnrecognized)
	}
}

func TestParserRejectsAmbiguousMonetaryAmounts(t *testing.T) {
	_, err := Parse(Snapshot{
		PackageName: BHIMPackage,
		PostedAt:    time.UnixMilli(1_788_200_000_000).UTC(),
		Text:        "Payment received ₹1.25. Available balance ₹100.37",
	})
	if !errors.Is(err, ErrAmbiguousAmount) {
		t.Fatalf("ambiguous notification error = %v, want %v", err, ErrAmbiguousAmount)
	}
}

func TestParserRejectsFailedIncomingLanguage(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	for _, text := range []string{
		"Payment failed but ₹100.37 received",
		"UPI payment declined: received ₹100.37",
		"Payment pending, amount ₹100.37 received",
	} {
		if _, err := Parse(Snapshot{PackageName: BHIMPackage, PostedAt: posted, Text: text}); !errors.Is(err, ErrUnrecognized) {
			t.Errorf("Parse(%q) error = %v, want %v", text, err, ErrUnrecognized)
		}
	}
}

func TestPayerUPIUsesIncomingPayerClause(t *testing.T) {
	got, err := Parse(Snapshot{
		PackageName: BHIMPackage,
		PostedAt:    time.UnixMilli(1_788_200_000_000).UTC(),
		Title:       "Merchant merchant@upi",
		Text:        "Received ₹1.25 from Alice (alice@upi)",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.PayerName != "Alice" || got.PayerUPIID != "alice@upi" {
		t.Fatalf("payer = %q / %q", got.PayerName, got.PayerUPIID)
	}
}

func TestPayerUPIPrefersFirstVPAInIncomingClause(t *testing.T) {
	got, err := Parse(Snapshot{
		PackageName: BHIMPackage,
		PostedAt:    time.UnixMilli(1_788_200_000_000).UTC(),
		Text:        "Received ₹1.25 from Alice (alice@upi) to merchant@upi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.PayerName != "Alice" || got.PayerUPIID != "alice@upi" {
		t.Fatalf("payer = %q / %q", got.PayerName, got.PayerUPIID)
	}
}
