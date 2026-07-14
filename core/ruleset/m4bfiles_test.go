package ruleset

import (
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckM4BSingleFile(t *testing.T) {
	tests := []struct {
		name        string
		meta        *metadata.Metadata
		wantViolLen int
		wantMsg     string
	}{
		{
			name: "Single M4B file",
			meta: &metadata.Metadata{
				Tracks: []metadata.Track{
					{
						Path:      "test/audiobook.m4b",
						Container: "M4B",
					},
				},
			},
			wantViolLen: 0,
		},
		{
			name: "Multiple M4B files with proper disc naming",
			meta: &metadata.Metadata{
				Tracks: []metadata.Track{
					{
						Path:      "test/Disc 1.m4b",
						Container: "M4B",
					},
					{
						Path:      "test/Disc 2.m4b",
						Container: "M4B",
					},
				},
			},
			wantViolLen: 0,
		},
		{
			name: "Multiple M4B files with lowercase disc naming",
			meta: &metadata.Metadata{
				Tracks: []metadata.Track{
					{
						Path:      "test/disc 1.m4b",
						Container: "M4B",
					},
					{
						Path:      "test/disc 2.m4b",
						Container: "M4B",
					},
				},
			},
			wantViolLen: 0,
		},
		{
			name: "Multiple M4B files with inconsistent disc naming",
			meta: &metadata.Metadata{
				Tracks: []metadata.Track{
					{
						Path:      "test/Disc 1.m4b",
						Container: "M4B",
					},
					{
						Path:      "test/Part 2.m4b",
						Container: "M4B",
					},
				},
			},
			wantViolLen: 1,
			wantMsg:     "should be single files or disc releases",
		},
		{
			name: "Multiple M4B files without disc naming",
			meta: &metadata.Metadata{
				Tracks: []metadata.Track{
					{
						Path:      "test/Part 1.m4b",
						Container: "M4B",
					},
					{
						Path:      "test/Part 2.m4b",
						Container: "M4B",
					},
				},
			},
			wantViolLen: 1,
			wantMsg:     "should be single files or disc releases",
		},
		{
			name: "MP3 container should be skipped",
			meta: &metadata.Metadata{
				Tracks: []metadata.Track{
					{
						Path:      "test/Part 1.mp3",
						Container: "MP3",
					},
					{
						Path:      "test/Part 2.mp3",
						Container: "MP3",
					},
				},
			},
			wantViolLen: 0,
		},
		{
			name: "FLAC container should be skipped",
			meta: &metadata.Metadata{
				Tracks: []metadata.Track{
					{
						Path:      "test/Part 1.flac",
						Container: "FLAC",
					},
					{
						Path:      "test/Part 2.flac",
						Container: "FLAC",
					},
				},
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
				Tracks: []metadata.Track{},
			},
			wantViolLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := CheckM4BSingleFile(tt.meta)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckM4BSingleFile() violations count = %v, want %v", len(violations), tt.wantViolLen)
			}

			if tt.wantViolLen > 0 && violations[0].Message == "" {
				t.Errorf("CheckM4BSingleFile() expected violation message, got empty")
			}

			if tt.wantMsg != "" && len(violations) > 0 {
				if !contains(violations[0].Message, tt.wantMsg) {
					t.Errorf("CheckM4BSingleFile() message = %v, want to contain %v", violations[0].Message, tt.wantMsg)
				}
			}
		})
	}
}
