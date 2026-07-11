package ruleset

import (
	"testing"

	"codeberg.org/Ether/zentag/core/metadata"
	"github.com/stretchr/testify/assert"
)

// fullMeta returns Metadata with every RULES.md §4 required field set; reused as a clean baseline tests break one field at a time.
func fullMeta() *metadata.Metadata {
	return &metadata.Metadata{
		Author:   []string{"Brandon Sanderson"},
		Title:    "The Way of Kings",
		Year:     2010,
		Narrator: []string{"Michael Kramer"},
		Language: "eng",
	}
}

func TestCheckRequiredTags_AllPresentClean(t *testing.T) {
	assert.Empty(t, CheckRequiredTags(fullMeta()))
}

func TestCheckRequiredTags_MissingAuthor(t *testing.T) {
	meta := fullMeta()
	meta.Author = nil
	violations := CheckRequiredTags(meta)
	assert.Len(t, violations, 1)
	assert.Equal(t, "required_tags", violations[0].Rule)
	assert.Contains(t, violations[0].Message, "author")
}

func TestCheckRequiredTags_MissingTitle(t *testing.T) {
	meta := fullMeta()
	meta.Title = ""
	violations := CheckRequiredTags(meta)
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "title")
}

func TestCheckRequiredTags_MissingYear(t *testing.T) {
	meta := fullMeta()
	meta.Year = 0
	violations := CheckRequiredTags(meta)
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "year")
}

func TestCheckRequiredTags_MissingNarrator(t *testing.T) {
	meta := fullMeta()
	meta.Narrator = nil
	violations := CheckRequiredTags(meta)
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "narrator")
}

func TestCheckRequiredTags_MissingLanguage(t *testing.T) {
	meta := fullMeta()
	meta.Language = ""
	violations := CheckRequiredTags(meta)
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "language")
}

func TestCheckRequiredTags_SeriesEntryMissingPart(t *testing.T) {
	meta := fullMeta()
	meta.Series = []metadata.SeriesEntry{{Name: "Stormlight Archive", Part: ""}}
	violations := CheckRequiredTags(meta)
	assert.Len(t, violations, 1)
	assert.Equal(t, `series part missing for series "Stormlight Archive"`, violations[0].Message)
}
