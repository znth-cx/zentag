package ruleset

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckCoverPlacement(t *testing.T) {
	tempDir := t.TempDir()

	largeCover := make([]byte, 3*1024*1024+1)
	smallCover := make([]byte, 100)

	tests := []struct {
		name        string
		meta        *metadata.Metadata
		setup       func(string) error
		wantViolLen int
		wantMsg     string
	}{
		{
			name: "M4B with embedded cover and no loose file",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   smallCover,
				Tracks: []metadata.Track{
					{Container: "M4B"},
				},
			},
			wantViolLen: 0,
		},
		{
			name: "M4B without embedded cover",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   []byte{},
				Tracks: []metadata.Track{
					{Container: "M4B"},
				},
			},
			wantViolLen: 1,
			wantMsg:     "must have embedded cover image",
		},
		{
			name: "M4B with embedded cover and loose cover.jpg",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   smallCover,
				Tracks: []metadata.Track{
					{Container: "M4B"},
				},
			},
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "cover.jpg"), smallCover, 0644)
			},
			wantViolLen: 1,
			wantMsg:     "should not have loose cover.jpg",
		},
		{
			name: "M4B with embedded cover over 3MB",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   largeCover,
				Tracks: []metadata.Track{
					{Container: "M4B"},
				},
			},
			wantViolLen: 1,
			wantMsg:     "exceeds 3MB size limit",
		},
		{
			name: "MP3 with cover.jpg in directory",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   []byte{},
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "cover.jpg"), smallCover, 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "MP3 without cover.jpg in directory",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   []byte{},
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			wantViolLen: 1,
			wantMsg:     "cover.jpg must be present",
		},
		{
			name: "FLAC with cover.jpg in directory",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   []byte{},
				Tracks: []metadata.Track{
					{Container: "FLAC"},
				},
			},
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "cover.jpg"), smallCover, 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "FLAC without cover.jpg in directory",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   []byte{},
				Tracks: []metadata.Track{
					{Container: "FLAC"},
				},
			},
			wantViolLen: 1,
			wantMsg:     "cover.jpg must be present",
		},
		{
			name: "cover.jpg over 3MB",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				CoverImage:   []byte{},
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "cover.jpg"), largeCover, 0644)
			},
			wantViolLen: 1,
			wantMsg:     "exceeds 3MB size limit",
		},
		{
			name:        "Nil metadata",
			meta:        nil,
			wantViolLen: 0,
		},
		{
			name: "Empty tracks",
			meta: &metadata.Metadata{
				OriginalPath: tempDir,
				Tracks:       []metadata.Track{},
			},
			wantViolLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			if tt.meta != nil {
				tt.meta.OriginalPath = testDir
			}

			if tt.setup != nil {
				if err := tt.setup(testDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			ctx := context.Background()
			violations := CheckCoverPlacement(ctx, tt.meta)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckCoverPlacement() violations count = %v, want %v", len(violations), tt.wantViolLen)
			}

			if tt.wantMsg != "" && len(violations) > 0 {
				if !contains(violations[0].Message, tt.wantMsg) {
					t.Errorf("CheckCoverPlacement() message = %v, want to contain %v", violations[0].Message, tt.wantMsg)
				}
			}
		})
	}
}
