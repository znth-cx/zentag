package naming

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/znth-cx/zentag/core/metadata"
)

func baseMeta() *metadata.Metadata {
	return &metadata.Metadata{
		OriginalPath: "/books/way-of-kings",
		Author:       []string{"Brandon Sanderson"},
		Title:        "The Way of Kings",
		Year:         2010,
		Narrator:     []string{"Michael Kramer"},
		Language:     "en",
		Source:       metadata.ReleaseSourceWEB,
		Tracks: []metadata.Track{
			{Path: "/books/way-of-kings/book.m4b", Container: "M4B", Codec: "AAC", Bitrate: 64},
		},
	}
}

func TestDirectoryName_NoEdition(t *testing.T) {
	got, err := DirectoryName(context.Background(), baseMeta())
	assert.NoError(t, err)
	assert.Equal(t, "Brandon Sanderson - The Way of Kings (2010) ENG {Michael Kramer} [WEB] M4B AAC 64kbps", got)
}

func TestDirectoryName_WithEdition(t *testing.T) {
	meta := baseMeta()
	meta.Edition = "Abridged"
	got, err := DirectoryName(context.Background(), meta)
	assert.NoError(t, err)
	assert.Equal(t, "Brandon Sanderson - The Way of Kings (2010) ENG Abridged {Michael Kramer} [WEB] M4B AAC 64kbps", got)
}

func TestDirectoryName_MP3ContainerOmitted(t *testing.T) {
	meta := baseMeta()
	meta.Tracks[0] = metadata.Track{Path: "book.mp3", Container: "", Codec: "MP3", Bitrate: 128}
	got, err := DirectoryName(context.Background(), meta)
	assert.NoError(t, err)
	assert.Equal(t, "Brandon Sanderson - The Way of Kings (2010) ENG {Michael Kramer} [WEB] MP3 128kbps", got)
}

func TestDirectoryName_FLACContainerOmitted(t *testing.T) {
	meta := baseMeta()
	meta.Tracks[0] = metadata.Track{Path: "book.flac", Container: "", Codec: "FLAC", Bitrate: 1000}
	got, err := DirectoryName(context.Background(), meta)
	assert.NoError(t, err)
	assert.Equal(t, "Brandon Sanderson - The Way of Kings (2010) ENG {Michael Kramer} [WEB] FLAC 1000kbps", got)
}

func TestDirectoryName_LanguageAlwaysRendersAsISO6393Token(t *testing.T) {
	// Every RULES.md §4-valid Language form must map to the canonical
	// directory name, else CheckNaming flags a correct dir as mismatch.
	for _, lang := range []string{"en", "eng", "ENG", "English", "english"} {
		meta := baseMeta()
		meta.Language = lang
		got, err := DirectoryName(context.Background(), meta)
		assert.NoError(t, err)
		assert.Contains(t, got, " ENG {", "Language %q must render as ENG", lang)
	}
}

func TestDirectoryName_SanitizesIllegalCharacters(t *testing.T) {
	meta := baseMeta()
	meta.Author = []string{"Vero/Nica Author"}
	got, err := DirectoryName(context.Background(), meta)
	assert.NoError(t, err)
	assert.NotContains(t, got, "/")
	assert.Contains(t, got, "Vero-Nica Author")
}

func TestDirectoryName_InconsistentTracksErrors(t *testing.T) {
	meta := baseMeta()
	meta.Tracks = append(meta.Tracks, metadata.Track{Path: "book2.mp3", Container: "", Codec: "MP3", Bitrate: 128})
	_, err := DirectoryName(context.Background(), meta)
	assert.ErrorContains(t, err, "inconsistent")
}

func TestDirectoryName_MissingAuthorErrors(t *testing.T) {
	meta := baseMeta()
	meta.Author = nil
	_, err := DirectoryName(context.Background(), meta)
	assert.ErrorContains(t, err, "author")
}

func TestDirectoryName_MissingTitleErrors(t *testing.T) {
	meta := baseMeta()
	meta.Title = ""
	_, err := DirectoryName(context.Background(), meta)
	assert.ErrorContains(t, err, "title")
}

func TestDirectoryName_MissingNarratorErrors(t *testing.T) {
	meta := baseMeta()
	meta.Narrator = nil
	_, err := DirectoryName(context.Background(), meta)
	assert.ErrorContains(t, err, "narrator")
}

func TestDirectoryName_NoTracksErrors(t *testing.T) {
	meta := baseMeta()
	meta.Tracks = nil
	_, err := DirectoryName(context.Background(), meta)
	assert.ErrorContains(t, err, "no tracks")
}

func multiFileMeta() *metadata.Metadata {
	return &metadata.Metadata{
		OriginalPath: "/books/example",
		Author:       []string{"Some Author"},
		Title:        "Example Title",
		Year:         2019,
		Narrator:     []string{"Some Narrator"},
		Language:     "en",
		Source:       metadata.ReleaseSourceWEB,
		Tracks: []metadata.Track{
			{Path: "/books/example/001.mp3", PartNumber: 1, Container: "", Codec: "MP3", Bitrate: 128},
			{Path: "/books/example/100.mp3", PartNumber: 100, Container: "", Codec: "MP3", Bitrate: 128},
		},
	}
}

func TestTrackName_SingleFile_EqualsDirectoryName(t *testing.T) {
	meta := baseMeta()
	dirName, err := DirectoryName(context.Background(), meta)
	assert.NoError(t, err)

	trackName, err := TrackName(context.Background(), meta, 0)
	assert.NoError(t, err)
	assert.Equal(t, dirName, trackName)
}

func TestTrackName_MultiFile_PartNumberPaddingAndChapterFallback(t *testing.T) {
	meta := multiFileMeta()
	got, err := TrackName(context.Background(), meta, 0)
	assert.NoError(t, err)
	assert.Equal(t, "001. Chapter 1 - Example Title (2019)", got)
}

func TestTrackName_MultiFile_UsesChapterTitleWhenPresent(t *testing.T) {
	meta := multiFileMeta()
	meta.Tracks[0].Chapters = []metadata.Chapter{{Title: "Prologue"}}
	got, err := TrackName(context.Background(), meta, 0)
	assert.NoError(t, err)
	assert.Equal(t, "001. Prologue - Example Title (2019)", got)
}

func TestTrackName_MultiFile_StripsLeadingNumberFromChapterTitle(t *testing.T) {
	meta := multiFileMeta()
	meta.Tracks[0].Chapters = []metadata.Chapter{{Title: "1. The Flame of Tar Valon"}}
	got, err := TrackName(context.Background(), meta, 0)
	assert.NoError(t, err)
	assert.Equal(t, "001. The Flame of Tar Valon - Example Title (2019)", got)
}

func TestTrackName_MultiFile_StripsZeroPaddedLeadingNumberFromChapterTitle(t *testing.T) {
	meta := multiFileMeta()
	meta.Tracks[0].Chapters = []metadata.Chapter{{Title: "01. The Flame of Tar Valon"}}
	got, err := TrackName(context.Background(), meta, 0)
	assert.NoError(t, err)
	assert.Equal(t, "001. The Flame of Tar Valon - Example Title (2019)", got)
}

func TestTrackName_MultiFile_LastTrackUsesFullWidthPadding(t *testing.T) {
	meta := multiFileMeta()
	got, err := TrackName(context.Background(), meta, 1)
	assert.NoError(t, err)
	assert.Equal(t, "100. Chapter 100 - Example Title (2019)", got)
}

func TestTrackName_OutOfRangeIndexErrors(t *testing.T) {
	meta := baseMeta()
	_, err := TrackName(context.Background(), meta, 5)
	assert.ErrorContains(t, err, "out of range")
}

func TestTrackName_MultiFile_InconsistentTracksErrors(t *testing.T) {
	meta := multiFileMeta()
	meta.Tracks[1].Codec = "FLAC"
	_, err := TrackName(context.Background(), meta, 0)
	assert.ErrorContains(t, err, "inconsistent")
}
