package operatorview_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/operatorview"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/store"
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

func TestPaymentQuerySupportsSearchFiltersPagingAndTypedDetail(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	db := store.NewPocketBase(app)
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	var targetID string
	seed := []struct {
		account                                 domain.PaymentAccount
		status                                  domain.PaymentStatus
		external, display, customer, email, rrn string
		amount                                  int64
	}{
		{domain.PaymentAccountKotak, domain.StatusPaid, "ORDER-ALPHA", "AI Workshop", "Alice Student", "alice@example.com", "111111111111", 10101},
		{domain.PaymentAccountSlice, domain.StatusPending, "ORDER-BETA", "Membership", "Bob Student", "bob@example.com", "", 20202},
		{domain.PaymentAccountPaytm, domain.StatusPaid, "ORDER-GAMMA", "Buildathon", "Carol Student", "carol@example.com", "333333333333", 30303},
	}
	if err := db.Write(t.Context(), func(tx store.UnitOfWork) error {
		for i, item := range seed {
			payment, err := tx.Payments().Create(store.NewPayment{
				Account: item.account, RequestedPaise: item.amount - 1, PayablePaise: item.amount,
				CreatedAt: now.Add(time.Duration(i) * time.Minute), ExpiresAt: now.Add(time.Hour), ReuseAfter: now.Add(24 * time.Hour),
				ExternalID: item.external, IdempotencyKey: item.external + "-idem", Metadata: map[string]any{"seed": i},
			})
			if err != nil {
				return err
			}
			payment.Status = item.status
			payment.DisplayName = item.display
			payment.CustomerName = item.customer
			payment.CustomerEmail = item.email
			payment.CustomerPhone = "+91-900000000" + string(rune('0'+i))
			payment.RRN = item.rrn
			payment.UPIId = strings.ToLower(strings.Fields(item.customer)[0]) + "@upi"
			payment.EvidenceReference = "evidence:" + item.external
			if item.external == "ORDER-GAMMA" {
				payment.PayerName = "Treasurer Alias"
				payment.Description = "Hardware security workshop registration"
				payment.AdminNote = "Priority reconciliation contact"
			}
			payment.Tags = []string{"seed", item.external}
			payment.CustomFields = map[string]any{"batch": "S7"}
			if err := tx.Payments().Save(payment); err != nil {
				return err
			}
			if item.external == "ORDER-GAMMA" {
				targetID = payment.ID
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	view := operatorview.New(app)
	page, err := view.QueryPayments(operatorview.PaymentQuery{Query: "Carol", Account: "paytm", Status: "paid", Sort: "newest", Limit: 25})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Payments) != 1 || page.Payments[0].ID != targetID || page.Payments[0].DisplayName != "Buildathon" {
		t.Fatalf("search page=%+v", page)
	}
	for _, query := range []string{"Treasurer Alias", "Hardware security", "Priority reconciliation"} {
		page, err = view.QueryPayments(operatorview.PaymentQuery{Query: query, Limit: 25})
		if err != nil || page.Total != 1 || len(page.Payments) != 1 || page.Payments[0].ID != targetID {
			t.Fatalf("expanded search %q page=%+v err=%v", query, page, err)
		}
	}
	page, err = view.QueryPayments(operatorview.PaymentQuery{Sort: "oldest", Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || page.Limit != 1 || page.Offset != 1 || len(page.Payments) != 1 || page.Payments[0].ExternalID != "ORDER-BETA" {
		t.Fatalf("paged=%+v", page)
	}
	if _, err := view.QueryPayments(operatorview.PaymentQuery{Sort: "status", Limit: 25}); err != nil {
		t.Fatalf("status sort: %v", err)
	}
	if _, err := view.QueryPayments(operatorview.PaymentQuery{Sort: "drop table"}); err == nil {
		t.Fatal("expected invalid sort")
	}
	if _, err := view.QueryPayments(operatorview.PaymentQuery{Account: "unknown"}); err == nil {
		t.Fatal("expected invalid account")
	}
	if _, err := view.QueryPayments(operatorview.PaymentQuery{Query: strings.Repeat("x", 256)}); err == nil {
		t.Fatal("expected long query rejection")
	}

	detail, err := view.GetPayment(targetID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.CustomerEmail != "carol@example.com" || detail.IdempotencyKey != "ORDER-GAMMA-idem" || detail.ReuseAfter == "" {
		t.Fatalf("detail=%+v", detail)
	}
	metadata, ok := detail.Metadata.(map[string]any)
	if !ok || metadata["seed"] == nil {
		t.Fatalf("metadata=%T %#v", detail.Metadata, detail.Metadata)
	}
	custom, ok := detail.CustomFields.(map[string]any)
	if !ok || custom["batch"] != "S7" {
		t.Fatalf("custom=%T %#v", detail.CustomFields, detail.CustomFields)
	}
}
