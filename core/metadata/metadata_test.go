package metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetadataOriginConstants(t *testing.T) {
	assert.EqualValues(t, "user_args", OriginUserArgs)
	assert.EqualValues(t, "audnexus", OriginAudnexus)
	assert.EqualValues(t, "file_metadata", OriginFileMetadata)
}

func TestReleaseSourceConstants(t *testing.T) {
	assert.EqualValues(t, "WEB", ReleaseSourceWEB)
	assert.EqualValues(t, "CD", ReleaseSourceCD)
	assert.EqualValues(t, "VINYL", ReleaseSourceVinyl)
}

func TestMetadataZeroValue(t *testing.T) {
	var m Metadata

	assert.Equal(t, 0, m.Year, "zero Year means unset")
	assert.Nil(t, m.Author)
	assert.Nil(t, m.Tracks)
	assert.Nil(t, m.Series)
	assert.Empty(t, m.MetadataOrigin)
	assert.Empty(t, m.Source)
}

func TestMetadataFullConstruction(t *testing.T) {
	m := Metadata{
		OriginalPath:   "/books/example",
		MetadataOrigin: OriginFileMetadata,
		Author:         []string{"Primary Author", "Second Author"},
		Title:          "Example Title",
		Subtitle:       "Example Subtitle",
		Publisher:      []string{"Example Publisher"},
		Year:           2019,
		Narrator:       []string{"Primary Narrator"},
		Description:    "Synopsis text.",
		Genre:          []string{"Fantasy"},
		Series: []SeriesEntry{
			{Name: "Example Series", Part: "1"},
		},
		Language:   "en",
		ISBN:       "9780000000000",
		ASIN:       "B000000000",
		CoverImage: []byte{0xFF, 0xD8},
		CoverMIME:  "image/jpeg",
		Edition:    "Abridged",
		Source:     ReleaseSourceWEB,
		Tracks: []Track{
			{
				Path:       "/books/example/file.m4b",
				PartNumber: 0,
				Container:  "M4B",
				Codec:      "AAC",
				Bitrate:    128,
				Chapters: []Chapter{
					{Title: "Chapter 1", Start: 0, End: 10 * time.Minute},
				},
			},
		},
	}

	assert.Equal(t, "Primary Author", m.Author[0])
	assert.Equal(t, "Primary Narrator", m.Narrator[0])
	assert.Equal(t, 2019, m.Year)
	assert.Len(t, m.Tracks, 1)
	assert.Equal(t, "Chapter 1", m.Tracks[0].Chapters[0].Title)
	assert.Equal(t, 10*time.Minute, m.Tracks[0].Chapters[0].End)
}
