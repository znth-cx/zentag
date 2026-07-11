package ruleset

import (
	"testing"

	"codeberg.org/Ether/zentag/core/metadata"
	"github.com/stretchr/testify/assert"
)

func TestCheckPrimaryKeys_MissingBothISBNAndASIN(t *testing.T) {
	violations := CheckPrimaryKeys(&metadata.Metadata{})
	assert.Len(t, violations, 1)
	assert.Equal(t, "primary_keys", violations[0].Rule)
	assert.Equal(t, SeverityTrumpable, violations[0].Severity)
	assert.Equal(t, "no ISBN or ASIN", violations[0].Message)
}

func TestCheckPrimaryKeys_ValidISBN13Clean(t *testing.T) {
	assert.Empty(t, CheckPrimaryKeys(&metadata.Metadata{ISBN: "9780306406157"}))
}

func TestCheckPrimaryKeys_InvalidISBNChecksum(t *testing.T) {
	violations := CheckPrimaryKeys(&metadata.Metadata{ISBN: "9780306406158"})
	assert.Len(t, violations, 1)
	assert.Equal(t, "ISBN checksum invalid", violations[0].Message)
}

func TestCheckPrimaryKeys_ASINOnlyClean(t *testing.T) {
	assert.Empty(t, CheckPrimaryKeys(&metadata.Metadata{ASIN: "B002V1S3G0"}))
}
