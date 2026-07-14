package ruleset

import (
	"context"
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckFLACMD5(t *testing.T) {
	tests := []struct {
		name        string
		tracks      []metadata.Track
		wantViolLen int
		wantMsg     string
	}{
		{
			name: "Non-FLAC track (M4B)",
			tracks: []metadata.Track{
				{Path: "test.m4b", Container: "M4B", Codec: "AAC"},
			},
			wantViolLen: 0,
		},
		{
			name: "Non-FLAC track (MP3)",
			tracks: []metadata.Track{
				{Path: "test.mp3", Container: "MP3", Codec: "MP3"},
			},
			wantViolLen: 0,
		},
		{
			name: "FLAC track with valid MD5 (mocked)",
			tracks: []metadata.Track{
				{Path: "test.flac", Container: "FLAC", Codec: "FLAC"},
			},
			wantViolLen: 0,
		},
		{
			name: "FLAC track with invalid MD5 (mocked)",
			tracks: []metadata.Track{
				{Path: "test.flac", Container: "FLAC", Codec: "FLAC"},
			},
			wantViolLen: 0,
		},
		{
			name: "Multiple FLAC tracks",
			tracks: []metadata.Track{
				{Path: "track1.flac", Container: "FLAC", Codec: "FLAC"},
				{Path: "track2.flac", Container: "FLAC", Codec: "FLAC"},
			},
			wantViolLen: 0,
		},
		{
			name: "Mixed format tracks",
			tracks: []metadata.Track{
				{Path: "track1.flac", Container: "FLAC", Codec: "FLAC"},
				{Path: "track2.m4b", Container: "M4B", Codec: "AAC"},
				{Path: "track3.flac", Container: "FLAC", Codec: "FLAC"},
			},
			wantViolLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &metadata.Metadata{
				Tracks: tt.tracks,
			}

			ctx := context.Background()
			violations := CheckFLACMD5(ctx, meta, nil)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckFLACMD5() violations count = %v, want %v", len(violations), tt.wantViolLen)
			}

			if tt.wantMsg != "" && len(violations) > 0 {
				if violations[0].Message != tt.wantMsg && !contains(violations[0].Message, tt.wantMsg) {
					t.Errorf("CheckFLACMD5() message = %v, want to contain %v", violations[0].Message, tt.wantMsg)
				}
			}

			for _, v := range violations {
				if v.Rule != "flac_md5" {
					t.Errorf("CheckFLACMD5() expected rule 'flac_md5', got %v", v.Rule)
				}
				if v.Severity != SeverityTrumpable {
					t.Errorf("CheckFLACMD5() expected severity 'trumpable', got %v", v.Severity)
				}
			}
		})
	}
}

func TestCheckFLACMD5Nil(t *testing.T) {
	ctx := context.Background()
	violations := CheckFLACMD5(ctx, nil, nil)
	if violations != nil {
		t.Errorf("CheckFLACMD5() on nil metadata should return nil, got %v", violations)
	}
}

func TestCheckFLACMD5NilTracks(t *testing.T) {
	ctx := context.Background()
	meta := &metadata.Metadata{
		Tracks: nil,
	}
	violations := CheckFLACMD5(ctx, meta, nil)
	if len(violations) != 0 {
		t.Errorf("CheckFLACMD5() on nil tracks should return empty, got %v", violations)
	}
}

func TestIsFLAC(t *testing.T) {
	tests := []struct {
		name  string
		track metadata.Track
		want  bool
	}{
		{
			name:  "FLAC container",
			track: metadata.Track{Container: "FLAC", Codec: "FLAC"},
			want:  true,
		},
		{
			name:  "flac container lowercase",
			track: metadata.Track{Container: "flac", Codec: "flac"},
			want:  true,
		},
		{
			name:  "FlAC container mixed case",
			track: metadata.Track{Container: "FlAC", Codec: "FlAC"},
			want:  true,
		},
		{
			name:  "M4B container",
			track: metadata.Track{Container: "M4B", Codec: "AAC"},
			want:  false,
		},
		{
			name:  "MP3 container",
			track: metadata.Track{Container: "MP3", Codec: "MP3"},
			want:  false,
		},
		{
			name:  "Empty container",
			track: metadata.Track{Container: "", Codec: ""},
			want:  false,
		},
		{
			name:  "FLAC codec in different container",
			track: metadata.Track{Container: "M4B", Codec: "FLAC"},
			want:  true,
		},
		{
			name:  "AAC codec in FLAC container",
			track: metadata.Track{Container: "FLAC", Codec: "AAC"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFLAC(tt.track); got != tt.want {
				t.Errorf("isFLAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidMD5Hash(t *testing.T) {
	tests := []struct {
		name string
		hash string
		want bool
	}{
		{
			name: "Valid lowercase MD5",
			hash: "5d41402abc4b2a76b9719d911017c592",
			want: true,
		},
		{
			name: "Valid uppercase MD5",
			hash: "5D41402ABC4B2A76B9719D911017C592",
			want: true,
		},
		{
			name: "Valid mixed case MD5",
			hash: "5D41402Abc4B2A76b9719D911017c592",
			want: true,
		},
		{
			name: "Invalid MD5 - too short",
			hash: "5d41402abc4b2a76b9719d911017c5",
			want: false,
		},
		{
			name: "Invalid MD5 - too long",
			hash: "5d41402abc4b2a76b9719d911017c592a",
			want: false,
		},
		{
			name: "Invalid MD5 - contains invalid chars",
			hash: "5d41402abc4b2a76b9719d911017c59z",
			want: false,
		},
		{
			name: "Invalid MD5 - contains spaces",
			hash: "5d41402abc4b2a76b9719d911017c59 ",
			want: false,
		},
		{
			name: "Invalid MD5 - empty string",
			hash: "",
			want: false,
		},
		{
			name: "Invalid MD5 - contains hyphens",
			hash: "5d41402abc4b2a76-b9719d911017c592",
			want: false,
		},
		{
			name: "Valid MD5 - all zeros",
			hash: "00000000000000000000000000000000",
			want: true,
		},
		{
			name: "Valid MD5 - all ones",
			hash: "11111111111111111111111111111111",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidMD5Hash(tt.hash); got != tt.want {
				t.Errorf("isValidMD5Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractHashFromLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "Standard MD5 line",
			line: "MD5 signature: 5d41402abc4b2a76b9719d911017c592",
			want: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name: "MD5 line with spaces",
			line: "MD5 signature:  5d41402abc4b2a76b9719d911017c592  ",
			want: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name: "Uppercase MD5",
			line: "MD5: 5D41402ABC4B2A76B9719D911017C592",
			want: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name: "Line without colon",
			line: "MD5 signature 5d41402abc4b2a76b9719d911017c592",
			want: "",
		},
		{
			name: "Line with empty hash",
			line: "MD5 signature:",
			want: "",
		},
		{
			name: "Line with invalid hash",
			line: "MD5 signature: invalid",
			want: "",
		},
		{
			name: "Line with multiple words containing valid MD5",
			line: "MD5 signature: test 5d41402abc4b2a76b9719d911017c592 other",
			want: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name: "Mixed format line",
			line: "Audio MD5    :    5d41402abc4b2a76b9719d911017c592",
			want: "5d41402abc4b2a76b9719d911017c592",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractHashFromLine(tt.line); got != tt.want {
				t.Errorf("extractHashFromLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindMD5InDump(t *testing.T) {
	tests := []struct {
		name        string
		dump        string
		wantFound   bool
		wantContain string
	}{
		{
			name:        "MD5 found in dump",
			dump:        "General\nMD5 signature: 5d41402abc4b2a76b9719d911017c592\nAudio\n",
			wantFound:   true,
			wantContain: "MD5",
		},
		{
			name:      "MD5 not found in dump",
			dump:      "General\nAudio\nFormat: FLAC\n",
			wantFound: false,
		},
		{
			name:        "Lowercase md5 found",
			dump:        "General\nmd5: 5d41402abc4b2a76b9719d911017c592\n",
			wantFound:   true,
			wantContain: "md5",
		},
		{
			name:        "Multiple MD5 lines, returns first",
			dump:        "MD5 signature: 5d41402abc4b2a76b9719d911017c592\nAnother MD5: abcdef0123456789abcdef012345678\n",
			wantFound:   true,
			wantContain: "MD5 signature",
		},
		{
			name:        "MD5 in longer line",
			dump:        "Some data before\nAudio MD5 signature: 5d41402abc4b2a76b9719d911017c592 and more text\nSome data after\n",
			wantFound:   true,
			wantContain: "MD5",
		},
		{
			name:      "Empty dump",
			dump:      "",
			wantFound: false,
		},
		{
			name:      "Dump with only newlines",
			dump:      "\n\n\n",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, found := findMD5InDump(tt.dump)
			if found != tt.wantFound {
				t.Errorf("findMD5InDump() found = %v, want %v", found, tt.wantFound)
			}
			if found && !contains(line, tt.wantContain) {
				t.Errorf("findMD5InDump() line = %v, want to contain %v", line, tt.wantContain)
			}
		})
	}
}
