package MP3Engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/znth-cx/zentag/core/ffmpeg"
	"github.com/znth-cx/zentag/core/metadata"
	"go.senan.xyz/taglib"
)

type fakeRunner struct {
	calls [][]string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	f.calls = append(f.calls, args)
	return nil, nil
}

type fakeMP3TagWriter struct{}

func (f *fakeMP3TagWriter) WriteTags(_ context.Context, path string, _ *metadata.Metadata, _ metadata.Track) error {
	if path == "" {
		return taglib.ErrSavingFile
	}
	return nil
}

func TestWrite_OneCallPerTrackNoCoverNoChapters(t *testing.T) {
	fr := &fakeRunner{}
	ft := &fakeMP3TagWriter{}
	w := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}

	meta := &metadata.Metadata{
		Title:      "Some Book",
		CoverImage: []byte{1, 2, 3},
		CoverMIME:  "image/jpeg",
		Tracks: []metadata.Track{
			{Path: "part1.mp3", Chapters: []metadata.Chapter{{Title: "Chapter One"}}},
			{Path: "part2.mp3"},
		},
	}

	writeMP3Tags = ft.WriteTags
	if err := Write(context.Background(), w, meta, "out", []string{"001. Chapter One - Some Book (0)", "002. Chapter Two - Some Book (0)"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if len(fr.calls) != 2 {
		t.Fatalf("Runner.Run called %d times, want 2", len(fr.calls))
	}

	wantInputs := []string{"part1.mp3", "part2.mp3"}
	wantOutputs := []string{
		filepath.Join("out", "001. Chapter One - Some Book (0).mp3"),
		filepath.Join("out", "002. Chapter Two - Some Book (0).mp3"),
	}
	for i, args := range fr.calls {
		if args[2] != wantInputs[i] {
			t.Errorf("call %d input = %q, want %q", i, args[2], wantInputs[i])
		}
		if args[len(args)-1] != wantOutputs[i] {
			t.Errorf("call %d output = %q, want %q", i, args[len(args)-1], wantOutputs[i])
		}
		for _, a := range args {
			if a == "attached_pic" || a == "-map_chapters" {
				t.Errorf("call %d: unexpected cover/chapter arg %q, MP3Engine must not embed either", i, a)
			}
		}
	}
}
