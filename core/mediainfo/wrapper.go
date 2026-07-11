// Package mediainfo wraps mediainfo CLI for technical file info.
package mediainfo

import (
	"context"
	"os/exec"
)

// Runner executes external commands.
type Runner interface {
	Run(ctx context.Context, binPath string, args []string) ([]byte, error)
}

type realRunner struct{}

func (realRunner) Run(ctx context.Context, binPath string, args []string) ([]byte, error) {
	return exec.CommandContext(ctx, binPath, args...).CombinedOutput()
}

// Wrapper runs mediainfo and parses its JSON output.
type Wrapper struct {
	BinPath string
	Runner  Runner
}

// New returns a Wrapper for the mediainfo binary at binPath.
func New(binPath string) *Wrapper {
	return &Wrapper{
		BinPath: binPath,
		Runner:  realRunner{},
	}
}
