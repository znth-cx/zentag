package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinTags(t *testing.T) {
	cases := []struct {
		name   string
		values []string
		want   string
	}{
		{"empty", nil, ""},
		{"single", []string{"Brandon Sanderson"}, "Brandon Sanderson"},
		{"multiple", []string{"Author A", "Author B"}, "Author A;Author B"},
		{"escapes semicolon", []string{"Smith; Jr"}, `Smith\; Jr`},
		{"escapes backslash", []string{`C:\books`}, `C:\\books`},
		{"escapes both", []string{`a;b\c`}, `a\;b\\c`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, JoinTags(tc.values))
		})
	}
}

func TestSplitTags(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single", "Brandon Sanderson", []string{"Brandon Sanderson"}},
		{"multiple", "Author A;Author B", []string{"Author A", "Author B"}},
		{"unescapes semicolon", `Smith\; Jr`, []string{"Smith; Jr"}},
		{"unescapes backslash", `C:\\books`, []string{`C:\books`}},
		{"unescapes both", `a\;b\\c`, []string{`a;b\c`}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, SplitTags(tc.in))
		})
	}
}

func TestJoinSplitTagsRoundTrip(t *testing.T) {
	cases := [][]string{
		nil,
		{"Solo Value"},
		{"Author A", "Author B", "Author C"},
		{"Smith; Jr", `C:\books`, "a;b\\c"},
	}

	for _, values := range cases {
		assert.Equal(t, values, SplitTags(JoinTags(values)))
	}
}
