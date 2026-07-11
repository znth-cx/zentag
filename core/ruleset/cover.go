package ruleset

import (
	"context"

	"github.com/znth-cx/zentag/core/cover"
	"github.com/znth-cx/zentag/core/metadata"
)

// CheckCover checks RULES.md §8: cover must be present and valid (decodable JPEG/PNG, under 3MB).
func CheckCover(ctx context.Context, meta *metadata.Metadata) []Violation {
	if len(meta.CoverImage) == 0 {
		return []Violation{{Rule: "cover", Severity: SeverityTrumpable, Message: "missing cover"}}
	}

	if ok, reason := cover.Validate(ctx, meta.CoverImage); !ok {
		return []Violation{{Rule: "cover", Severity: SeverityTrumpable, Message: reason}}
	}
	return nil
}
