package hw02unpackstring

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidString = errors.New("invalid string")
	errAtDigit       = fmt.Errorf("found digit when symbol to decode is undefined: %w", ErrInvalidString)
	errAtBackslash   = fmt.Errorf("only digits or backslash are allowed after escape symbol '\\': %w", ErrInvalidString)
	outputSB         strings.Builder
)

func applyRune(r rune, count int) {
	for i := 0; i < count; i++ {
		outputSB.WriteRune(r)
	}
}

func isDigit(r rune) bool {
	if r >= '0' && r <= '9' {
		return true
	}
	return false
}

/*
	Convert input line to rune slice, then process runes one-by-one.

Put rune in cache, then process it at next iteration,
in case if next rune will be digit then add cached rune "digit" times or delete if digit is 0.
If next rune isn't number add cached rune once.
*/
func Unpack(inputString string) (string, error) {
	outputSB.Reset()
	var runeCached rune
	runes := []rune(inputString)
	for pos := 0; pos < len(runes); pos++ {
		if isDigit(runes[pos]) {
			if runeCached == 0 {
				return "", errAtDigit
			}

			count := int(runes[pos]) - '0'
			if count != 0 {
				applyRune(runeCached, count)
			}
			runeCached = 0
			continue
		}

		if runeCached != 0 {
			applyRune(runeCached, 1)
		}

		// In case if line ended with backslash, then backslash will be processed as regular symbol.
		if runes[pos] == '\\' && pos < len(runes)-1 {
			pos++
			if !isDigit(runes[pos]) && runes[pos] != '\\' {
				return "", errAtBackslash
			}
		}

		runeCached = runes[pos]
	}
	if runeCached != 0 {
		applyRune(runeCached, 1)
	}
	return outputSB.String(), nil
}
