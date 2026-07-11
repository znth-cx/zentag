package cover

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
)

// Load reads cover bytes from URL or filepath and sniffs MIME type.
func Load(ctx context.Context, source string) ([]byte, string, error) {
	if isURL(source) {
		return loadFromURL(ctx, source)
	}
	return loadFromPath(source)
}

// isURL reports whether source is an http(s) URL.
func isURL(source string) bool {
	u, err := url.Parse(source)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func loadFromURL(ctx context.Context, source string) ([]byte, string, error) {
	slog.DebugContext(ctx, "cover load fetching URL", "url", source)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, "", fmt.Errorf("cover: load %q: %w", source, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("cover: load %q: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("cover: load %q: unexpected status %d", source, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("cover: load %q: %w", source, err)
	}

	slog.DebugContext(ctx, "cover load fetched URL", "url", source, "bytes", len(data))
	return data, http.DetectContentType(data), nil
}

func loadFromPath(source string) ([]byte, string, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, "", fmt.Errorf("cover: load %q: %w", source, err)
	}
	return data, http.DetectContentType(data), nil
}
