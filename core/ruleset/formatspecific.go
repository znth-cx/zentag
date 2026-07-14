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

type tagCheck struct {
	fieldName string
	tagName   string
	getter    func(*metadata.Metadata) (bool, string)
}

func getTagChecks(container string) []tagCheck {
	switch container {
	case "M4B":
		return []tagCheck{
			{fieldName: "author", tagName: ".ART (author)", getter: func(m *metadata.Metadata) (bool, string) {
				return len(m.Author) == 0 || m.Author[0] == "", "missing required tag .ART (author)"
			}},
			{fieldName: "title", tagName: ".nam (title)", getter: func(m *metadata.Metadata) (bool, string) { return m.Title == "", "missing required tag .nam (title)" }},
			{fieldName: "year", tagName: ".day (year)", getter: func(m *metadata.Metadata) (bool, string) { return m.Year == 0, "missing required tag .day (year)" }},
			{fieldName: "narrator", tagName: ".wrt (narrator)", getter: func(m *metadata.Metadata) (bool, string) {
				return len(m.Narrator) == 0 || m.Narrator[0] == "", "missing required tag .wrt (narrator)"
			}},
			{fieldName: "language", tagName: "Language (mdhd)", getter: func(m *metadata.Metadata) (bool, string) {
				return m.Language == "", "missing required tag Language (mdhd)"
			}},
		}
	case "MP3":
		return []tagCheck{
			{fieldName: "title", tagName: "TIT2 (title)", getter: func(m *metadata.Metadata) (bool, string) { return m.Title == "", "missing required tag TIT2 (title)" }},
			{fieldName: "author", tagName: "TPE1 (artist/author)", getter: func(m *metadata.Metadata) (bool, string) {
				return len(m.Author) == 0 || m.Author[0] == "", "missing required tag TPE1 (artist/author)"
			}},
			{fieldName: "year", tagName: "TDRC (year)", getter: func(m *metadata.Metadata) (bool, string) { return m.Year == 0, "missing required tag TDRC (year)" }},
			{fieldName: "language", tagName: "TLAN (language)", getter: func(m *metadata.Metadata) (bool, string) {
				return m.Language == "", "missing required tag TLAN (language)"
			}},
			{fieldName: "narrator", tagName: "TCOM (composer/narrator)", getter: func(m *metadata.Metadata) (bool, string) {
				return len(m.Narrator) == 0 || m.Narrator[0] == "", "missing required tag TCOM (composer/narrator)"
			}},
		}
	case "FLAC":
		return []tagCheck{
			{fieldName: "author", tagName: "author", getter: func(m *metadata.Metadata) (bool, string) {
				return len(m.Author) == 0 || m.Author[0] == "", "missing required tag author"
			}},
			{fieldName: "title", tagName: "title", getter: func(m *metadata.Metadata) (bool, string) { return m.Title == "", "missing required tag title" }},
			{fieldName: "year", tagName: "year", getter: func(m *metadata.Metadata) (bool, string) { return m.Year == 0, "missing required tag year" }},
			{fieldName: "narrator", tagName: "narrator", getter: func(m *metadata.Metadata) (bool, string) {
				return len(m.Narrator) == 0 || m.Narrator[0] == "", "missing required tag narrator"
			}},
			{fieldName: "language", tagName: "language", getter: func(m *metadata.Metadata) (bool, string) { return m.Language == "", "missing required tag language" }},
		}
	default:
		return nil
	}
}

func getSeriesTag(container string) (tagName, message string) {
	switch container {
	case "M4B":
		return "----com.apple.iTunes:SERIES-PART", "missing required tag ----com.apple.iTunes:SERIES-PART"
	case "MP3":
		return "TXXX:SERIES-PART", "missing required tag TXXX:SERIES-PART"
	case "FLAC":
		return "series-part", "missing required tag series-part"
	default:
		return "", ""
	}
}

func getISBNTag(container string) (tagName, message string) {
	switch container {
	case "M4B":
		return "----com.apple.iTunes:ISBN or ----com.apple.iTunes:ASIN", "missing required tag ----com.apple.iTunes:ISBN or ----com.apple.iTunes:ASIN"
	case "MP3":
		return "TXXX:ISBN or TXXX:ASIN", "missing required tag TXXX:ISBN or TXXX:ASIN"
	case "FLAC":
		return "isbn or asin", "missing required tag isbn or asin"
	default:
		return "", ""
	}
}

func getCoverTag(container string) (tagName, message string) {
	switch container {
	case "M4B":
		return "covr (cover image)", "missing required tag covr (cover image)"
	default:
		return "", ""
	}
}

// CheckFormatSpecificTags validates format-specific required tags per RULES.md §4.
// It dispatches to the appropriate format-specific checker based on the container type.
func CheckFormatSpecificTags(meta *metadata.Metadata) []Violation {
	if meta == nil || len(meta.Tracks) == 0 {
		return nil
	}

	container := meta.Tracks[0].Container
	checks := getTagChecks(container)
	if checks == nil {
		return nil
	}

	var violations []Violation
	containerPrefix := container + ": "

	for _, check := range checks {
		if missing, msg := check.getter(meta); missing {
			violations = append(violations, Violation{
				Rule:     "format_specific_tags",
				Severity: SeverityTrumpable,
				Message:  containerPrefix + msg,
			})
		}
	}

	seriesTag, seriesMsg := getSeriesTag(container)
	if seriesTag != "" && len(meta.Series) > 0 {
		for _, s := range meta.Series {
			if s.Part == "" {
				violations = append(violations, Violation{
					Rule:     "format_specific_tags",
					Severity: SeverityTrumpable,
					Message:  containerPrefix + seriesMsg,
				})
			}
		}
	}

	isbnTag, isbnMsg := getISBNTag(container)
	if isbnTag != "" && meta.ISBN == "" && meta.ASIN == "" {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  containerPrefix + isbnMsg,
		})
	}

	coverTag, coverMsg := getCoverTag(container)
	if coverTag != "" && len(meta.CoverImage) == 0 {
		violations = append(violations, Violation{
			Rule:     "format_specific_tags",
			Severity: SeverityTrumpable,
			Message:  containerPrefix + coverMsg,
		})
	}

	return violations
}
