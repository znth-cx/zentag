package ruleset

import (
	"fmt"

	"codeberg.org/Ether/zentag/core/lang"
	"codeberg.org/Ether/zentag/core/metadata"
)

// CheckLanguage checks RULES.md §3/§4: language must be a valid ISO-639-3 code or recognized English name.
// Empty Language is CheckRequiredTags's violation, not repeated here.
func CheckLanguage(meta *metadata.Metadata) []Violation {
	if meta.Language == "" || lang.ValidNameOrCode(meta.Language) {
		return nil
	}
	return []Violation{{
		Rule:     "language",
		Severity: SeverityTrumpable,
		Message:  fmt.Sprintf("invalid language code: %q", meta.Language),
	}}
}
