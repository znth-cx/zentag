package ffmpeg

import (
	"reflect"
	"testing"

	"codeberg.org/Ether/zentag/core/metadata"
	"codeberg.org/Ether/zentag/internal/version"
)

func TestMetadataArgs_FullFields(t *testing.T) {
	m := &metadata.Metadata{
		Author:      []string{"Brandon Sanderson"},
		Title:       "The Way of Kings",
		Subtitle:    "Stormlight Archive 1",
		Publisher:   []string{"Macmillan"},
		Year:        2010,
		Narrator:    []string{"Michael Kramer", "Kate Reading"},
		Description: "A war-torn world.",
		Genre:       []string{"Fantasy"},
		Series: []metadata.SeriesEntry{
			{Name: "Stormlight Archive", Part: "1"},
		},
		Language: "en",
		ISBN:     "9780765326355",
		ASIN:     "B0041OW6EG",
	}

	got := metadataArgs(m)
	want := []string{
		"-metadata", "author=Brandon Sanderson",
		"-metadata", "artist=Brandon Sanderson",
		"-metadata", "title=The Way of Kings",
		"-metadata", "subtitle=Stormlight Archive 1",
		"-metadata", "publisher=Macmillan",
		"-metadata", "year=2010",
		"-metadata", "narrator=Michael Kramer;Kate Reading",
		"-metadata", "composer=Michael Kramer;Kate Reading",
		"-metadata", "description=A war-torn world.",
		"-metadata", "genre=Fantasy",
		"-metadata", "series=Stormlight Archive",
		"-metadata", "series-part=1",
		"-metadata", "language=en",
		"-metadata", "isbn=9780765326355",
		"-metadata", "asin=B0041OW6EG",
		"-metadata", "zentag=" + version.Version,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("metadataArgs() =\n%q\nwant\n%q", got, want)
	}
}

func TestMetadataArgs_AlwaysIncludesZentagVersion(t *testing.T) {
	got := metadataArgs(&metadata.Metadata{})
	want := "zentag=" + version.Version
	for i := 0; i < len(got); i += 2 {
		if got[i] == "-metadata" && got[i+1] == want {
			return
		}
	}
	t.Errorf("metadataArgs() missing %q, got %q", want, got)
}

func TestMetadataArgs_OmitsYearZeroAndEmptyASIN(t *testing.T) {
	m := &metadata.Metadata{
		Title: "Some Book",
		Year:  0,
		ASIN:  "",
	}

	got := metadataArgs(m)

	for i := 0; i < len(got); i += 2 {
		if got[i] == "-metadata" && (got[i+1] == "year=0" || got[i+1] == "asin=") {
			t.Errorf("metadataArgs() included omitted field: %q", got[i+1])
		}
	}
}
