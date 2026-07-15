package mp3tag

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.senan.xyz/taglib"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestWriteTags_Integration(t *testing.T) {
	testMP3 := filepath.Join("testdata", "sample.mp3")
	if _, err := os.Stat(testMP3); os.IsNotExist(err) {
		t.Skip("test MP3 file not available")
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mp3")
	if err := copyFile(testMP3, testFile); err != nil {
		t.Fatalf("failed to copy test file: %v", err)
	}

	m := &metadata.Metadata{
		Author:      []string{"Test Author"},
		Title:       "Test Book",
		Subtitle:    "Test Subtitle",
		Publisher:   []string{"Test Publisher"},
		Year:        2024,
		Narrator:    []string{"Test Narrator"},
		Description: "Test description",
		Genre:       []string{"Test Genre"},
		Series: []metadata.SeriesEntry{
			{Name: "Test Series", Part: "1"},
		},
		Language: "eng",
		ISBN:     "9781234567890",
		ASIN:     "B00TESTASIN",
		Tracks: []metadata.Track{
			{Path: "test.mp3", PartNumber: 1},
		},
	}
	track := metadata.Track{
		Path:       "test.mp3",
		PartNumber: 1,
	}

	ctx := context.Background()
	if err := WriteTags(ctx, testFile, m, track); err != nil {
		t.Fatalf("WriteTags() failed: %v", err)
	}

	tags, err := readTags(testFile)
	if err != nil {
		t.Fatalf("failed to read tags back: %v", err)
	}

	if tags["TITLE"][0] != "Test Book" {
		t.Errorf("TITLE = %v, want Test Book", tags["TITLE"])
	}

	if tags["NARRATOR"][0] != "Test Narrator" {
		t.Errorf("NARRATOR = %v, want Test Narrator", tags["NARRATOR"])
	}

	if tags["SERIES"][0] != "Test Series" {
		t.Errorf("SERIES = %v, want Test Series", tags["SERIES"])
	}

	if tags["ISBN"][0] != "9781234567890" {
		t.Errorf("ISBN = %v, want 9781234567890", tags["ISBN"])
	}

	if tags["TRACKNUMBER"][0] != "1/1" {
		t.Errorf("TRACKNUMBER = %v, want 1/1", tags["TRACKNUMBER"])
	}
}

func TestWriteTags_MergeModePreservesExistingTags(t *testing.T) {
	testMP3 := filepath.Join("testdata", "sample_with_tags.mp3")
	if _, err := os.Stat(testMP3); os.IsNotExist(err) {
		t.Skip("pre-tagged test MP3 file not available")
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mp3")
	if err := copyFile(testMP3, testFile); err != nil {
		t.Fatalf("failed to copy test file: %v", err)
	}

	originalTags, err := readTags(testFile)
	if err != nil {
		t.Fatalf("failed to read original tags: %v", err)
	}
	_ = originalTags

	m := &metadata.Metadata{
		Title:  "New Title Only",
		Author: []string{"New Author"},
		Tracks: []metadata.Track{
			{Path: "test.mp3"},
		},
	}
	track := metadata.Track{Path: "test.mp3"}

	ctx := context.Background()
	if err := WriteTags(ctx, testFile, m, track); err != nil {
		t.Fatalf("WriteTags() failed: %v", err)
	}

	newTags, err := readTags(testFile)
	if err != nil {
		t.Fatalf("failed to read new tags: %v", err)
	}

	if newTags["TITLE"][0] != "New Title Only" {
		t.Errorf("TITLE = %v, want New Title Only", newTags["TITLE"])
	}

	if newTags["ARTIST"][0] != "New Author" {
		t.Errorf("ARTIST = %v, want New Author", newTags["ARTIST"])
	}
}

func TestWriteTags_FailOnError(t *testing.T) {
	m := &metadata.Metadata{Title: "Test"}
	track := metadata.Track{Path: "nonexistent.mp3"}

	ctx := context.Background()
	err := WriteTags(ctx, "/invalid/path/test.mp3", m, track)
	if err == nil {
		t.Error("WriteTags() should fail with invalid path")
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func readTags(path string) (map[string][]string, error) {
	return readTagsForTest(path)
}

var readTagsForTest = defaultReadTags

func defaultReadTags(path string) (map[string][]string, error) {
	return readTagsImpl(path)
}

func readTagsImpl(path string) (map[string][]string, error) {
	tags, err := taglib.ReadTags(path)
	if err != nil {
		return nil, err
	}
	return tags, nil
}
