package sms

import (
	"errors"
	"regexp"
	"strings"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/money"
)

var (
	// Intentionally anchored to credit language. We do not infer payments from
	// arbitrary messages that merely contain a rupee amount.
	bankCreditPattern = regexp.MustCompile(`(?i)(?:payment\s+for\s+)?received\s*(?:rs\.?|inr|₹)\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)`)
	rrnPattern        = regexp.MustCompile(`(?i)(?:upi\s*ref(?:erence)?|rrn|ref(?:erence)?)(?:\s*(?:no|number|id))?\s*[:.#\- ]*([0-9]{8,24})`)
	upiPattern        = regexp.MustCompile(`(?i)[a-z0-9][a-z0-9._-]{0,127}@[a-z0-9][a-z0-9._-]{0,127}`)
	fromPattern       = regexp.MustCompile(`(?i)\bfrom\s+(.+?)(?:\s+on\s+|\s+upi\s+ref|\.|$)`)
)

var ErrUnrecognized = errors.New("SMS is not a recognized bank credit message")

func LooksLikeBankCredit(body string) bool {
	return bankCreditPattern.MatchString(body)
}

func Parse(body string) (domain.ParsedSMS, error) {
	match := bankCreditPattern.FindStringSubmatch(body)
	if len(match) < 2 {
		return domain.ParsedSMS{}, ErrUnrecognized
	}
	amount, err := money.ParseAmount(match[1])
	if err != nil || amount <= 0 {
		if err != nil {
			return domain.ParsedSMS{}, err
		}
		return domain.ParsedSMS{}, money.ErrInvalidAmount
	}

	rrn := ""
	if match := rrnPattern.FindStringSubmatch(body); len(match) > 1 {
		rrn = strings.TrimSpace(match[1])
	}
	upiID := strings.TrimSpace(upiPattern.FindString(body))
	payerName := ""
	if from := fromPattern.FindStringSubmatch(body); len(from) > 1 {
		payerName = strings.TrimSpace(from[1])
		if strings.EqualFold(payerName, upiID) {
			payerName = ""
		}
	}

	return domain.ParsedSMS{
		AmountPaise: amount,
		RRN:         rrn,
		UPIId:       truncateRunes(upiID, 255),
		PayerName:   truncateRunes(payerName, 255),
	}, nil
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
