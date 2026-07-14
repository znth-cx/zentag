// Package ruleset validates metadata.Metadata against RULES.md, returns severity-tagged violations.
package ruleset

import (
	"context"
	"log/slog"

	"github.com/znth-cx/zentag/core/mediainfo"
	"github.com/znth-cx/zentag/core/metadata"
)

// Severity is RULES.md §6's classification for a violation.
type Severity string

const (
	SeverityTrumpable  Severity = "trumpable"  // fixable via re-tag/re-upload
	SeverityUpgradable Severity = "upgradable" // needs better source, not just tag fix
	SeverityProhibited Severity = "prohibited" // not fixable, not allowed
	// SeverityWarn: advisory finding, not a RULES.md §6 category.
	SeverityWarn Severity = "warn"
)

// Violation is one RULES.md rule failure found in a Metadata object.
type Violation struct {
	Rule     string // short machine-friendly identifier, e.g. "primary_keys", "banned_content"
	Severity Severity
	Message  string
}

// Validate runs meta through every rule check, returns all violations (nil if none).
func Validate(ctx context.Context, meta *metadata.Metadata) []Violation {
	slog.DebugContext(ctx, "ruleset: validating metadata", "path", meta.OriginalPath)

	var violations []Violation
	violations = append(violations, CheckPrimaryKeys(meta)...)
	violations = append(violations, CheckRequiredTags(meta)...)
	violations = append(violations, CheckLanguage(meta)...)
	violations = append(violations, CheckCover(ctx, meta)...)
	violations = append(violations, CheckCoverPlacement(ctx, meta)...)
	violations = append(violations, CheckChapters(meta)...)
	violations = append(violations, CheckAudnexusChapters(meta)...)
	violations = append(violations, CheckBannedContent(meta)...)
	violations = append(violations, CheckNaming(ctx, meta)...)
	violations = append(violations, CheckSource(meta)...)
	violations = append(violations, CheckFormatSpecificTags(meta)...)
	violations = append(violations, CheckM4BSingleFile(meta)...)
	violations = append(violations, CheckExtraFiles(ctx, meta)...)
	violations = append(violations, CheckBitrate(meta)...)
	violations = append(violations, CheckLossyContainer(meta)...)
	violations = append(violations, CheckTagSeparators(meta)...)
	violations = append(violations, CheckChapterFormat(ctx, meta)...)

	slog.DebugContext(ctx, "ruleset: validation complete", "path", meta.OriginalPath, "violations", len(violations))
	return violations
}

// ValidateWithMediaInfo runs meta through every rule check, including FLAC MD5 validation, returns all violations (nil if none).
func ValidateWithMediaInfo(ctx context.Context, meta *metadata.Metadata, mi *mediainfo.Wrapper) []Violation {
	slog.DebugContext(ctx, "ruleset: validating metadata with mediainfo", "path", meta.OriginalPath)

	var violations []Violation
	violations = append(violations, CheckPrimaryKeys(meta)...)
	violations = append(violations, CheckRequiredTags(meta)...)
	violations = append(violations, CheckLanguage(meta)...)
	violations = append(violations, CheckCover(ctx, meta)...)
	violations = append(violations, CheckCoverPlacement(ctx, meta)...)
	violations = append(violations, CheckChapters(meta)...)
	violations = append(violations, CheckAudnexusChapters(meta)...)
	violations = append(violations, CheckBannedContent(meta)...)
	violations = append(violations, CheckNaming(ctx, meta)...)
	violations = append(violations, CheckSource(meta)...)
	violations = append(violations, CheckFormatSpecificTags(meta)...)
	violations = append(violations, CheckM4BSingleFile(meta)...)
	violations = append(violations, CheckExtraFiles(ctx, meta)...)
	violations = append(violations, CheckBitrate(meta)...)
	violations = append(violations, CheckLossyContainer(meta)...)
	violations = append(violations, CheckFLACMD5(ctx, meta, mi)...)
	violations = append(violations, CheckTagSeparators(meta)...)
	violations = append(violations, CheckChapterFormat(ctx, meta)...)

	slog.DebugContext(ctx, "ruleset: validation complete", "path", meta.OriginalPath, "violations", len(violations))
	return violations
}
