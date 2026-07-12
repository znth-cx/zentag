package mp4tag

import (
	"testing"

	mp4 "github.com/Sorrow446/go-mp4tag"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestBuildTags_AllFieldsSetNoCover(t *testing.T) {
	meta := &metadata.Metadata{
		Author:      []string{"Robert Jordan"},
		Title:       "The Eye of the World",
		Subtitle:    "Book One",
		Publisher:   []string{"Tor"},
		Year:        1990,
		Narrator:    []string{"Michael Kramer"},
		Description: "A fantasy epic",
		Genre:       []string{"Fantasy"},
		Series:      []metadata.SeriesEntry{{Name: "Wheel of Time", Part: "1"}},
		Language:    "eng",
		ISBN:        "9780812511819",
		ASIN:        "B000TEST00",
	}

	tags := buildTags(meta)

	if tags.Title != "The Eye of the World" {
		t.Errorf("Title = %q", tags.Title)
	}
	if tags.Artist != "Robert Jordan" {
		t.Errorf("Artist = %q, want mirrored author", tags.Artist)
	}
	if tags.Year != 1990 {
		t.Errorf("Year = %d", tags.Year)
	}
	if tags.Composer != "Michael Kramer" {
		t.Errorf("Composer = %q, want mirrored narrator", tags.Composer)
	}
	if tags.Description != "A fantasy epic" {
		t.Errorf("Description = %q", tags.Description)
	}
	if tags.CustomGenre != "Fantasy" {
		t.Errorf("CustomGenre = %q", tags.CustomGenre)
	}

	want := map[string]string{
		"AUTHOR":      "Robert Jordan",
		"NARRATOR":    "Michael Kramer",
		"PUBLISHER":   "Tor",
		"SUBTITLE":    "Book One",
		"SERIES":      "Wheel of Time",
		"SERIES-PART": "1",
		"LANGUAGE":    "eng",
		"ISBN":        "9780812511819",
		"ASIN":        "B000TEST00",
	}
	for k, v := range want {
		if got := tags.Custom[k]; got != v {
			t.Errorf("Custom[%q] = %q, want %q", k, got, v)
		}
	}
	if _, ok := tags.Custom["ZENTAG"]; !ok {
		t.Error("Custom[\"ZENTAG\"] missing")
	}
	if len(tags.Pictures) != 0 {
		t.Errorf("Pictures = %v, want none set", tags.Pictures)
	}
}

func TestBuildTags_YearZeroOmitted(t *testing.T) {
	tags := buildTags(&metadata.Metadata{Title: "Some Book", Year: 0})
	if tags.Year != 0 {
		t.Errorf("Year = %d, want 0 (zero value, not written)", tags.Year)
	}
}

func TestBuildTags_YearOutOfRangeOmitted(t *testing.T) {
	for _, y := range []int{-1, metadata.MaxYear + 1, 99999999999} {
		tags := buildTags(&metadata.Metadata{Title: "Some Book", Year: y})
		if tags.Year != 0 || tags.Date != "" {
			t.Errorf("Year %d: got Year=%d Date=%q, want omitted", y, tags.Year, tags.Date)
		}
	}
}

func TestBuildTags_EmptyASINOmittedFromCustom(t *testing.T) {
	tags := buildTags(&metadata.Metadata{Title: "Some Book", ASIN: ""})
	if _, ok := tags.Custom["ASIN"]; ok {
		t.Error("Custom[\"ASIN\"] set despite empty ASIN")
	}
}

func TestBuildTags_CoverPresentSetsPicture(t *testing.T) {
	tags := buildTags(&metadata.Metadata{
		Title:      "Some Book",
		CoverImage: []byte{1, 2, 3},
		CoverMIME:  "image/png",
	})
	if len(tags.Pictures) != 1 {
		t.Fatalf("Pictures = %v, want exactly 1", tags.Pictures)
	}
	pic := tags.Pictures[0]
	if string(pic.Data) != "\x01\x02\x03" {
		t.Errorf("Pictures[0].Data = %v, want cover bytes", pic.Data)
	}
	if pic.Format != mp4.ImageTypePNG {
		t.Errorf("Pictures[0].Format = %v, want ImageTypePNG", pic.Format)
	}
}

func TestBuildTags_CoverAbsentNoPictures(t *testing.T) {
	tags := buildTags(&metadata.Metadata{Title: "Some Book"})
	if len(tags.Pictures) != 0 {
		t.Errorf("Pictures = %v, want none", tags.Pictures)
	}
}

func TestCoverFormat(t *testing.T) {
	cases := map[string]mp4.ImageType{
		"image/jpeg": mp4.ImageTypeJPEG,
		"image/png":  mp4.ImageTypePNG,
		"image/gif":  mp4.ImageTypeAuto,
		"":           mp4.ImageTypeAuto,
	}
	for mime, want := range cases {
		if got := coverFormat(mime); got != want {
			t.Errorf("coverFormat(%q) = %v, want %v", mime, got, want)
		}
	}
}
