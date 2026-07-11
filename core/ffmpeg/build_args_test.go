package ffmpeg

import (
	"os"
	"testing"
	"time"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestBuildArgs_M4BWithChapters(t *testing.T) {
	meta := &metadata.Metadata{Title: "The Way of Kings"}
	track := metadata.Track{
		Path: "in.m4b",
		Chapters: []metadata.Chapter{
			{Title: "Chapter One", Start: 0, End: 60 * time.Second},
		},
	}
	opts := WriteOpts{
		InputPath:     "in.m4b",
		OutputPath:    "out.m4b",
		Metadata:      meta,
		Track:         track,
		SkipMetadata:  true,
		EmbedChapters: true,
	}

	args, cleanup, err := buildArgs(opts)
	if err != nil {
		t.Fatalf("buildArgs() error = %v", err)
	}
	defer cleanup()

	if got, want := args[0], "-y"; got != want {
		t.Errorf("args[0] = %q, want %q", got, want)
	}
	if got, want := args[1], "-i"; got != want {
		t.Errorf("args[1] = %q, want %q", got, want)
	}
	if got, want := args[2], "in.m4b"; got != want {
		t.Errorf("args[2] = %q, want %q", got, want)
	}

	// second -i input: chapter file
	if got, want := args[3], "-i"; got != want {
		t.Errorf("args[3] = %q, want %q", got, want)
	}
	chapterPath := args[4]
	data, err := os.ReadFile(chapterPath)
	if err != nil {
		t.Fatalf("chapter file %q not readable: %v", chapterPath, err)
	}
	if string(data) != chapterMetadataContent(track.Chapters) {
		t.Errorf("chapter file content = %q, want %q", data, chapterMetadataContent(track.Chapters))
	}

	rest := args[5:]
	wantRest := []string{
		"-map", "0:a",
		"-map_metadata", "1",
		"-c", "copy",
	}
	for i, w := range wantRest {
		if rest[i] != w {
			t.Errorf("rest[%d] = %q, want %q (rest=%q)", i, rest[i], w, rest)
		}
	}
	if len(rest) != len(wantRest)+1 { // +1 for the trailing output path
		t.Errorf("rest = %q, want exactly %q + output path (SkipMetadata must omit metadataArgs)", rest, wantRest)
	}

	if args[len(args)-1] != "out.m4b" {
		t.Errorf("last arg = %q, want output path", args[len(args)-1])
	}

	cleanup()
	if _, err := os.Stat(chapterPath); !os.IsNotExist(err) {
		t.Error("cleanup() did not remove chapter temp file")
	}
}

func TestBuildArgs_SkipMetadataFalseStillWritesMetadataArgs(t *testing.T) {
	meta := &metadata.Metadata{Title: "The Way of Kings", ASIN: "B0REALASIN"}
	track := metadata.Track{Path: "in.m4b"}
	opts := WriteOpts{
		InputPath:  "in.m4b",
		OutputPath: "out.m4b",
		Metadata:   meta,
		Track:      track,
	}

	args, cleanup, err := buildArgs(opts)
	if err != nil {
		t.Fatalf("buildArgs() error = %v", err)
	}
	defer cleanup()

	found := false
	for _, a := range args {
		if a == "asin=B0REALASIN" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected metadataArgs' asin=B0REALASIN in args, got %q", args)
	}
}

func TestBuildArgs_MP3NoChapters(t *testing.T) {
	meta := &metadata.Metadata{Title: "Some Book"}
	track := metadata.Track{Path: "part1.mp3"}
	opts := WriteOpts{
		InputPath:     "part1.mp3",
		OutputPath:    "out1.mp3",
		Metadata:      meta,
		Track:         track,
		EmbedChapters: false,
	}

	args, cleanup, err := buildArgs(opts)
	if err != nil {
		t.Fatalf("buildArgs() error = %v", err)
	}
	defer cleanup()

	want := []string{"-y", "-i", "part1.mp3", "-map", "0:a", "-map_metadata", "0", "-c", "copy"}
	for i, w := range want {
		if args[i] != w {
			t.Errorf("args[%d] = %q, want %q", i, args[i], w)
		}
	}

	for _, a := range args {
		if a == "-map_chapters" || a == "attached_pic" || a == "use_metadata_tags" {
			t.Errorf("unexpected arg %q in no-chapter MP3 build", a)
		}
	}

	if args[len(args)-1] != "out1.mp3" {
		t.Errorf("last arg = %q, want output path", args[len(args)-1])
	}
}

func TestBuildArgs_StreamLanguageSetRegardlessOfSkipMetadata(t *testing.T) {
	meta := &metadata.Metadata{Title: "The Way of Kings", Language: "eng"}
	track := metadata.Track{Path: "in.m4b"}
	opts := WriteOpts{
		InputPath:    "in.m4b",
		OutputPath:   "out.m4b",
		Metadata:     meta,
		Track:        track,
		SkipMetadata: true,
	}

	args, cleanup, err := buildArgs(opts)
	if err != nil {
		t.Fatalf("buildArgs() error = %v", err)
	}
	defer cleanup()

	found := false
	for i, a := range args {
		if a == "-metadata:s:a:0" && i+1 < len(args) && args[i+1] == "language=eng" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected -metadata:s:a:0 language=eng even under SkipMetadata (mp4tag can't set mdhd), got %q", args)
	}
}

func TestBuildArgs_NoStreamLanguageArgWhenUnset(t *testing.T) {
	meta := &metadata.Metadata{Title: "Some Book"}
	track := metadata.Track{Path: "part1.mp3"}
	opts := WriteOpts{
		InputPath:  "part1.mp3",
		OutputPath: "out1.mp3",
		Metadata:   meta,
		Track:      track,
	}

	args, cleanup, err := buildArgs(opts)
	if err != nil {
		t.Fatalf("buildArgs() error = %v", err)
	}
	defer cleanup()

	for _, a := range args {
		if a == "-metadata:s:a:0" {
			t.Errorf("unexpected -metadata:s:a:0 arg when Language is unset, got %q", args)
		}
	}
}
