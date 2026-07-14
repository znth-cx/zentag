package ruleset

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckExtraFiles(t *testing.T) {
	tests := []struct {
		name        string
		meta        *metadata.Metadata
		setup       func(string) error
		wantViolLen int
		wantMsg     string
	}{
		{
			name: "M4B directory with single .m4b file",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "M4B"},
				},
			},
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "audiobook.m4b"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "M4B directory with extra .txt file",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "M4B"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "audiobook.m4b"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "description.txt"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "M4B directory with extra .jpg (not cover.jpg)",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "M4B"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "audiobook.m4b"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "back.jpg"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "M4B directory with extra .m4a file",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "M4B"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "audiobook.m4b"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "preview.m4a"), []byte("test"), 0644)
			},
			wantViolLen: 1,
			wantMsg:     "unexpected file",
		},
		{
			name: "MP3 directory with multiple .mp3 files and cover.jpg",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "02.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "cover.jpg"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "MP3 directory with extra .m4b file",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "bonus.m4b"), []byte("test"), 0644)
			},
			wantViolLen: 1,
			wantMsg:     "unexpected file",
		},
		{
			name: "FLAC directory with multiple .flac files and cover.jpg",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "FLAC"},
					{Container: "FLAC"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.flac"), []byte("test"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "02.flac"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "cover.jpg"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "Directory with hidden files",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, ".hidden.txt"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "Directory with session file",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "session.json"), []byte("test"), 0644)
			},
			wantViolLen: 1,
			wantMsg:     "unexpected file",
		},
		{
			name: "Directory with .nfo file",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "info.nfo"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "Directory with metadata.json",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "Directory with uppercase METADATA.JSON",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "METADATA.JSON"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "Directory with .m3u playlist",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "playlist.m3u"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name: "Directory with .m3u8 playlist",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "MP3"},
				},
			},
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "01.mp3"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "playlist.m3u8"), []byte("test"), 0644)
			},
			wantViolLen: 0,
		},
		{
			name:        "Nil metadata",
			meta:        nil,
			wantViolLen: 0,
		},
		{
			name: "Empty tracks",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks:       []metadata.Track{},
			},
			wantViolLen: 0,
		},
		{
			name: "Unknown container",
			meta: &metadata.Metadata{
				OriginalPath: "",
				Tracks: []metadata.Track{
					{Container: "UNKNOWN"},
				},
			},
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "file.mp3"), []byte("test"), 0644)
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
			violations := CheckExtraFiles(ctx, tt.meta)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckExtraFiles() violations count = %v, want %v", len(violations), tt.wantViolLen)
			}

			if tt.wantMsg != "" && len(violations) > 0 {
				if !contains(violations[0].Message, tt.wantMsg) {
					t.Errorf("CheckExtraFiles() message = %v, want to contain %v", violations[0].Message, tt.wantMsg)
				}
			}
		})
	}
}
