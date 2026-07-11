// Package ffmpeg translates metadata.Metadata into ffmpeg args (tags, cover, chapters) and runs them via os/exec.
package ffmpeg

import (
	"context"
	"os/exec"
)

// Runner executes a command, returns combined output; injectable for tests without a real ffmpeg binary.
type Runner interface {
	Run(ctx context.Context, binPath string, args []string) ([]byte, error)
}

type realRunner struct{}

func (realRunner) Run(ctx context.Context, binPath string, args []string) ([]byte, error) {
	return exec.CommandContext(ctx, binPath, args...).CombinedOutput()
}

// Wrapper builds ffmpeg/ffprobe args from metadata.Metadata, runs via Runner.
type Wrapper struct {
	BinPath      string // ffmpeg: writing tags/cover/chapters, reading cover
	ProbeBinPath string // ffprobe: reading chapters
	Runner       Runner
}

// New returns a Wrapper for binPath/probeBinPath using the real exec.CommandContext runner.
func New(binPath, probeBinPath string) *Wrapper {
	return &Wrapper{
		BinPath:      binPath,
		ProbeBinPath: probeBinPath,
		Runner:       realRunner{},
	}
}
