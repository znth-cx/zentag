// Package metadata's merge.go combines multiple Metadata sources into one: agreeing fields set directly, disagreeing fields reported as Conflicts for ApplyResolutions.
package metadata

import (
	"context"
	"log/slog"
	"slices"
	"strconv"
	"strings"
)

// Conflict is one field where Merge's sources disagreed; Values/Origins/rawValues align by index in source precedence order.
type Conflict struct {
	Field       string
	Values      []string // display strings, aligned with Origins
	Origins     []MetadataOrigin
	Recommended int // index into Values/Origins; -1 = hard tie

	rawValues []any // typed values aligned with Values/Origins; consumed by ApplyResolutions
}

type fieldValue[T any] struct {
	origin MetadataOrigin
	value  T
}

// collectField gathers non-zero values per source, preserving Merge's precedence order (index 0 = highest).
func collectField[T any](sources []*Metadata, isZero func(T) bool, get func(*Metadata) T) []fieldValue[T] {
	var out []fieldValue[T]
	for _, s := range sources {
		v := get(s)
		if !isZero(v) {
			out = append(out, fieldValue[T]{origin: s.MetadataOrigin, value: v})
		}
	}
	return out
}

// resolveField turns candidates into a settled value, or a Conflict if they disagree. equal checks agreement; display renders a value for Conflict.Values.
func resolveField[T any](field string, candidates []fieldValue[T], equal func(a, b T) bool, display func(T) string) (T, *Conflict) {
	var zero T
	if len(candidates) == 0 {
		return zero, nil
	}

	allEqual := true
	for _, c := range candidates[1:] {
		if !equal(c.value, candidates[0].value) {
			allEqual = false
			break
		}
	}
	if allEqual {
		return candidates[0].value, nil
	}

	conflict := &Conflict{Field: field, Recommended: 0}
	for _, c := range candidates {
		conflict.Values = append(conflict.Values, display(c.value))
		conflict.Origins = append(conflict.Origins, c.origin)
		conflict.rawValues = append(conflict.rawValues, c.value)
	}
	return zero, conflict
}

// resolveSliceField wraps resolveField for slice fields, cloning the settled value and conflict rawValues so merged never aliases a source's backing array (Merge and ApplyResolutions both flow through here).
func resolveSliceField[E any](field string, candidates []fieldValue[[]E], equal func(a, b []E) bool, display func([]E) string) ([]E, *Conflict) {
	v, c := resolveField(field, candidates, equal, display)
	if c != nil {
		for i := range c.rawValues {
			c.rawValues[i] = slices.Clone(c.rawValues[i].([]E))
		}
	}
	return slices.Clone(v), c
}

func isZeroString(v string) bool    { return v == "" }
func equalString(a, b string) bool  { return a == b }
func displayString(v string) string { return v }

func isZeroInt(v int) bool    { return v == 0 }
func equalInt(a, b int) bool  { return a == b }
func displayInt(v int) string { return strconv.Itoa(v) }

func isZeroStrings(v []string) bool   { return len(v) == 0 }
func equalStrings(a, b []string) bool { return slices.Equal(a, b) }
func displayStrings(v []string) string {
	return strings.Join(v, "; ")
}

func isZeroSeries(v []SeriesEntry) bool   { return len(v) == 0 }
func equalSeries(a, b []SeriesEntry) bool { return slices.Equal(a, b) }
func displaySeries(v []SeriesEntry) string {
	parts := make([]string, len(v))
	for i, e := range v {
		parts[i] = e.Name + " #" + e.Part
	}
	return strings.Join(parts, "; ")
}

func isZeroSource(v ReleaseSource) bool   { return v == "" }
func equalSource(a, b ReleaseSource) bool { return a == b }
func displaySource(v ReleaseSource) string {
	return string(v)
}

// Merge combines sources (nil skipped) into one Metadata: agreements set directly, disagreements become Conflicts (index 0 = highest precedence). Cover fields and Tracks/OriginalPath come from a single source, never mixed.
func Merge(ctx context.Context, sources ...*Metadata) (*Metadata, []Conflict) {
	var active []*Metadata
	for _, s := range sources {
		if s != nil {
			active = append(active, s)
		}
	}
	slog.DebugContext(ctx, "metadata: merging sources", "count", len(active))

	merged := &Metadata{}
	var conflicts []Conflict
	addConflict := func(c *Conflict) {
		if c != nil {
			conflicts = append(conflicts, *c)
		}
	}

	var c *Conflict
	merged.Author, c = resolveSliceField("Author", collectField(active, isZeroStrings, func(m *Metadata) []string { return m.Author }), equalStrings, displayStrings)
	addConflict(c)
	merged.Title, c = resolveField("Title", collectField(active, isZeroString, func(m *Metadata) string { return m.Title }), equalString, displayString)
	addConflict(c)
	merged.Subtitle, c = resolveField("Subtitle", collectField(active, isZeroString, func(m *Metadata) string { return m.Subtitle }), equalString, displayString)
	addConflict(c)
	merged.Publisher, c = resolveSliceField("Publisher", collectField(active, isZeroStrings, func(m *Metadata) []string { return m.Publisher }), equalStrings, displayStrings)
	addConflict(c)
	merged.Year, c = resolveField("Year", collectField(active, isZeroInt, func(m *Metadata) int { return m.Year }), equalInt, displayInt)
	addConflict(c)
	merged.Narrator, c = resolveSliceField("Narrator", collectField(active, isZeroStrings, func(m *Metadata) []string { return m.Narrator }), equalStrings, displayStrings)
	addConflict(c)
	merged.Description, c = resolveField("Description", collectField(active, isZeroString, func(m *Metadata) string { return m.Description }), equalString, displayString)
	addConflict(c)
	merged.Genre, c = resolveSliceField("Genre", collectField(active, isZeroStrings, func(m *Metadata) []string { return m.Genre }), equalStrings, displayStrings)
	addConflict(c)
	merged.Series, c = resolveSliceField("Series", collectField(active, isZeroSeries, func(m *Metadata) []SeriesEntry { return m.Series }), equalSeries, displaySeries)
	addConflict(c)
	merged.Language, c = resolveField("Language", collectField(active, isZeroString, func(m *Metadata) string { return strings.ToLower(m.Language) }), equalString, displayString)
	addConflict(c)
	merged.ISBN, c = resolveField("ISBN", collectField(active, isZeroString, func(m *Metadata) string { return m.ISBN }), equalString, displayString)
	addConflict(c)
	merged.ASIN, c = resolveField("ASIN", collectField(active, isZeroString, func(m *Metadata) string { return m.ASIN }), equalString, displayString)
	addConflict(c)
	merged.Edition, c = resolveField("Edition", collectField(active, isZeroString, func(m *Metadata) string { return m.Edition }), equalString, displayString)
	addConflict(c)
	merged.Source, c = resolveField("Source", collectField(active, isZeroSource, func(m *Metadata) ReleaseSource { return m.Source }), equalSource, displaySource)
	addConflict(c)

	for _, s := range active {
		if len(s.CoverImage) > 0 {
			merged.CoverImage = s.CoverImage
			merged.CoverMIME = s.CoverMIME
			break
		}
	}
	for _, s := range active {
		if len(s.Tracks) > 0 {
			// clone Tracks and per-track Chapters so editing merged never touches source
			merged.Tracks = slices.Clone(s.Tracks)
			for i := range merged.Tracks {
				merged.Tracks[i].Chapters = slices.Clone(merged.Tracks[i].Chapters)
			}
			merged.OriginalPath = s.OriginalPath
			break
		}
	}
	for _, s := range active {
		if s.AudnexusChapterCount > 0 {
			merged.AudnexusChapterCount = s.AudnexusChapterCount
			break
		}
	}

	slog.DebugContext(ctx, "metadata: merge complete", "conflicts", len(conflicts))
	return merged, conflicts
}

// ApplyResolutions applies choices to conflicts on merged, filling in every conflicted field; a missing choice uses Recommended, -1 omits the field (zero value).
func ApplyResolutions(merged *Metadata, conflicts []Conflict, choices map[string]int) *Metadata {
	for _, c := range conflicts {
		choice, ok := choices[c.Field]
		if !ok {
			choice = c.Recommended
		}
		if choice < 0 || choice >= len(c.rawValues) {
			continue
		}
		val := c.rawValues[choice]
		switch c.Field {
		case "Author":
			merged.Author = val.([]string)
		case "Title":
			merged.Title = val.(string)
		case "Subtitle":
			merged.Subtitle = val.(string)
		case "Publisher":
			merged.Publisher = val.([]string)
		case "Year":
			merged.Year = val.(int)
		case "Narrator":
			merged.Narrator = val.([]string)
		case "Description":
			merged.Description = val.(string)
		case "Genre":
			merged.Genre = val.([]string)
		case "Series":
			merged.Series = val.([]SeriesEntry)
		case "Language":
			merged.Language = val.(string)
		case "ISBN":
			merged.ISBN = val.(string)
		case "ASIN":
			merged.ASIN = val.(string)
		case "Edition":
			merged.Edition = val.(string)
		case "Source":
			merged.Source = val.(ReleaseSource)
		}
	}
	return merged
}
