package ruleset

import (
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckLossyContainer(t *testing.T) {
	tests := []struct {
		name        string
		tracks      []metadata.Track
		wantViolLen int
		wantMsg     string
	}{
		{
			name: "M4B with AAC codec",
			tracks: []metadata.Track{
				{Path: "test.m4b", Container: "M4B", Codec: "AAC"},
			},
			wantViolLen: 0,
		},
		{
			name: "m4b with aac codec (lowercase)",
			tracks: []metadata.Track{
				{Path: "test.m4b", Container: "m4b", Codec: "aac"},
			},
			wantViolLen: 0,
		},
		{
			name: "M4B with aac codec (mixed case)",
			tracks: []metadata.Track{
				{Path: "test.m4b", Container: "M4b", Codec: "Aac"},
			},
			wantViolLen: 0,
		},
		{
			name: "MP3 container",
			tracks: []metadata.Track{
				{Path: "test.mp3", Container: "MP3", Codec: "MP3"},
			},
			wantViolLen: 1,
			wantMsg:     "container \"MP3\" with codec \"MP3\" should be M4B",
		},
		{
			name: "M4A with AAC codec",
			tracks: []metadata.Track{
				{Path: "test.m4a", Container: "M4A", Codec: "AAC"},
			},
			wantViolLen: 1,
			wantMsg:     "container \"M4A\" with codec \"AAC\" should be M4B",
		},
		{
			name: "FLAC container",
			tracks: []metadata.Track{
				{Path: "test.flac", Container: "FLAC", Codec: "FLAC"},
			},
			wantViolLen: 0,
		},
		{
			name: "M4B with OPUS codec",
			tracks: []metadata.Track{
				{Path: "test.m4b", Container: "M4B", Codec: "OPUS"},
			},
			wantViolLen: 0,
		},
		{
			name: "MP3 with OPUS codec",
			tracks: []metadata.Track{
				{Path: "test.mp3", Container: "MP3", Codec: "OPUS"},
			},
			wantViolLen: 1,
			wantMsg:     "container \"MP3\" with codec \"OPUS\" should be M4B",
		},
		{
			name: "Multi-file with mixed containers",
			tracks: []metadata.Track{
				{Path: "part1.m4b", Container: "M4B", Codec: "AAC"},
				{Path: "part2.mp3", Container: "MP3", Codec: "MP3"},
				{Path: "part3.m4b", Container: "M4B", Codec: "AAC"},
			},
			wantViolLen: 1,
		},
		{
			name: "Multi-file with multiple invalid containers",
			tracks: []metadata.Track{
				{Path: "part1.mp3", Container: "MP3", Codec: "MP3"},
				{Path: "part2.m4a", Container: "M4A", Codec: "AAC"},
			},
			wantViolLen: 2,
		},
		{
			name: "ALAC codec (lossless)",
			tracks: []metadata.Track{
				{Path: "test.m4a", Container: "M4A", Codec: "ALAC"},
			},
			wantViolLen: 0,
		},
		{
			name: "Empty container and codec",
			tracks: []metadata.Track{
				{Path: "test.unknown", Container: "", Codec: ""},
			},
			wantViolLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &metadata.Metadata{
				Tracks: tt.tracks,
			}
			violations := CheckLossyContainer(meta)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckLossyContainer() violations count = %v, want %v", len(violations), tt.wantViolLen)
			}

			if tt.wantViolLen > 0 && violations[0].Message == "" {
				t.Errorf("CheckLossyContainer() expected violation message, got empty")
			}

			if tt.wantMsg != "" && len(violations) > 0 {
				if violations[0].Message != tt.wantMsg && !contains(violations[0].Message, tt.wantMsg) {
					t.Errorf("CheckLossyContainer() message = %v, want to contain %v", violations[0].Message, tt.wantMsg)
				}
			}

			for _, v := range violations {
				if v.Rule != "lossy_container" {
					t.Errorf("CheckLossyContainer() expected rule 'lossy_container', got %v", v.Rule)
				}
				if v.Severity != SeverityUpgradable {
					t.Errorf("CheckLossyContainer() expected severity 'upgradable', got %v", v.Severity)
				}
			}
		})
	}
}

func TestCheckLossyContainerNil(t *testing.T) {
	violations := CheckLossyContainer(nil)
	if violations != nil {
		t.Errorf("CheckLossyContainer() on nil metadata should return nil, got %v", violations)
	}
}

func TestCheckLossyContainerNilTracks(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: nil,
	}
	violations := CheckLossyContainer(meta)
	if len(violations) != 0 {
		t.Errorf("CheckLossyContainer() on nil tracks should return empty, got %v", violations)
	}
}

func TestCheckLossyContainerEmptyTracks(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{},
	}
	violations := CheckLossyContainer(meta)
	if len(violations) != 0 {
		t.Errorf("CheckLossyContainer() on empty tracks should return empty, got %v", violations)
	}
}
