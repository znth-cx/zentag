package mp4tag

import (
	"fmt"

	mp4 "github.com/Sorrow446/go-mp4tag"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/internal/version"
)

// buildTags translates m to go-mp4tag's MP4Tags per RULES.md §4. Fields with no native atom (author, subtitle, series, series-part, language, isbn, asin, zentag) go through Custom freeform atoms; narrator too, since go-mp4tag's Narrator field never actually writes the "©nrt" atom (library bug). Year omitted when 0, ASIN omitted when empty; other fields always written.
func buildTags(m *metadata.Metadata) *mp4.MP4Tags {
	author := metadata.JoinTags(m.Author)
	narrator := metadata.JoinTags(m.Narrator)

	seriesNames, seriesParts := metadata.SeriesNamesParts(m.Series)

	tags := &mp4.MP4Tags{
		Title:       m.Title,
		Album:       m.Title,
		Artist:      author,
		AlbumArtist: author,
		Composer:    narrator,
		Description: m.Description,
		Comment:     m.Description,
		CustomGenre: metadata.JoinTags(m.Genre),
		Custom:      map[string]string{},
	}

	put := func(name, val string) { tags.Custom[name] = val }
	put("AUTHOR", author)
	if m.Year > 0 && m.Year <= metadata.MaxYear {
		tags.Year = int32(m.Year)
		tags.Date = fmt.Sprint(m.Year)
		put("YEAR", fmt.Sprint(m.Year))
	}
	put("NARRATOR", narrator)
	put("SUBTITLE", m.Subtitle)
	put("SERIES", metadata.JoinTags(seriesNames))
	put("SERIES-PART", metadata.JoinTags(seriesParts))
	put("LANGUAGE", m.Language)
	put("ISBN", m.ISBN)
	put("PUBLISHER", metadata.JoinTags(m.Publisher))
	if m.ASIN != "" {
		put("ASIN", m.ASIN)
	}
	put("ZENTAG", version.Version)

	if len(m.CoverImage) > 0 {
		tags.Pictures = []*mp4.MP4Picture{{
			Format: coverFormat(m.CoverMIME),
			Data:   m.CoverImage,
		}}
	}

	return tags
}

// coverFormat maps MIME to go-mp4tag's image type enum; unknown falls back to ImageTypeAuto (sniffed from magic bytes).
func coverFormat(mime string) mp4.ImageType {
	switch mime {
	case "image/jpeg":
		return mp4.ImageTypeJPEG
	case "image/png":
		return mp4.ImageTypePNG
	default:
		return mp4.ImageTypeAuto
	}
}
