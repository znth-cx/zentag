package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/znth-cx/zentag/core/metadata"
)

func TestBuildEbookEdit(t *testing.T) {
	got := buildEbookEdit(ebookFields{
		author: "A One, B Two", title: "T", year: "2006", isbn: "9780765311788",
		seriesName: "Mistborn", seriesPart: "1", edition: "Revised", retail: true,
		publisher: "Tor", language: "eng", description: "d", tags: "Fantasy, Epic", asin: "B002GYI9C4",
	}, metadata.OriginUserArgs, "in.epub")
	m := got.meta
	assert.Equal(t, []string{"A One", "B Two"}, m.Author)
	assert.Equal(t, 2006, m.Year)
	assert.Equal(t, []metadata.SeriesEntry{{Name: "Mistborn", Part: "1"}}, m.Series)
	assert.Equal(t, []string{"Fantasy", "Epic"}, m.Genre)
	assert.True(t, got.retail)
	assert.Equal(t, "in.epub", m.OriginalPath)
}

func TestEbookKeyMap_UpDownNavigatesFields(t *testing.T) {
	km := ebookKeyMap()
	assert.Contains(t, km.Input.Next.Keys(), "down")
	assert.Contains(t, km.Input.Prev.Keys(), "up")
	assert.Contains(t, km.Confirm.Next.Keys(), "down")
	assert.Contains(t, km.Confirm.Prev.Keys(), "up")
	// Text keeps up/down for line movement, not field nav.
	assert.NotContains(t, km.Text.Next.Keys(), "down")
	assert.NotContains(t, km.Text.Prev.Keys(), "up")
}
