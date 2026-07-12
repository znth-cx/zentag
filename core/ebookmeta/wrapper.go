// Package ebookmeta wraps calibre's ebook-meta CLI.
package ebookmeta

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, binPath string, args []string) ([]byte, error)
}

type realRunner struct{}

func (realRunner) Run(ctx context.Context, binPath string, args []string) ([]byte, error) {
	return exec.CommandContext(ctx, binPath, args...).CombinedOutput()
}

// Wrapper runs the ebook-meta binary at BinPath.
type Wrapper struct {
	BinPath string
	Runner  Runner
}

func New(binPath string) *Wrapper {
	return &Wrapper{BinPath: binPath, Runner: realRunner{}}
}

var versionRE = regexp.MustCompile(`calibre \d+\.\d+`)

// Validate confirms BinPath is calibre's ebook-meta.
func (w *Wrapper) Validate(ctx context.Context) error {
	out, err := w.Runner.Run(ctx, w.BinPath, []string{"--version"})
	if err != nil {
		return fmt.Errorf("ebook-meta %q could not be run: %w", w.BinPath, err)
	}
	if !versionRE.Match(out) {
		return fmt.Errorf("binary is not calibre's ebook-meta (got: %s)", strings.TrimSpace(string(out)))
	}
	return nil
}

// Read extracts metadata via temp OPF file.
func (w *Wrapper) Read(ctx context.Context, file string) (*OPFMetadata, error) {
	tmp, err := os.CreateTemp("", "zentag-opf-*.opf")
	if err != nil {
		return nil, fmt.Errorf("create temp opf: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	out, err := w.Runner.Run(ctx, w.BinPath, []string{file, "--to-opf=" + tmpPath})
	if err != nil {
		return nil, fmt.Errorf("ebook-meta read %q: %w: %s", file, err, strings.TrimSpace(string(out)))
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("read generated opf: %w", err)
	}
	return parseOPF(data)
}

// Write sets metadata on file in place; args are ebook-meta flags.
func (w *Wrapper) Write(ctx context.Context, file string, args []string) error {
	out, err := w.Runner.Run(ctx, w.BinPath, append([]string{file}, args...))
	if err != nil {
		return fmt.Errorf("ebook-meta write %q: %w: %s", file, err, strings.TrimSpace(string(out)))
	}
	return nil
}
