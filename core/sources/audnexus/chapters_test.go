package audnexus

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chapterListJSON = `{
  "asin": "B08G9PRS1K",
  "chapters": [
    {"title": "Chapter 1", "lengthMs": 1000, "startOffsetMs": 0, "startOffsetSec": 0},
    {"title": "Chapter 2", "lengthMs": 1000, "startOffsetMs": 1000, "startOffsetSec": 1}
  ]
}`

func TestFetchChapterCount_Success(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "/books/B08G9PRS1K/chapters")
			return jsonResponse(http.StatusOK, chapterListJSON), nil
		},
	}}

	count, err := fetchChapterCount(context.Background(), w, "B08G9PRS1K", "us")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestFetchChapterCount_NotFoundReturnsZero(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusNotFound, ""), nil
		},
	}}

	count, err := fetchChapterCount(context.Background(), w, "B08G9PRS1K", "us")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestFetchChapterCount_UnexpectedStatus(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusInternalServerError, ""), nil
		},
	}}

	_, err := fetchChapterCount(context.Background(), w, "B08G9PRS1K", "us")
	assert.Error(t, err)
}

func TestGather_SetsAudnexusChapterCount(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "/chapters") {
				return jsonResponse(http.StatusOK, chapterListJSON), nil
			}
			return jsonResponse(http.StatusOK, minimalBookJSON), nil
		},
	}}

	got, err := Gather(context.Background(), w, "B08G9PRS1K", "")
	require.NoError(t, err)
	assert.Equal(t, 2, got.AudnexusChapterCount)
}
