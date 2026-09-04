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
func TestParseKotakGoogleMessagesNotification(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	got, err := Parse(Snapshot{
		PackageName: GoogleMessagesPackage,
		PostedAt:    posted,
		Title:       "VM-KOTAKB",
		Text:        "Kotak: Received Rs. 1,250.50 from Maya UPI Ref No. 123456789012",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "kotak_sms" || got.CollectionProfileID != "kotak" || got.AmountPaise != 125050 {
		t.Fatalf("observation = %+v", got)
	}
	if got.PayerName != "Maya" || !got.OccurredAt.Equal(posted) || got.OccurredAtSource != "notification_posted_at" {
		t.Fatalf("parsed Kotak details = %+v", got)
	}
}

func TestKotakExtractsUPIWithoutRequiringReference(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	got, err := Parse(Snapshot{
		PackageName: GoogleMessagesPackage,
		PostedAt:    posted,
		Title:       "KOTAK",
		Text:        "Payment for Received INR 75.37 from a@upi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AmountPaise != 7537 || got.PayerUPIID != "a@upi" {
		t.Fatalf("observation = %+v", got)
	}
}
func TestMessagesAcceptAnyIncomingBankCreditButStillRejectUnsafeMoneyText(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	accepted := []struct{ title, text string }{
		{"VM-HDFCBK", "Received Rs.100.37 from Rahul"},
		{"JD-SBIUPI-S", "A/c credited Rs.100.37 through UPI Ref 123456789012"},
		{"JK-SBIUPI-S", "A/c credited Rs.100.37 through UPI Ref 123456789012"},
	}
	for _, tc := range accepted {
		got, err := Parse(Snapshot{PackageName: GoogleMessagesPackage, PostedAt: posted, Title: tc.title, Text: tc.text})
		if err != nil || got.Source != GenericMessageSource {
			t.Errorf("Parse(%q,%q)=%+v err=%v", tc.title, tc.text, got, err)
		}
	}
	rejected := []struct {
		title, text string
		want        error
	}{
		{"VM-KOTAKB", "Your OTP is 123456 for Rs.100.37", ErrUnrecognized},
		{"VM-KOTAKB", "Rs.100.37 debited from your account", ErrUnrecognized},
		{"VM-KOTAKB", "Cashback of INR 100.37 credited to your account", ErrUnrecognized},
		{"VM-KOTAKB", "Received Rs.100.00 from Rahul", ErrNonPayGateAmount},
	}
	for _, tc := range rejected {
		_, err := Parse(Snapshot{PackageName: GoogleMessagesPackage, PostedAt: posted, Title: tc.title, Text: tc.text})
		if !errors.Is(err, tc.want) {
			t.Errorf("Parse(%q,%q) error=%v want=%v", tc.title, tc.text, err, tc.want)
		}
	}
}

func TestUnknownPackageSplitTitleAndAmountCanProvideIncomingPaymentEvidence(t *testing.T) {
	posted := time.Now().UTC()
	got, err := Parse(Snapshot{PackageName: "com.example.wallet", PostedAt: posted, Title: "received", Text: "₹98765.43"})
	if err != nil || got.Source != GenericNotificationSource || got.AmountPaise != 9876543 {
		t.Fatalf("split-field generic observation=%+v err=%v", got, err)
	}
}

func TestUnknownPackageCanProvideIncomingPaymentEvidence(t *testing.T) {
	got, err := Parse(Snapshot{PackageName: "com.example.wallet", PostedAt: time.Now().UTC(), Text: "₹100.37 received from Rahul"})
	if err != nil || got.Source != GenericNotificationSource || got.AmountPaise != 10037 {
		t.Fatalf("generic package observation=%+v err=%v", got, err)
	}
}

func TestParseGenericIncomingPaymentApplications(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	cases := []struct{ pkg, text, source string }{
		{"in.amazon.mShop.android.shopping", "Payment received: ₹499.37 from Rahul", GenericNotificationSource},
		{"com.phonepe.app", "You received INR 250.41 from Maya via UPI", GenericNotificationSource},
		{"com.google.android.apps.nbu.paisa.user", "₹99.23 received from user@okaxis", GenericNotificationSource},
		{GoogleMessagesPackage, "HDFC: A/c credited with Rs. 701.19 from Arun UPI Ref 123456789012", GenericMessageSource},
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
