package FLACEngine

import (
	"context"
	"path/filepath"
	"testing"

	"codeberg.org/Ether/zentag/core/ffmpeg"
	"codeberg.org/Ether/zentag/core/metadata"
)

type fakeRunner struct {
	calls [][]string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	f.calls = append(f.calls, args)
	return nil, nil
}

func TestWrite_OneCallPerTrackNoCoverNoChapters(t *testing.T) {
	fr := &fakeRunner{}
	w := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}

	meta := &metadata.Metadata{
		Title:      "Some Book",
		CoverImage: []byte{1, 2, 3},
		CoverMIME:  "image/jpeg",
		Tracks: []metadata.Track{
			{Path: "part1.flac", Chapters: []metadata.Chapter{{Title: "Chapter One"}}},
			{Path: "part2.flac"},
		},
	}

	if err := Write(context.Background(), w, meta, "out", []string{"001. Chapter One - Some Book (0)", "002. Chapter Two - Some Book (0)"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if len(fr.calls) != 2 {
		t.Fatalf("Runner.Run called %d times, want 2", len(fr.calls))
	}

	wantInputs := []string{"part1.flac", "part2.flac"}
	wantOutputs := []string{
		filepath.Join("out", "001. Chapter One - Some Book (0).flac"),
		filepath.Join("out", "002. Chapter Two - Some Book (0).flac"),
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
				t.Errorf("call %d: unexpected cover/chapter arg %q, FLACEngine must not embed either", i, a)
			}
		}
	}
}
