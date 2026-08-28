package operatorview_test

import (
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/operatorview"
	"github.com/Phloraxx/payment-api/internal/payments"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func TestOverviewAndPaymentViewsUseTypedContract(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	cfg := config.Config{
		TestMode:         true,
		PaymentTTL:       5 * time.Minute,
		AmountQuarantine: 24 * time.Hour,
	}
	paymentService := payments.NewService(app, cfg, nil)
	paymentService.SuffixStart = func() (int64, error) { return 1, nil }
	payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 125, PaymentAccount: "kotak"})
	if err != nil {
		t.Fatal(err)
	}

	view := operatorview.New(app)
	overview, err := view.Overview(5)
	if err != nil {
		t.Fatal(err)
	}
	if overview.PaymentCounts["total"] != 1 || overview.PaymentCounts["pending"] != 1 {
		t.Fatalf("counts=%+v", overview.PaymentCounts)
	}
	if len(overview.Recent) != 1 || overview.Recent[0].ID != payment.ID {
		t.Fatalf("recent=%+v", overview.Recent)
	}

	pending, err := view.ListPayments("pending", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].PayableAmountPaise != payment.PayablePaise {
		t.Fatalf("pending=%+v", pending)
	}

	detail, err := view.GetPayment(payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != payment.ID || detail.PaymentAccount != "kotak" || detail.RRN != "" {
		t.Fatalf("detail=%+v", detail)
	}
}
