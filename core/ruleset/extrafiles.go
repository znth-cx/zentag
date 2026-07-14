package ruleset

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/znth-cx/zentag/core/metadata"
)

var ignoredPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^metadata\.json$`),
	regexp.MustCompile(`\.txt$`),
	regexp.MustCompile(`\.nfo$`),
	regexp.MustCompile(`\.log$`),
	regexp.MustCompile(`\.m3u8?$`),
}

func CheckExtraFiles(ctx context.Context, meta *metadata.Metadata) []Violation {
	if meta == nil || meta.OriginalPath == "" {
		return nil
	}

	var violations []Violation

	entries, err := os.ReadDir(meta.OriginalPath)
	if err != nil {
		slog.WarnContext(ctx, "ruleset: cannot read directory", "path", meta.OriginalPath, "error", err)
		return nil
	}

	var allowedExtensions map[string]bool
	container := ""
	if len(meta.Tracks) > 0 {
		container = meta.Tracks[0].Container
	}

	switch container {
	case "M4B":
		allowedExtensions = map[string]bool{
			".m4b": true,
			".jpg": true,
		}
	case "MP3":
		allowedExtensions = map[string]bool{
			".mp3": true,
			".jpg": true,
		}
	case "FLAC":
		allowedExtensions = map[string]bool{
			".flac": true,
			".jpg":  true,
		}
	default:
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if strings.HasPrefix(name, ".") {
			continue
		}

		shouldIgnore := false
		for _, pattern := range ignoredPatterns {
			if pattern.MatchString(name) {
				shouldIgnore = true
				break
			}
		}
		if shouldIgnore {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		baseName := strings.ToLower(strings.TrimSuffix(name, ext))

		if container == "MP3" || container == "FLAC" {
			if baseName == "cover" && ext == ".jpg" {
				continue
			}
		}

		if allowedExtensions[ext] {
			continue
		}

		violations = append(violations, Violation{
			Rule:     "extra_files",
			Severity: SeverityTrumpable,
			Message:  "unexpected file found: " + name,
		})
	}

	return violations
}
