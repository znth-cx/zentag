package ruleset

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckChapterFormat_M4BWithChaptersClean(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{
			{
				Path:      "book.m4b",
				Container: "M4B",
				Chapters: []metadata.Chapter{
					{Title: "Chapter 1", Start: 0, End: 10 * time.Minute},
				},
			},
		},
	}

	assert.Empty(t, CheckChapterFormat(context.Background(), meta))
}

func TestCheckChapterFormat_M4BWithoutChaptersSkipped(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{
			{
				Path:      "book.m4b",
				Container: "M4B",
				Chapters:  []metadata.Chapter{},
			},
		},
	}

	assert.Empty(t, CheckChapterFormat(context.Background(), meta))
}

func TestCheckChapterFormat_MP3Skipped(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{
			{
				Path:      "book.mp3",
				Container: "MP3",
				Chapters: []metadata.Chapter{
					{Title: "Chapter 1", Start: 0, End: 10 * time.Minute},
				},
			},
		},
	}

	assert.Empty(t, CheckChapterFormat(context.Background(), meta))
}

func TestCheckChapterFormat_FLACSkipped(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{
			{
				Path:      "book.flac",
				Container: "FLAC",
				Chapters: []metadata.Chapter{
					{Title: "Chapter 1", Start: 0, End: 10 * time.Minute},
				},
			},
		},
	}

	assert.Empty(t, CheckChapterFormat(context.Background(), meta))
}

func TestCheckChapterFormat_NilTrackSkipped(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{},
	}

	assert.Empty(t, CheckChapterFormat(context.Background(), meta))
}

func TestCheckChapterFormat_M4BWithMultipleChaptersClean(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{
			{
				Path:      "book.m4b",
				Container: "M4B",
				Chapters: []metadata.Chapter{
					{Title: "Chapter 1", Start: 0, End: 10 * time.Minute},
					{Title: "Chapter 2", Start: 10 * time.Minute, End: 20 * time.Minute},
					{Title: "Chapter 3", Start: 20 * time.Minute, End: 30 * time.Minute},
				},
			},
		},
	}

	assert.Empty(t, CheckChapterFormat(context.Background(), meta))
}
