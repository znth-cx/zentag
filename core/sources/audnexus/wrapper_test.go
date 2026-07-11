package audnexus

import "testing"

func TestNew_DefaultsBaseURL(t *testing.T) {
	w := New("")
	if w.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", w.BaseURL, defaultBaseURL)
	}
	if w.Client == nil {
		t.Error("Client is nil, want default *http.Client")
	}
}

func TestNew_CustomBaseURL(t *testing.T) {
	w := New("https://example.test")
	if w.BaseURL != "https://example.test" {
		t.Errorf("BaseURL = %q, want %q", w.BaseURL, "https://example.test")
	}
}
