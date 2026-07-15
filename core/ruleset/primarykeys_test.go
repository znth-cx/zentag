package ruleset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/znth-cx/zentag/core/metadata"
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

func TestCheckPrimaryKeys_ValidISBN10Clean(t *testing.T) {
	assert.Empty(t, CheckPrimaryKeys(&metadata.Metadata{ISBN: "0306406152"}))
}

func TestCheckPrimaryKeys_InvalidISBNChecksum(t *testing.T) {
	violations := CheckPrimaryKeys(&metadata.Metadata{ISBN: "9780306406158"})
	assert.Len(t, violations, 1)
	assert.Equal(t, "ISBN checksum invalid", violations[0].Message)
}

func TestCheckPrimaryKeys_ASINOnlyClean(t *testing.T) {
	assert.Empty(t, CheckPrimaryKeys(&metadata.Metadata{ASIN: "B002V1S3G0"}))
}

func TestCheckPrimaryKeys_ISBN10WithASINWarns(t *testing.T) {
	violations := CheckPrimaryKeys(&metadata.Metadata{
		ISBN: "0306406152",
		ASIN: "B002V1S3G0",
	})
	assert.Len(t, violations, 1)
	assert.Equal(t, "primary_keys", violations[0].Rule)
	assert.Equal(t, SeverityWarn, violations[0].Severity)
	assert.Contains(t, violations[0].Message, "ISBN-13 is preferred")
}

func TestCheckPrimaryKeys_ISBN13WithASINClean(t *testing.T) {
	assert.Empty(t, CheckPrimaryKeys(&metadata.Metadata{
		ISBN: "9780306406157",
		ASIN: "B002V1S3G0",
	}))
}

func TestCheckPrimaryKeys_ISBN10AloneClean(t *testing.T) {
	assert.Empty(t, CheckPrimaryKeys(&metadata.Metadata{ISBN: "0306406152"}))
}

func TestCheckPrimaryKeys_ISBN13AloneClean(t *testing.T) {
	assert.Empty(t, CheckPrimaryKeys(&metadata.Metadata{ISBN: "9780306406157"}))
}
