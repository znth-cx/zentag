package audnexus

import (
	"context"
	"log/slog"
	"time"

	"github.com/znth-cx/zentag/core/lang"
	"github.com/znth-cx/zentag/core/metadata"
)

type person struct {
	Name string `json:"name"`
}

// genre: Type ("genre"/"tag") folded into Metadata.Genre, distinction dropped.
type genre struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type series struct {
	Name     string `json:"name"`
	Position string `json:"position"`
}

type book struct {
	ASIN            string   `json:"asin"`
	Authors         []person `json:"authors"`
	Description     string   `json:"description"`
	FormatType      string   `json:"formatType"`
	Genres          []genre  `json:"genres"`
	Image           string   `json:"image"`
	ISBN            string   `json:"isbn"`
	Language        string   `json:"language"`
	Narrators       []person `json:"narrators"`
	PublisherName   string   `json:"publisherName"`
	ReleaseDate     string   `json:"releaseDate"`
	SeriesPrimary   *series  `json:"seriesPrimary"`
	SeriesSecondary *series  `json:"seriesSecondary"`
	Subtitle        string   `json:"subtitle"`
	Summary         string   `json:"summary"`
	Title           string   `json:"title"`
}

// toMetadata: bad language/date logged and left zero, not fatal. Reports what audnexus returned.
func (b book) toMetadata(ctx context.Context) *metadata.Metadata {
	m := &metadata.Metadata{
		MetadataOrigin: metadata.OriginAudnexus,
		Title:          b.Title,
		Subtitle:       b.Subtitle,
		Description:    b.Summary,
		ISBN:           b.ISBN,
		ASIN:           b.ASIN,
	}

	for _, a := range b.Authors {
		m.Author = append(m.Author, a.Name)
	}
	for _, n := range b.Narrators {
		m.Narrator = append(m.Narrator, n.Name)
	}
	if b.PublisherName != "" {
		m.Publisher = []string{b.PublisherName}
	}
	for _, g := range b.Genres {
		m.Genre = append(m.Genre, g.Name)
	}
	if b.SeriesPrimary != nil {
		m.Series = append(m.Series, metadata.SeriesEntry{Name: b.SeriesPrimary.Name, Part: b.SeriesPrimary.Position})
	}
	if b.SeriesSecondary != nil {
		m.Series = append(m.Series, metadata.SeriesEntry{Name: b.SeriesSecondary.Name, Part: b.SeriesSecondary.Position})
	}

	if code, ok := lang.CodeForName(b.Language); ok {
		m.Language = code
	} else if b.Language != "" {
		slog.WarnContext(ctx, "audnexus: unrecognized language name", "language", b.Language)
	}

	if t, err := time.Parse(time.RFC3339, b.ReleaseDate); err == nil {
		m.Year = t.Year()
	} else if b.ReleaseDate != "" {
		slog.WarnContext(ctx, "audnexus: unparseable release date", "releaseDate", b.ReleaseDate)
	}

	if b.FormatType == "abridged" {
		m.Edition = "Abridged"
	}

	return m
}
