package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var wholeRupees = regexp.MustCompile(`^[1-9][0-9]*$`)
var paisaAmount = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)

var ErrInvalidAmount = errors.New("amount must be a positive whole number of INR rupees")

var maxRequestedRupees = (int64(^uint64(0)>>1) - 99) / 100

func ParseWholeRupees(raw json.RawMessage) (int64, error) {
	value := strings.TrimSpace(string(raw))
	if len(value) >= 2 && value[0] == '"' {
		var decoded string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return 0, ErrInvalidAmount
		}
		value = strings.TrimSpace(decoded)
	}
	if !wholeRupees.MatchString(value) {
		return 0, ErrInvalidAmount
	}
	r, err := strconv.ParseInt(value, 10, 64)
	if err != nil || r <= 0 || r > maxRequestedRupees {
		return 0, ErrInvalidAmount
	}
	return r, nil
}

func RupeesToPaise(rupees int64) (int64, error) {
	if rupees <= 0 || rupees > maxRequestedRupees {
		return 0, ErrInvalidAmount
	}
	return rupees * 100, nil
}

func ParseAmount(text string) (int64, error) {
	text = strings.ReplaceAll(strings.TrimSpace(text), ",", "")
	if !paisaAmount.MatchString(text) {
		return 0, fmt.Errorf("invalid INR amount %q", text)
	}
	parts := strings.SplitN(text, ".", 2)
	r, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || r < 0 || r > maxRequestedRupees {
		return 0, fmt.Errorf("invalid INR amount %q", text)
	}
	p := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		p, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid INR amount %q", text)
		}
	}
	return r*100 + p, nil
}

func FormatPaise(paise int64) string { return fmt.Sprintf("%d.%02d", paise/100, paise%100) }
