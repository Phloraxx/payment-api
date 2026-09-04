package observations

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/money"
)

const (
	PaytmBusinessPackage          = "com.paytm.business"
	GoogleMessagesPackage         = "com.google.android.apps.messaging"
	GenericNotificationSource     = "android_notification"
	GenericMessageSource          = "android_message"
	paytmPostTimeRefinementWindow = time.Minute
)

var (
	ErrUnrecognized     = errors.New("notification is not a recognized incoming PayGate payment")
	ErrNonPayGateAmount = errors.New("incoming amount is not a PayGate decimal amount")
)

type Snapshot struct {
	PackageName string
	PostedAt    time.Time
	Title       string
	Text        string
	BigText     string
}

type Observation struct {
	Source              string
	CollectionProfileID string
	AmountPaise         int64
	PayerName           string
	PayerUPIID          string
	OccurredAt          time.Time
	OccurredAtSource    string
}

var (
	currencyAmount   = `(?:rs\.?|inr|₹)\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)`
	incomingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:payment\s+)?received\b.{0,120}?` + currencyAmount),
		regexp.MustCompile(`(?i)` + currencyAmount + `.{0,80}?\b(?:received|credited|deposited)\b`),
		regexp.MustCompile(`(?i)\b(?:received|credited|deposited)\b.{0,120}?` + currencyAmount),
		regexp.MustCompile(`(?i)\byou\s+(?:have\s+)?received\b.{0,100}?` + currencyAmount),
		regexp.MustCompile(`(?i)\bmoney\s+received\b.{0,100}?` + currencyAmount),
		regexp.MustCompile(`(?i)\bpaid\s+you\b.{0,120}?` + currencyAmount),
	}
	paytmAmountPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:payment\s+)?received\b.{0,100}?` + currencyAmount),
		regexp.MustCompile(`(?i)` + currencyAmount + `.{0,60}?\breceived\b`),
		regexp.MustCompile(`(?i)\breceived\b.{0,60}?` + currencyAmount),
	}
	nonPaymentPattern    = regexp.MustCompile(`(?i)\b(?:reversal|reversed|refund(?:ed)?|cashback|reward|interest|salary|chargeback|settlement|settled|loan|emi|bill|due|reminder)\b`)
	debitPattern         = regexp.MustCompile(`(?i)\b(?:debited|sent|you\s+paid|paid\s+to|paid\s+for|withdrawn|purchase|spent|transferred\s+to)\b`)
	upiPattern           = regexp.MustCompile(`(?i)[a-z0-9][a-z0-9._-]{0,127}@[a-z0-9][a-z0-9._-]{0,127}`)
	fromPattern          = regexp.MustCompile(`(?i)\b(?:from|by)\s+(.+?)(?:\s+at\s+\d{1,2}:\d{2}(?:\s*[ap]m)?\b|\s+on\s+|\s+(?:upi\s+)?(?:ref|rrn|utr)|[!|\n]|\.(?:\s|$)|$)`)
	paidYouPayerPattern  = regexp.MustCompile(`(?i)^(.{1,120}?)\s+paid\s+you\b`)
	paytmOccurredPattern = regexp.MustCompile(`(?i)\breceived\s+on\s+(\d{1,2}\s+[A-Za-z]{3}\s+\d{4}\s+\d{1,2}:\d{2}\s+(?:AM|PM))\b`)
)

func Parse(snapshot Snapshot) (Observation, error) {
	text := combinedText(snapshot)
	pkg := strings.TrimSpace(snapshot.PackageName)
	if pkg == "" || strings.TrimSpace(text) == "" {
		return Observation{}, ErrUnrecognized
	}
	if pkg == PaytmBusinessPackage {
		return parsePaytm(text, snapshot.PostedAt)
	}
	if pkg == GoogleMessagesPackage && strings.Contains(strings.ToLower(text), "kotak") {
		return parseKotak(text, snapshot.PostedAt)
	}
	source := GenericNotificationSource
	if pkg == GoogleMessagesPackage {
		source = GenericMessageSource
	}
	return parseGeneric(text, snapshot.PostedAt, source)
}

func parseGeneric(text string, postedAt time.Time, source string) (Observation, error) {
	if rejectedTransactionText(text) {
		return Observation{}, ErrUnrecognized
	}
	amountText := firstAmount(text, incomingPatterns)
	if amountText == "" {
		return Observation{}, ErrUnrecognized
	}
	amount, err := parsePayGateAmount(amountText)
	if err != nil {
		return Observation{}, err
	}
	payerName, payerUPI := extractPayer(text)
	return Observation{
		Source: source, AmountPaise: amount, PayerName: payerName, PayerUPIID: payerUPI,
		OccurredAt: postedAt.UTC(), OccurredAtSource: "notification_posted_at",
	}, nil
}

func parsePaytm(text string, postedAt time.Time) (Observation, error) {
	if rejectedTransactionText(text) {
		return Observation{}, ErrUnrecognized
	}
	amountText := firstAmount(text, paytmAmountPatterns)
	if amountText == "" {
		return Observation{}, ErrUnrecognized
	}
	amount, err := parsePayGateAmount(amountText)
	if err != nil {
		return Observation{}, err
	}
	payerName, payerUPI := extractPayer(text)
	occurredAt, source := parsePaytmOccurredAt(text)
	if !occurredAt.IsZero() && canRefinePaytmMinuteWithPostedAt(occurredAt, postedAt) {
		occurredAt = postedAt.UTC()
		source = "notification_posted_at"
	}
	if occurredAt.IsZero() {
		occurredAt = postedAt.UTC()
		source = "notification_posted_at"
	}
	return Observation{Source: "paytm_notification", CollectionProfileID: "paytm", AmountPaise: amount,
		PayerName: payerName, PayerUPIID: payerUPI, OccurredAt: occurredAt, OccurredAtSource: source}, nil
}

func parseKotak(text string, postedAt time.Time) (Observation, error) {
	if rejectedTransactionText(text) {
		return Observation{}, ErrUnrecognized
	}
	amountText := firstAmount(text, incomingPatterns)
	if amountText == "" {
		return Observation{}, ErrUnrecognized
	}
	amount, err := parsePayGateAmount(amountText)
	if err != nil {
		return Observation{}, err
	}
	payerName, payerUPI := extractPayer(text)
	return Observation{Source: "kotak_sms", CollectionProfileID: "kotak", AmountPaise: amount,
		PayerName: payerName, PayerUPIID: payerUPI, OccurredAt: postedAt.UTC(), OccurredAtSource: "notification_posted_at"}, nil
}

func rejectedTransactionText(text string) bool {
	return strings.TrimSpace(text) == "" || nonPaymentPattern.MatchString(text) || debitPattern.MatchString(text)
}

func parsePayGateAmount(value string) (int64, error) {
	amount, err := money.ParseAmount(value)
	if err != nil || amount <= 0 {
		if err != nil {
			return 0, err
		}
		return 0, ErrUnrecognized
	}
	if amount%100 == 0 {
		return 0, ErrNonPayGateAmount
	}
	return amount, nil
}

func firstAmount(text string, patterns []*regexp.Regexp) string {
	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(text); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

func extractPayer(text string) (string, string) {
	upiID := strings.TrimSpace(upiPattern.FindString(text))
	payerName := ""
	if match := paidYouPayerPattern.FindStringSubmatch(text); len(match) > 1 {
		payerName = cleanPayer(match[1], upiID)
	} else if match := fromPattern.FindStringSubmatch(text); len(match) > 1 {
		payerName = cleanPayer(match[1], upiID)
	}
	return truncateRunes(payerName, 255), truncateRunes(upiID, 255)
}
func cleanPayer(value, upiID string) string {
	value = strings.Trim(strings.TrimSpace(value), " ,;:-")
	if upiID != "" {
		parenthesizedUPI := "(" + upiID + ")"
		if strings.Contains(value, parenthesizedUPI) {
			value = strings.Replace(value, parenthesizedUPI, "", 1)
		} else {
			value = strings.Replace(value, upiID, "", 1)
		}
		value = strings.TrimSpace(value)
	}
	return value
}
func canRefinePaytmMinuteWithPostedAt(minuteStart, postedAt time.Time) bool {
	if minuteStart.IsZero() || postedAt.IsZero() {
		return false
	}
	posted, start := postedAt.UTC(), minuteStart.UTC()
	return !posted.Before(start) && posted.Before(start.Add(paytmPostTimeRefinementWindow))
}
func parsePaytmOccurredAt(text string) (time.Time, string) {
	match := paytmOccurredPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return time.Time{}, ""
	}
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return time.Time{}, ""
	}
	parsed, err := time.ParseInLocation("2 Jan 2006 03:04 PM", strings.TrimSpace(match[1]), loc)
	if err != nil {
		return time.Time{}, ""
	}
	return parsed.UTC(), "notification_text"
}
func combinedText(snapshot Snapshot) string {
	seen := map[string]struct{}{}
	parts := make([]string, 0, 3)
	for _, value := range []string{snapshot.Title, snapshot.Text, snapshot.BigText} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		parts = append(parts, value)
	}
	return strings.Join(parts, " ")
}
func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
