package ruleset

import (
	"context"
	"os"
	"path/filepath"

	"github.com/znth-cx/zentag/core/metadata"
)

const maxCoverSize = 3 * 1024 * 1024

// itemDir returns path if it is a directory, else its parent (single-file M4B).
func itemDir(path string) string {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return filepath.Dir(path)
	}
	return path
}

func CheckCoverPlacement(ctx context.Context, meta *metadata.Metadata) []Violation {
	if meta == nil || len(meta.Tracks) == 0 {
		return nil
	}

	var violations []Violation

	hasEmbeddedCover := len(meta.CoverImage) > 0
	container := meta.Tracks[0].Container

	if hasEmbeddedCover {
		if len(meta.CoverImage) > maxCoverSize {
			violations = append(violations, Violation{
				Rule:     "cover_placement",
				Severity: SeverityTrumpable,
				Message:  "cover image exceeds 3MB size limit",
			})
		}
	}

	switch container {
	case "M4B":
		if !hasEmbeddedCover {
			violations = append(violations, Violation{
				Rule:     "cover_placement",
				Severity: SeverityTrumpable,
				Message:  "M4B files must have embedded cover image",
			})
		}

		looseCover := filepath.Join(itemDir(meta.OriginalPath), "cover.jpg")
		if _, err := os.Stat(looseCover); err == nil {
			violations = append(violations, Violation{
				Rule:     "cover_placement",
				Severity: SeverityTrumpable,
				Message:  "M4B files should not have loose cover.jpg file (cover must be embedded)",
			})
		}

	case "MP3", "FLAC":
		coverPath := filepath.Join(meta.OriginalPath, "cover.jpg")
		fileInfo, err := os.Stat(coverPath)
		if os.IsNotExist(err) {
			violations = append(violations, Violation{
				Rule:     "cover_placement",
				Severity: SeverityTrumpable,
				Message:  "cover.jpg must be present in root directory",
			})
		} else if err == nil {
			if fileInfo.Size() > int64(maxCoverSize) {
				violations = append(violations, Violation{
					Rule:     "cover_placement",
					Severity: SeverityTrumpable,
					Message:  "cover.jpg exceeds 3MB size limit",
				})
			}
		}
	}

	return violations
}
