package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTitleCase_LowersMinorWordsMidTitle(t *testing.T) {
	got := TitleCase("the way of kings", "en")
	assert.Equal(t, "The Way of Kings", got)
}

func TestTitleCase_CapitalizesFirstLastAndAfterColon(t *testing.T) {
	got := TitleCase("a study in scarlet: a novel", "en")
	assert.Equal(t, "A Study in Scarlet: A Novel", got)
}

func TestTitleCase_PreservesGraphicAudioPartSuffix(t *testing.T) {
	got := TitleCase("shadowfall (1 of 5)", "en")
	assert.Equal(t, "Shadowfall (1 of 5)", got)
}

func TestTitleCase_UnknownLanguageFallsBackGracefully(t *testing.T) {
	got := TitleCase("the way of kings", "")
	assert.Equal(t, "The Way of Kings", got)
}

func TestTitleCase_EmptyTitleReturnsEmpty(t *testing.T) {
	got := TitleCase("", "en")
	assert.Equal(t, "", got)
}

func TestTitleCase_LowersIfMidTitle(t *testing.T) {
	got := TitleCase("what happens if you fall", "en")
	assert.Equal(t, "What Happens if You Fall", got)
}
