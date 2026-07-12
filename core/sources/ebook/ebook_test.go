package ebook

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/znth-cx/zentag/core/ebookmeta"
)

type fakeRunner struct{ opf string }

func (f fakeRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	for _, a := range args {
		if strings.HasPrefix(a, "--to-opf=") {
			return nil, os.WriteFile(strings.TrimPrefix(a, "--to-opf="), []byte(f.opf), 0o644)
		}
	}
	return nil, nil
}

func TestGather(t *testing.T) {
	opf := `<?xml version="1.0"?><package><metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">` +
		`<dc:title>The Final Empire</dc:title><dc:creator opf:role="aut">Brandon Sanderson</dc:creator>` +
		`<dc:language>eng</dc:language><dc:date>2006-07-17</dc:date>` +
		`<meta name="calibre:series" content="Mistborn"/><meta name="calibre:series_index" content="1"/>` +
		`</metadata></package>`
	w := &ebookmeta.Wrapper{BinPath: "x", Runner: fakeRunner{opf: opf}}
	m, err := Gather(context.Background(), w, "b.epub")
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	if m.Title != "The Final Empire" || m.Year != 2006 || m.Language != "eng" {
		t.Fatalf("scalar fields wrong: %+v", m)
	}
	if len(m.Series) != 1 || m.Series[0].Name != "Mistborn" || m.Series[0].Part != "1" {
		t.Fatalf("series wrong: %+v", m.Series)
	}
	if m.MetadataOrigin != "file_metadata" {
		t.Fatalf("origin wrong: %s", m.MetadataOrigin)
	}
}
