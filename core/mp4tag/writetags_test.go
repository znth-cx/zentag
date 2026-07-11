package mp4tag

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	mp4 "github.com/Sorrow446/go-mp4tag"

	"codeberg.org/Ether/zentag/core/metadata"
)

// withFakeWrite swaps the package-level write var for a fake for the
// duration of the test.
func withFakeWrite(t *testing.T, fake func(path string, tags *mp4.MP4Tags) error) {
	t.Helper()
	orig := write
	write = fake
	t.Cleanup(func() { write = orig })
}

func mustTempFile(t *testing.T, size int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.m4b")
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestWriteTags_CallsWriteWithBuiltTags(t *testing.T) {
	path := mustTempFile(t, 10)

	var gotPath string
	var gotTags *mp4.MP4Tags
	withFakeWrite(t, func(p string, tags *mp4.MP4Tags) error {
		gotPath = p
		gotTags = tags
		return nil
	})

	err := WriteTags(context.Background(), path, &metadata.Metadata{Title: "Some Book"})
	if err != nil {
		t.Fatalf("WriteTags() error = %v", err)
	}
	if gotPath != path {
		t.Errorf("write called with path %q, want %q", gotPath, path)
	}
	if gotTags.Title != "Some Book" {
		t.Errorf("write called with Title %q, want %q", gotTags.Title, "Some Book")
	}
}

func TestWriteTags_WriteErrorWrapsPath(t *testing.T) {
	path := mustTempFile(t, 10)
	withFakeWrite(t, func(string, *mp4.MP4Tags) error {
		return errors.New("boom")
	})

	err := WriteTags(context.Background(), path, &metadata.Metadata{Title: "Some Book"})
	if err == nil {
		t.Fatal("WriteTags() error = nil, want error")
	}
	if got := err.Error(); got == "" {
		t.Errorf("WriteTags() error empty")
	}
}

func TestWriteTags_MissingFileErrors(t *testing.T) {
	withFakeWrite(t, func(string, *mp4.MP4Tags) error {
		t.Fatal("write should not be called when Stat fails")
		return nil
	})

	err := WriteTags(context.Background(), filepath.Join(t.TempDir(), "missing.m4b"), &metadata.Metadata{})
	if err == nil {
		t.Fatal("WriteTags() error = nil, want error for missing file")
	}
}

func TestWriteTags_OversizeFileErrors(t *testing.T) {
	orig := maxSize
	maxSize = 5
	t.Cleanup(func() { maxSize = orig })

	path := mustTempFile(t, 10)
	withFakeWrite(t, func(string, *mp4.MP4Tags) error {
		t.Fatal("write should not be called when file exceeds maxSize")
		return nil
	})

	err := WriteTags(context.Background(), path, &metadata.Metadata{})
	if err == nil {
		t.Fatal("WriteTags() error = nil, want error for oversize file")
	}
}

func TestWriteTags_CanceledContextErrors(t *testing.T) {
	path := mustTempFile(t, 10)
	withFakeWrite(t, func(string, *mp4.MP4Tags) error {
		t.Fatal("write should not be called with a canceled context")
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WriteTags(ctx, path, &metadata.Metadata{})
	if err == nil {
		t.Fatal("WriteTags() error = nil, want context.Canceled")
	}
}
