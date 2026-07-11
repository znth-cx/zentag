package M4BEngine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/znth-cx/zentag/core/ffmpeg"
	"github.com/znth-cx/zentag/core/metadata"
)

type fakeRunner struct {
	calls [][]string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	f.calls = append(f.calls, args)
	return nil, nil
}

// withFakeMP4Tag swaps writeMP4Tags for a fake for test duration.
func withFakeMP4Tag(t *testing.T, fake func(ctx context.Context, path string, m *metadata.Metadata) error) {
	t.Helper()
	orig := writeMP4Tags
	writeMP4Tags = fake
	t.Cleanup(func() { writeMP4Tags = orig })
}

func TestWrite_SingleTrackRemuxesChaptersOnlyThenWritesMP4Tags(t *testing.T) {
	fr := &fakeRunner{}
	w := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}

	var gotPath string
	var gotMeta *metadata.Metadata
	withFakeMP4Tag(t, func(_ context.Context, path string, m *metadata.Metadata) error {
		gotPath = path
		gotMeta = m
		return nil
	})

	meta := &metadata.Metadata{
		Title:      "The Way of Kings",
		CoverImage: []byte{1, 2, 3},
		CoverMIME:  "image/jpeg",
		Tracks: []metadata.Track{
			{Path: "book.m4b", Chapters: []metadata.Chapter{{Title: "Chapter One"}}},
		},
	}

	if err := Write(context.Background(), w, meta, "out", []string{"The Way of Kings"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if len(fr.calls) != 1 {
		t.Fatalf("Runner.Run called %d times, want 1", len(fr.calls))
	}
	args := fr.calls[0]
	if args[2] != "book.m4b" {
		t.Errorf("input path arg = %q, want %q", args[2], "book.m4b")
	}
	wantOut := filepath.Join("out", "The Way of Kings.m4b")
	if got := args[len(args)-1]; got != wantOut {
		t.Errorf("output path arg = %q, want %q", got, wantOut)
	}

	foundMapMetadataChapterInput := false
	for i, a := range args {
		if a == "-map_chapters" {
			t.Error("-map_chapters produces an orphaned chapter track with this ffmpeg's mov muxer; chapters must ride in via -map_metadata instead")
		}
		if a == "-map_metadata" && i+1 < len(args) && args[i+1] == "1" {
			foundMapMetadataChapterInput = true
		}
		if a == "asin=" || a == "title=The Way of Kings" {
			t.Errorf("ffmpeg remux must not write metadata tags (mp4tag owns those), found %q", a)
		}
	}
	if !foundMapMetadataChapterInput {
		t.Error("expected -map_metadata 1 (chapter temp file), so chapters carry over as a resolvable QT chapter track")
	}

	if gotPath != wantOut {
		t.Errorf("writeMP4Tags called with path %q, want %q", gotPath, wantOut)
	}
	if gotMeta != meta {
		t.Errorf("writeMP4Tags called with different *metadata.Metadata than Write received")
	}
}

func TestWrite_MP4TagErrorPropagates(t *testing.T) {
	fr := &fakeRunner{}
	w := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}

	wantErr := errors.New("mp4tag boom")
	withFakeMP4Tag(t, func(context.Context, string, *metadata.Metadata) error {
		return wantErr
	})

	// Pre-place the remux product; a tag failure must remove it, matching FLAC/MP3 cleanup.
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "Some Book.m4b")
	if err := os.WriteFile(outPath, []byte("remuxed"), 0o644); err != nil {
		t.Fatalf("pre-create output: %v", err)
	}

	meta := &metadata.Metadata{Title: "Some Book", Tracks: []metadata.Track{{Path: "book.m4b"}}}
	if err := Write(context.Background(), w, meta, outDir, []string{"Some Book"}); err == nil {
		t.Fatal("Write() error = nil, want mp4tag's error to propagate")
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Errorf("output %q still exists after tag failure, want removed", outPath)
	}
}

func TestWrite_RejectsNonSingleTrack(t *testing.T) {
	fr := &fakeRunner{}
	w := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}

	meta := &metadata.Metadata{Tracks: []metadata.Track{}}

	if err := Write(context.Background(), w, meta, "out", nil); err == nil {
		t.Fatal("Write() error = nil, want error for zero tracks")
	}
	if len(fr.calls) != 0 {
		t.Errorf("Runner.Run called %d times, want 0", len(fr.calls))
	}
}
