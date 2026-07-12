package ebookmeta

import "testing"

const sampleOPF = `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>The Final Empire</dc:title>
    <dc:creator opf:role="aut">Brandon Sanderson</dc:creator>
    <dc:identifier opf:scheme="ISBN">9780765311788</dc:identifier>
    <dc:identifier opf:scheme="MOBI-ASIN">B002GYI9C4</dc:identifier>
    <dc:language>eng</dc:language>
    <dc:date>2006-07-17T00:00:00+00:00</dc:date>
    <dc:publisher>Tor Books</dc:publisher>
    <dc:description>Book one of Mistborn.</dc:description>
    <dc:subject>Fantasy</dc:subject>
    <dc:subject>Epic</dc:subject>
    <meta name="calibre:series" content="Mistborn"/>
    <meta name="calibre:series_index" content="1"/>
  </metadata>
</package>`

func TestParseOPF(t *testing.T) {
	md, err := parseOPF([]byte(sampleOPF))
	if err != nil {
		t.Fatalf("parseOPF() error: %v", err)
	}
	if md.Title != "The Final Empire" {
		t.Errorf("Title = %q", md.Title)
	}
	if len(md.Authors) != 1 || md.Authors[0] != "Brandon Sanderson" {
		t.Errorf("Authors = %v", md.Authors)
	}
	if md.ISBN != "9780765311788" {
		t.Errorf("ISBN = %q", md.ISBN)
	}
	if md.ASIN != "B002GYI9C4" {
		t.Errorf("ASIN = %q", md.ASIN)
	}
	if md.Language != "eng" {
		t.Errorf("Language = %q", md.Language)
	}
	if md.Year != 2006 {
		t.Errorf("Year = %d", md.Year)
	}
	if len(md.Publisher) != 1 || md.Publisher[0] != "Tor Books" {
		t.Errorf("Publisher = %v", md.Publisher)
	}
	if len(md.Tags) != 2 || md.Tags[0] != "Fantasy" || md.Tags[1] != "Epic" {
		t.Errorf("Tags = %v", md.Tags)
	}
	if md.Series != "Mistborn" || md.SeriesIndex != 1 {
		t.Errorf("Series = %q index = %v", md.Series, md.SeriesIndex)
	}
}

func TestParseOPF_Empty(t *testing.T) {
	md, err := parseOPF([]byte(`<package><metadata></metadata></package>`))
	if err != nil {
		t.Fatalf("parseOPF() error: %v", err)
	}
	if md.Title != "" || len(md.Authors) != 0 || md.Year != 0 {
		t.Errorf("expected zero-value metadata, got %+v", md)
	}
}
