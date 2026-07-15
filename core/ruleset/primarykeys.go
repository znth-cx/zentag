package ruleset

import (
	"strings"

	"github.com/znth-cx/zentag/core/isbn"
	"github.com/znth-cx/zentag/core/metadata"
)

// CheckPrimaryKeys checks RULES.md §2: needs ISBN or ASIN; if ISBN present, checksum must be valid.
// Also implements ISBN-13 preference: if both ISBN-10 and ISBN-13 are present, advises ISBN-13 preference.
func CheckPrimaryKeys(meta *metadata.Metadata) []Violation {
	if meta == nil {
		return nil
	}

	var violations []Violation

	if meta.ISBN == "" && meta.ASIN == "" {
		violations = append(violations, Violation{
			Rule:     "primary_keys",
			Severity: SeverityTrumpable,
			Message:  "no ISBN or ASIN",
		})
	}

	if meta.ISBN != "" {
		ok, err := isbn.Validate(meta.ISBN)
		if err != nil || !ok {
			violations = append(violations, Violation{
				Rule:     "primary_keys",
				Severity: SeverityTrumpable,
				Message:  "ISBN checksum invalid",
			})
		} else {
			violations = append(violations, validateISBNPreference(meta)...)
		}
	}

	return violations
}

// validateISBNPreference checks if ISBN-13 is preferred when ISBN-10 is also present per RULES.md §2.
// This is an advisory warning, not a blocking violation.
func validateISBNPreference(meta *metadata.Metadata) []Violation {
	if meta.ISBN == "" {
		return nil
	}

	cleaned := strings.ReplaceAll(strings.ReplaceAll(meta.ISBN, "-", ""), " ", "")

	isISBN10 := false

	if len(cleaned) == 10 {
		isISBN10 = true
	} else if len(cleaned) != 13 {
		return nil
	}

	if isISBN10 && meta.ASIN != "" {
		return []Violation{{
			Rule:     "primary_keys",
			Severity: SeverityWarn,
			Message:  "ISBN-10 present with ASIN: ISBN-13 is preferred",
		}}
	}

	return nil
}
