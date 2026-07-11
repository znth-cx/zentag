package audnexus

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func nopCloserBytes(b []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(b))
}

func TestFetchCover_Success(t *testing.T) {
	pngBytes := []byte("\x89PNG\r\n\x1a\n" + "rest of a fake png")
	client := &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://example.test/cover.jpg", req.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       nopCloserBytes(pngBytes),
			}, nil
		},
	}

	data, mime, err := fetchCover(context.Background(), client, "https://example.test/cover.jpg")
	require.NoError(t, err)
	assert.Equal(t, pngBytes, data)
	assert.Equal(t, "image/png", mime)
}

func TestFetchCover_UnexpectedStatus(t *testing.T) {
	client := &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusNotFound, Body: nopCloserBytes(nil)}, nil
		},
	}

	_, _, err := fetchCover(context.Background(), client, "https://example.test/cover.jpg")
	require.Error(t, err)
}

func TestFetchCover_TransportError(t *testing.T) {
	client := &fakeHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		},
	}

	_, _, err := fetchCover(context.Background(), client, "https://example.test/cover.jpg")
	require.Error(t, err)
}
