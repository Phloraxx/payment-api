package evidenceshadow

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/store"
)

const GoogleMessagesPackage = "com.google.android.apps.messaging"
const correlationWindow = 5 * time.Minute

type MetricsService struct {
	Store store.Database
	Now   func() time.Time
}

func (s MetricsService) Current(days int) (domain.EvidenceShadowMetrics, error) {
	if days <= 0 {
		days = 14
	}
	if days > 30 {
		days = 30
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	since := now.Add(-time.Duration(days) * 24 * time.Hour)
	metrics := domain.EvidenceShadowMetrics{WindowStart: since, WindowDays: days, RemovalGate: "collect_more"}
	var android []*domain.RelayEvent
	var libgm []*domain.SMSEvent
	err := s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var err error
		android, err = uow.RelayEvents().ListByPackageSince(GoogleMessagesPackage, since, 5000)
		if err != nil {
			return err
		}
		libgm, err = uow.SMSEvents().ListBySourceSince("gmessages", since, 5000)
		return err
	})
	if err != nil {
		return metrics, err
	}
	return calculate(metrics, android, libgm), nil
}

type comparable struct {
	amount int64
	hash   string
	at     time.Time
}

func calculate(metrics domain.EvidenceShadowMetrics, android []*domain.RelayEvent, libgm []*domain.SMSEvent) domain.EvidenceShadowMetrics {
	metrics.AndroidObserved = len(android)
	metrics.LibGMObserved = len(libgm)
	androidComplete := make([]comparable, 0, len(android))
	for _, event := range android {
		annotation, ok := annotationFrom(event.ProviderResult)
		if !ok || annotation.Provider != Provider {
			continue
		}
		if annotation.AmountPaise > 0 {
			metrics.AndroidParseable++
		}
		if annotation.ParseStatus == "complete" && annotation.AmountPaise > 0 && annotation.ReferenceHash != "" {
			metrics.AndroidComplete++
			at := event.NotificationWhen
			if at.IsZero() {
				at = event.CreatedAt
			}
			androidComplete = append(androidComplete, comparable{amount: annotation.AmountPaise, hash: annotation.ReferenceHash, at: at})
		}
	}
	libComplete := make([]comparable, 0, len(libgm))
	for _, event := range libgm {
		if event.AmountPaise <= 0 || event.RRN == "" {
			continue
		}
		metrics.LibGMComplete++
		libComplete = append(libComplete, comparable{amount: event.AmountPaise, hash: HashReference(event.RRN), at: event.MessageTime})
	}
	metrics.ExactMatches = correlate(androidComplete, libComplete)
	metrics.AndroidOnlyComplete = metrics.AndroidComplete - metrics.ExactMatches
	metrics.LibGMOnlyComplete = metrics.LibGMComplete - metrics.ExactMatches
	metrics.ReferenceCoveragePercent = percent(metrics.AndroidComplete, metrics.AndroidParseable)
	metrics.ExactParityPercent = percent(metrics.ExactMatches, metrics.LibGMComplete)
	metrics.RemovalReady = metrics.LibGMComplete >= 100 && metrics.AndroidParseable >= 100 && metrics.LibGMOnlyComplete == 0 && metrics.ReferenceCoveragePercent == 100 && metrics.ExactParityPercent == 100
	if metrics.RemovalReady {
		metrics.RemovalGate = "eligible_for_manual_removal_review"
	} else if metrics.LibGMComplete >= 100 || metrics.AndroidParseable >= 100 {
		metrics.RemovalGate = "keep_libgm_parity_incomplete"
	}
	return metrics
}

func annotationFrom(value any) (Annotation, bool) {
	data, err := json.Marshal(value)
	if err != nil {
		return Annotation{}, false
	}
	var result Annotation
	if json.Unmarshal(data, &result) != nil {
		return Annotation{}, false
	}
	return result, result.Provider != ""
}

func correlate(android, libgm []comparable) int {
	sort.Slice(android, func(i, j int) bool { return android[i].at.Before(android[j].at) })
	sort.Slice(libgm, func(i, j int) bool { return libgm[i].at.Before(libgm[j].at) })
	used := make([]bool, len(android))
	matches := 0
	for _, bank := range libgm {
		best, bestDelta := -1, time.Duration(math.MaxInt64)
		for i, relay := range android {
			if used[i] || bank.amount != relay.amount || bank.hash != relay.hash {
				continue
			}
			delta := bank.at.Sub(relay.at)
			if delta < 0 {
				delta = -delta
			}
			if delta <= correlationWindow && delta < bestDelta {
				best, bestDelta = i, delta
			}
		}
		if best >= 0 {
			used[best] = true
			matches++
		}
	}
	return matches
}

func percent(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return math.Round((float64(numerator)/float64(denominator))*10000) / 100
}

func ExplainGate(metrics domain.EvidenceShadowMetrics) string {
	return fmt.Sprintf("%s: %d exact pairs, %d libgm-only complete events, %.2f%% Android reference coverage over %d days", metrics.RemovalGate, metrics.ExactMatches, metrics.LibGMOnlyComplete, metrics.ReferenceCoveragePercent, metrics.WindowDays)
}
