package ffmpeg

import (
	"os"
	"testing"
	"time"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestChapterMetadataContent(t *testing.T) {
	chapters := []metadata.Chapter{
		{Title: "Chapter One", Start: 0, End: 125 * time.Second},
		{Title: "Chapter Two", Start: 125 * time.Second, End: 260500 * time.Millisecond},
	}

	got := chapterMetadataContent(chapters)
	want := ";FFMETADATA1\n" +
		"[CHAPTER]\n" +
		"TIMEBASE=1/1000\n" +
		"START=0\n" +
		"END=125000\n" +
		"title=Chapter One\n" +
		"\n" +
		"[CHAPTER]\n" +
		"TIMEBASE=1/1000\n" +
		"START=125000\n" +
		"END=260500\n" +
		"title=Chapter Two\n"

	if got != want {
		t.Errorf("chapterMetadataContent() =\n%q\nwant\n%q", got, want)
	}
}

func TestChapterMetadataContentEscaping(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"a;b", `title=a\;b`},
		{"a=b", `title=a\=b`},
		{"a#b", `title=a\#b`},
		{`a\b`, `title=a\\b`},
		{"a\nb", "title=a\\\nb"},
	}
	for _, tt := range tests {
		got := chapterMetadataContent([]metadata.Chapter{
			{Title: tt.title, Start: 0, End: time.Second},
		})
		want := ";FFMETADATA1\n" +
			"[CHAPTER]\n" +
			"TIMEBASE=1/1000\n" +
			"START=0\n" +
			"END=1000\n" +
			tt.want + "\n"
		if got != want {
			t.Errorf("chapterMetadataContent(title=%q) =\n%q\nwant\n%q", tt.title, got, want)
		}
	}
}

func TestWriteChapterFile(t *testing.T) {
	chapters := []metadata.Chapter{
		{Title: "Chapter One", Start: 0, End: 60 * time.Second},
	}

	path, cleanup, err := writeChapterFile(chapters)
	if err != nil {
		t.Fatalf("writeChapterFile() error = %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(data) != chapterMetadataContent(chapters) {
		t.Errorf("file content = %q, want %q", data, chapterMetadataContent(chapters))
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("cleanup() did not remove temp file %q", path)
	}
}
