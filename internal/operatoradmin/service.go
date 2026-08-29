package operatoradmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/store"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

type UpdatePaymentInput struct {
	PaymentID     string
	Actor         audit.Actor
	DisplayName   string
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	Description   string
	AdminNote     string
	Tags          []string
	CustomFields  map[string]any
}

type Service struct {
	Store store.Database
	Now   func() time.Time
}

func (s *Service) UpdatePayment(ctx context.Context, input UpdatePaymentInput) (*domain.Payment, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("operator admin store is required")
	}
	normalized, err := normalize(input)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	var updated *domain.Payment
	err = s.Store.Write(ctx, func(tx store.UnitOfWork) error {
		payment, err := tx.Payments().Get(normalized.PaymentID)
		if err != nil {
			return err
		}
		changed := applyProfile(payment, normalized)
		if len(changed) == 0 {
			updated = payment
			return nil
		}
		if err := tx.Payments().Save(payment); err != nil {
			return err
		}
		auditService := audit.Service{Now: s.Now}
		if err := auditService.RecordUoW(tx, audit.Entry{
			Action: "payment.profile.updated", Actor: normalized.Actor,
			EntityType: "payment", EntityID: payment.ID,
			Summary: "Updated payment business details",
			Details: map[string]any{"fields": changed}, OccurredAt: now,
		}); err != nil {
			return err
		}
		updated = payment
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func normalize(input UpdatePaymentInput) (UpdatePaymentInput, error) {
	input.PaymentID = strings.TrimSpace(input.PaymentID)
	if input.PaymentID == "" {
		return input, invalid("paymentId", "is required")
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.CustomerName = strings.TrimSpace(input.CustomerName)
	input.CustomerEmail = strings.TrimSpace(input.CustomerEmail)
	input.CustomerPhone = strings.TrimSpace(input.CustomerPhone)
	input.Description = strings.TrimSpace(input.Description)
	input.AdminNote = strings.TrimSpace(input.AdminNote)
	for field, value := range map[string]string{
		"displayName": input.DisplayName, "customerName": input.CustomerName,
		"customerPhone": input.CustomerPhone,
	} {
		max := 255
		if field == "customerPhone" {
			max = 64
		}
		if runeLen(value) > max {
			return input, invalid(field, fmt.Sprintf("must be at most %d characters", max))
		}
	}
	if runeLen(input.CustomerEmail) > 254 {
		return input, invalid("customerEmail", "must be at most 254 characters")
	}
	if input.CustomerEmail != "" {
		address, err := mail.ParseAddress(input.CustomerEmail)
		if err != nil || !strings.EqualFold(address.Address, input.CustomerEmail) {
			return input, invalid("customerEmail", "must be a valid email address")
		}
	}
	if runeLen(input.Description) > 4096 {
		return input, invalid("description", "must be at most 4096 characters")
	}
	if runeLen(input.AdminNote) > 4096 {
		return input, invalid("adminNote", "must be at most 4096 characters")
	}
	tags, err := normalizeTags(input.Tags)
	if err != nil {
		return input, err
	}
	input.Tags = tags
	if input.CustomFields == nil {
		input.CustomFields = map[string]any{}
	}
	if err := validateJSONObject("customFields", input.CustomFields, 256*1024); err != nil {
		return input, err
	}
	return input, nil
}

func applyProfile(payment *domain.Payment, input UpdatePaymentInput) []string {
	changed := make([]string, 0, 10)
	setString := func(name string, dst *string, value string) {
		if *dst != value {
			*dst = value
			changed = append(changed, name)
		}
	}
	setString("displayName", &payment.DisplayName, input.DisplayName)
	setString("customerName", &payment.CustomerName, input.CustomerName)
	setString("customerEmail", &payment.CustomerEmail, input.CustomerEmail)
	setString("customerPhone", &payment.CustomerPhone, input.CustomerPhone)
	setString("description", &payment.Description, input.Description)
	setString("adminNote", &payment.AdminNote, input.AdminNote)
	if !sameStrings(payment.Tags, input.Tags) {
		payment.Tags = append([]string(nil), input.Tags...)
		changed = append(changed, "tags")
	}
	if !sameJSONObject(payment.CustomFields, input.CustomFields) {
		payment.CustomFields = input.CustomFields
		changed = append(changed, "customFields")
	}
	return changed
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sameJSONObject(existing any, desired map[string]any) bool {
	if existing == nil {
		existing = map[string]any{}
	}
	left, leftErr := json.Marshal(existing)
	right, rightErr := json.Marshal(desired)
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}

func normalizeTags(values []string) ([]string, error) {
	if len(values) > 32 {
		return nil, invalid("tags", "must contain at most 32 tags")
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if runeLen(value) > 64 {
			return nil, invalid("tags", "each tag must be at most 64 characters")
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

func validateJSONObject(field string, value map[string]any, max int) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return invalid(field, "must contain valid JSON values")
	}
	if len(encoded) > max {
		return invalid(field, fmt.Sprintf("must be at most %d bytes", max))
	}
	return nil
}

func invalid(field, message string) error { return &ValidationError{Field: field, Message: message} }
func runeLen(value string) int            { return len([]rune(value)) }
