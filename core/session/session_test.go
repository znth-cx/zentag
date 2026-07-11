package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleMeta() *metadata.Metadata {
	return &metadata.Metadata{
		Title:      "The Hobbit",
		Author:     []string{"J.R.R. Tolkien"},
		Year:       1937,
		CoverImage: []byte{0x01, 0x02, 0x03},
		CoverMIME:  "image/jpeg",
		Tracks: []metadata.Track{{
			Path:  "/books/hobbit.m4b",
			Codec: "AAC",
			Chapters: []metadata.Chapter{
				{Title: "An Unexpected Party", Start: 0, End: 10 * time.Minute},
			},
		}},
	}
}

func TestPath_SlugifiesAbsolutePathAndIsStable(t *testing.T) {
	dir := t.TempDir()
	p1, err := Path(dir, "/books/The Hobbit")
	require.NoError(t, err)
	p2, err := Path(dir, "/books/The Hobbit")
	require.NoError(t, err)

	assert.Equal(t, p1, p2, "same item path must key to the same file")
	assert.Equal(t, dir, filepath.Dir(p1))
	assert.True(t, strings.HasSuffix(p1, ".json"))
	// Space and separators are replaced with '_'.
	assert.NotContains(t, filepath.Base(p1), " ")
	assert.NotContains(t, filepath.Base(p1), string(filepath.Separator))
}

func TestPath_DistinctPathsDistinctFiles(t *testing.T) {
	dir := t.TempDir()
	a, err := Path(dir, "/books/Dune")
	require.NoError(t, err)
	b, err := Path(dir, "/books/Hobbit")
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestFileName_LongPathTruncatedWithSuffix(t *testing.T) {
	long := "/" + strings.Repeat("a", 400)
	name := fileName(long)
	assert.LessOrEqual(t, len(name), 200+len(".json"))
	assert.True(t, strings.HasSuffix(name, ".json"))
}

func TestFileName_ShortPathUnchangedScheme(t *testing.T) {
	// existing saved sessions must still resume: no hash suffix under maxSlug
	assert.Equal(t, "_books_The_Hobbit.json", fileName("/books/The Hobbit"))
}

func TestFileName_LongPathsSharedPrefixDoNotCollide(t *testing.T) {
	// identical first 191 chars and identical tail: only full-path hash disambiguates
	prefix := "/" + strings.Repeat("a", 200)
	suffix := strings.Repeat("z", 20)
	n1 := fileName(prefix + "one" + suffix)
	n2 := fileName(prefix + "two" + suffix)
	assert.NotEqual(t, n1, n2)
	assert.LessOrEqual(t, len(n1), 200+len(".json"))
	assert.LessOrEqual(t, len(n2), 200+len(".json"))
}

func TestSave_AtomicValidJSONNoTempLeftover(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	item := "/books/atomic"
	require.NoError(t, Save(ctx, dir, item, sampleMeta()))
	// overwrite same item: rename must replace, not fail
	require.NoError(t, Save(ctx, dir, item, sampleMeta()))

	p, err := Path(dir, item)
	require.NoError(t, err)
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.True(t, json.Valid(data), "target must be complete valid JSON")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temp files left behind")
	assert.Equal(t, filepath.Base(p), entries[0].Name())
}

func TestSaveThenLoad_RoundTrips(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions") // not pre-created
	ctx := context.Background()
	item := "/books/The Hobbit"

	require.NoError(t, Save(ctx, dir, item, sampleMeta()))

	got, found, err := Load(ctx, dir, item)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "The Hobbit", got.Title)
	assert.Equal(t, []string{"J.R.R. Tolkien"}, got.Author)
	assert.Equal(t, 1937, got.Year)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, got.CoverImage)
	require.Len(t, got.Tracks, 1)
	require.Len(t, got.Tracks[0].Chapters, 1)
	assert.Equal(t, "An Unexpected Party", got.Tracks[0].Chapters[0].Title)
	assert.Equal(t, 10*time.Minute, got.Tracks[0].Chapters[0].End)
}

func TestLoad_MissingSessionNotFoundNoError(t *testing.T) {
	got, found, err := Load(context.Background(), t.TempDir(), "/books/never-saved")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestLoad_CorruptFileErrors(t *testing.T) {
	dir := t.TempDir()
	item := "/books/corrupt"
	p, err := Path(dir, item)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p, []byte("{not json"), 0o644))

	_, found, err := Load(context.Background(), dir, item)
	assert.Error(t, err)
	assert.False(t, found)
}

func TestClean_RemovesOnlyThatItem(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	keep := "/books/Dune"
	drop := "/books/Hobbit"
	require.NoError(t, Save(ctx, dir, keep, sampleMeta()))
	require.NoError(t, Save(ctx, dir, drop, sampleMeta()))

	require.NoError(t, Clean(ctx, dir, drop))

	_, found, err := Load(ctx, dir, drop)
	require.NoError(t, err)
	assert.False(t, found, "cleaned item's session should be gone")

	_, found, err = Load(ctx, dir, keep)
	require.NoError(t, err)
	assert.True(t, found, "other items' sessions must survive")
}

func TestClean_MissingFileNoError(t *testing.T) {
	assert.NoError(t, Clean(context.Background(), t.TempDir(), "/books/never-saved"))
}
