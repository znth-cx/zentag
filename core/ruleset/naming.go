package ruleset

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/naming"
)

// CheckNaming checks RULES.md §3: title must be APA title case, dir/track names must match naming.DirectoryName/naming.TrackName.
// Returns nil if Author/Title/Narrator/Tracks empty: CheckRequiredTags/CheckPrimaryKeys already flags that gap.
func CheckNaming(ctx context.Context, meta *metadata.Metadata) []Violation {
	if len(meta.Author) == 0 || meta.Author[0] == "" || meta.Title == "" ||
		len(meta.Narrator) == 0 || meta.Narrator[0] == "" || len(meta.Tracks) == 0 {
		return nil
	}

	slog.DebugContext(ctx, "ruleset: checking naming", "path", meta.OriginalPath)

	var violations []Violation

	expectedTitle := naming.TitleCase(meta.Title, meta.Language)
	if meta.Title != expectedTitle {
		violations = append(violations, Violation{
			Rule:     "naming",
			Severity: SeverityTrumpable,
			Message:  fmt.Sprintf("title not in APA title case, expected %q", expectedTitle),
		})
	}

	if len(meta.Tracks) == 1 && meta.OriginalPath == meta.Tracks[0].Path {
		violations = append(violations, Violation{
			Rule:     "naming",
			Severity: SeverityTrumpable,
			Message:  "single file must be inside a directory",
		})
		return violations
	}

	expectedDir, err := naming.DirectoryName(ctx, meta)
	if err != nil {
		slog.WarnContext(ctx, "ruleset: could not build expected directory name", "path", meta.OriginalPath, "error", err)
		return violations
	}

	actualDir := filepath.Base(meta.OriginalPath)
	if actualDir != expectedDir {
		violations = append(violations, Violation{
			Rule:     "naming",
			Severity: SeverityTrumpable,
			Message:  fmt.Sprintf("directory name does not match expected %q", expectedDir),
		})
		return violations
	}

	violations = append(violations, validateSourceToken(ctx, meta)...)

	baseName := func(p string) string { return strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)) }

	if len(meta.Tracks) == 1 {
		actualTrack := baseName(meta.Tracks[0].Path)
		if actualTrack != expectedDir {
			violations = append(violations, Violation{
				Rule:     "naming",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("track %s name does not match expected %q", meta.Tracks[0].Path, expectedDir),
			})
		}
		return violations
	}

	violations = append(violations, validatePartNumberFormat(meta)...)

	for i, tr := range meta.Tracks {
		expectedTrack, err := naming.TrackName(ctx, meta, i)
		if err != nil {
			slog.WarnContext(ctx, "ruleset: could not build expected track name", "path", tr.Path, "error", err)
			continue
		}
		actualTrack := baseName(tr.Path)
		if actualTrack != expectedTrack {
			violations = append(violations, Violation{
				Rule:     "naming",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("track %s name does not match expected %q", tr.Path, expectedTrack),
			})
		}
	}

	return violations
}

// validateSourceToken validates the [Source] token in naming convention per RULES.md §3.
func validateSourceToken(ctx context.Context, meta *metadata.Metadata) []Violation {
	if meta.Source == "" {
		return nil
	}

	expectedSource := fmt.Sprintf("[%s]", meta.Source)
	actualDir := filepath.Base(meta.OriginalPath)

	if !strings.Contains(actualDir, expectedSource) {
		return []Violation{{
			Rule:     "naming",
			Severity: SeverityTrumpable,
			Message:  fmt.Sprintf("directory name missing source token %q", expectedSource),
		}}
	}

	return nil
}

// validatePartNumberFormat validates PartNumber padding per RULES.md §3.
func validatePartNumberFormat(meta *metadata.Metadata) []Violation {
	if len(meta.Tracks) <= 1 {
		return nil
	}

	maxPart := 0
	for _, tr := range meta.Tracks {
		if tr.PartNumber > maxPart {
			maxPart = tr.PartNumber
		}
	}

	expectedWidth := len(strconv.Itoa(maxPart))

	var violations []Violation
	for _, tr := range meta.Tracks {
		baseName := filepath.Base(tr.Path)
		ext := filepath.Ext(baseName)
		nameWithoutExt := strings.TrimSuffix(baseName, ext)

		re := regexp.MustCompile(`^(\d+)\.`)
		matches := re.FindStringSubmatch(nameWithoutExt)
		if len(matches) < 2 {
			continue
		}

		if len(matches[1]) != expectedWidth {
			violations = append(violations, Violation{
				Rule:     "naming",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("track %s has part number width %d, expected %d based on max part %d", tr.Path, len(matches[1]), expectedWidth, maxPart),
			})
		}
	}

	return violations
}
