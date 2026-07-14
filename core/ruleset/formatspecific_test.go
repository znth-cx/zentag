package ruleset

import (
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckFormatSpecificTags(t *testing.T) {
	tests := []struct {
		name        string
		container   string
		setupMeta   func(*metadata.Metadata)
		wantViolLen int
		wantRule    string
	}{
		{
			name:      "M4B with all required tags",
			container: "M4B",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = "1234567890123"
				m.CoverImage = []byte{0x01}
			},
			wantViolLen: 0,
		},
		{
			name:      "M4B missing .ART (author)",
			container: "M4B",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = "1234567890123"
				m.CoverImage = []byte{0x01}
			},
			wantViolLen: 1,
			wantRule:    "format_specific_tags",
		},
		{
			name:      "M4B missing .wrt (narrator)",
			container: "M4B",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{}
				m.Language = "eng"
				m.ISBN = "1234567890123"
				m.CoverImage = []byte{0x01}
			},
			wantViolLen: 1,
			wantRule:    "format_specific_tags",
		},
		{
			name:      "M4B with SERIES but missing SERIES-PART",
			container: "M4B",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = "1234567890123"
				m.CoverImage = []byte{0x01}
				m.Series = []metadata.SeriesEntry{{Name: "Series Name", Part: ""}}
			},
			wantViolLen: 1,
			wantRule:    "format_specific_tags",
		},
		{
			name:      "M4B with neither ISBN nor ASIN",
			container: "M4B",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = ""
				m.ASIN = ""
				m.CoverImage = []byte{0x01}
			},
			wantViolLen: 1,
			wantRule:    "format_specific_tags",
		},
		{
			name:      "M4B without embedded cover",
			container: "M4B",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = "1234567890123"
				m.CoverImage = []byte{}
			},
			wantViolLen: 1,
			wantRule:    "format_specific_tags",
		},
		{
			name:      "MP3 with all required tags",
			container: "MP3",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = "1234567890123"
			},
			wantViolLen: 0,
		},
		{
			name:      "MP3 missing TPE1 (author)",
			container: "MP3",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = "1234567890123"
			},
			wantViolLen: 1,
			wantRule:    "format_specific_tags",
		},
		{
			name:      "MP3 missing TCOM (narrator)",
			container: "MP3",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{}
				m.Language = "eng"
				m.ISBN = "1234567890123"
			},
			wantViolLen: 1,
			wantRule:    "format_specific_tags",
		},
		{
			name:      "FLAC with all required tags",
			container: "FLAC",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = "1234567890123"
			},
			wantViolLen: 0,
		},
		{
			name:      "FLAC missing author",
			container: "FLAC",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{}
				m.Title = "Title"
				m.Year = 2024
				m.Narrator = []string{"Narrator"}
				m.Language = "eng"
				m.ISBN = "1234567890123"
			},
			wantViolLen: 1,
			wantRule:    "format_specific_tags",
		},
		{
			name:      "Unknown container type",
			container: "WAV",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
			},
			wantViolLen: 0,
		},
		{
			name:        "Nil metadata",
			container:   "M4B",
			setupMeta:   nil,
			wantViolLen: 0,
		},
		{
			name:      "Empty tracks",
			container: "M4B",
			setupMeta: func(m *metadata.Metadata) {
				m.Author = []string{"Author"}
				m.Tracks = []metadata.Track{}
			},
			wantViolLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var meta *metadata.Metadata
			if tt.setupMeta != nil {
				meta = &metadata.Metadata{
					Tracks: []metadata.Track{{Container: tt.container}},
				}
				tt.setupMeta(meta)
			}

			violations := CheckFormatSpecificTags(meta)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckFormatSpecificTags() violations count = %v, want %v. Violations: %+v", len(violations), tt.wantViolLen, violations)
			}

			if tt.wantRule != "" && len(violations) > 0 {
				if violations[0].Rule != tt.wantRule {
					t.Errorf("CheckFormatSpecificTags() rule = %v, want %v", violations[0].Rule, tt.wantRule)
				}
			}
		})
	}
}

func TestMultiValueTags(t *testing.T) {
	// Test the infrastructure map exists
	if MultiValueTags == nil {
		t.Fatal("MultiValueTags map is nil")
	}

	formats := []string{"M4B", "MP3", "FLAC"}
	for _, format := range formats {
		if _, ok := MultiValueTags[format]; !ok {
			t.Errorf("MultiValueTags missing definition for format %s", format)
		}
	}

	// Verify author is multi-value for M4B
	if !containsTag(MultiValueTags["M4B"], "author") {
		t.Error("M4B should have author as multi-value tag")
	}
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
