package ruleset

import (
	"github.com/znth-cx/zentag/core/metadata"
)

// MultiValueTags defines which fields are multi-value for each container format.
// This infrastructure is used by CheckTagSeparators (Phase 4).
var MultiValueTags = map[string][]string{
	"M4B":  {"author", "narrator", "genre", "publisher", "series"},
	"MP3":  {"author", "narrator", "genre", "publisher", "series"},
	"FLAC": {"author", "narrator", "genre", "publisher", "series"},
}

// CheckFormatSpecificTags validates format-specific required tags per RULES.md §4.
// It dispatches to the appropriate format-specific checker based on the container type.
func CheckFormatSpecificTags(meta *metadata.Metadata) []Violation {
	if meta == nil || len(meta.Tracks) == 0 {
		return nil
	}

	container := meta.Tracks[0].Container
	switch container {
	case "M4B":
		return CheckM4BTags(meta)
	case "MP3":
		return CheckMP3Tags(meta)
	case "FLAC":
		return CheckFLACTags(meta)
	default:
		// Unknown container type, skip gracefully
		return nil
	}
}

// CheckM4BTags validates M4B-specific required tags per RULES.md §4.
// Checks: Author, Title, Year, Narrator, Series/Part, Language, ISBN/ASIN, Cover.
func CheckM4BTags(meta *metadata.Metadata) []Violation {
	var violations []Violation

	// .ART (author)
	if len(meta.Author) == 0 || meta.Author[0] == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "M4B: missing required tag .ART (author)",
		})
	}

	// .nam (title)
	if meta.Title == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "M4B: missing required tag .nam (title)",
		})
	}

	// .day (year)
	if meta.Year == 0 {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "M4B: missing required tag .day (year)",
		})
	}

	// .wrt (narrator)
	if len(meta.Narrator) == 0 || meta.Narrator[0] == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "M4B: missing required tag .wrt (narrator)",
		})
	}

	// Language (mdhd)
	if meta.Language == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "M4B: missing required tag Language (mdhd)",
		})
	}

	// Series/Part
	if len(meta.Series) > 0 {
		for _, s := range meta.Series {
			if s.Part == "" {
				violations = append(violations, Violation{
					Rule:     "format_specific_tags",
					Severity: SeverityTrumpable,
					Message:  "M4B: missing required tag ----com.apple.iTunes:SERIES-PART",
				})
			}
		}
	}

	// ISBN/ASIN (at least one required)
	if meta.ISBN == "" && meta.ASIN == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "M4B: missing required tag ----com.apple.iTunes:ISBN or ----com.apple.iTunes:ASIN",
		})
	}

	// covr (embedded cover)
	if len(meta.CoverImage) == 0 {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "M4B: missing required tag covr (cover image)",
		})
	}

	return violations
}

// CheckMP3Tags validates MP3-specific required tags per RULES.md §4.
// Checks: Title, Author, Year, Language, Narrator, Series/Part, ISBN/ASIN.
func CheckMP3Tags(meta *metadata.Metadata) []Violation {
	var violations []Violation

	// TIT2 (title)
	if meta.Title == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "MP3: missing required tag TIT2 (title)",
		})
	}

	// TPE1 (artist/author)
	if len(meta.Author) == 0 || meta.Author[0] == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "MP3: missing required tag TPE1 (artist/author)",
		})
	}

	// TDRC (recording time/year)
	if meta.Year == 0 {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "MP3: missing required tag TDRC (year)",
		})
	}

	// TLAN (language)
	if meta.Language == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "MP3: missing required tag TLAN (language)",
		})
	}

	// TCOM (composer/narrator)
	if len(meta.Narrator) == 0 || meta.Narrator[0] == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "MP3: missing required tag TCOM (composer/narrator)",
		})
	}

	// Series/Part
	if len(meta.Series) > 0 {
		for _, s := range meta.Series {
			if s.Part == "" {
				violations = append(violations, Violation{
					Rule:     "format_specific_tags",
					Severity: SeverityTrumpable,
					Message:  "MP3: missing required tag TXXX:SERIES-PART",
				})
			}
		}
	}

	// ISBN/ASIN (at least one required)
	if meta.ISBN == "" && meta.ASIN == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "MP3: missing required tag TXXX:ISBN or TXXX:ASIN",
		})
	}

	return violations
}

// CheckFLACTags validates FLAC-specific required tags per RULES.md §4.
// Checks: Author, Title, Year, Narrator, Series/Part, Language, ISBN/ASIN.
func CheckFLACTags(meta *metadata.Metadata) []Violation {
	var violations []Violation

	// author
	if len(meta.Author) == 0 || meta.Author[0] == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "FLAC: missing required tag author",
		})
	}

	// title
	if meta.Title == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "FLAC: missing required tag title",
		})
	}

	// year
	if meta.Year == 0 {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "FLAC: missing required tag year",
		})
	}

	// narrator
	if len(meta.Narrator) == 0 || meta.Narrator[0] == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "FLAC: missing required tag narrator",
		})
	}

	// language
	if meta.Language == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "FLAC: missing required tag language",
		})
	}

	// series/series-part
	if len(meta.Series) > 0 {
		for _, s := range meta.Series {
			if s.Part == "" {
				violations = append(violations, Violation{
					Rule:     "format_specific_tags",
					Severity: SeverityTrumpable,
					Message:  "FLAC: missing required tag series-part",
				})
			}
		}
	}

	// isbn/asin (at least one required)
	if meta.ISBN == "" && meta.ASIN == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  "FLAC: missing required tag isbn or asin",
		})
	}

	return violations
}
