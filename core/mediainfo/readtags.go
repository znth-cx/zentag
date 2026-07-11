package mediainfo

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"codeberg.org/Ether/zentag/core/metadata"
)

// TagSet holds book metadata fields read from a file's tags.
type TagSet struct {
	Author, Narrator, Publisher, Genre []string
	Title, Subtitle, Description       string
	Year                               int
	Series                             []metadata.SeriesEntry
	Language, ISBN, ASIN               string

	// Extra holds all custom tags not in RULES.md fields. Writers preserve unrecognized tags to avoid data loss.
	Extra map[string]string
}

// extraValue does a case-insensitive linear scan. Use only for one-off lookups; parseTagSet builds a lowered map instead.
func extraValue(extra map[string]string, key string) string {
	for k, v := range extra {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

func parseTagSet(general, audio *mediainfoTrack) TagSet {
	// lowered-key copy for O(1) case-insensitive lookups; TagSet.Extra keeps the original map.
	extra := make(map[string]string, len(general.Extra))
	for k, v := range general.Extra {
		extra[strings.ToLower(k)] = v
	}

	title := general.Title
	if title == "" {
		title = extra["title"]
	}

	language := extra["language"]
	if language == "" {
		// ID3 TLAN / Vorbis LANGUAGE surface only as the Audio track's named Language; not a redundant fallback.
		language = audio.Language
	}

	// Description is a named field (©cmt), not extra. Must fall back to it.
	description := extra["description"]
	if description == "" {
		description = general.Comment
	}

	tags := TagSet{
		Author:      metadata.SplitTags(extra["author"]),
		Narrator:    metadata.SplitTags(extra["narrator"]),
		Publisher:   metadata.SplitTags(extra["publisher"]),
		Genre:       metadata.SplitTags(general.Genre),
		Title:       title,
		Subtitle:    extra["subtitle"],
		Description: description,
		Language:    language,
		ISBN:        extra["isbn"],
		ASIN:        extra["asin"],
		Extra:       general.Extra,
	}

	if year := extra["year"]; year != "" {
		if y, err := strconv.Atoi(year); err == nil {
			tags.Year = y
		}
	}

	names := metadata.SplitTags(extra["series"])
	seriesPart := extra["series-part"]
	if seriesPart == "" {
		seriesPart = extra["seriespart"]
	}
	parts := metadata.SplitTags(seriesPart)
	for i := range names {
		part := ""
		if i < len(parts) {
			part = parts[i]
		}
		tags.Series = append(tags.Series, metadata.SeriesEntry{Name: names[i], Part: part})
	}

	return tags
}

// readTagSet is the shared probe+parse flow behind ReadTags and ReadTagsGreedy.
func (w *Wrapper) readTagSet(ctx context.Context, path string) (TagSet, error) {
	general, audio, err := w.runAndFindTracks(ctx, path)
	if err != nil {
		return TagSet{}, err
	}
	return parseTagSet(general, audio), nil
}

// ReadTags reads book tags from mediainfo's RULES.md §4 locations.
func (w *Wrapper) ReadTags(ctx context.Context, path string) (TagSet, error) {
	slog.DebugContext(ctx, "mediainfo read tags starting", "path", path)

	tags, err := w.readTagSet(ctx, path)
	if err != nil {
		return TagSet{}, fmt.Errorf("mediainfo read tags: %w", err)
	}

	slog.DebugContext(ctx, "mediainfo read tags succeeded", "path", path)
	return tags, nil
}

// ReadTagsGreedy reads tags with fallbacks for non-standard tag locations.
func (w *Wrapper) ReadTagsGreedy(ctx context.Context, path string) (TagSet, error) {
	slog.DebugContext(ctx, "mediainfo read tags (greedy) starting", "path", path)

	tags, err := w.readTagSet(ctx, path)
	if err != nil {
		return TagSet{}, fmt.Errorf("mediainfo read tags greedy: %w", err)
	}

	if tags.ASIN == "" {
		if asin := extraValue(tags.Extra, "CDEK"); asin != "" {
			slog.DebugContext(ctx, "mediainfo read tags (greedy): ASIN found in CDEK tag", "path", path)
			tags.ASIN = asin
		}
	}

	slog.DebugContext(ctx, "mediainfo read tags (greedy) succeeded", "path", path)
	return tags, nil
}
