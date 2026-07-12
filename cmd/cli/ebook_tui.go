package main

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/znth-cx/zentag/core/metadata"
)

// ThemeCharm's focus bar is near-invisible gray (238); recolor for visibility
func ebookFormTheme() *huh.Theme {
	t := huh.ThemeCharm()
	bar := lipgloss.Color("212")
	t.Focused.Base = t.Focused.Base.BorderForeground(bar)
	t.Focused.Title = t.Focused.Title.Foreground(bar).Bold(true)
	return t
}

// up/down move fields; Text (Description) untouched so they still move lines
func ebookKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Input.Next.SetKeys(append(km.Input.Next.Keys(), "down")...)
	km.Input.Prev.SetKeys(append(km.Input.Prev.Keys(), "up")...)
	km.Confirm.Next.SetKeys(append(km.Confirm.Next.Keys(), "down")...)
	km.Confirm.Prev.SetKeys(append(km.Confirm.Prev.Keys(), "up")...)
	return km
}

// retail tracked separately; not a Metadata field
type ebookEdit struct {
	meta   *metadata.Metadata
	retail bool
}

type ebookFields struct {
	author, title, year, isbn                    string
	seriesName, seriesPart, edition              string
	retail                                       bool
	publisher, language, description, tags, asin string
}

func buildEbookEdit(f ebookFields, origin metadata.MetadataOrigin, originalPath string) ebookEdit {
	m := &metadata.Metadata{
		MetadataOrigin: origin,
		OriginalPath:   originalPath,
		Author:         splitCSV(f.author),
		Title:          strings.TrimSpace(f.title),
		ISBN:           strings.TrimSpace(f.isbn),
		Edition:        strings.TrimSpace(f.edition),
		Publisher:      splitCSV(f.publisher),
		Language:       strings.TrimSpace(f.language),
		Description:    f.description,
		Genre:          splitCSV(f.tags),
		ASIN:           strings.TrimSpace(f.asin),
	}
	if y, err := strconv.Atoi(strings.TrimSpace(f.year)); err == nil {
		m.Year = y
	}
	if name := strings.TrimSpace(f.seriesName); name != "" {
		m.Series = []metadata.SeriesEntry{{Name: name, Part: strings.TrimSpace(f.seriesPart)}}
	}
	return ebookEdit{meta: m, retail: f.retail}
}

// accepted=false on abort (ctrl+c/esc); m never mutated
func runEbookEditForm(ctx context.Context, in io.Reader, out io.Writer, m *metadata.Metadata, retail bool) (ebookEdit, bool, error) {
	f := ebookFields{
		author:      strings.Join(m.Author, ", "),
		title:       m.Title,
		isbn:        m.ISBN,
		edition:     m.Edition,
		retail:      retail,
		publisher:   strings.Join(m.Publisher, ", "),
		language:    m.Language,
		description: m.Description,
		tags:        strings.Join(m.Genre, ", "),
		asin:        m.ASIN,
	}
	if m.Year != 0 {
		f.year = strconv.Itoa(m.Year)
	}
	if len(m.Series) > 0 {
		f.seriesName, f.seriesPart = m.Series[0].Name, m.Series[0].Part
	}

	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Author").Description("comma-separated, primary first").Value(&f.author),
		huh.NewInput().Title("Title").Value(&f.title),
		huh.NewInput().Title("Year").Validate(validateYear).Value(&f.year),
		huh.NewInput().Title("ISBN").Validate(validateISBN).Value(&f.isbn),
		huh.NewInput().Title("Series name").Value(&f.seriesName),
		huh.NewInput().Title("Series part").Value(&f.seriesPart),
		huh.NewInput().Title("Edition").Value(&f.edition),
		huh.NewConfirm().Title("Retail release?").Value(&f.retail),
		huh.NewInput().Title("Publisher").Description("comma-separated").Value(&f.publisher),
		huh.NewInput().Title("Language").Description("ISO-639-3, e.g. eng").Validate(validateLanguage).Value(&f.language),
		huh.NewText().Title("Description").Value(&f.description),
		huh.NewInput().Title("Tags").Description("comma-separated").Value(&f.tags),
		huh.NewInput().Title("ASIN").Value(&f.asin),
	).Title("Edit ebook metadata")).
		WithTheme(ebookFormTheme()).
		WithKeyMap(ebookKeyMap()).
		WithShowHelp(true).
		WithInput(in).WithOutput(out)

	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return ebookEdit{}, false, nil
		}
		return ebookEdit{}, false, err
	}
	return buildEbookEdit(f, m.MetadataOrigin, m.OriginalPath), true, nil
}
