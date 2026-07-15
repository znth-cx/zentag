package ruleset

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/znth-cx/zentag/core/mediainfo"
	"github.com/znth-cx/zentag/core/metadata"
)

var (
	md5HashPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
)

// CheckFLACMD5 validates FLAC files have proper MD5 hash per RULES.md §6.
func CheckFLACMD5(ctx context.Context, meta *metadata.Metadata, mi *mediainfo.Wrapper) []Violation {
	if meta == nil {
		return nil
	}

	if len(meta.Tracks) == 0 {
		return nil
	}

	var violations []Violation

	for i, track := range meta.Tracks {
		if !isFLAC(track) {
			continue
		}

		md5Hash, err := extractFLACMD5(ctx, track.Path, mi)
		if err != nil {
			slog.WarnContext(ctx, "failed to extract FLAC MD5", "path", track.Path, "error", err)
			continue
		}

		if md5Hash == "" {
			violations = append(violations, Violation{
				Rule:     "flac_md5",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("track %d (FLAC): missing MD5 hash", i+1),
			})
			continue
		}

		if !isValidMD5Hash(md5Hash) {
			violations = append(violations, Violation{
				Rule:     "flac_md5",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("track %d (FLAC): invalid MD5 hash format %q", i+1, md5Hash),
			})
		}
	}

	return violations
}

func isFLAC(track metadata.Track) bool {
	return strings.EqualFold(track.Container, "FLAC") || strings.EqualFold(track.Codec, "FLAC")
}

func extractFLACMD5(ctx context.Context, path string, mi *mediainfo.Wrapper) (string, error) {
	if mi == nil {
		return "", fmt.Errorf("mediainfo wrapper is nil")
	}

	dump, err := mi.Dump(ctx, path)
	if err != nil {
		return "", fmt.Errorf("mediainfo dump failed: %w", err)
	}

	md5Line, found := findMD5InDump(dump)
	if !found {
		return "", nil
	}

	md5Hash := extractHashFromLine(md5Line)
	return md5Hash, nil
}

func findMD5InDump(dump string) (string, bool) {
	lines := strings.Split(dump, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "md5") && strings.Contains(line, ":") {
			return line, true
		}
	}
	return "", false
}

func extractHashFromLine(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return ""
	}

	hash := strings.TrimSpace(parts[1])

	if md5HashPattern.MatchString(hash) {
		return strings.ToLower(hash)
	}

	words := strings.Fields(hash)
	for _, word := range words {
		if md5HashPattern.MatchString(word) {
			return strings.ToLower(word)
		}
	}

	return ""
}

func isValidMD5Hash(hash string) bool {
	return md5HashPattern.MatchString(hash)
}
