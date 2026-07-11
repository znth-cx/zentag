package ruleset

import (
	"fmt"
	"strings"

	edlib "github.com/hbollon/go-edlib"

	"github.com/znth-cx/zentag/core/metadata"
)

// bannedAuthors: RULES.md §1 list, lowercased for fuzzy match.
var bannedAuthors = []string{
	"j.r.r. tolkien",
	"anne perry",
	"simon scarrow",
	"sara gruen",
	"joan elliott",
	"alan dart",
	"chris mead",
	"paul moore & gavin jones",
	"noah k sturdevant",
	"benedict brown",
	"erika t wurth",
	"randolph lalonde",
	"unpublished works of j. d. salinger",
	"andrea sfiligoi",
	"ana-maria babanica",
}

// bannedWorks: RULES.md §1 list, lowercased for fuzzy match.
var bannedWorks = []string{
	"four against darkness expanded edition, and all associated content",
}

// bannedMatchThreshold: min Levenshtein similarity (0-1) to match banned name. Tolerates typos, avoids over-matching.
const bannedMatchThreshold = 0.85

// CheckBannedContent checks RULES.md §1: author/title must not fuzzy-match a banned author or work.
func CheckBannedContent(meta *metadata.Metadata) []Violation {
	var violations []Violation

	if len(meta.Author) > 0 && meta.Author[0] != "" {
		// error always nil for Levenshtein (only Hamming errors on unequal length); discarded.
		match, _ := edlib.FuzzySearchThreshold(strings.ToLower(meta.Author[0]), bannedAuthors, bannedMatchThreshold, edlib.Levenshtein)
		if match != "" {
			violations = append(violations, Violation{
				Rule:     "banned_content",
				Severity: SeverityProhibited,
				Message:  fmt.Sprintf("author/work matches banned list: %q", meta.Author[0]),
			})
		}
	}

	if meta.Title != "" {
		match, _ := edlib.FuzzySearchThreshold(strings.ToLower(meta.Title), bannedWorks, bannedMatchThreshold, edlib.Levenshtein)
		if match != "" {
			violations = append(violations, Violation{
				Rule:     "banned_content",
				Severity: SeverityProhibited,
				Message:  fmt.Sprintf("author/work matches banned list: %q", meta.Title),
			})
		}
	}

	return violations
}
