package ruleset

import (
	"path/filepath"
	"regexp"

	"github.com/znth-cx/zentag/core/metadata"
)

var discPattern = regexp.MustCompile(`(?i)^disc\s*\d+$`)

func CheckM4BSingleFile(meta *metadata.Metadata) []Violation {
	if meta == nil || len(meta.Tracks) == 0 {
		return nil
	}

	container := meta.Tracks[0].Container
	if container != "M4B" {
		return nil
	}

	if len(meta.Tracks) == 1 {
		return nil
	}

	allDiscNamed := true
	for _, tr := range meta.Tracks {
		baseName := filepath.Base(tr.Path)
		nameWithoutExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]

		if !discPattern.MatchString(nameWithoutExt) {
			allDiscNamed = false
			break
		}
	}

	if allDiscNamed {
		return nil
	}

	return []Violation{
		{
			Rule:     "m4b_split_file",
			Severity: SeverityTrumpable,
			Message:  "M4B uploads should be single files or disc releases with proper disc naming (e.g., \"Disc 1.m4b\", \"Disc 2.m4b\")",
		},
	}
}
