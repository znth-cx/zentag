package ebookmeta

import (
	"context"
	"os"
	"strings"
	"testing"
)

type fakeRunner struct {
	version string
	opf     string
	writes  [][]string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	for _, a := range args {
		if a == "--version" {
			return []byte(f.version), nil
		}
		if strings.HasPrefix(a, "--to-opf=") {
			path := strings.TrimPrefix(a, "--to-opf=")
			return nil, os.WriteFile(path, []byte(f.opf), 0o644)
		}
	}
	f.writes = append(f.writes, args)
	return nil, nil
}

func TestValidate(t *testing.T) {
	ctx := context.Background()
	w := &Wrapper{BinPath: "ebook-meta", Runner: &fakeRunner{version: "ebook-meta (calibre 9.11.0)"}}
	if err := w.Validate(ctx); err != nil {
		t.Fatalf("Validate ok case: %v", err)
	}
	w2 := &Wrapper{BinPath: "x", Runner: &fakeRunner{version: "GNU coreutils 9.0"}}
	if err := w2.Validate(ctx); err == nil {
		t.Fatalf("Validate should reject non-calibre binary")
	}
}

func TestReadWrite(t *testing.T) {
	ctx := context.Background()
	opf := `<?xml version="1.0"?><package><metadata xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>T</dc:title><dc:creator opf:role="aut">A</dc:creator>` +
		`<dc:identifier opf:scheme="ISBN">9780756404741</dc:identifier></metadata></package>`
	f := &fakeRunner{opf: opf}
	w := &Wrapper{BinPath: "ebook-meta", Runner: f}
	md, err := w.Read(ctx, "book.epub")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if md.Title != "T" || md.ISBN != "9780756404741" || len(md.Authors) != 1 {
		t.Fatalf("parsed wrong: %+v", md)
	}
	if err := w.Write(ctx, "book.epub", []string{"--title", "T"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if len(f.writes) != 1 || f.writes[0][0] != "book.epub" {
		t.Fatalf("Write args wrong: %v", f.writes)
	}
}
