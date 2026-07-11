package ruleset

import (
	"testing"
	"time"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/stretchr/testify/assert"
)

func TestCheckChapters_M4BTrackWithNoChaptersFlagged(t *testing.T) {
	meta := &metadata.Metadata{Tracks: []metadata.Track{{Path: "book.m4b", Container: "M4B"}}}
	violations := CheckChapters(meta)
	assert.Len(t, violations, 1)
	assert.Equal(t, "chapters", violations[0].Rule)
	assert.Equal(t, SeverityWarn, violations[0].Severity, "RULES.md §9 only mandates chapters when the source had them, so warn not fail")
	assert.Contains(t, violations[0].Message, "book.m4b")
}

func TestCheckChapters_M4BTrackWithChaptersClean(t *testing.T) {
	meta := &metadata.Metadata{Tracks: []metadata.Track{
		{Path: "book.m4b", Container: "M4B", Chapters: []metadata.Chapter{{Title: "Ch1", Start: 0, End: time.Minute}}},
	}}
	assert.Empty(t, CheckChapters(meta))
}

func TestCheckChapters_MultiTrackOnlyFlagsMissingOnes(t *testing.T) {
	meta := &metadata.Metadata{Tracks: []metadata.Track{
		{Path: "001.m4b", Container: "M4B", Chapters: []metadata.Chapter{{Title: "Ch1"}}},
		{Path: "002.m4b", Container: "M4B"},
	}}
	violations := CheckChapters(meta)
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "002.m4b")
}

func TestCheckChapters_MP3AndFLACNeverFlaggedEvenWithoutChapters(t *testing.T) {
	meta := &metadata.Metadata{Tracks: []metadata.Track{
		{Path: "001.mp3", Codec: "MP3"},
		{Path: "001.flac", Codec: "FLAC"},
	}}
	assert.Empty(t, CheckChapters(meta), "MP3/FLAC don't embed chapter markers, so a missing one isn't a violation")
}

func TestCheckAudnexusChapters_NoCountSkips(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{{Path: "book.m4b", Container: "M4B", Chapters: []metadata.Chapter{{Title: "Ch1"}}}},
	}
	assert.Empty(t, CheckAudnexusChapters(meta), "AudnexusChapterCount 0 means not looked up — nothing to compare")
}

func TestCheckAudnexusChapters_M4BMismatchWarns(t *testing.T) {
	meta := &metadata.Metadata{
		AudnexusChapterCount: 3,
		Tracks: []metadata.Track{{Path: "book.m4b", Container: "M4B", Chapters: []metadata.Chapter{
			{Title: "Ch1"}, {Title: "Ch2"},
		}}},
	}
	violations := CheckAudnexusChapters(meta)
	assert.Len(t, violations, 1)
	assert.Equal(t, "audnexus_chapters", violations[0].Rule)
	assert.Equal(t, SeverityWarn, violations[0].Severity)
	assert.Contains(t, violations[0].Message, "3")
	assert.Contains(t, violations[0].Message, "2")
}

func TestCheckAudnexusChapters_M4BMatchClean(t *testing.T) {
	meta := &metadata.Metadata{
		AudnexusChapterCount: 2,
		Tracks: []metadata.Track{{Path: "book.m4b", Container: "M4B", Chapters: []metadata.Chapter{
			{Title: "Ch1"}, {Title: "Ch2"},
		}}},
	}
	assert.Empty(t, CheckAudnexusChapters(meta))
}

func TestCheckAudnexusChapters_MultiFileComparesPartCount(t *testing.T) {
	meta := &metadata.Metadata{
		AudnexusChapterCount: 5,
		Tracks: []metadata.Track{
			{Path: "001.mp3", Codec: "MP3", PartNumber: 1},
			{Path: "002.mp3", Codec: "MP3", PartNumber: 2},
		},
	}
	violations := CheckAudnexusChapters(meta)
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "5")
	assert.Contains(t, violations[0].Message, "2")
}

func TestCheckAudnexusChapters_SingleFileNonM4BSkips(t *testing.T) {
	meta := &metadata.Metadata{
		AudnexusChapterCount: 12,
		Tracks:               []metadata.Track{{Path: "book.mp3", Codec: "MP3"}},
	}
	assert.Empty(t, CheckAudnexusChapters(meta), "a single MP3/FLAC file has no part/chapter count to compare against")
}
