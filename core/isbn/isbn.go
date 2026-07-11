// Package isbn validates ISBN-10 and ISBN-13 checksums.
package isbn

import (
	"fmt"
	"strings"
)

// Validate strips hyphens/spaces and checks ISBN-10 or ISBN-13 checksum.
func Validate(isbn string) (bool, error) {
	cleaned := strings.ReplaceAll(strings.ReplaceAll(isbn, "-", ""), " ", "")

	switch len(cleaned) {
	case 10:
		return validateISBN10(cleaned)
	case 13:
		return validateISBN13(cleaned)
	default:
		return false, fmt.Errorf("isbn: %q is not a valid length (want 10 or 13 digits, got %d)", isbn, len(cleaned))
	}
}

func validateISBN10(s string) (bool, error) {
	sum := 0
	for i := 0; i < 10; i++ {
		var digit int
		switch {
		case i == 9 && (s[i] == 'X' || s[i] == 'x'):
			digit = 10
		case s[i] >= '0' && s[i] <= '9':
			digit = int(s[i] - '0')
		default:
			return false, fmt.Errorf("isbn: %q contains a non-digit character at position %d", s, i+1)
		}
		sum += digit * (10 - i)
	}
	return sum%11 == 0, nil
}

func validateISBN13(s string) (bool, error) {
	sum := 0
	for i := 0; i < 13; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false, fmt.Errorf("isbn: %q contains a non-digit character at position %d", s, i+1)
		}
		digit := int(s[i] - '0')
		weight := 1
		if i%2 == 1 {
			weight = 3
		}
		sum += digit * weight
	}
	return sum%10 == 0, nil
}
