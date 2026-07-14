package ruleset

import (
	"fmt"

	"github.com/znth-cx/zentag/core/metadata"
)

const (
	minBitrateKBPS = 60
)

// CheckBitrate validates bitrate meets minimum requirements per RULES.md §6.
func CheckBitrate(meta *metadata.Metadata) []Violation {
	if meta == nil {
		return nil
	}

	var violations []Violation

	for i, track := range meta.Tracks {
		if track.Bitrate > 0 && track.Bitrate < minBitrateKBPS {
			violations = append(violations, Violation{
				Rule:     "bitrate",
				Severity: SeverityUpgradable,
				Message:  fmt.Sprintf("track %d has bitrate %d kbps (minimum %d kbps)", i+1, track.Bitrate, minBitrateKBPS),
			})
		}
	}

	return violations
}
