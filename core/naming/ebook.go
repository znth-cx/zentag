package naming

import (
	"context"
	"strconv"
	"strings"
)

// Format is an ebook container format in the naming vocabulary.
type Format int

const (
	FormatUnset Format = iota
	FormatEPUB
	FormatPDF
	FormatDJVU
	FormatMOBI
	FormatAZW3
)

func (f Format) String() string {
	switch f {
	case FormatEPUB:
		return "EPUB"
	case FormatPDF:
		return "PDF"
	case FormatDJVU:
		return "DJVU"
	case FormatMOBI:
		return "MOBI"
	case FormatAZW3:
		return "AZW3"
	default:
		return ""
	}
}

// Writable reports whether ebook-meta can write metadata into this format.
func (f Format) Writable() bool {
	switch f {
	case FormatEPUB, FormatPDF, FormatMOBI, FormatAZW3:
		return true
	default:
		return false
	}
}

func FormatFromExtension(ext string) (Format, bool) {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "epub":
		return FormatEPUB, true
	case "pdf":
		return FormatPDF, true
	case "djvu":
		return FormatDJVU, true
	case "mobi":
		return FormatMOBI, true
	case "azw3":
		return FormatAZW3, true
	default:
		return FormatUnset, false
	}
}

// EbookNameParams collects the fields for the ebook name template.
type EbookNameParams struct {
	Author     string
	Series     string
	SeriesPart string
	Title      string
	Year       int
	Language   string
	Edition    string
	Format     Format
	ISBN       string
	ASIN       string
	Retail     bool
}

// EbookFolderName builds the output directory name:
//
//	Author - [Series - ]Title [LANG FORMAT]
func EbookFolderName(ctx context.Context, p EbookNameParams) string {
	var b strings.Builder
	b.WriteString(p.Author)
	b.WriteString(" - ")
	if p.Series != "" {
		b.WriteString(p.Series)
		b.WriteString(" - ")
	}
	b.WriteString(p.Title)

	var parts []string
	if p.Language != "" {
		parts = append(parts, strings.ToUpper(p.Language))
	}
	if s := p.Format.String(); s != "" {
		parts = append(parts, s)
	}
	writeBracket(&b, parts)
	return sanitize(ctx, b.String())
}

func writeBracket(b *strings.Builder, parts []string) {
	b.WriteString(" [")
	b.WriteString(strings.Join(parts, " "))
	b.WriteString("]")
}

// EbookFileName builds the file basename; detailed so books stay distinguishable:
//
//	Author - [Series #Part - ]Title (Year) [Language Edition Format ISBN|ASIN Retail]
//
// Identifier: ISBN when present, else ASIN only for retail.
func EbookFileName(ctx context.Context, p EbookNameParams) string {
	var b strings.Builder
	b.WriteString(p.Author)
	b.WriteString(" - ")
	if p.Series != "" {
		b.WriteString(p.Series)
		b.WriteString(" #")
		b.WriteString(p.SeriesPart)
		b.WriteString(" - ")
	}
	b.WriteString(p.Title)
	b.WriteString(" (")
	b.WriteString(strconv.Itoa(p.Year))
	b.WriteString(")")

	var parts []string
	if p.Language != "" {
		parts = append(parts, strings.ToUpper(p.Language))
	}
	if p.Edition != "" {
		parts = append(parts, p.Edition)
	}
	if s := p.Format.String(); s != "" {
		parts = append(parts, s)
	}
	if p.ISBN != "" {
		parts = append(parts, p.ISBN)
	} else if p.Retail && p.ASIN != "" {
		parts = append(parts, p.ASIN)
	}
	if p.Retail {
		parts = append(parts, "Retail")
	}
	writeBracket(&b, parts)
	return sanitize(ctx, b.String())
}
