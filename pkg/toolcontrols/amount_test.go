package toolcontrols

import (
	"errors"
	"testing"
)

func TestParseWholeAmountAcceptsPositiveWholeNumbers(t *testing.T) {
	for text, want := range map[string]int64{
		"1":                   1,
		"60":                  60,
		"5.00":                5,
		"100.0":               100,
		"9223372036854775807": 9223372036854775807,
	} {
		got, err := ParseWholeAmount(text)
		if err != nil || got != want {
			t.Errorf("ParseWholeAmount(%q) = %d, %v; want %d", text, got, err, want)
		}
	}
}

func TestParseWholeAmountRefusesEverythingElse(t *testing.T) {
	for _, text := range []string{
		"0", "0.00", "-0", "-0.00", // zero
		"-1", "-5.00", // negative
		"0.5", "1.5", "2.25", "0.00000001", // fractional
		"", " ", " 5", "5 ", "+5", "05", "5.", ".5", "1e3", "1E3", "0x10", "NaN", "Inf", "-Inf", "five", "5abc", "1_000", // malformed
		"9223372036854775808", "99999999999999999999", "123456789012345678901234567890123456789", // out of range
	} {
		got, err := ParseWholeAmount(text)
		if err == nil {
			t.Errorf("ParseWholeAmount(%q) = %d, want an error", text, got)
			continue
		}
		if !errors.Is(err, ErrNotWholeAmount) {
			t.Errorf("ParseWholeAmount(%q) error = %v, want ErrNotWholeAmount", text, err)
		}
	}
}
