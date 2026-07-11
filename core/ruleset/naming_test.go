package ruleset

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/naming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "directory name")
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
