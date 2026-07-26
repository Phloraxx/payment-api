package money

import (
	"encoding/json"
	"testing"
)

func TestParseWholeRupeesRejectsPaiseAndNonIntegers(t *testing.T) {
	for _, raw := range []string{`100.01`, `100.0`, `1e2`, `0`, `-1`, `"100.01"`, `"+100"`} {
		if _, err := ParseWholeRupees(json.RawMessage(raw)); err == nil {
			t.Errorf("ParseWholeRupees(%s) accepted fractional or invalid amount", raw)
		}
	}
	for _, raw := range []string{`100`, `"100"`} {
		if got, err := ParseWholeRupees(json.RawMessage(raw)); err != nil || got != 100 {
			t.Errorf("ParseWholeRupees(%s) = %d, %v; want 100", raw, got, err)
		}
	}
}

func TestParseAmountUsesIntegerPaise(t *testing.T) {
	cases := map[string]int64{"1": 100, "1.5": 150, "1.05": 105, "1,234.50": 123450}
	for input, want := range cases {
		got, err := ParseAmount(input)
		if err != nil || got != want {
			t.Errorf("ParseAmount(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
}

func TestRequestedAmountReservesDDMSuffixHeadroom(t *testing.T) {
	if _, err := RupeesToPaise(maxRequestedRupees); err != nil {
		t.Fatalf("maximum safe requested rupees rejected: %v", err)
	}
	if _, err := RupeesToPaise(maxRequestedRupees + 1); err == nil {
		t.Fatal("requested amount that can overflow when adding a DDM suffix was accepted")
	}
}
