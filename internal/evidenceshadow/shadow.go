package evidenceshadow

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/sms"
)

const Provider = "google_messages_android_shadow"

type Annotation struct {
	Provider      string `json:"provider"`
	Parser        string `json:"parser"`
	ParseStatus   string `json:"parseStatus"`
	AmountPaise   int64  `json:"amountPaise,omitempty"`
	ReferenceHash string `json:"referenceHash,omitempty"`
}

func Annotate(event *domain.RelayEvent, text string) Annotation {
	result := Annotation{Provider: Provider, Parser: "bank_sms_v1", ParseStatus: "unrecognized"}
	parsed, err := sms.Parse(strings.TrimSpace(text))
	if errors.Is(err, sms.ErrUnrecognized) {
		event.ProviderResult = result
		return result
	}
	if err != nil {
		result.ParseStatus = "parse_error"
		event.ProviderResult = result
		return result
	}
	result.AmountPaise = parsed.AmountPaise
	if strings.TrimSpace(parsed.RRN) == "" {
		result.ParseStatus = "amount_only"
	} else {
		result.ParseStatus = "complete"
		result.ReferenceHash = HashReference(parsed.RRN)
	}
	event.ProviderResult = result
	return result
}

func HashReference(reference string) string {
	normalized := strings.ToUpper(strings.TrimSpace(reference))
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
