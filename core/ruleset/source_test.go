package ruleset

import (
	"testing"

	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckSource(t *testing.T) {
	tests := []struct {
		name        string
		source      metadata.ReleaseSource
		wantViolLen int
		wantMsg     string
	}{
		{
			name:        "Valid WEB source",
			source:      metadata.ReleaseSourceWEB,
			wantViolLen: 0,
		},
		{
			name:        "Valid CD source",
			source:      metadata.ReleaseSourceCD,
			wantViolLen: 0,
		},
		{
			name:        "Valid VINYL source",
			source:      metadata.ReleaseSourceVinyl,
			wantViolLen: 0,
		},
		{
			name:        "Invalid CASSETTE source",
			source:      metadata.ReleaseSourceCassette,
			wantViolLen: 1,
			wantMsg:     "invalid source",
		},
		{
			name:        "Empty source",
			source:      "",
			wantViolLen: 1,
			wantMsg:     "source field is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &metadata.Metadata{
				Source: tt.source,
			}
			violations := CheckSource(meta)

			if len(violations) != tt.wantViolLen {
				t.Errorf("CheckSource() violations count = %v, want %v", len(violations), tt.wantViolLen)
			}

			if tt.wantViolLen > 0 && violations[0].Message == "" {
				t.Errorf("CheckSource() expected violation message, got empty")
			}

			if tt.wantMsg != "" && len(violations) > 0 {
				if violations[0].Message != tt.wantMsg && !contains(violations[0].Message, tt.wantMsg) {
					t.Errorf("CheckSource() message = %v, want to contain %v", violations[0].Message, tt.wantMsg)
				}
			}
		})
	}
}

func TestCheckSourceNil(t *testing.T) {
	violations := CheckSource(nil)
	if violations != nil {
		t.Errorf("CheckSource() on nil metadata should return nil, got %v", violations)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
