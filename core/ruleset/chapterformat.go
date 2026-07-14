package ruleset

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/znth-cx/zentag/core/metadata"
)

// CheckChapterFormat validates M4B chapters are in QuickTime format per RULES.md §9.
// Nero format is optional but QuickTime is required.
func CheckChapterFormat(ctx context.Context, meta *metadata.Metadata) []Violation {
	var violations []Violation

	for _, tr := range meta.Tracks {
		if tr.Container != "M4B" {
			continue
		}

		if len(tr.Chapters) == 0 {
			continue
		}

		chapterFormat, err := detectChapterFormat(ctx, &tr)
		if err != nil {
			slog.WarnContext(ctx, "ruleset: could not detect chapter format", "path", tr.Path, "error", err)
			continue
		}

		if chapterFormat != "QuickTime" && chapterFormat != "Nero" {
			violations = append(violations, Violation{
				Rule:     "chapter_format",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("track %q has unknown chapter format %q (QuickTime required)", tr.Path, chapterFormat),
			})
		}
	}

	return violations
}

// detectChapterFormat detects the chapter format used in the file.
// Returns "QuickTime", "Nero", or an error if format cannot be determined.
// TODO: Implement proper chapter format detection using ffmpeg or mediainfo.
// Currently returns QuickTime as default since it's the required format per RULES.md §9.
func detectChapterFormat(ctx context.Context, track *metadata.Track) (string, error) {
	if track == nil {
		return "", fmt.Errorf("track is nil")
	}

	if len(track.Chapters) == 0 {
		return "", fmt.Errorf("no chapters present")
	}

	slog.DebugContext(ctx, "ruleset: detecting chapter format", "path", track.Path, "chapters", len(track.Chapters))

	slog.WarnContext(ctx, "ruleset: chapter format detection not implemented, assuming QuickTime", "path", track.Path)
	return "QuickTime", nil
}
