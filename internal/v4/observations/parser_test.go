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
func TestKotakRequiresKotakMarkerAndIncomingCredit(t *testing.T) {
	posted := time.UnixMilli(1_788_200_000_000).UTC()
	cases := []struct {
		title string
		text  string
		want  error
	}{
		{"VM-HDFCBK", "Received Rs.100.37 from Rahul", ErrUnrecognized},
		{"JD-SBIUPI-S", "A/c credited Rs.100.37 through UPI Ref 123456789012", ErrUnrecognized},
		{"JK-SBIUPI-S", "A/c credited Rs.100.37 through UPI Ref 123456789012", ErrUnrecognized},
		{"VM-KOTAKB", "Your OTP is 123456 for Rs.100.37", ErrUnrecognized},
		{"VM-KOTAKB", "Rs.100.37 debited from your account", ErrUnrecognized},
		{"VM-KOTAKB", "Cashback of INR 100.37 credited to your account", ErrUnrecognized},
		{"VM-KOTAKB", "Received Rs.100.00 from Rahul", ErrNonPayGateAmount},
	}
	for _, tc := range cases {
		_, err := Parse(Snapshot{PackageName: GoogleMessagesPackage, PostedAt: posted, Title: tc.title, Text: tc.text})
		if !errors.Is(err, tc.want) {
			t.Errorf("Parse(%q,%q) error=%v want=%v", tc.title, tc.text, err, tc.want)
		}
	}
}

func TestUnsupportedPackageIsIgnored(t *testing.T) {
	_, err := Parse(Snapshot{PackageName: "com.google.android.apps.nbu.paisa.user", Text: "₹100.37 received"})
	if !errors.Is(err, ErrUnrecognized) {
		t.Fatalf("unsupported package error = %v", err)
	}
}
