package mp3tag

import (
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestBuildID3Tags_FullFields(t *testing.T) {
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
		Language: "eng",
		ISBN:     "9780765326355",
		ASIN:     "B0041OW6EG",
		Tracks: []metadata.Track{
			{Path: "part1.mp3"},
		},
	}
	track := metadata.Track{
		Path:       "part1.mp3",
		PartNumber: 1,
	}

	got := buildID3Tags(m, track)

	if got["TITLE"][0] != "The Way of Kings" {
		t.Errorf("TITLE = %v, want The Way of Kings", got["TITLE"])
	}

	if got["TXXX:NARRATOR"][0] != "Michael Kramer;Kate Reading" {
		t.Errorf("TXXX:NARRATOR = %v, want Michael Kramer;Kate Reading", got["TXXX:NARRATOR"])
	}

	if got["TXXX:SERIES"][0] != "Stormlight Archive" {
		t.Errorf("TXXX:SERIES = %v, want Stormlight Archive", got["TXXX:SERIES"])
	}

	if _, exists := got["DISCNUMBER"]; exists {
		t.Error("DISCNUMBER should not exist")
	}

	if got["TRACKNUMBER"][0] != "1/1" {
		t.Errorf("TRACKNUMBER = %v, want 1/1", got["TRACKNUMBER"])
	}

	if got["TITLESORT"][0] != "Stormlight Archive 1" {
		t.Errorf("TITLESORT = %v, want Stormlight Archive 1", got["TITLESORT"])
	}

	if got["TXXX:ISBN"][0] != "9780765326355" {
		t.Errorf("TXXX:ISBN = %v, want 9780765326355", got["TXXX:ISBN"])
	}

	if got["TXXX:ASIN"][0] != "B0041OW6EG" {
		t.Errorf("TXXX:ASIN = %v, want B0041OW6EG", got["TXXX:ASIN"])
	}

	if got["LANGUAGE"][0] != "eng" {
		t.Errorf("LANGUAGE = %v, want eng", got["LANGUAGE"])
	}

	if got["LABEL"][0] != "Macmillan" {
		t.Errorf("LABEL = %v, want Macmillan", got["LABEL"])
	}

	if got["COMMENT"][0] != "A war-torn world." {
		t.Errorf("COMMENT = %v, want A war-torn world.", got["COMMENT"])
	}
}

func TestBuildID3Tags_OmitsEmptyFields(t *testing.T) {
	m := &metadata.Metadata{
		Title: "Some Book",
		Year:  0,
		ASIN:  "",
	}
	track := metadata.Track{Path: "test.mp3"}

	got := buildID3Tags(m, track)

	if _, exists := got["DATE"]; exists {
		t.Error("DATE should not exist when Year=0")
	}

	if _, exists := got["TXXX:ASIN"]; exists {
		t.Error("TXXX:ASIN should not exist when empty")
	}

	if _, exists := got["TXXX:ISBN"]; exists {
		t.Error("TXXX:ISBN should not exist when empty")
	}

	if _, exists := got["TXXX:NARRATOR"]; exists {
		t.Error("TXXX:NARRATOR should not exist when empty")
	}

	if _, exists := got["TXXX:SERIES"]; exists {
		t.Error("TXXX:SERIES should not exist when empty")
	}

	if _, exists := got["TXXX:SERIES-PART"]; exists {
		t.Error("TXXX:SERIES-PART should not exist when empty")
	}

	if _, exists := got["COMMENT"]; exists {
		t.Error("COMMENT should not exist when empty")
	}

	if _, exists := got["TITLESORT"]; exists {
		t.Error("TITLESORT should not exist when Subtitle is empty")
	}
}

func TestBuildID3Tags_MultiValueFields(t *testing.T) {
	m := &metadata.Metadata{
		Author:    []string{"Author One", "Author Two"},
		Narrator:  []string{"Narrator One", "Narrator Two"},
		Genre:     []string{"Fantasy", "Adventure"},
		Publisher: []string{"Publisher One", "Publisher Two"},
	}
	track := metadata.Track{Path: "test.mp3"}

	got := buildID3Tags(m, track)

	if len(got["ARTIST"]) != 2 {
		t.Errorf("ARTIST length = %d, want 2", len(got["ARTIST"]))
	}

	if len(got["COMPOSER"]) != 2 {
		t.Errorf("COMPOSER length = %d, want 2", len(got["COMPOSER"]))
	}

	if len(got["GENRE"]) != 2 {
		t.Errorf("GENRE length = %d, want 2", len(got["GENRE"]))
	}

	if len(got["LABEL"]) != 2 {
		t.Errorf("LABEL length = %d, want 2", len(got["LABEL"]))
	}
}

func TestBuildID3Tags_MultipleSeriesEntries(t *testing.T) {
	m := &metadata.Metadata{
		Title: "Test Book",
		Series: []metadata.SeriesEntry{
			{Name: "Series One", Part: "1"},
			{Name: "Series Two", Part: "3"},
		},
	}
	track := metadata.Track{Path: "test.mp3"}

	got := buildID3Tags(m, track)

	if len(got["TXXX:SERIES"]) != 2 {
		t.Errorf("TXXX:SERIES length = %d, want 2", len(got["TXXX:SERIES"]))
	}

	if len(got["TXXX:SERIES-PART"]) != 2 {
		t.Errorf("TXXX:SERIES-PART length = %d, want 2", len(got["TXXX:SERIES-PART"]))
	}
}

func TestBuildID3Tags_TrackNumberFormatting(t *testing.T) {
	m := &metadata.Metadata{
		Title:  "Test Book",
		Tracks: []metadata.Track{{}, {}, {}},
	}
	track := metadata.Track{Path: "test.mp3", PartNumber: 2}

	got := buildID3Tags(m, track)

	if got["TRACKNUMBER"][0] != "2/3" {
		t.Errorf("TRACKNUMBER = %v, want 2/3", got["TRACKNUMBER"])
	}
}

func TestBuildID3Tags_NoTrackNumberWhenZero(t *testing.T) {
	m := &metadata.Metadata{
		Title: "Test Book",
	}
	track := metadata.Track{Path: "test.mp3", PartNumber: 0}

	got := buildID3Tags(m, track)

	if _, exists := got["TRACKNUMBER"]; exists {
		t.Error("TRACKNUMBER should not exist when PartNumber=0")
	}
}

func TestBuildID3Tags_YearBounds(t *testing.T) {
	m := &metadata.Metadata{
		Title: "Test Book",
		Year:  10000,
	}
	track := metadata.Track{Path: "test.mp3"}

	got := buildID3Tags(m, track)

	if _, exists := got["DATE"]; exists {
		t.Error("DATE should not exist when Year > MaxYear")
	}
}

func TestBuildID3Tags_EscapedFields(t *testing.T) {
	m := &metadata.Metadata{
		Title:    "Test; Book",
		Author:   []string{"Author\\With;Backslashes"},
		Narrator: []string{"Narrator;With;Semicolons"},
	}
	track := metadata.Track{Path: "test.mp3"}

	got := buildID3Tags(m, track)

	expectedNarrator := "Narrator\\;With\\;Semicolons"
	if got["TXXX:NARRATOR"][0] != expectedNarrator {
		t.Errorf("TXXX:NARRATOR = %v, want %v", got["TXXX:NARRATOR"][0], expectedNarrator)
	}
}
