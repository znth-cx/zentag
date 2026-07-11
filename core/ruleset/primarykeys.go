package ruleset

import (
	"codeberg.org/Ether/zentag/core/isbn"
	"codeberg.org/Ether/zentag/core/metadata"
)

// CheckPrimaryKeys checks RULES.md §2: needs ISBN or ASIN; if ISBN present, checksum must be valid.
func CheckPrimaryKeys(meta *metadata.Metadata) []Violation {
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
		}
	}

	return violations
}
