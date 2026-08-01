package main

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestMergeManagedRateLimitRulesIsIdempotentAndPreservesCustomRules(t *testing.T) {
	initial := []core.RateLimitRule{
		{Label: "POST /api/events/sms", MaxRequests: 1, Duration: 1},
		{Label: "custom", MaxRequests: 9, Duration: 9},
		{Label: "POST /api/payments", MaxRequests: 2, Duration: 2},
	}
	first := mergeManagedRateLimitRules(initial)
	second := mergeManagedRateLimitRules(first)
	if len(first) != 4 || len(second) != 4 {
		t.Fatalf("lengths first=%d second=%d", len(first), len(second))
	}
	counts := map[string]int{}
	for _, rule := range second {
		counts[rule.Label]++
	}
	for _, label := range []string{"POST /api/events/sms", "POST /api/webhook", "POST /api/payments", "custom"} {
		if counts[label] != 1 {
			t.Fatalf("label %s count=%d", label, counts[label])
		}
	}
	if second[0].MaxRequests != 60 || second[2].MaxRequests != 120 {
		t.Fatalf("managed rules not restored: %+v", second)
	}
}
