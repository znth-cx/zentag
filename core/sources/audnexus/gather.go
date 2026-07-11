package audnexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/znth-cx/zentag/core/cover"
	"github.com/znth-cx/zentag/core/metadata"
)

// ErrNotFound indicates audnexus has no book for the given ASIN.
var ErrNotFound = errors.New("audnexus: book not found")

// newBooksRequest builds GET /books/{ASIN}, shared by Gather and FetchBookJSON.
// region: au/ca/de/es/fr/in/it/jp/us/uk; empty defaults to us.
func newBooksRequest(ctx context.Context, w *Wrapper, asin, region string) (*http.Request, error) {
	if region == "" {
		region = "us"
	}

	reqURL, err := url.Parse(w.BaseURL + "/books/" + url.PathEscape(asin))
	if err != nil {
		return nil, fmt.Errorf("audnexus %s: %w", asin, err)
	}
	q := reqURL.Query()
	q.Set("region", region)
	reqURL.RawQuery = q.Encode()

	return http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
}

// FetchBookJSON fetches raw /books/{ASIN} body undecoded, for previewing before a full Gather.
func FetchBookJSON(ctx context.Context, w *Wrapper, asin, region string) ([]byte, error) {
	req, err := newBooksRequest(ctx, w, asin, region)
	if err != nil {
		return nil, err
	}

	resp, err := w.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("audnexus fetch %s: %w", asin, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("audnexus fetch %s: %w", asin, ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audnexus fetch %s: unexpected status %d", asin, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("audnexus fetch %s: reading response: %w", asin, err)
	}
	return body, nil
}

// Gather looks up asin via audnexus API, maps result to metadata.Metadata.
// region: audnexus region code, empty defaults to us.
func Gather(ctx context.Context, w *Wrapper, asin, region string) (*metadata.Metadata, error) {
	slog.DebugContext(ctx, "audnexus gather starting", "asin", asin, "region", region)

	if region == "" {
		region = "us" // reused below by fetchChapterCount
	}

	req, err := newBooksRequest(ctx, w, asin, region)
	if err != nil {
		return nil, fmt.Errorf("audnexus gather %s: %w", asin, err)
	}

	resp, err := w.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("audnexus gather %s: %w", asin, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("audnexus gather %s: %w", asin, ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audnexus gather %s: unexpected status %d", asin, resp.StatusCode)
	}

	var b book
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, fmt.Errorf("audnexus gather %s: decoding response: %w", asin, err)
	}

	m := b.toMetadata(ctx)

	if b.Image != "" {
		data, mime, err := fetchCover(ctx, w.Client, b.Image)
		if err != nil {
			slog.WarnContext(ctx, "audnexus: cover fetch failed", "asin", asin, "url", b.Image, "error", err)
		} else if ok, reason := cover.Validate(ctx, data); !ok {
			// garbage bytes (HTML error page etc.) fail here, not at write time
			slog.WarnContext(ctx, "audnexus: cover invalid, skipping", "asin", asin, "url", b.Image, "reason", reason)
		} else {
			m.CoverImage = data
			m.CoverMIME = mime
		}
	}

	if count, err := fetchChapterCount(ctx, w, asin, region); err != nil {
		slog.WarnContext(ctx, "audnexus: chapter lookup failed, continuing without it", "asin", asin, "error", err)
	} else {
		m.AudnexusChapterCount = count
	}

	slog.DebugContext(ctx, "audnexus gather succeeded", "asin", asin)
	return m, nil
}
