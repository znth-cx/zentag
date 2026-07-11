package ruleset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckLanguage_InvalidCode(t *testing.T) {
	meta := fullMeta()
	meta.Language = "xx"
	violations := CheckLanguage(meta)
	assert.Len(t, violations, 1)
	assert.Equal(t, "language", violations[0].Rule)
	assert.Equal(t, SeverityTrumpable, violations[0].Severity)
}

func TestCheckLanguage_ValidCodeClean(t *testing.T) {
	meta := fullMeta()
	meta.Language = "eng"
	assert.Empty(t, CheckLanguage(meta))
}

func TestCheckLanguage_ValidNameClean(t *testing.T) {
	meta := fullMeta()
	meta.Language = "English"
	assert.Empty(t, CheckLanguage(meta))
}

func TestCheckLanguage_EnglishAliasClean(t *testing.T) {
	meta := fullMeta()
	meta.Language = "en"
	assert.Empty(t, CheckLanguage(meta), `"en" must always be accepted as English`)
}

func TestCheckLanguage_BannedCodeStillInvalid(t *testing.T) {
	meta := fullMeta()
	meta.Language = "enc"
	violations := CheckLanguage(meta)
	assert.Len(t, violations, 1)
	assert.Equal(t, "language", violations[0].Rule)
}

func TestCheckLanguage_EmptyLanguageCleanFromThisCheck(t *testing.T) {
	meta := fullMeta()
	meta.Language = ""
	assert.Empty(t, CheckLanguage(meta))
}
