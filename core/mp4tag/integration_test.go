package mp4tag

import (
	"bytes"
	"context"
	"encoding/base64"
	"os/exec"
	"path/filepath"
	"testing"

	mp4 "github.com/Sorrow446/go-mp4tag"

	"github.com/znth-cx/zentag/core/metadata"
)

// requireBinary skips the test if name isn't on PATH.
func requireBinary(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not found on PATH, skipping integration test: %v", name, err)
	}
	return path
}

// tinyPNGBase64: 1x1 red PNG, base64, for an inline test fixture.
const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	if err != nil {
		t.Fatalf("decode tiny PNG fixture: %v", err)
	}
	return data
}

// TestWriteTags_Real_M4B: real go-mp4tag round-trip against a real M4B; cover stays byte-exact across a second edit with no cover set.
func TestWriteTags_Real_M4B(t *testing.T) {
	ffmpegPath := requireBinary(t, "ffmpeg")

	dir := t.TempDir()
	m4bPath := filepath.Join(dir, "test.m4b")
	runOK(t, ffmpegPath, "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:a", "aac", "-b:a", "64k", m4bPath, "-loglevel", "error")

	coverBytes := tinyPNG(t)
	meta := &metadata.Metadata{
		Author:      []string{"Robert Jordan"},
		Title:       "The Eye of the World",
		Subtitle:    "Book One",
		Publisher:   []string{"Tor"},
		Year:        1990,
		Narrator:    []string{"Michael Kramer"},
		Description: "A fantasy epic",
		Genre:       []string{"Fantasy"},
		Series:      []metadata.SeriesEntry{{Name: "Wheel of Time", Part: "1"}},
		Language:    "eng",
		ISBN:        "9780812511819",
		ASIN:        "B000TEST00",
		CoverImage:  coverBytes,
		CoverMIME:   "image/png",
	}

	if err := WriteTags(context.Background(), m4bPath, meta); err != nil {
		t.Fatalf("WriteTags() error = %v", err)
	}

	got := readBack(t, m4bPath)
	if got.Title != "The Eye of the World" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.Artist != "Robert Jordan" {
		t.Errorf("Artist = %q", got.Artist)
	}
	if got.Composer != "Michael Kramer" {
		t.Errorf("Composer = %q", got.Composer)
	}
	if got.Custom["NARRATOR"] != "Michael Kramer" && got.Custom["narrator"] != "Michael Kramer" {
		t.Errorf("Custom NARRATOR = %q", got.Custom)
	}
	if got.Custom["ASIN"] != "B000TEST00" && got.Custom["asin"] != "B000TEST00" {
		t.Errorf("Custom ASIN = %q", got.Custom)
	}
	if got.Custom["ISBN"] != "9780812511819" && got.Custom["isbn"] != "9780812511819" {
		t.Errorf("Custom ISBN = %q", got.Custom)
	}
	if len(got.Pictures) != 1 {
		t.Fatalf("Pictures = %v, want exactly 1", got.Pictures)
	}
	if !bytes.Equal(got.Pictures[0].Data, coverBytes) {
		t.Error("extracted cover does not byte-match source")
	}

	// Second edit, no cover: cover/custom fields survive (go-mp4tag merges with existing atoms); Title still applies.
	if err := WriteTags(context.Background(), m4bPath, &metadata.Metadata{Title: "New Title"}); err != nil {
		t.Fatalf("second WriteTags() error = %v", err)
	}
	got2 := readBack(t, m4bPath)
	if got2.Title != "New Title" {
		t.Errorf("Title after second edit = %q, want %q", got2.Title, "New Title")
	}
	if len(got2.Pictures) != 1 || !bytes.Equal(got2.Pictures[0].Data, coverBytes) {
		t.Error("cover did not survive second edit")
	}
}

func readBack(t *testing.T, path string) *mp4.MP4Tags {
	t.Helper()
	m, err := mp4.Open(path)
	if err != nil {
		t.Fatalf("mp4tag.Open(%q) error = %v", path, err)
	}
	defer m.Close()
	tags, err := m.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	return tags
}

func runOK(t *testing.T, bin string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\noutput:\n%s", bin, args, err, out)
	}
	return out
}
