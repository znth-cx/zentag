package ruleset

import (
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckBitrate(t *testing.T) {
	tests := []struct {
		name        string
		tracks      []metadata.Track
		wantViolLen int
		wantMsg     string
	}{
		{
			name: "Single track at 128kbps",
			tracks: []metadata.Track{
				{Path: "test.m4b", Bitrate: 128},
			},
			wantViolLen: 0,
		},
		{
			name: "Single track at 64kbps",
			tracks: []metadata.Track{
				{Path: "test.m4b", Bitrate: 64},
			},
			wantViolLen: 0,
		},
		{
			name: "Single track at 60kbps",
			tracks: []metadata.Track{
				{Path: "test.m4b", Bitrate: 60},
			},
			wantViolLen: 0,
		},
		{
			name: "Single track at 59kbps",
			tracks: []metadata.Track{
				{Path: "test.m4b", Bitrate: 59},
			},
			wantViolLen: 1,
			wantMsg:     "bitrate 59 kbps",
		},
		{
			name: "Single track at 32kbps",
			tracks: []metadata.Track{
				{Path: "test.m4b", Bitrate: 32},
			},
			wantViolLen: 1,
			wantMsg:     "bitrate 32 kbps",
		},
		{
			name: "Multi-file with one low bitrate track",
			tracks: []metadata.Track{
				{Path: "part1.m4b", Bitrate: 128},
				{Path: "part2.m4b", Bitrate: 32},
				{Path: "part3.m4b", Bitrate: 128},
			},
			wantViolLen: 1,
			wantMsg:     "track 2 has bitrate 32 kbps",
		},
		{
			name: "Multi-file with multiple low bitrate tracks",
			tracks: []metadata.Track{
				{Path: "part1.m4b", Bitrate: 32},
				{Path: "part2.m4b", Bitrate: 48},
				{Path: "part3.m4b", Bitrate: 128},
			},
			wantViolLen: 2,
		},
		{
			name: "Track with bitrate 0 (unset)",
			tracks: []metadata.Track{
				{Path: "test.m4b", Bitrate: 0},
			},
			wantViolLen: 0,
		},
		{
			name: "All tracks above minimum",
			tracks: []metadata.Track{
				{Path: "part1.m4b", Bitrate: 128},
				{Path: "part2.m4b", Bitrate: 256},
				{Path: "part3.m4b", Bitrate: 192},
			},
			wantViolLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &metadata.Metadata{
				Tracks: tt.tracks,
			}
			violations := CheckBitrate(meta)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckBitrate() violations count = %v, want %v", len(violations), tt.wantViolLen)
			}

			if tt.wantViolLen > 0 && violations[0].Message == "" {
				t.Errorf("CheckBitrate() expected violation message, got empty")
			}

			if tt.wantMsg != "" && len(violations) > 0 {
				if violations[0].Message != tt.wantMsg && !contains(violations[0].Message, tt.wantMsg) {
					t.Errorf("CheckBitrate() message = %v, want to contain %v", violations[0].Message, tt.wantMsg)
				}
			}

			for _, v := range violations {
				if v.Rule != "bitrate" {
					t.Errorf("CheckBitrate() expected rule 'bitrate', got %v", v.Rule)
				}
				if v.Severity != SeverityUpgradable {
					t.Errorf("CheckBitrate() expected severity 'upgradable', got %v", v.Severity)
				}
			}
		})
	}
}

func TestCheckBitrateNil(t *testing.T) {
	violations := CheckBitrate(nil)
	if violations != nil {
		t.Errorf("CheckBitrate() on nil metadata should return nil, got %v", violations)
	}
}

func TestCheckBitrateNilTracks(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: nil,
	}
	violations := CheckBitrate(meta)
	if len(violations) != 0 {
		t.Errorf("CheckBitrate() on nil tracks should return empty, got %v", violations)
	}
}
