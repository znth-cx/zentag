package ffmpeg

import (
	"context"
	"testing"
)

func TestNew_SetsBinPathsAndDefaultRunner(t *testing.T) {
	w := New("ffmpeg", "ffprobe")

	if w.BinPath != "ffmpeg" {
		t.Errorf("BinPath = %q, want %q", w.BinPath, "ffmpeg")
	}
	if w.ProbeBinPath != "ffprobe" {
		t.Errorf("ProbeBinPath = %q, want %q", w.ProbeBinPath, "ffprobe")
	}
	if w.Runner == nil {
		t.Error("Runner is nil, want default real runner")
	}
}

func TestRealRunner_Run_NonexistentBinary(t *testing.T) {
	r := &realRunner{}

	_, err := r.Run(context.Background(), "zentag-definitely-not-a-real-binary-xyz", nil)

	if err == nil {
		t.Error("Run() with nonexistent binary: got nil error, want error")
	}
}

func TestRealRunner_Run_Success(t *testing.T) {
	r := &realRunner{}

	out, err := r.Run(context.Background(), "go", []string{"version"})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(out) == 0 {
		t.Error("Run() output empty, want go version output")
	}
}
