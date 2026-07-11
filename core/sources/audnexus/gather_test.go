package audnexus

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tinyPNG: 1x1 red PNG; must decode for real since Gather validates covers.
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode tiny PNG fixture: %v", err)
	}
	return data
}

type fakeHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f.do(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

const minimalBookJSON = `{
  "asin": "B08G9PRS1K",
  "title": "Project Hail Mary",
  "language": "english",
  "formatType": "unabridged"
}`

const bookWithCoverJSON = `{
  "asin": "B08G9PRS1K",
  "title": "Project Hail Mary",
  "language": "english",
  "formatType": "unabridged",
  "image": "https://example.test/cover.jpg"
}`

func TestGather_HappyPath(t *testing.T) {
	cover := tinyPNG(t)
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "/books/") {
				return jsonResponse(http.StatusOK, bookWithCoverJSON), nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(cover))}, nil
		},
	}}

	got, err := Gather(context.Background(), w, "B08G9PRS1K", "")
	require.NoError(t, err)
	assert.Equal(t, "B08G9PRS1K", got.ASIN)
	assert.Equal(t, "Project Hail Mary", got.Title)
	assert.Equal(t, cover, got.CoverImage)
	assert.Equal(t, "image/png", got.CoverMIME)
}

func TestGather_CoverFetchFailure_LeavesCoverNilButSucceeds(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "/books/") {
				return jsonResponse(http.StatusOK, bookWithCoverJSON), nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
		},
	}}

	got, err := Gather(context.Background(), w, "B08G9PRS1K", "")
	require.NoError(t, err)
	assert.Nil(t, got.CoverImage)
}

func TestGather_NoImageURL_SkipsCoverFetch(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, minimalBookJSON), nil
		},
	}}

	got, err := Gather(context.Background(), w, "B08G9PRS1K", "")
	require.NoError(t, err)
	assert.Nil(t, got.CoverImage)
}

func TestGather_DefaultsRegionToUS(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Query().Get("region"), "us")
			return jsonResponse(http.StatusOK, minimalBookJSON), nil
		},
	}}

	_, err := Gather(context.Background(), w, "B08G9PRS1K", "")
	require.NoError(t, err)
}

func TestGather_UsesProvidedRegion(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "uk", req.URL.Query().Get("region"))
			return jsonResponse(http.StatusOK, minimalBookJSON), nil
		},
	}}

	_, err := Gather(context.Background(), w, "B08G9PRS1K", "uk")
	require.NoError(t, err)
}

func TestGather_NotFound_ReturnsErrNotFound(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusNotFound, `{"message":"Bad ASIN"}`), nil
		},
	}}

	_, err := Gather(context.Background(), w, "BADASIN", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestGather_UnexpectedStatus_ReturnsError(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusInternalServerError, `{}`), nil
		},
	}}

	_, err := Gather(context.Background(), w, "B08G9PRS1K", "")
	require.Error(t, err)
}

func TestGather_MalformedJSON_ReturnsError(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{not valid json`), nil
		},
	}}

	_, err := Gather(context.Background(), w, "B08G9PRS1K", "")
	require.Error(t, err)
}

func TestGather_TransportError_ReturnsError(t *testing.T) {
	w := &Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		},
	}}

	_, err := Gather(context.Background(), w, "B08G9PRS1K", "")
	require.Error(t, err)
}
