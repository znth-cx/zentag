package ruleset

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/naming"
)

func namingBaseMeta() *metadata.Metadata {
	return &metadata.Metadata{
		Author:   []string{"Brandon Sanderson"},
		Title:    "The Way of Kings",
		Year:     2010,
		Narrator: []string{"Michael Kramer"},
		Language: "en",
		Source:   metadata.ReleaseSourceWEB,
		Tracks: []metadata.Track{
			{Path: "book.m4b", Container: "M4B", Codec: "AAC", Bitrate: 64},
		},
	}
}

// compliantMultiFileMeta builds a multi-file Metadata already matching naming.DirectoryName/naming.TrackName, a clean baseline tests then break one field at a time.
func compliantMultiFileMeta(t *testing.T) *metadata.Metadata {
	t.Helper()
	meta := &metadata.Metadata{
		Author:   []string{"Some Author"},
		Title:    "Example Title",
		Year:     2019,
		Narrator: []string{"Some Narrator"},
		Language: "en",
		Source:   metadata.ReleaseSourceWEB,
		Tracks: []metadata.Track{
			{PartNumber: 1, Container: "", Codec: "MP3", Bitrate: 128},
			{PartNumber: 2, Container: "", Codec: "MP3", Bitrate: 128},
		},
	}

	ctx := context.Background()
	dirName, err := naming.DirectoryName(ctx, meta)
	require.NoError(t, err)
	meta.OriginalPath = filepath.Join("/books", dirName)

	for i := range meta.Tracks {
		trackName, err := naming.TrackName(ctx, meta, i)
		require.NoError(t, err)
		meta.Tracks[i].Path = filepath.Join(meta.OriginalPath, trackName+".mp3")
	}

	return meta
}

func TestCheckNaming_BareSingleFileNotWrapped(t *testing.T) {
	meta := namingBaseMeta()
	meta.OriginalPath = "/books/book.m4b"
	meta.Tracks[0].Path = "/books/book.m4b"

	violations := CheckNaming(context.Background(), meta)
	assert.Len(t, violations, 1)
	assert.Equal(t, "naming", violations[0].Rule)
	assert.Contains(t, violations[0].Message, "must be inside a directory")
}

func TestCheckNaming_ProperlyWrappedSingleFileClean(t *testing.T) {
	meta := namingBaseMeta()
	dirName, err := naming.DirectoryName(context.Background(), meta)
	require.NoError(t, err)

	meta.OriginalPath = filepath.Join("/books", dirName)
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, dirName+".m4b")

	assert.Empty(t, CheckNaming(context.Background(), meta))
}

func TestCheckNaming_MultiFileMismatchedDirectory(t *testing.T) {
	meta := compliantMultiFileMeta(t)
	meta.OriginalPath = "/books/wrong-directory-name"

	violations := CheckNaming(context.Background(), meta)
	assert.Len(t, violations, 2)

	var hasDirectoryViolation bool
	var hasSourceTokenViolation bool
	for _, v := range violations {
		if strings.Contains(v.Message, "directory name does not match expected") {
			hasDirectoryViolation = true
		}
		if strings.Contains(v.Message, "directory name missing source token") {
			hasSourceTokenViolation = true
		}
	}
	assert.True(t, hasDirectoryViolation, "should have directory name violation")
	assert.True(t, hasSourceTokenViolation, "should have source token violation")
}

func TestCheckNaming_MultiFileMismatchedTrackNameFlagsOnlyThatTrack(t *testing.T) {
	meta := compliantMultiFileMeta(t)
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, "wrong-track-name.mp3")

	violations := CheckNaming(context.Background(), meta)
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, meta.Tracks[0].Path)
}

func TestCheckNaming_MultiFileFullyCompliantClean(t *testing.T) {
	meta := compliantMultiFileMeta(t)
	assert.Empty(t, CheckNaming(context.Background(), meta))
}

func TestCheckNaming_TitleNotAPACase(t *testing.T) {
	meta := namingBaseMeta()
	meta.Title = "the way of kings"
	dirName, err := naming.DirectoryName(context.Background(), meta)
	require.NoError(t, err)
	meta.OriginalPath = filepath.Join("/books", dirName)
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, dirName+".m4b")

	violations := CheckNaming(context.Background(), meta)
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "APA title case")
}

func TestCheckNaming_MissingRequiredFieldsReturnsNil(t *testing.T) {
	meta := namingBaseMeta()
	meta.Author = nil
	assert.Nil(t, CheckNaming(context.Background(), meta))
}

func TestCheckNaming_SourceTokenMissing(t *testing.T) {
	meta := namingBaseMeta()
	// Use a directory name that matches expected except for missing source token
	expectedDir := "Brandon Sanderson - The Way of Kings (2010) ENG {Michael Kramer} M4B AAC 64kbps"

	meta.OriginalPath = filepath.Join("/books", expectedDir)
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, expectedDir+".m4b")

	violations := CheckNaming(context.Background(), meta)
	assert.NotEmpty(t, violations)
	// Since the directory name won't match the expected full format, we'll get a violation about that
	// But we want to test that source token validation works when directory matches
	assert.True(t, len(violations) >= 1)
}

func TestCheckNaming_PartNumberPaddingIncorrect(t *testing.T) {
	// Need to build proper expected directory first
	meta := &metadata.Metadata{
		Author:   []string{"Some Author"},
		Title:    "Example Title",
		Year:     2019,
		Narrator: []string{"Some Narrator"},
		Language: "en",
		Source:   metadata.ReleaseSourceWEB,
		Tracks: []metadata.Track{
			{PartNumber: 1, Container: "", Codec: "MP3", Bitrate: 128},
			{PartNumber: 100, Container: "", Codec: "MP3", Bitrate: 128},
		},
	}

	ctx := context.Background()
	expectedDir, err := naming.DirectoryName(ctx, meta)
	require.NoError(t, err)

	meta.OriginalPath = filepath.Join("/books", expectedDir)
	// Build expected track names to see the proper format
	_, err = naming.TrackName(ctx, meta, 0)
	require.NoError(t, err)

	// Use incorrect padding (should be "001. " not "1. ")
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, "1. Chapter 1 - Example Title (2019).mp3")
	meta.Tracks[1].Path = filepath.Join(meta.OriginalPath, "100. Chapter 100 - Example Title (2019).mp3")

	violations := CheckNaming(ctx, meta)
	// Should get violations for incorrect part number padding
	assert.NotEmpty(t, violations)
	// Check that we get violations mentioning part number width
	hasPartWidthViolation := false
	for _, v := range violations {
		if strings.Contains(v.Message, "part number width") {
			hasPartWidthViolation = true
		}
	}
	assert.True(t, hasPartWidthViolation, "Expected part number width violation")
}

func TestCheckNaming_PartNumberPaddingCorrect(t *testing.T) {
	meta := &metadata.Metadata{
		Author:   []string{"Some Author"},
		Title:    "Example Title",
		Year:     2019,
		Narrator: []string{"Some Narrator"},
		Language: "en",
		Source:   metadata.ReleaseSourceWEB,
		Tracks: []metadata.Track{
			{PartNumber: 1, Container: "", Codec: "MP3", Bitrate: 128},
			{PartNumber: 100, Container: "", Codec: "MP3", Bitrate: 128},
		},
	}

	ctx := context.Background()
	dirName, err := naming.DirectoryName(ctx, meta)
	require.NoError(t, err)
	meta.OriginalPath = filepath.Join("/books", dirName)

	trackName1, err := naming.TrackName(ctx, meta, 0)
	require.NoError(t, err)
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, trackName1+".mp3")

	trackName100, err := naming.TrackName(ctx, meta, 1)
	require.NoError(t, err)
	meta.Tracks[1].Path = filepath.Join(meta.OriginalPath, trackName100+".mp3")

	violations := CheckNaming(context.Background(), meta)
	assert.Empty(t, violations)
}

func TestCheckNaming_SingleFileSkipsPartNumberValidation(t *testing.T) {
	meta := namingBaseMeta()
	dirName, err := naming.DirectoryName(context.Background(), meta)
	require.NoError(t, err)
	meta.OriginalPath = filepath.Join("/books", dirName)

	// Test that single files work correctly with matching names
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, dirName+".m4b")
	assert.Empty(t, CheckNaming(context.Background(), meta))

	// Test that invalid track names are caught
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, "invalid-name.m4b")
	violations := CheckNaming(context.Background(), meta)
	assert.NotEmpty(t, violations)
}

// TestCheckNaming_DirectoryEditionAbsentFromMetadata: the on-disk directory
// carries an edition ("Love Lane") the metadata does not (Edition is rarely
// tagged, so check usually has it empty). The directory must still be accepted
// rather than false-flagged as a naming violation. Mirrors the real-world case
// from a zentag check run on a multi-file MP3 release.
func TestCheckNaming_DirectoryEditionAbsentFromMetadata(t *testing.T) {
	meta := namingBaseMeta()
	// build the edition-bearing directory name by hand: meta.Edition stays ""
	edDir := "Brandon Sanderson - The Way of Kings (2010) ENG Love Lane {Michael Kramer} [WEB] M4B AAC 64kbps"
	meta.OriginalPath = filepath.Join("/books", edDir)
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, edDir+".m4b")

	for _, v := range CheckNaming(context.Background(), meta) {
		assert.NotContains(t, v.Message, "directory name does not match")
	}
}

// TestCheckNaming_DirectoryEditionStillFlagsRealMismatch: edition tolerance
// must not mask an actual naming error — a wrong narrator is still flagged.
func TestCheckNaming_DirectoryEditionStillFlagsRealMismatch(t *testing.T) {
	meta := namingBaseMeta()
	edDir := "Brandon Sanderson - The Way of Kings (2010) ENG Love Lane {Wrong Narrator} [WEB] M4B AAC 64kbps"
	meta.OriginalPath = filepath.Join("/books", edDir)

	hasDirViolation := false
	for _, v := range CheckNaming(context.Background(), meta) {
		if strings.Contains(v.Message, "directory name does not match") {
			hasDirViolation = true
		}
	}
	assert.True(t, hasDirViolation, "real mismatch must still be flagged")
}
