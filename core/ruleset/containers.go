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
