package audnexus

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// fetchCover downloads imageURL, sniffs MIME. Mirrors ffmpeg.ReadCover's approach.
func fetchCover(ctx context.Context, client HTTPClient, imageURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d fetching cover", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return data, http.DetectContentType(data), nil
}
