package ruleset

import (
	"fmt"

	"codeberg.org/Ether/zentag/core/metadata"
)

// CheckRequiredTags checks RULES.md §4: each missing required field is its own Trumpable violation.
func CheckRequiredTags(meta *metadata.Metadata) []Violation {
	var violations []Violation

	if len(meta.Author) == 0 || meta.Author[0] == "" {
		violations = append(violations, Violation{Rule: "required_tags", Severity: SeverityTrumpable, Message: "missing required tag: author"})
	}
	if meta.Title == "" {
		violations = append(violations, Violation{Rule: "required_tags", Severity: SeverityTrumpable, Message: "missing required tag: title"})
	}
	if meta.Year == 0 {
		violations = append(violations, Violation{Rule: "required_tags", Severity: SeverityTrumpable, Message: "missing required tag: year"})
	}
	if len(meta.Narrator) == 0 || meta.Narrator[0] == "" {
		violations = append(violations, Violation{Rule: "required_tags", Severity: SeverityTrumpable, Message: "missing required tag: narrator"})
	}
	if meta.Language == "" {
		violations = append(violations, Violation{Rule: "required_tags", Severity: SeverityTrumpable, Message: "missing required tag: language"})
	}
	for _, s := range meta.Series {
		if s.Part == "" {
			violations = append(violations, Violation{
				Rule:     "required_tags",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("series part missing for series %q", s.Name),
			})
		}
	}

	return violations
}
