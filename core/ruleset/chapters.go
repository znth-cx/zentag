package ruleset

import (
	"fmt"

	"codeberg.org/Ether/zentag/core/metadata"
)

// CheckChapters checks RULES.md §9: M4B tracks need >=1 chapter.
// MP3/FLAC skipped: chapters not embedded there (see ffmpeg.WriteOpts.EmbedChapters).
func CheckChapters(meta *metadata.Metadata) []Violation {
	var violations []Violation
	for _, tr := range meta.Tracks {
		if tr.Container != "M4B" {
			continue
		}
		if len(tr.Chapters) == 0 {
			violations = append(violations, Violation{
				Rule: "chapters",
				// warn only: RULES.md §9 requires chapters iff source had them, unknowable from metadata
				Severity: SeverityWarn,
				Message:  fmt.Sprintf("track %q has no chapters", tr.Path),
			})
		}
	}
	return violations
}

// CheckAudnexusChapters compares audnexus's chapter count against actual chapters/parts. Single-file MP3/FLAC skipped (no count to compare).
// Not a RULES.md rule (audnexus data can be wrong), so SeverityWarn, not a fixability category.
func CheckAudnexusChapters(meta *metadata.Metadata) []Violation {
	if meta.AudnexusChapterCount <= 0 || len(meta.Tracks) == 0 {
		return nil
	}

	var have int
	if meta.Tracks[0].Container == "M4B" {
		for _, tr := range meta.Tracks {
			have += len(tr.Chapters)
		}
	} else if len(meta.Tracks) > 1 {
		have = len(meta.Tracks)
	} else {
		return nil
	}

	if have == meta.AudnexusChapterCount {
		return nil
	}
	return []Violation{{
		Rule:     "audnexus_chapters",
		Severity: SeverityWarn,
		Message:  fmt.Sprintf("audnexus reports %d chapters, item has %d", meta.AudnexusChapterCount, have),
	}}
}
