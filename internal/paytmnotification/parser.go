package paytmnotification

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/money"
)

var (
	currencyAmount  = `(?:rs\.?|inr|₹)\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)`
	paymentPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:payment\s+)?received\b.{0,80}?` + currencyAmount),
		regexp.MustCompile(`(?i)` + currencyAmount + `.{0,40}?\b(?:received|paid)\b`),
		regexp.MustCompile(`(?i)\b(?:received|paid)\b.{0,40}?` + currencyAmount),
	}
	payerPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bpaid\s+by\s+(.+?)(?:[.!|\n]|$)`),
		regexp.MustCompile(`(?i)\b(?:received|payment\s+received)\b.{0,100}?\bfrom\s+(.+?)(?:[.!|\n]|$)`),
		regexp.MustCompile(`(?i)` + currencyAmount + `.{0,80}?\bfrom\s+(.+?)(?:[.!|\n]|$)`),
	}
	occurredAtPattern = regexp.MustCompile(`(?i)\breceived\s+on\s+(\d{1,2}\s+[A-Za-z]{3}\s+\d{4}\s+\d{1,2}:\d{2}\s+(?:AM|PM))\b`)
	nonPaymentPattern = regexp.MustCompile(`(?i)\b(?:refund(?:ed)?|reversal|reversed|cashback|reward|settlement|settled|loan|emi|interest|chargeback)\b`)
)

var ErrUnrecognized = errors.New("notification is not a recognized Paytm customer payment")

type Parsed struct {
	AmountPaise int64
	PayerName   string
	OccurredAt  time.Time
}

func Parse(text string) (Parsed, error) {
	text = strings.TrimSpace(text)
	if text == "" || nonPaymentPattern.MatchString(text) {
		return Parsed{}, ErrUnrecognized
	}
	amountText := ""
	for _, pattern := range paymentPatterns {
		if match := pattern.FindStringSubmatch(text); len(match) > 1 {
			amountText = match[1]
			break
		}
	}
	if amountText == "" {
		return Parsed{}, ErrUnrecognized
	}
	amount, err := money.ParseAmount(amountText)
	if err != nil || amount <= 0 {
		if err != nil {
			return Parsed{}, err
		}
		return Parsed{}, money.ErrInvalidAmount
	}
	payer := ""
	for _, pattern := range payerPatterns {
		if match := pattern.FindStringSubmatch(text); len(match) > 1 {
			payer = cleanPayer(match[len(match)-1])
			if payer != "" {
				break
			}
		}
	}
	occurredAt := parseOccurredAt(text)
	return Parsed{AmountPaise: amount, PayerName: payer, OccurredAt: occurredAt}, nil
}

func parseOccurredAt(text string) time.Time {
	match := occurredAtPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return time.Time{}
	}
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return time.Time{}
	}
	parsed, err := time.ParseInLocation("2 Jan 2006 03:04 PM", strings.TrimSpace(match[1]), loc)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func cleanPayer(value string) string {
	value = strings.TrimSpace(strings.Trim(value, " ,;:-"))
	if len([]rune(value)) > 255 {
		value = string([]rune(value)[:255])
	}
	return value
}
