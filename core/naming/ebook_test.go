package naming

import (
	"context"
	"testing"
)

func TestFormatFromExtension(t *testing.T) {
	cases := map[string]struct {
		want Format
		ok   bool
	}{
		"epub": {FormatEPUB, true}, "EPUB": {FormatEPUB, true}, ".pdf": {FormatPDF, true},
		"djvu": {FormatDJVU, true}, "mobi": {FormatMOBI, true}, "azw3": {FormatAZW3, true},
		"txt": {FormatUnset, false},
	}
	for ext, c := range cases {
		got, ok := FormatFromExtension(ext)
		if got != c.want || ok != c.ok {
			t.Errorf("FormatFromExtension(%q) = %v,%v want %v,%v", ext, got, ok, c.want, c.ok)
		}
	}
	if !FormatEPUB.Writable() || FormatDJVU.Writable() {
		t.Errorf("Writable: EPUB must be true, DJVU false")
	}
	if FormatEPUB.String() != "EPUB" || FormatUnset.String() != "" {
		t.Errorf("String mapping wrong")
	}
}

func TestEbookFileName(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		p    EbookNameParams
		want string
	}{
		{"standalone isbn", EbookNameParams{Author: "Patrick Rothfuss", Title: "The Name of the Wind", Year: 2007, Language: "eng", Format: FormatEPUB, ISBN: "9780756404741"},
			"Patrick Rothfuss - The Name of the Wind (2007) [ENG EPUB 9780756404741]"},
		{"series edition retail isbn wins over asin", EbookNameParams{Author: "Brandon Sanderson", Series: "Mistborn", SeriesPart: "1", Title: "The Final Empire", Year: 2006, Language: "eng", Edition: "Revised", Format: FormatEPUB, ISBN: "9780765311788", ASIN: "B002GYI9C4", Retail: true},
			"Brandon Sanderson - Mistborn #1 - The Final Empire (2006) [ENG Revised EPUB 9780765311788 Retail]"},
		{"retail no isbn uses asin", EbookNameParams{Author: "A B", Title: "T", Year: 2020, Language: "eng", Format: FormatMOBI, ASIN: "B002GYI9C4", Retail: true},
			"A B - T (2020) [ENG MOBI B002GYI9C4 Retail]"},
		{"no language", EbookNameParams{Author: "A B", Title: "T", Year: 2020, Format: FormatPDF, ISBN: "9780765311788"},
			"A B - T (2020) [PDF 9780765311788]"},
	}
	for _, c := range cases {
		if got := EbookFileName(ctx, c.p); got != c.want {
			t.Errorf("%s: EbookFileName = %q want %q", c.name, got, c.want)
		}
	}
}

func TestEbookFolderName(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		p    EbookNameParams
		want string
	}{
		{"series uses series and title, drops year/isbn/edition/retail/part", EbookNameParams{Author: "Brandon Sanderson", Series: "Mistborn", SeriesPart: "1", Title: "The Final Empire", Year: 2006, Language: "eng", Edition: "Revised", Format: FormatEPUB, ISBN: "9780765311788", ASIN: "B002GYI9C4", Retail: true},
			"Brandon Sanderson - Mistborn - The Final Empire [ENG EPUB]"},
		{"standalone uses title", EbookNameParams{Author: "Patrick Rothfuss", Title: "The Name of the Wind", Year: 2007, Language: "eng", Format: FormatEPUB, ISBN: "9780756404741"},
			"Patrick Rothfuss - The Name of the Wind [ENG EPUB]"},
		{"standalone no language", EbookNameParams{Author: "A B", Title: "T", Year: 2020, Format: FormatPDF, ISBN: "9780765311788"},
			"A B - T [PDF]"},
		{"series no language", EbookNameParams{Author: "A B", Series: "S", SeriesPart: "2", Title: "T", Format: FormatMOBI},
			"A B - S - T [MOBI]"},
	}
	for _, c := range cases {
		if got := EbookFolderName(ctx, c.p); got != c.want {
			t.Errorf("%s: EbookFolderName = %q want %q", c.name, got, c.want)
		}
	}
}
