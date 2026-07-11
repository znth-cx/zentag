package audnexus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// chapterList: only count used, titles/timings not modeled.
type chapterList struct {
	Chapters []struct {
		Title string `json:"title"`
	} `json:"chapters"`
}

// fetchChapterCount hits audnexus's separate /chapters endpoint (not in /books response).
// 404 = no chapter data, returns (0, nil) not error; other failures returned to caller.
func fetchChapterCount(ctx context.Context, w *Wrapper, asin, region string) (int, error) {
	reqURL, err := url.Parse(w.BaseURL + "/books/" + url.PathEscape(asin) + "/chapters")
	if err != nil {
		return 0, err
	}
	q := reqURL.Query()
	q.Set("region", region)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return 0, err
	}

	resp, err := w.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status %d fetching chapters", resp.StatusCode)
	}

	var cl chapterList
	if err := json.NewDecoder(resp.Body).Decode(&cl); err != nil {
		return 0, fmt.Errorf("decoding chapters response: %w", err)
	}
	return len(cl.Chapters), nil
}
