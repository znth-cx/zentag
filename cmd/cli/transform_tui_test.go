package main

import (
	"testing"
	"time"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "0:00:00"},
		{5 * time.Second, "0:00:05"},
		{90 * time.Second, "0:01:30"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03"},
		{-1 * time.Second, "0:00:00"}, // clamped
	}
	for _, c := range cases {
		assert.Equal(t, c.want, formatDuration(c.in), "d=%s", c.in)
	}
}

func TestGroupIndexFromKey(t *testing.T) {
	cases := []struct {
		key  string
		want int
	}{
		{"g0_title", 0},
		{"g1_5", 1},
		{"g2_confirm", 2},
		{"g10_x", 10},
		{"title", -1},   // no g prefix
		{"g_title", -1}, // no digits
		{"", -1},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, groupIndexFromKey(c.key), "key=%q", c.key)
	}
}

func TestValidateYear(t *testing.T) {
	assert.NoError(t, validateYear(""))   // unset ok
	assert.NoError(t, validateYear("  ")) // blank ok
	assert.NoError(t, validateYear("1937"))
	assert.Error(t, validateYear("0"))
	assert.Error(t, validateYear("-5"))
	assert.Error(t, validateYear("nineteen"))
}

func TestValidateISBN(t *testing.T) {
	assert.NoError(t, validateISBN(""))                  // unset ok
	assert.NoError(t, validateISBN("978-0-7653-2635-5")) // valid ISBN-13 with hyphens
	assert.NoError(t, validateISBN("0306406152"))        // valid ISBN-10
	assert.Error(t, validateISBN("9780765326354"))       // bad checksum
	assert.Error(t, validateISBN("12345"))               // wrong length
	assert.Error(t, validateISBN("978076532635X"))       // non-digit in ISBN-13
}

func TestValidateLanguage(t *testing.T) {
	assert.NoError(t, validateLanguage(""))        // unset ok
	assert.NoError(t, validateLanguage("eng"))     // valid code
	assert.NoError(t, validateLanguage("ENG"))     // case-insensitive code
	assert.NoError(t, validateLanguage("English")) // resolvable name
	assert.NoError(t, validateLanguage("english")) // case-insensitive name
	assert.NoError(t, validateLanguage("en"))      // ISO-639-1 alias for English
	assert.NoError(t, validateLanguage("En"))      // alias wins over banned name "En" (code "enc")
	assert.Error(t, validateLanguage("enc"))       // banned code
	assert.Error(t, validateLanguage("zz"))        // unknown code
	assert.Error(t, validateLanguage("clingon"))   // unknown name
}

func TestResolveLanguage(t *testing.T) {
	code, ok := resolveLanguage("English")
	assert.True(t, ok)
	assert.Equal(t, "eng", code)

	code, ok = resolveLanguage("ENG")
	assert.True(t, ok)
	assert.Equal(t, "eng", code)

	code, ok = resolveLanguage("fra")
	assert.True(t, ok)
	assert.Equal(t, "fra", code)

	code, ok = resolveLanguage("en")
	assert.True(t, ok)
	assert.Equal(t, "eng", code, `"en" must always resolve to English`)

	_, ok = resolveLanguage("enc")
	assert.False(t, ok, "banned code must still be rejected")

	_, ok = resolveLanguage("")
	assert.False(t, ok)
	_, ok = resolveLanguage("zz")
	assert.False(t, ok)
}

func TestCoverStatus(t *testing.T) {
	assert.Equal(t, "current: none", coverStatus(&metadata.Metadata{}))
	assert.Equal(t, "current: image/jpeg, 3 bytes",
		coverStatus(&metadata.Metadata{CoverImage: []byte{1, 2, 3}, CoverMIME: "image/jpeg"}))
}

func TestApplyUserOverrides_SetFieldsWin(t *testing.T) {
	dst := &metadata.Metadata{
		Title:    "Session Title",
		Subtitle: "Session Subtitle",
		Author:   []string{"Session Author"},
		Year:     2000,
	}
	src := &metadata.Metadata{
		Title:  "Flag Title",
		Author: []string{"Flag Author"},
		Year:   2020,
	}

	applyUserOverrides(dst, src)
	assert.Equal(t, "Flag Title", dst.Title)
	assert.Equal(t, []string{"Flag Author"}, dst.Author)
	assert.Equal(t, 2020, dst.Year)
	// unset fields keep session value
	assert.Equal(t, "Session Subtitle", dst.Subtitle)
}

func TestApplyUserOverrides_UnsetFieldsDoNotClobber(t *testing.T) {
	dst := &metadata.Metadata{
		Subtitle:  "Session Subtitle",
		Publisher: []string{"Session Pub"},
		Source:    metadata.ReleaseSourceCD,
	}
	applyUserOverrides(dst, &metadata.Metadata{}) // no flags set

	assert.Equal(t, "Session Subtitle", dst.Subtitle)
	assert.Equal(t, []string{"Session Pub"}, dst.Publisher)
	assert.Equal(t, metadata.ReleaseSourceCD, dst.Source)
}

func TestApplyUserOverrides_CoverReplacedWithMIME(t *testing.T) {
	dst := &metadata.Metadata{CoverImage: []byte{1}, CoverMIME: "image/png"}
	src := &metadata.Metadata{CoverImage: []byte{9, 9}, CoverMIME: "image/jpeg"}

	applyUserOverrides(dst, src)
	assert.Equal(t, []byte{9, 9}, dst.CoverImage)
	assert.Equal(t, "image/jpeg", dst.CoverMIME)
}
