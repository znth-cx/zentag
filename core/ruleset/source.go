package ruleset

import (
	"fmt"
	"strings"

	"github.com/znth-cx/zentag/core/metadata"
)

// CheckSource validates the Source field contains only allowed values per RULES.md §3.
func CheckSource(meta *metadata.Metadata) []Violation {
	if meta == nil {
		return nil
	}

	if meta.Source == "" {
		return []Violation{
			{
				Rule:     "source",
				Severity: SeverityTrumpable,
				Message:  "source field is empty",
			},
		}
	}

	allowedSources := map[metadata.ReleaseSource]bool{
		metadata.ReleaseSourceWEB:   true,
		metadata.ReleaseSourceCD:    true,
		metadata.ReleaseSourceVinyl: true,
	}

	if !allowedSources[meta.Source] {
		return []Violation{
			{
				Rule:     "source",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("invalid source %q: must be one of %s", meta.Source, strings.Join([]string{string(metadata.ReleaseSourceWEB), string(metadata.ReleaseSourceCD), string(metadata.ReleaseSourceVinyl)}, ", ")),
			},
		}
	}

	return nil
}
