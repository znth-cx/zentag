package ruleset

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/naming"
)

// fullyBrokenMeta trips all seven Check* funcs: bad ISBN, missing year, invalid language, missing cover, no chapters, banned author/title, bare single-file path.
func fullyBrokenMeta() *metadata.Metadata {
	return &metadata.Metadata{
		OriginalPath: "/books/book.m4b",
		Author:       []string{"J.R.R. Tolkien"},
		Title:        "four against darkness expanded edition, and all associated content",
		Narrator:     []string{"Some Narrator"},
		Language:     "xx",
		ISBN:         "9780306406158", // bad checksum
		Tracks: []metadata.Track{
			{Path: "/books/book.m4b", Container: "M4B", Codec: "AAC", Bitrate: 64},
		},
	}
}

func TestValidate_FullyBrokenFixtureTriggersEveryCheck(t *testing.T) {
	violations := Validate(context.Background(), fullyBrokenMeta())

	rules := make(map[string]bool)
	for _, v := range violations {
		rules[v.Rule] = true
	}

	for _, want := range []string{"primary_keys", "required_tags", "language", "cover", "chapters", "banned_content", "naming"} {
		assert.True(t, rules[want], "expected a violation for rule %q, got rules %v", want, rules)
	}
}

// fullyCleanMeta satisfies all seven Check* funcs: valid ISBN, all required tags, valid language, small valid JPEG cover, a chapter, unrelated author/title, matching dir/track names.
func fullyCleanMeta(t *testing.T) *metadata.Metadata {
	t.Helper()
	meta := &metadata.Metadata{
		Author:     []string{"Brandon Sanderson"},
		Title:      "The Way of Kings",
		Year:       2010,
		Narrator:   []string{"Michael Kramer"},
		Language:   "eng",
		ISBN:       "9780306406157",
		Source:     metadata.ReleaseSourceWEB,
		CoverImage: noiseJPEG(t, 20, 20, 90),
		Tracks: []metadata.Track{
			{Container: "M4B", Codec: "AAC", Bitrate: 64, Chapters: []metadata.Chapter{{Title: "Ch1"}}},
		},
	}

	ctx := context.Background()
	dirName, err := naming.DirectoryName(ctx, meta)
	require.NoError(t, err)
	meta.OriginalPath = filepath.Join("/books", dirName)
	meta.Tracks[0].Path = filepath.Join(meta.OriginalPath, dirName+".m4b")

	return meta
}

func TestValidate_FullyCleanFixtureReturnsNoViolations(t *testing.T) {
	assert.Empty(t, Validate(context.Background(), fullyCleanMeta(t)))
}
