package ruleset

import (
	"fmt"
	"strings"

	"github.com/znth-cx/zentag/core/metadata"
)

// CheckLossyContainer validates lossy slot contains M4B per RULES.md §6.
func CheckLossyContainer(meta *metadata.Metadata) []Violation {
	if meta == nil {
		return nil
	}

	if len(meta.Tracks) == 0 {
		return nil
	}

	var violations []Violation
	seen := make(map[string]bool)

	for _, track := range meta.Tracks {
		container := strings.ToLower(track.Container)
		codec := strings.ToLower(track.Codec)

		isLossy := codec == "aac" || codec == "mp3" || codec == "opus"

		if !isLossy {
			continue
		}

		if container != "m4b" {
			key := container + "|" + codec
			if seen[key] {
				continue
			}
			seen[key] = true
			violations = append(violations, Violation{
				Rule:     "lossy_container",
				Severity: SeverityUpgradable,
				Message:  fmt.Sprintf("container %q with codec %q should be M4B", track.Container, track.Codec),
			})
		}
	}

	return violations
}

// CheckMixedFormat validates every track shares one container+codec combo.
// A directory mixing formats (e.g. MP3 and FLAC) is structurally invalid per RULES.md §5 slots.
func CheckMixedFormat(meta *metadata.Metadata) []Violation {
	if meta == nil {
		return nil
	}

	if len(meta.Tracks) < 2 {
		return nil
	}

	type fmtCombo struct{ container, codec string }
	var distinct []fmtCombo
	seen := make(map[[2]string]bool)

	for _, track := range meta.Tracks {
		k := [2]string{strings.ToLower(track.Container), strings.ToLower(track.Codec)}
		if seen[k] {
			continue
		}
		seen[k] = true
		distinct = append(distinct, fmtCombo{track.Container, track.Codec})
	}

	if len(distinct) < 2 {
		return nil
	}

	parts := make([]string, len(distinct))
	for i, c := range distinct {
		parts[i] = fmt.Sprintf("container %q codec %q", c.container, c.codec)
	}

	return []Violation{{
		Rule:     "mixed_format",
		Severity: SeverityProhibited,
		Message:  "tracks have mixed formats: " + strings.Join(parts, ", "),
	}}
}
