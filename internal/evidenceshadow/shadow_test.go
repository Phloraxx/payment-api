package evidenceshadow

import (
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
)

func TestAnnotateStoresOnlyHashedReferenceMetadata(t *testing.T) {
	event := &domain.RelayEvent{}
	annotation := Annotate(event, "Received Rs.100.01 from Person UPI Ref:123456789012")
	if annotation.ParseStatus != "complete" || annotation.AmountPaise != 10001 {
		t.Fatalf("annotation=%+v", annotation)
	}
	if annotation.ReferenceHash == "" || annotation.ReferenceHash == "123456789012" {
		t.Fatalf("reference was not irreversibly represented: %+v", annotation)
	}
	if event.ProviderResult != annotation {
		t.Fatalf("provider result=%+v", event.ProviderResult)
	}
}

func TestCalculateRequiresCompleteExactParityForRemovalReview(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	android := make([]*domain.RelayEvent, 0, 100)
	libgm := make([]*domain.SMSEvent, 0, 100)
	for i := 0; i < 100; i++ {
		ref := "REF" + string(rune('A'+i%26)) + string(rune('A'+(i/26)%26)) + "12345678"
		at := now.Add(time.Duration(i) * time.Minute)
		android = append(android, &domain.RelayEvent{NotificationWhen: at, ProviderResult: Annotation{Provider: Provider, Parser: "bank_sms_v1", ParseStatus: "complete", AmountPaise: 10001 + int64(i), ReferenceHash: HashReference(ref)}})
		libgm = append(libgm, &domain.SMSEvent{MessageTime: at.Add(10 * time.Second), AmountPaise: 10001 + int64(i), RRN: ref})
	}
	metrics := calculate(domain.EvidenceShadowMetrics{WindowDays: 14}, android, libgm)
	if !metrics.RemovalReady || metrics.ExactMatches != 100 || metrics.ExactParityPercent != 100 || metrics.ReferenceCoveragePercent != 100 {
		t.Fatalf("metrics=%+v", metrics)
	}
	android[0].ProviderResult = Annotation{Provider: Provider, Parser: "bank_sms_v1", ParseStatus: "amount_only", AmountPaise: 10001}
	metrics = calculate(domain.EvidenceShadowMetrics{WindowDays: 14}, android, libgm)
	if metrics.RemovalReady || metrics.LibGMOnlyComplete != 1 || metrics.ReferenceCoveragePercent >= 100 {
		t.Fatalf("incomplete metrics=%+v", metrics)
	}
}
