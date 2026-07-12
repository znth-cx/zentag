// Package EbookEngine writes ebook tags via ebook-meta.
// Edition, Retail, Format are naming-only, never written.
package EbookEngine

import (
	"context"
	"strconv"
	"strings"

	"github.com/znth-cx/zentag/core/ebookmeta"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/internal/version"
)

// Write tags a writable ebook at outFile with meta's writable fields.
func Write(ctx context.Context, w *ebookmeta.Wrapper, meta *metadata.Metadata, outFile string) error {
	return w.Write(ctx, outFile, Args(meta))
}

// Args builds the ebook-meta flag list for meta's writable fields.
func Args(meta *metadata.Metadata) []string {
	var args []string
	if len(meta.Author) > 0 {
		args = append(args, "--authors", strings.Join(meta.Author, " & "))
	}
	if meta.Title != "" {
		args = append(args, "--title", meta.Title)
	}
	if meta.Year != 0 {
		args = append(args, "--date", strconv.Itoa(meta.Year))
	}
	if meta.ISBN != "" {
		args = append(args, "--isbn", meta.ISBN)
	}
	if len(meta.Series) > 0 && meta.Series[0].Name != "" {
		args = append(args, "--series", meta.Series[0].Name)
		if meta.Series[0].Part != "" {
			args = append(args, "--index", meta.Series[0].Part)
		}
	}
	if len(meta.Publisher) > 0 {
		args = append(args, "--publisher", strings.Join(meta.Publisher, "; "))
	}
	if meta.Language != "" {
		args = append(args, "--language", meta.Language)
	}
	if meta.Description != "" {
		args = append(args, "--comments", meta.Description)
	}
	if len(meta.Genre) > 0 {
		args = append(args, "--tags", strings.Join(meta.Genre, ","))
	}
	if meta.ASIN != "" {
		args = append(args, "--identifier", "asin:"+meta.ASIN)
	}
	// Version stamp; matches audio writers.
	args = append(args, "--identifier", "zentag:"+version.Version)
	return args
}
