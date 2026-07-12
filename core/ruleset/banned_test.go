package ruleset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/znth-cx/zentag/core/metadata"
)

func TestCheckBannedContent_ExactAuthorMatch(t *testing.T) {
	violations := CheckBannedContent(&metadata.Metadata{Author: []string{"J.R.R. Tolkien"}})
	assert.Len(t, violations, 1)
	assert.Equal(t, "banned_content", violations[0].Rule)
	assert.Equal(t, SeverityProhibited, violations[0].Severity)
}

func TestCheckBannedContent_FuzzyAuthorTypoMatch(t *testing.T) {
	// Single-character substitution against banned author "Sara Gruen".
	violations := CheckBannedContent(&metadata.Metadata{Author: []string{"Sara Gruem"}})
	assert.Len(t, violations, 1)
}

func TestCheckBannedContent_UnrelatedAuthorClean(t *testing.T) {
	assert.Empty(t, CheckBannedContent(&metadata.Metadata{Author: []string{"Brandon Sanderson"}}))
}

func TestCheckBannedContent_ExactWorkMatch(t *testing.T) {
	violations := CheckBannedContent(&metadata.Metadata{
		Title: "Four Against Darkness Expanded Edition, and all associated content",
	})
	assert.Len(t, violations, 1)
	assert.Equal(t, SeverityProhibited, violations[0].Severity)
}

func TestCheckBannedContent_FuzzyWorkTypoMatch(t *testing.T) {
	// Single-character deletion ("Darknes" missing the final 's').
	violations := CheckBannedContent(&metadata.Metadata{
		Title: "Four Against Darknes Expanded Edition, and all associated content",
	})
	assert.Len(t, violations, 1)
}

func TestCheckBannedContent_UnrelatedTitleClean(t *testing.T) {
	assert.Empty(t, CheckBannedContent(&metadata.Metadata{Title: "The Way of Kings"}))
}
