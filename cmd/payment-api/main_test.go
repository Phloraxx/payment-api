package main

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

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
	if len(first) != 11 || len(second) != 11 {
		t.Fatalf("lengths first=%d second=%d", len(first), len(second))
	}
	counts := map[string]int{}
	for _, rule := range second {
		counts[rule.Label]++
	}
	for _, label := range []string{"POST /api/events/sms", "POST /api/events/paytm-notification", "POST /api/relay/v1/enroll", "POST /api/relay/v1/events", "POST /api/relay/v1/heartbeat", "POST /api/events/email", "POST /api/webhook", "POST /api/payments", "POST /api/razorpay/test/orders", "POST /api/razorpay/test/webhook", "custom"} {
		if counts[label] != 1 {
			t.Fatalf("label %s count=%d", label, counts[label])
		}
	}
	if second[0].MaxRequests != 60 || second[7].MaxRequests != 120 {
		t.Fatalf("managed rules not restored: %+v", second)
	}
}

type testBackgroundRunner struct{ started chan struct{} }

func (r *testBackgroundRunner) Run(ctx context.Context) {
	close(r.started)
	<-ctx.Done()
}

func TestStartBackgroundRunnersStartsEveryRunner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := &testBackgroundRunner{started: make(chan struct{})}
	second := &testBackgroundRunner{started: make(chan struct{})}
	startBackgroundRunners(ctx, first, second)
	for index, started := range []<-chan struct{}{first.started, second.started} {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("runner %d did not start", index)
		}
	}
}

func TestWithServeOriginsInjectsCheckoutAllowlist(t *testing.T) {
	want := []string{"https://payment.mulearnscet.in", "https://pay.ieeesahrdaya.com"}
	got, changed := withServeOrigins([]string{"serve", "--http=127.0.0.1:3000"}, want)
	if !changed {
		t.Fatal("expected serve origins injection")
	}
	if !slices.Contains(got, "--origins="+strings.Join(want, ",")) {
		t.Fatalf("args=%v", got)
	}
}

func TestWithServeOriginsPreservesExplicitOrNonServeArgs(t *testing.T) {
	for _, tc := range [][]string{
		{"serve", "--origins=https://custom.example"},
		{"migrate", "up"},
	} {
		got, changed := withServeOrigins(tc, []string{"https://payment.example.com"})
		if changed || !slices.Equal(got, tc) {
			t.Fatalf("args=%v got=%v changed=%v", tc, got, changed)
		}
	}
}
