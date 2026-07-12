package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/znth-cx/zentag/core/ebookmeta"
)

type fakeEbookRunner struct {
	opf    string
	writes [][]string
}

func (f *fakeEbookRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	for _, a := range args {
		if a == "--version" {
			return []byte("ebook-meta (calibre 9.11.0)"), nil
		}
		if strings.HasPrefix(a, "--to-opf=") {
			return nil, os.WriteFile(strings.TrimPrefix(a, "--to-opf="), []byte(f.opf), 0o644)
		}
	}
	f.writes = append(f.writes, args)
	return nil, nil
}

func swapEbookWrapper(t *testing.T, r ebookmeta.Runner) {
	t.Helper()
	old := newEbookmetaWrapper
	newEbookmetaWrapper = func(binPath string) *ebookmeta.Wrapper { return &ebookmeta.Wrapper{BinPath: binPath, Runner: r} }
	oldTTY := stdinIsInteractive
	stdinIsInteractive = func() bool { return false }
	t.Cleanup(func() { newEbookmetaWrapper = old; stdinIsInteractive = oldTTY })
}

func TestEbookCmd_HappyPath(t *testing.T) {
	setTestConfig(t)
	out := t.TempDir()
	cfg.OutputDir = out
	swapEbookWrapper(t, &fakeEbookRunner{opf: `<?xml version="1.0"?><package><metadata></metadata></package>`})

	src := filepath.Join(t.TempDir(), "in.epub")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o644))

	resetEbookFlags(t)
	require.NoError(t, ebookCmd.Flags().Set("author", "Patrick Rothfuss"))
	require.NoError(t, ebookCmd.Flags().Set("title", "The Name of the Wind"))
	require.NoError(t, ebookCmd.Flags().Set("year", "2007"))
	require.NoError(t, ebookCmd.Flags().Set("isbn", "9780756404741"))
	require.NoError(t, ebookCmd.Flags().Set("language", "eng"))

	var buf bytes.Buffer
	ebookCmd.SetContext(context.Background())
	ebookCmd.SetOut(&buf)
	require.NoError(t, ebookCmd.RunE(ebookCmd, []string{src}))

	folder := "Patrick Rothfuss - The Name of the Wind [ENG EPUB]"
	file := "Patrick Rothfuss - The Name of the Wind (2007) [ENG EPUB 9780756404741]"
	dir := filepath.Join(out, folder)
	assert.DirExists(t, dir)
	assert.FileExists(t, filepath.Join(dir, file+".epub"))
}

// User flag beats file metadata on conflict. Regression: discarded Merge
// conflicts left conflicted fields zeroed.
func TestEbookCmd_UserFlagsWinConflict(t *testing.T) {
	setTestConfig(t)
	out := t.TempDir()
	cfg.OutputDir = out
	opf := `<?xml version="1.0"?><package><metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">` +
		`<dc:title>File Title</dc:title>` +
		`<meta name="calibre:series" content="File Series"/><meta name="calibre:series_index" content="9"/>` +
		`</metadata></package>`
	runner := &fakeEbookRunner{opf: opf}
	swapEbookWrapper(t, runner)

	src := filepath.Join(t.TempDir(), "in.epub")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o644))

	resetEbookFlags(t)
	require.NoError(t, ebookCmd.Flags().Set("author", "Test Author"))
	require.NoError(t, ebookCmd.Flags().Set("title", "User Title"))
	require.NoError(t, ebookCmd.Flags().Set("year", "2020"))
	require.NoError(t, ebookCmd.Flags().Set("isbn", "9780756404741"))
	require.NoError(t, ebookCmd.Flags().Set("language", "eng"))
	require.NoError(t, ebookCmd.Flags().Set("series", "User Series"))
	require.NoError(t, ebookCmd.Flags().Set("series-part", "1"))

	var buf bytes.Buffer
	ebookCmd.SetContext(context.Background())
	ebookCmd.SetOut(&buf)
	require.NoError(t, ebookCmd.RunE(ebookCmd, []string{src}))

	folder := "Test Author - User Series - User Title [ENG EPUB]"
	file := "Test Author - User Series #1 - User Title (2020) [ENG EPUB 9780756404741]"
	dir := filepath.Join(out, folder)
	assert.DirExists(t, dir)
	assert.FileExists(t, filepath.Join(dir, file+".epub"))

	var titleArgs, seriesArgs []string
	for _, w := range runner.writes {
		for i, a := range w {
			if a == "--title" && i+1 < len(w) {
				titleArgs = append(titleArgs, w[i+1])
			}
			if a == "--series" && i+1 < len(w) {
				seriesArgs = append(seriesArgs, w[i+1])
			}
		}
	}
	assert.Contains(t, titleArgs, "User Title")
	assert.NotContains(t, titleArgs, "File Title")
	assert.Contains(t, seriesArgs, "User Series")
	assert.NotContains(t, seriesArgs, "File Series")
}

type badVersionRunner struct{}

func (badVersionRunner) Run(_ context.Context, _ string, _ []string) ([]byte, error) {
	return []byte("not calibre"), nil
}

func TestEbookCmd_Errors(t *testing.T) {
	cases := []struct {
		name, ext, wantErr string
		runner             ebookmeta.Runner
	}{
		{"missing required fields", "epub", "missing required fields", &fakeEbookRunner{opf: `<package><metadata></metadata></package>`}},
		{"unsupported format", "txt", "unsupported ebook format", &fakeEbookRunner{opf: `<package><metadata></metadata></package>`}},
		{"bad ebook-meta binary", "epub", "install calibre", badVersionRunner{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setTestConfig(t)
			cfg.OutputDir = t.TempDir()
			swapEbookWrapper(t, tc.runner)
			src := filepath.Join(t.TempDir(), "in."+tc.ext)
			require.NoError(t, os.WriteFile(src, []byte("x"), 0o644))
			resetEbookFlags(t)
			ebookCmd.SetContext(context.Background())
			err := ebookCmd.RunE(ebookCmd, []string{src})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// resetEbookFlags restores every ebook flag to its default between tests
// (cobra flag state is package-global on the shared command).
func resetEbookFlags(t *testing.T) {
	t.Helper()
	for _, n := range []string{"author", "title", "year", "isbn", "series", "series-part", "edition", "retail", "publisher", "language", "description", "tags", "asin"} {
		f := ebookCmd.Flags().Lookup(n)
		require.NoError(t, ebookCmd.Flags().Set(n, f.DefValue))
		f.Changed = false
	}
}
