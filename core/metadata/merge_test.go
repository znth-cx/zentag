package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge_SingleSourceNoConflicts(t *testing.T) {
	src := &Metadata{
		MetadataOrigin: OriginFileMetadata,
		Author:         []string{"Brandon Sanderson"},
		Title:          "The Way of Kings",
		Year:           2010,
		Narrator:       []string{"Michael Kramer"},
		Language:       "en",
		Tracks:         []Track{{Path: "book.m4b"}},
		OriginalPath:   "/books/way-of-kings",
	}

	merged, conflicts := Merge(context.Background(), src)
	assert.Empty(t, conflicts)
	assert.Equal(t, []string{"Brandon Sanderson"}, merged.Author)
	assert.Equal(t, "The Way of Kings", merged.Title)
	assert.Equal(t, 2010, merged.Year)
	assert.Equal(t, []string{"Michael Kramer"}, merged.Narrator)
	assert.Equal(t, "en", merged.Language)
	assert.Equal(t, "/books/way-of-kings", merged.OriginalPath)
	assert.Len(t, merged.Tracks, 1)
}

func TestMerge_AgreeingSourcesNoConflict(t *testing.T) {
	a := &Metadata{MetadataOrigin: OriginUserArgs, Title: "Same Title"}
	b := &Metadata{MetadataOrigin: OriginFileMetadata, Title: "Same Title"}

	merged, conflicts := Merge(context.Background(), a, b)
	assert.Empty(t, conflicts)
	assert.Equal(t, "Same Title", merged.Title)
}

func TestMerge_DisagreeingSourcesEmitsConflictWithRecommended(t *testing.T) {
	userArgs := &Metadata{MetadataOrigin: OriginUserArgs, Title: "User Title"}
	file := &Metadata{MetadataOrigin: OriginFileMetadata, Title: "File Title"}

	merged, conflicts := Merge(context.Background(), userArgs, file)
	require.Len(t, conflicts, 1)
	c := conflicts[0]
	assert.Equal(t, "Title", c.Field)
	assert.Equal(t, []string{"User Title", "File Title"}, c.Values)
	assert.Equal(t, []MetadataOrigin{OriginUserArgs, OriginFileMetadata}, c.Origins)
	assert.Equal(t, 0, c.Recommended)
	assert.Equal(t, "", merged.Title) // left zero until resolved
}

func TestMerge_SliceFieldConflictDisplaysJoined(t *testing.T) {
	a := &Metadata{MetadataOrigin: OriginUserArgs, Author: []string{"A", "B"}}
	b := &Metadata{MetadataOrigin: OriginFileMetadata, Author: []string{"C"}}

	_, conflicts := Merge(context.Background(), a, b)
	require.Len(t, conflicts, 1)
	assert.Equal(t, "Author", conflicts[0].Field)
	assert.Equal(t, []string{"A; B", "C"}, conflicts[0].Values)
}

func TestMerge_LanguageCaseInsensitiveNoConflict(t *testing.T) {
	audnexus := &Metadata{MetadataOrigin: OriginAudnexus, Language: "en"}
	file := &Metadata{MetadataOrigin: OriginFileMetadata, Language: "EN"}

	merged, conflicts := Merge(context.Background(), audnexus, file)
	assert.Empty(t, conflicts)
	assert.Equal(t, "en", merged.Language)
}

func TestMerge_NilSourcesSkipped(t *testing.T) {
	file := &Metadata{MetadataOrigin: OriginFileMetadata, Title: "Only Source"}
	merged, conflicts := Merge(context.Background(), nil, file, nil)
	assert.Empty(t, conflicts)
	assert.Equal(t, "Only Source", merged.Title)
}

func TestMerge_CoverTakesHighestPrecedenceNonNilNoConflict(t *testing.T) {
	userArgs := &Metadata{MetadataOrigin: OriginUserArgs}
	audnexus := &Metadata{MetadataOrigin: OriginAudnexus, CoverImage: []byte{1, 2, 3}, CoverMIME: "image/jpeg"}
	file := &Metadata{MetadataOrigin: OriginFileMetadata, CoverImage: []byte{4, 5, 6}, CoverMIME: "image/png"}

	merged, conflicts := Merge(context.Background(), userArgs, audnexus, file)
	assert.Empty(t, conflicts)
	assert.Equal(t, []byte{1, 2, 3}, merged.CoverImage)
	assert.Equal(t, "image/jpeg", merged.CoverMIME, "CoverMIME must come from the same source as CoverImage, never mixed")
}

func TestMerge_TracksAndOriginalPathComeFromFileSourceOnly(t *testing.T) {
	userArgs := &Metadata{MetadataOrigin: OriginUserArgs}
	file := &Metadata{
		MetadataOrigin: OriginFileMetadata,
		OriginalPath:   "/books/item",
		Tracks:         []Track{{Path: "book.m4b"}},
	}

	merged, conflicts := Merge(context.Background(), userArgs, file)
	assert.Empty(t, conflicts)
	assert.Equal(t, "/books/item", merged.OriginalPath)
	assert.Len(t, merged.Tracks, 1)
}

func TestMerge_SeriesFieldConflict(t *testing.T) {
	a := &Metadata{MetadataOrigin: OriginUserArgs, Series: []SeriesEntry{{Name: "The Stormlight Archive", Part: "1"}}}
	b := &Metadata{MetadataOrigin: OriginFileMetadata, Series: []SeriesEntry{{Name: "Stormlight Archive", Part: "1"}}}

	_, conflicts := Merge(context.Background(), a, b)
	require.Len(t, conflicts, 1)
	assert.Equal(t, "Series", conflicts[0].Field)
}

func TestMerge_YearFieldConflictDisplaysAsNumber(t *testing.T) {
	a := &Metadata{MetadataOrigin: OriginUserArgs, Year: 2010}
	b := &Metadata{MetadataOrigin: OriginFileMetadata, Year: 2011}

	_, conflicts := Merge(context.Background(), a, b)
	require.Len(t, conflicts, 1)
	assert.Equal(t, []string{"2010", "2011"}, conflicts[0].Values)
}

func TestApplyResolutions_ChosenIndexSetsField(t *testing.T) {
	userArgs := &Metadata{MetadataOrigin: OriginUserArgs, Title: "User Title"}
	file := &Metadata{MetadataOrigin: OriginFileMetadata, Title: "File Title"}
	merged, conflicts := Merge(context.Background(), userArgs, file)

	resolved := ApplyResolutions(merged, conflicts, map[string]int{"Title": 1})
	assert.Equal(t, "File Title", resolved.Title)
}

func TestApplyResolutions_MissingChoiceUsesRecommended(t *testing.T) {
	userArgs := &Metadata{MetadataOrigin: OriginUserArgs, Title: "User Title"}
	file := &Metadata{MetadataOrigin: OriginFileMetadata, Title: "File Title"}
	merged, conflicts := Merge(context.Background(), userArgs, file)

	resolved := ApplyResolutions(merged, conflicts, map[string]int{})
	assert.Equal(t, "User Title", resolved.Title) // Recommended == 0 == "User Title"
}

func TestApplyResolutions_NegativeOneOmitsField(t *testing.T) {
	userArgs := &Metadata{MetadataOrigin: OriginUserArgs, Title: "User Title"}
	file := &Metadata{MetadataOrigin: OriginFileMetadata, Title: "File Title"}
	merged, conflicts := Merge(context.Background(), userArgs, file)

	resolved := ApplyResolutions(merged, conflicts, map[string]int{"Title": -1})
	assert.Equal(t, "", resolved.Title)
}

func TestApplyResolutions_SliceFieldChoice(t *testing.T) {
	a := &Metadata{MetadataOrigin: OriginUserArgs, Author: []string{"A"}}
	b := &Metadata{MetadataOrigin: OriginFileMetadata, Author: []string{"B"}}
	merged, conflicts := Merge(context.Background(), a, b)

	resolved := ApplyResolutions(merged, conflicts, map[string]int{"Author": 1})
	assert.Equal(t, []string{"B"}, resolved.Author)
}

func TestMerge_MergedDoesNotAliasSourceSlices(t *testing.T) {
	a := &Metadata{
		MetadataOrigin: OriginUserArgs,
		Author:         []string{"Original Author"},
	}
	b := &Metadata{
		MetadataOrigin: OriginFileMetadata,
		Author:         []string{"Original Author"},
		Tracks:         []Track{{Path: "book.m4b", Chapters: []Chapter{{Title: "Chapter 1"}}}},
	}

	merged, conflicts := Merge(context.Background(), a, b)
	require.Empty(t, conflicts)

	merged.Author[0] = "Mutated"
	merged.Tracks[0].Chapters[0].Title = "Mutated"

	assert.Equal(t, []string{"Original Author"}, a.Author)
	assert.Equal(t, []string{"Original Author"}, b.Author)
	assert.Equal(t, "Chapter 1", b.Tracks[0].Chapters[0].Title)
}

func TestApplyResolutions_ResolvedSliceDoesNotAliasSource(t *testing.T) {
	a := &Metadata{MetadataOrigin: OriginUserArgs, Author: []string{"A"}}
	b := &Metadata{MetadataOrigin: OriginFileMetadata, Author: []string{"B"}}
	merged, conflicts := Merge(context.Background(), a, b)

	resolved := ApplyResolutions(merged, conflicts, map[string]int{"Author": 1})
	resolved.Author[0] = "Mutated"
	assert.Equal(t, []string{"B"}, b.Author)
}

func TestApplyResolutions_MultipleConflictsResolvedIndependently(t *testing.T) {
	a := &Metadata{MetadataOrigin: OriginUserArgs, Title: "User Title", Year: 2010}
	b := &Metadata{MetadataOrigin: OriginFileMetadata, Title: "File Title", Year: 2011}
	merged, conflicts := Merge(context.Background(), a, b)
	require.Len(t, conflicts, 2)

	resolved := ApplyResolutions(merged, conflicts, map[string]int{"Title": 0, "Year": 1})
	assert.Equal(t, "User Title", resolved.Title)
	assert.Equal(t, 2011, resolved.Year)
}
