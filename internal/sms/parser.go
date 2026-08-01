package sms

import (
	"errors"
	"regexp"
	"strings"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/money"
)

var (
	currencyAmount       = `(?:rs\.?|inr|₹)\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)`
	creditAmountPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:payment\s+for\s+)?received\s*(?:payment\s+of\s*)?` + currencyAmount),
		regexp.MustCompile(`(?i)` + currencyAmount + `\s*(?:has\s+been\s+)?(?:received|credited|deposited)\b`),
		regexp.MustCompile(`(?i)\b(?:credited|deposited)\b.{0,120}?` + currencyAmount),
		regexp.MustCompile(`(?i)\b(?:a/?c|account)\b.{0,100}?\bcredited\b.{0,80}?` + currencyAmount),
	}
	// Reversal/refund/cashback credits are not customer checkout payments and
	// must never satisfy a PayGate order merely because the amount happens to match.
	nonPaymentCreditPattern = regexp.MustCompile(`(?i)\b(?:reversal|reversed|refund(?:ed)?|cashback|reward|interest|salary|chargeback)\b`)
	debitPattern            = regexp.MustCompile(`(?i)\b(?:debited|sent|paid\s+to|withdrawn|purchase)\b`)
	rrnPatterns             = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bupi\s*(?:ref(?:erence)?|txn|transaction)(?:\s*(?:no|number|id))?\s*[:.#\- ]*([a-z0-9]{8,35})`),
		regexp.MustCompile(`(?i)\brrn(?:\s*(?:no|number|id))?\s*[:.#\- ]*([a-z0-9]{8,35})`),
		regexp.MustCompile(`(?i)\butr(?:\s*(?:no|number|id))?\s*[:.#\- ]*([a-z0-9]{8,35})`),
		regexp.MustCompile(`(?i)\bref(?:erence)?(?:\s*(?:no|number|id))?\s*[:.#\- ]*([a-z0-9]{8,35})`),
	}
	upiPattern  = regexp.MustCompile(`(?i)[a-z0-9][a-z0-9._-]{0,127}@[a-z0-9][a-z0-9._-]{0,127}`)
	fromPattern = regexp.MustCompile(`(?i)\b(?:from|by)\s+(.+?)(?:\s+on\s+|\s+(?:upi\s+)?(?:ref|rrn|utr)|\.|$)`)
)

var ErrUnrecognized = errors.New("SMS is not a recognized bank credit message")

func LooksLikeBankCredit(body string) bool {
	if nonPaymentCreditPattern.MatchString(body) || debitPattern.MatchString(body) {
		return false
	}
	return findCreditAmount(body) != ""
}

func Parse(body string) (domain.ParsedSMS, error) {
	if !LooksLikeBankCredit(body) {
		return domain.ParsedSMS{}, ErrUnrecognized
	}
	amountText := findCreditAmount(body)
	amount, err := money.ParseAmount(amountText)
	if err != nil || amount <= 0 {
		if err != nil {
			return domain.ParsedSMS{}, err
		}
		return domain.ParsedSMS{}, money.ErrInvalidAmount
	}

	rrn := ExtractReference(body)
	upiID := strings.TrimSpace(upiPattern.FindString(body))
	payerName := ""
	if from := fromPattern.FindStringSubmatch(body); len(from) > 1 {
		payerName = cleanPayerName(from[1])
		if strings.EqualFold(payerName, upiID) {
			payerName = ""
		}
	}

	return domain.ParsedSMS{
		AmountPaise: amount,
		RRN:         truncateRunes(rrn, 64),
		UPIId:       truncateRunes(upiID, 255),
		PayerName:   truncateRunes(payerName, 255),
	}, nil
}

func ExtractReference(body string) string {
	for _, pattern := range rrnPatterns {
		if match := pattern.FindStringSubmatch(body); len(match) > 1 {
			return truncateRunes(strings.ToUpper(strings.TrimSpace(match[1])), 64)
		}
	}
	return ""
}

func findCreditAmount(body string) string {
	for _, pattern := range creditAmountPatterns {
		if match := pattern.FindStringSubmatch(body); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

func cleanPayerName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, " ,;:-")
	if upi := upiPattern.FindString(value); upi != "" {
		value = strings.TrimSpace(strings.Replace(value, upi, "", 1))
	}
	return value
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
