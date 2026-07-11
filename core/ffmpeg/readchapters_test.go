package ffmpeg

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/znth-cx/zentag/core/metadata"
)

const chaptersJSON = `{
  "chapters": [
    {"start_time": "0.000000", "end_time": "60.000000", "tags": {"title": "Chapter One"}},
    {"start_time": "60.000000", "end_time": "125.500000", "tags": {"title": "Chapter Two"}}
  ]
}`

const noChaptersJSON = `{"chapters": []}`

func TestReadChapters_HappyPath(t *testing.T) {
	fr := &fakeRunner{returnOut: []byte(chaptersJSON)}
	w := &Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: fr}

	got, err := w.ReadChapters(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadChapters() error = %v", err)
	}

	want := []metadata.Chapter{
		{Title: "Chapter One", Start: 0, End: 60 * time.Second},
		{Title: "Chapter Two", Start: 60 * time.Second, End: 125500 * time.Millisecond},
	}
	if len(got) != len(want) {
		t.Fatalf("ReadChapters() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("chapter[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	if fr.gotBinPath != "ffprobe" {
		t.Errorf("binPath = %q, want %q", fr.gotBinPath, "ffprobe")
	}
	wantArgs := []string{"-v", "quiet", "-show_chapters", "-print_format", "json", "book.m4b"}
	if len(fr.gotArgs) != len(wantArgs) {
		t.Fatalf("args = %q, want %q", fr.gotArgs, wantArgs)
	}
	for i := range wantArgs {
		if fr.gotArgs[i] != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q", i, fr.gotArgs[i], wantArgs[i])
		}
	}
}

func TestReadChapters_NoChaptersReturnsEmptyNotError(t *testing.T) {
	fr := &fakeRunner{returnOut: []byte(noChaptersJSON)}
	w := &Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: fr}

	got, err := w.ReadChapters(context.Background(), "book.mp3")
	if err != nil {
		t.Fatalf("ReadChapters() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ReadChapters() = %+v, want empty", got)
	}
}

func TestProbeDump_HappyPath(t *testing.T) {
	fr := &fakeRunner{returnOut: []byte("Input #0, mov,mp4,m4a...\n")}
	w := &Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: fr}

	got, err := w.ProbeDump(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ProbeDump() error = %v", err)
	}
	if got != "Input #0, mov,mp4,m4a...\n" {
		t.Errorf("ProbeDump() = %q", got)
	}

	if fr.gotBinPath != "ffprobe" {
		t.Errorf("binPath = %q, want %q", fr.gotBinPath, "ffprobe")
	}
	wantArgs := []string{"-hide_banner", "-show_format", "-show_streams", "-show_chapters", "book.m4b"}
	if len(fr.gotArgs) != len(wantArgs) {
		t.Fatalf("args = %q, want %q", fr.gotArgs, wantArgs)
	}
	for i := range wantArgs {
		if fr.gotArgs[i] != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q", i, fr.gotArgs[i], wantArgs[i])
		}
	}
}

func TestProbeDump_RunnerError(t *testing.T) {
	fr := &fakeRunner{returnErr: errors.New("exit status 1")}
	w := &Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: fr}

	if _, err := w.ProbeDump(context.Background(), "book.m4b"); err == nil {
		t.Fatal("ProbeDump() error = nil, want error")
	}
}

func TestReadChapters_ErrorCases(t *testing.T) {
	cases := []struct {
		name string
		out  string
		err  error
	}{
		{"runner error", "", errors.New("exit status 1")},
		{"malformed json", "{not valid json", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := &fakeRunner{returnOut: []byte(tc.out), returnErr: tc.err}
			w := &Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: fr}

			_, err := w.ReadChapters(context.Background(), "book.m4b")
			if err == nil {
				t.Fatal("ReadChapters() error = nil, want error")
			}
		})
	}
}
