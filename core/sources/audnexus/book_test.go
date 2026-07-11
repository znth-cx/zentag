package audnexus

import (
	"context"
	"reflect"
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestBookToMetadata_FullMapping(t *testing.T) {
	b := book{
		ASIN:          "B08G9PRS1K",
		Authors:       []person{{Name: "Andy Weir"}},
		Narrators:     []person{{Name: "Ray Porter"}},
		PublisherName: "Audible Studios",
		Genres: []genre{
			{Name: "Science Fiction & Fantasy", Type: "genre"},
			{Name: "Space Opera", Type: "tag"},
		},
		ISBN:            "9781603935470",
		Language:        "english",
		FormatType:      "unabridged",
		ReleaseDate:     "2021-05-04T00:00:00.000Z",
		SeriesPrimary:   &series{Name: "Standalone", Position: "1"},
		SeriesSecondary: &series{Name: "Omnibus", Position: "1-3"},
		Subtitle:        "A Novel",
		Summary:         "<p>Full <b>synopsis</b>.</p>",
		Title:           "Project Hail Mary",
	}

	got := b.toMetadata(context.Background())

	want := &metadata.Metadata{
		MetadataOrigin: metadata.OriginAudnexus,
		Author:         []string{"Andy Weir"},
		Title:          "Project Hail Mary",
		Subtitle:       "A Novel",
		Publisher:      []string{"Audible Studios"},
		Year:           2021,
		Narrator:       []string{"Ray Porter"},
		Description:    "<p>Full <b>synopsis</b>.</p>",
		Genre:          []string{"Science Fiction & Fantasy", "Space Opera"},
		Series: []metadata.SeriesEntry{
			{Name: "Standalone", Part: "1"},
			{Name: "Omnibus", Part: "1-3"},
		},
		Language: "eng",
		ISBN:     "9781603935470",
		ASIN:     "B08G9PRS1K",
		Edition:  "",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("toMetadata() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestBookToMetadata_AbridgedSetsEdition(t *testing.T) {
	b := book{FormatType: "abridged"}
	got := b.toMetadata(context.Background())
	if got.Edition != "Abridged" {
		t.Errorf("Edition = %q, want %q", got.Edition, "Abridged")
	}
}

func TestBookToMetadata_SinglePrimarySeries(t *testing.T) {
	b := book{SeriesPrimary: &series{Name: "Standalone", Position: "1"}}
	got := b.toMetadata(context.Background())
	want := []metadata.SeriesEntry{{Name: "Standalone", Part: "1"}}
	if !reflect.DeepEqual(got.Series, want) {
		t.Errorf("Series = %+v, want %+v", got.Series, want)
	}
}

func TestBookToMetadata_UnknownLanguageLeavesEmpty(t *testing.T) {
	b := book{Language: "not-a-real-language"}
	got := b.toMetadata(context.Background())
	if got.Language != "" {
		t.Errorf("Language = %q, want empty", got.Language)
	}
}

func TestBookToMetadata_UnparseableReleaseDateLeavesYearZero(t *testing.T) {
	b := book{ReleaseDate: "not-a-date"}
	got := b.toMetadata(context.Background())
	if got.Year != 0 {
		t.Errorf("Year = %d, want 0", got.Year)
	}
}
