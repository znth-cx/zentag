package EbookEngine

import (
	"strings"
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/internal/version"
)

func TestArgs(t *testing.T) {
	m := &metadata.Metadata{
		Author:    []string{"A One", "B Two"},
		Title:     "T",
		Year:      2006,
		ISBN:      "9780765311788",
		Series:    []metadata.SeriesEntry{{Name: "Mistborn", Part: "1"}},
		Publisher: []string{"Tor"},
		Language:  "eng",
		Genre:     []string{"Fantasy", "Epic"},
		ASIN:      "B002GYI9C4",
	}
	got := strings.Join(Args(m), "\x00")
	for _, want := range []string{
		"--authors\x00A One & B Two", "--title\x00T", "--date\x002006",
		"--isbn\x009780765311788", "--series\x00Mistborn", "--index\x001",
		"--publisher\x00Tor", "--language\x00eng", "--tags\x00Fantasy,Epic",
		"--identifier\x00asin:B002GYI9C4",
		"--identifier\x00zentag:" + version.Version,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Args missing %q in %q", want, got)
		}
	}
}
