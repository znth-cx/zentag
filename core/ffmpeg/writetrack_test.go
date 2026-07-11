package ffmpeg

import (
	"context"
	"errors"
	"os"
	"testing"

	"codeberg.org/Ether/zentag/core/metadata"
)

type fakeRunner struct {
	gotBinPath string
	gotArgs    []string
	returnErr  error
	returnOut  []byte
}

func (f *fakeRunner) Run(_ context.Context, binPath string, args []string) ([]byte, error) {
	f.gotBinPath = binPath
	f.gotArgs = args
	return f.returnOut, f.returnErr
}

func TestWriteTrack_Success(t *testing.T) {
	fr := &fakeRunner{}
	w := &Wrapper{BinPath: "ffmpeg", Runner: fr}

	opts := WriteOpts{
		InputPath:  "in.mp3",
		OutputPath: "out.mp3",
		Metadata:   &metadata.Metadata{Title: "Some Book"},
		Track:      metadata.Track{Path: "in.mp3"},
	}

	if err := w.WriteTrack(context.Background(), opts); err != nil {
		t.Fatalf("WriteTrack() error = %v", err)
	}

	if fr.gotBinPath != "ffmpeg" {
		t.Errorf("Runner got binPath = %q, want %q", fr.gotBinPath, "ffmpeg")
	}
	if fr.gotArgs[0] != "-y" || fr.gotArgs[2] != "in.mp3" {
		t.Errorf("Runner got unexpected args = %q", fr.gotArgs)
	}
}

func TestWriteTrack_RunnerErrorPropagates(t *testing.T) {
	fr := &fakeRunner{returnErr: errors.New("exit status 1"), returnOut: []byte("ffmpeg: invalid argument")}
	w := &Wrapper{BinPath: "ffmpeg", Runner: fr}

	opts := WriteOpts{
		InputPath:  "in.mp3",
		OutputPath: "out.mp3",
		Metadata:   &metadata.Metadata{Title: "Some Book"},
		Track:      metadata.Track{Path: "in.mp3"},
	}

	err := w.WriteTrack(context.Background(), opts)
	if err == nil {
		t.Fatal("WriteTrack() error = nil, want error")
	}
}

func TestWriteTrack_SamePathRefused(t *testing.T) {
	fr := &fakeRunner{}
	w := &Wrapper{BinPath: "ffmpeg", Runner: fr}

	opts := WriteOpts{
		InputPath:  "in.mp3",
		OutputPath: "in.mp3",
		Metadata:   &metadata.Metadata{Title: "Some Book"},
		Track:      metadata.Track{Path: "in.mp3"},
	}

	err := w.WriteTrack(context.Background(), opts)
	if err == nil {
		t.Fatal("WriteTrack() error = nil, want error")
	}
	if fr.gotArgs != nil {
		t.Errorf("Runner invoked with args = %q, want no invocation", fr.gotArgs)
	}
}

func TestWriteTrack_CleansUpTempFilesOnSuccess(t *testing.T) {
	fr := &fakeRunner{}
	w := &Wrapper{BinPath: "ffmpeg", Runner: fr}

	opts := WriteOpts{
		InputPath:  "in.m4b",
		OutputPath: "out.m4b",
		Metadata: &metadata.Metadata{
			Title: "Some Book",
		},
		Track: metadata.Track{
			Path: "in.m4b",
			Chapters: []metadata.Chapter{
				{Title: "Chapter One", Start: 0, End: 1},
			},
		},
		EmbedChapters: true,
	}

	if err := w.WriteTrack(context.Background(), opts); err != nil {
		t.Fatalf("WriteTrack() error = %v", err)
	}

	// args[4] = chapter temp path (per buildArgs ordering)
	if _, err := os.Stat(fr.gotArgs[4]); !os.IsNotExist(err) {
		t.Errorf("temp file %q not cleaned up after WriteTrack", fr.gotArgs[4])
	}
}
