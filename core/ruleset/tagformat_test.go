package ruleset

import (
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckTagSeparators(t *testing.T) {
	tests := []struct {
		name        string
		container   string
		author      []string
		narrator    []string
		genre       []string
		publisher   []string
		series      []metadata.SeriesEntry
		wantViolLen int
		wantMsg     string
	}{
		{
			name:        "Valid single value",
			container:   "M4B",
			author:      []string{"Author Name"},
			narrator:    []string{"Narrator Name"},
			genre:       []string{"Fiction"},
			publisher:   []string{"Publisher"},
			series:      []metadata.SeriesEntry{{Name: "Series Name", Part: "1"}},
			wantViolLen: 0,
		},
		{
			name:        "Unescaped semicolon in single value",
			container:   "M4B",
			author:      []string{"Author;One"},
			wantViolLen: 1,
			wantMsg:     "unescaped semicolon",
		},
		{
			name:        "Unescaped backslash",
			container:   "M4B",
			author:      []string{"Author\\Test"},
			wantViolLen: 1,
			wantMsg:     "unescaped backslash",
		},
		{
			name:        "Backslash at end of string",
			container:   "M4B",
			author:      []string{"Author\\"},
			wantViolLen: 1,
			wantMsg:     "backslash at end",
		},
		{
			name:        "Multiple violations in one field",
			container:   "M4B",
			author:      []string{"Author\\Test;Another"},
			wantViolLen: 2,
		},
		{
			name:        "Violations in multiple fields",
			container:   "M4B",
			author:      []string{"Author\\Test"},
			narrator:    []string{"Narrator;One"},
			genre:       []string{"Fiction\\Adventure"},
			wantViolLen: 3,
		},
		{
			name:        "Unknown container type",
			container:   "UNKNOWN",
			author:      []string{"Author\\Test"},
			wantViolLen: 0,
		},
		{
			name:        "Empty multi-value field",
			container:   "M4B",
			author:      []string{""},
			wantViolLen: 0,
		},
		{
			name:        "Empty slices are fine",
			container:   "M4B",
			author:      []string{},
			narrator:    []string{},
			genre:       []string{},
			publisher:   []string{},
			wantViolLen: 0,
		},
		{
			name:        "Valid MP3 format",
			container:   "MP3",
			author:      []string{"Author One;Author Two"},
			wantViolLen: 1,
			wantMsg:     "unescaped semicolon",
		},
		{
			name:        "Valid FLAC format",
			container:   "FLAC",
			author:      []string{"Author One;Author Two"},
			wantViolLen: 1,
			wantMsg:     "unescaped semicolon",
		},
		{
			name:        "Multiple array elements with violations",
			container:   "M4B",
			author:      []string{"Author One", "Author;Two", "Author\\Three"},
			wantViolLen: 2,
		},
		{
			name:        "Series field with violations",
			container:   "M4B",
			series:      []metadata.SeriesEntry{{Name: "Series;One", Part: "1"}},
			wantViolLen: 1,
			wantMsg:     "unescaped semicolon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &metadata.Metadata{
				Tracks: []metadata.Track{
					{Container: tt.container},
				},
				Author:    tt.author,
				Narrator:  tt.narrator,
				Genre:     tt.genre,
				Publisher: tt.publisher,
				Series:    tt.series,
			}

			violations := CheckTagSeparators(meta)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckTagSeparators() violations count = %v, want %v", len(violations), tt.wantViolLen)
				for i, v := range violations {
					t.Logf("  Violation %d: %s", i, v.Message)
				}
			}

			if tt.wantMsg != "" && len(violations) > 0 {
				found := false
				for _, v := range violations {
					if contains(v.Message, tt.wantMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("CheckTagSeparators() expected violation message to contain %q, got violations:", tt.wantMsg)
					for i, v := range violations {
						t.Logf("  Violation %d: %s", i, v.Message)
					}
				}
			}
		})
	}
}

func TestCheckTagSeparatorsNil(t *testing.T) {
	violations := CheckTagSeparators(nil)
	if violations != nil {
		t.Errorf("CheckTagSeparators() on nil metadata should return nil, got %v", violations)
	}
}

func TestCheckTagSeparatorsNoTracks(t *testing.T) {
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{},
		Author: []string{"Author\\Test"},
	}
	violations := CheckTagSeparators(meta)
	if violations != nil {
		t.Errorf("CheckTagSeparators() on metadata with no tracks should return nil, got %v", violations)
	}
}

func TestParseMultiFieldTag(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "Single value",
			value:    "Author Name",
			expected: []string{"Author Name"},
		},
		{
			name:     "Multiple values",
			value:    "Author One;Author Two",
			expected: []string{"Author One", "Author Two"},
		},
		{
			name:     "Value with escaped semicolon",
			value:    "Author\\;With;Semicolon",
			expected: []string{"Author;With", "Semicolon"},
		},
		{
			name:     "Value with escaped backslash",
			value:    "Author\\\\Backslash",
			expected: []string{"Author\\Backslash"},
		},
		{
			name:     "Multiple values with escapes",
			value:    "Author\\;One;Author\\\\Two",
			expected: []string{"Author;One", "Author\\Two"},
		},
		{
			name:     "Empty string",
			value:    "",
			expected: []string{},
		},
		{
			name:     "Trailing semicolon",
			value:    "Author One;",
			expected: []string{"Author One", ""},
		},
		{
			name:     "Leading semicolon",
			value:    ";Author Two",
			expected: []string{"", "Author Two"},
		},
		{
			name:     "Consecutive semicolons",
			value:    "Author One;;Author Two",
			expected: []string{"Author One", "", "Author Two"},
		},
		{
			name:     "Only semicolon",
			value:    ";",
			expected: []string{"", ""},
		},
		{
			name:     "Escaped backslash before semicolon",
			value:    "Author\\\\;Author Two",
			expected: []string{"Author\\", "Author Two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMultiFieldTag(tt.value)

			if len(result) != len(tt.expected) {
				t.Errorf("parseMultiFieldTag() result length = %v, want %v", len(result), len(tt.expected))
				t.Logf("  Got:      %v", result)
				t.Logf("  Expected: %v", tt.expected)
				return
			}

			for i, val := range result {
				if val != tt.expected[i] {
					t.Errorf("parseMultiFieldTag() result[%d] = %v, want %v", i, val, tt.expected[i])
				}
			}
		})
	}
}
