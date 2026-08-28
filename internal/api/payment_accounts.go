package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/store"
)

type paymentAccountOption struct {
	ID                string `json:"id"`
	Label             string `json:"label"`
	Verification      string `json:"verification"`
	Flow              string `json:"flow"`
	Ready             bool   `json:"ready"`
	UnavailableReason string `json:"unavailableReason,omitempty"`
}

func (a *API) paymentAccountOptions() ([]paymentAccountOption, error) {
	accounts := a.Config.PaymentAccounts()
	result := make([]paymentAccountOption, 0, len(accounts))
	for _, account := range accounts {
		ready, reason, err := a.paymentAccountReady(account)
		if err != nil {
			return nil, err
		}
		result = append(result, paymentAccountOption{
			ID: account.ID, Label: account.Label, Verification: account.Verification,
			Flow: account.Flow, Ready: ready, UnavailableReason: reason,
		})
	}
	return result, nil
}

func (a *API) paymentAccountReady(account config.PaymentAccount) (bool, string, error) {
	if account.ID == "slice" {
		if a.Config.EmailEvidenceEnabled && a.Email != nil {
			return true, "", nil
		}
		return false, "Slice is temporarily unavailable. Choose another payment account.", nil
	}
	if account.ID != "paytm" {
		return true, "", nil
	}
	if a.Config.TestMode {
		return true, "", nil
	}
	// The legacy static merchant-QR path is paired with the legacy signed
	// notification webhook. The modern exact-amount QR path intentionally
	// requires a recently active Android relay before a checkout is created.
	if account.Flow == "merchant_qr" {
		if strings.TrimSpace(a.Config.PaytmNotificationWebhookSecret) != "" {
			return true, "", nil
		}
		return false, "Paytm is temporarily unavailable. Choose another payment account.", nil
	}
	if a.AndroidRelay == nil {
		return false, "Paytm is temporarily unavailable. Choose another payment account.", nil
	}
	ready, err := a.AndroidRelay.Ready(a.Config.AndroidRelayStaleAfter)
	if err != nil {
		return false, "", err
	}
	if !ready {
		return false, "Paytm is temporarily unavailable. Choose another payment account.", nil
	}
	return true, "", nil
}

func (a *API) paymentAccountReadyUoW(uow store.UnitOfWork, account config.PaymentAccount) (bool, string, error) {
	if account.ID == "slice" {
		if a.Config.EmailEvidenceEnabled && a.Email != nil {
			return true, "", nil
		}
		return false, "Slice is temporarily unavailable. Choose another payment account.", nil
	}
	if account.ID != "paytm" || a.Config.TestMode {
		return true, "", nil
	}
	if account.Flow == "merchant_qr" {
		if strings.TrimSpace(a.Config.PaytmNotificationWebhookSecret) != "" {
			return true, "", nil
		}
		return false, "Paytm is temporarily unavailable. Choose another payment account.", nil
	}
	if a.AndroidRelay == nil {
		return false, "Paytm is temporarily unavailable. Choose another payment account.", nil
	}
	ready, err := a.AndroidRelay.ReadyUoW(uow, a.Config.AndroidRelayStaleAfter)
	if err != nil {
		return false, "", err
	}
	if !ready {
		return false, "Paytm is temporarily unavailable. Choose another payment account.", nil
	}
	return true, "", nil
}

func (a *API) ensurePaymentAccountReady(requested string) error {
	if a.Payments != nil && a.Payments.Store != nil {
		return a.Payments.Store.View(context.Background(), func(uow store.UnitOfWork) error {
			return a.ensurePaymentAccountReadyUoW(uow, requested)
		})
	}
	accountID := strings.ToLower(strings.TrimSpace(requested))
	if accountID == "" {
		accountID = strings.ToLower(strings.TrimSpace(a.Config.DefaultPaymentAccount))
		if accountID == "" {
			accountID = "kotak"
		}
	}
	account, ok := a.Config.PaymentAccount(accountID)
	if !ok {
		return nil
	}
	ready, reason, err := a.paymentAccountReady(account)
	if err != nil {
		return domain.New("PAYMENT_ACCOUNT_UNAVAILABLE", "payment verification is temporarily unavailable", http.StatusServiceUnavailable)
	}
	if !ready {
		if reason == "" {
			reason = "payment verification is temporarily unavailable"
		}
		return domain.New("PAYMENT_ACCOUNT_UNAVAILABLE", reason, http.StatusServiceUnavailable)
	}
	return nil
}

func (a *API) ensurePaymentAccountReadyUoW(uow store.UnitOfWork, requested string) error {
	accountID := strings.ToLower(strings.TrimSpace(requested))
	if accountID == "" {
		accountID = strings.ToLower(strings.TrimSpace(a.Config.DefaultPaymentAccount))
		if accountID == "" {
			accountID = "kotak"
		}
	}
	account, ok := a.Config.PaymentAccount(accountID)
	if !ok {
		return nil // payments.Service returns the canonical invalid/disabled-account error.
	}
	ready, reason, err := a.paymentAccountReadyUoW(uow, account)
	if err != nil {
		return domain.New("PAYMENT_ACCOUNT_UNAVAILABLE", "payment verification is temporarily unavailable", http.StatusServiceUnavailable)
	}
	if !ready {
		if reason == "" {
			reason = "payment verification is temporarily unavailable"
		}
		return domain.New("PAYMENT_ACCOUNT_UNAVAILABLE", reason, http.StatusServiceUnavailable)
	}
	return nil
}
