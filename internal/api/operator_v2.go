package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Phloraxx/payment-api/internal/operatorview"
	"github.com/Phloraxx/payment-api/internal/reviews"
	"github.com/pocketbase/pocketbase/core"
)

func (a *API) operatorV2Overview(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	view, err := operatorview.New(e.App).Overview(queryLimit(e, 8))
	if err != nil {
		return e.InternalServerError("failed to load operator overview", err)
	}
	payload := map[string]any{"overview": view, "connector": a.connectorStatus()}
	if a.Payments != nil {
		capacity, capacityErr := a.Payments.Capacity()
		if capacityErr != nil {
			return e.InternalServerError("failed to load payment capacity", capacityErr)
		}
		payload["capacity"] = capacity
	}
	if a.Backups != nil {
		backup, backupErr := a.Backups.GetStatus(e.Request.Context(), false)
		if backupErr != nil {
			payload["backup"] = map[string]any{"enabled": a.Config.BackupCron != "", "error": backupErr.Error()}
		} else {
			payload["backup"] = backup
		}
	}
	if a.AndroidRelay != nil {
		relay, relayErr := a.AndroidRelay.Status(a.Config.AndroidRelayStaleAfter)
		if relayErr != nil {
			return e.InternalServerError("failed to load relay status", relayErr)
		}
		payload["relay"] = relay
	}
	return e.JSON(http.StatusOK, payload)
}
func (a *API) operatorV2Payments(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	items, err := operatorview.New(e.App).ListPayments(e.Request.URL.Query().Get("status"), queryLimit(e, 50))
	if err != nil {
		if strings.Contains(err.Error(), "invalid payment status") {
			return e.BadRequestError("invalid payment status", nil)
		}
		return e.InternalServerError("failed to list payments", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"payments": items})
}

func (a *API) operatorV2Payment(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	item, err := operatorview.New(e.App).GetPayment(e.Request.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		return e.NotFoundError("payment not found", nil)
	}
	if err != nil {
		return e.InternalServerError("failed to load payment", err)
	}
	return e.JSON(http.StatusOK, item)
}
func (a *API) operatorV2Reviews(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	items, err := operatorview.New(e.App).ListReviews(e.Request.URL.Query().Get("status"), queryLimit(e, 50))
	if err != nil {
		if strings.Contains(err.Error(), "invalid review status") {
			return e.BadRequestError("invalid review status", nil)
		}
		return e.InternalServerError("failed to list reviews", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"reviews": items})
}

func (a *API) operatorV2Alerts(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	items, err := operatorview.New(e.App).ListAlerts(e.Request.URL.Query().Get("status"), queryLimit(e, 50))
	if err != nil {
		if strings.Contains(err.Error(), "invalid alert status") {
			return e.BadRequestError("invalid alert status", nil)
		}
		return e.InternalServerError("failed to list alerts", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"alerts": items})
}
func (a *API) operatorV2Relay(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	if a.AndroidRelay == nil {
		return e.JSON(http.StatusOK, map[string]any{"enabled": false, "ready": false})
	}
	status, err := a.AndroidRelay.Status(a.Config.AndroidRelayStaleAfter)
	if err != nil {
		return e.InternalServerError("failed to load relay status", err)
	}
	return e.JSON(http.StatusOK, status)
}

func queryLimit(e *core.RequestEvent, fallback int) int {
	value := strings.TrimSpace(e.Request.URL.Query().Get("limit"))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > 100 {
		return 100
	}
	return parsed
}

func (a *API) operatorV2CancelPayment(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	payment, err := a.Payments.Cancel(e.Request.PathValue("id"))
	if err != nil {
		return writeDomainError(e, err)
	}
	item, err := operatorview.New(e.App).GetPayment(payment.ID)
	if err != nil {
		return e.InternalServerError("payment cancelled but operator view could not be loaded", err)
	}
	return e.JSON(http.StatusOK, item)
}

type operatorDismissReviewBody struct {
	Note string `json:"note"`
}

func (a *API) operatorV2DismissReview(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	if a.Reviews == nil {
		return e.NotFoundError("review service is unavailable", nil)
	}
	var body operatorDismissReviewBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	result, err := a.Reviews.Resolve(reviews.ResolveInput{
		CaseID: e.Request.PathValue("id"), Action: "dismissed", Note: body.Note, Actor: a.actor(e),
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func (a *API) operatorV2Review(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	item, err := operatorview.New(e.App).GetReview(e.Request.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		return e.NotFoundError("review case not found", nil)
	}
	if err != nil {
		return e.InternalServerError("failed to load review case", err)
	}
	return e.JSON(http.StatusOK, item)
}

func (a *API) operatorV2ResolveReview(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	if a.Reviews == nil {
		return e.NotFoundError("review service is unavailable", nil)
	}
	var body reviewResolutionBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	result, err := a.Reviews.Resolve(reviews.ResolveInput{
		CaseID: e.Request.PathValue("id"), Action: body.Action, PaymentID: body.PaymentID,
		BankReference: body.BankReference, Note: body.Note, Actor: a.actor(e),
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func (a *API) operatorV2ReconciliationRuns(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	items, err := operatorview.New(e.App).ListReconciliationRuns(queryLimit(e, 50))
	if err != nil {
		return e.InternalServerError("failed to list reconciliation runs", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"runs": items})
}
func (a *API) operatorV2ReconciliationEntries(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	items, err := operatorview.New(e.App).ListReconciliationEntries(e.Request.PathValue("id"), queryLimit(e, 250))
	if err != nil {
		return e.InternalServerError("failed to list reconciliation entries", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"entries": items})
}
func (a *API) operatorV2Refunds(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	items, err := operatorview.New(e.App).ListRefunds(queryLimit(e, 50))
	if err != nil {
		return e.InternalServerError("failed to list refunds", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"refunds": items})
}

func (a *API) operatorV2OperationalRecords(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	items, err := operatorview.New(e.App).ListOperationalRecords(e.Request.PathValue("kind"), queryLimit(e, 50))
	if err != nil {
		if strings.Contains(err.Error(), "invalid operational record kind") {
			return e.NotFoundError("record view not found", nil)
		}
		return e.InternalServerError("failed to list operational records", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"records": items})
}

func (a *API) operatorV2RazorpayOrders(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("operator authentication is required", nil)
	}
	items, err := operatorview.New(e.App).ListRazorpayOrders(e.Request.PathValue("mode"), queryLimit(e, 50))
	if err != nil {
		if strings.Contains(err.Error(), "invalid razorpay mode") {
			return e.NotFoundError("razorpay mode not found", nil)
		}
		return e.InternalServerError("failed to list razorpay orders", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"orders": items})
}
