// Package audnexus wraps the audnexus API (https://api.audnex.us) for looking up book metadata by ASIN.
package audnexus

import "net/http"

const defaultBaseURL = "https://api.audnex.us"

// HTTPClient is the subset of *http.Client Gather needs; injectable for test fakes.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Wrapper calls the audnexus API and maps its responses to
// metadata.Metadata.
type Wrapper struct {
	BaseURL string
	Client  HTTPClient
}

// New returns a Wrapper for baseURL (default: public audnexus API), using a real *http.Client.
func New(baseURL string) *Wrapper {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Wrapper{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}
