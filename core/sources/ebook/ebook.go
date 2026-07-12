// Package ebook gathers metadata from an ebook file via ebook-meta into
// canonical metadata.Metadata.
package ebook

import (
	"context"
	"strconv"

	"github.com/znth-cx/zentag/core/ebookmeta"
	"github.com/znth-cx/zentag/core/metadata"
)

// Gather reads path's metadata into a *Metadata tagged OriginFileMetadata.
func Gather(ctx context.Context, w *ebookmeta.Wrapper, path string) (*metadata.Metadata, error) {
	opf, err := w.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	m := &metadata.Metadata{
		OriginalPath:   path,
		MetadataOrigin: metadata.OriginFileMetadata,
		Author:         opf.Authors,
		Title:          opf.Title,
		Year:           opf.Year,
		ISBN:           opf.ISBN,
		ASIN:           opf.ASIN,
		Publisher:      opf.Publisher,
		Language:       opf.Language,
		Description:    opf.Description,
		Genre:          opf.Tags,
	}
	if opf.Series != "" {
		m.Series = []metadata.SeriesEntry{{Name: opf.Series, Part: formatIndex(opf.SeriesIndex)}}
	}
	return m, nil
}

// formatIndex renders a numeric series index without trailing zeros; 0 -> "".
func formatIndex(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
