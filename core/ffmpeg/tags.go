package ffmpeg

import (
	"strconv"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/internal/version"
)

// metadataArgs builds -metadata args per RULES.md §4; artist/composer mirror author/narrator; year=0 and empty ASIN omitted.
func metadataArgs(m *metadata.Metadata) []string {
	author := metadata.JoinTags(m.Author)
	narrator := metadata.JoinTags(m.Narrator)

	seriesNames, seriesParts := metadata.SeriesNamesParts(m.Series)

	tag := func(key, value string) []string {
		return []string{"-metadata", key + "=" + value}
	}

	args := tag("author", author)
	args = append(args, tag("artist", author)...)
	args = append(args, tag("title", m.Title)...)
	args = append(args, tag("subtitle", m.Subtitle)...)
	args = append(args, tag("publisher", metadata.JoinTags(m.Publisher))...)
	if m.Year != 0 {
		args = append(args, tag("year", strconv.Itoa(m.Year))...)
	}
	args = append(args, tag("narrator", narrator)...)
	args = append(args, tag("composer", narrator)...)
	args = append(args, tag("description", m.Description)...)
	args = append(args, tag("genre", metadata.JoinTags(m.Genre))...)
	args = append(args, tag("series", metadata.JoinTags(seriesNames))...)
	args = append(args, tag("series-part", metadata.JoinTags(seriesParts))...)
	args = append(args, tag("language", m.Language)...)
	args = append(args, tag("isbn", m.ISBN)...)
	if m.ASIN != "" {
		args = append(args, tag("asin", m.ASIN)...)
	}
	args = append(args, tag("zentag", version.Version)...)

	return args
}
