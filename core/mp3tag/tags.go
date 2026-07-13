package mp3tag

import (
	"strconv"

	"go.senan.xyz/taglib"

	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/internal/version"
)

func buildID3Tags(m *metadata.Metadata, track metadata.Track) map[string][]string {
	tags := make(map[string][]string)

	tags[taglib.Title] = []string{m.Title}
	tags[taglib.Artist] = m.Author
	tags[taglib.Album] = []string{m.Title}
	tags[taglib.AlbumArtist] = m.Author
	tags[taglib.Label] = m.Publisher

	if m.Year > 0 && m.Year <= metadata.MaxYear {
		tags[taglib.Date] = []string{strconv.Itoa(m.Year)}
	}

	tags[taglib.Composer] = m.Narrator

	if m.Description != "" {
		tags[taglib.Comment] = []string{m.Description}
	}

	if len(m.Genre) > 0 {
		tags[taglib.Genre] = m.Genre
	}

	if m.Language != "" {
		tags[taglib.Language] = []string{m.Language}
	}

	if track.PartNumber > 0 {
		total := strconv.Itoa(len(m.Tracks))
		current := strconv.Itoa(track.PartNumber)
		tags[taglib.TrackNumber] = []string{current + "/" + total}
	}

	if m.Subtitle != "" {
		tags[taglib.TitleSort] = []string{m.Subtitle}
	}

	if len(m.Narrator) > 0 {
		tags["NARRATOR"] = []string{metadata.JoinTags(m.Narrator)}
	}

	if len(m.Series) > 0 {
		seriesNames, seriesParts := metadata.SeriesNamesParts(m.Series)
		if len(seriesNames) > 0 {
			tags["SERIES"] = seriesNames
		}
		if len(seriesParts) > 0 {
			tags["SERIES-PART"] = seriesParts
		}
	}

	if m.ISBN != "" {
		tags["ISBN"] = []string{m.ISBN}
	}

	if m.ASIN != "" {
		tags["ASIN"] = []string{m.ASIN}
	}

	tags["ZENTAG"] = []string{version.Version}

	return tags
}
