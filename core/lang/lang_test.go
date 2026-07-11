package lang

import "testing"

func TestValidCode(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"eng", true},
		{"ENG", true},
		{"en", false},  // ISO 639-1, not accepted (639-3 only)
		{"enc", false}, // banned: real code but a typo trap for "en"
		{"ENC", false}, // banned, case-insensitive
		{"xx", false},
		{"", false},
	}
	for _, c := range cases {
		if got := ValidCode(c.code); got != c.want {
			t.Errorf("ValidCode(%q) = %v, want %v", c.code, got, c.want)
		}
	}
}

func TestCodeForName_Lowercase(t *testing.T) {
	code, ok := CodeForName("english")
	if !ok || code != "eng" {
		t.Errorf("CodeForName(%q) = (%q, %v), want (%q, true)", "english", code, ok, "eng")
	}
}

func TestCodeForName_MixedCase(t *testing.T) {
	code, ok := CodeForName("English")
	if !ok || code != "eng" {
		t.Errorf("CodeForName(%q) = (%q, %v), want (%q, true)", "English", code, ok, "eng")
	}
}

func TestCodeForName_Unknown(t *testing.T) {
	_, ok := CodeForName("not-a-real-language")
	if ok {
		t.Error("CodeForName(unknown) ok = true, want false")
	}
}

func TestCodeForName_BannedNameRejected(t *testing.T) {
	_, ok := CodeForName("En")
	if ok {
		t.Error(`CodeForName("En") ok = true, want false (banned code "enc")`)
	}
}

func TestValidNameOrCode(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"eng", true},
		{"ENG", true},
		{"English", true},
		{"english", true},
		{"en", true},   // ISO 639-1 alias for English, accepted
		{"EN", true},   // alias, case-insensitive
		{"En", true},   // alias wins over the banned name "En" (code "enc")
		{"enc", false}, // banned code, still rejected
		{"not-a-language", false},
	}
	for _, c := range cases {
		if got := ValidNameOrCode(c.s); got != c.want {
			t.Errorf("ValidNameOrCode(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestResolveNameOrCode(t *testing.T) {
	cases := []struct {
		s        string
		wantCode string
		wantOK   bool
	}{
		{"eng", "eng", true},
		{"ENG", "eng", true},
		{"en", "eng", true},
		{"En", "eng", true}, // alias wins over the banned name
		{"English", "eng", true},
		{"enc", "", false}, // banned code, still rejected
		{"xx", "", false},
	}
	for _, c := range cases {
		got, ok := ResolveNameOrCode(c.s)
		if got != c.wantCode || ok != c.wantOK {
			t.Errorf("ResolveNameOrCode(%q) = (%q, %v), want (%q, %v)", c.s, got, ok, c.wantCode, c.wantOK)
		}
	}
}

func TestNormalizeToPart3(t *testing.T) {
	cases := []struct {
		code     string
		wantCode string
		wantOK   bool
	}{
		{"en", "eng", true},  // ISO 639-1 -> 639-3
		{"EN", "eng", true},  // case-insensitive
		{"ger", "deu", true}, // ISO 639-2/B -> 639-3
		{"eng", "eng", true}, // already 639-3, unchanged
		{"deu", "deu", true}, // already 639-3, unchanged
		{"zz", "", false},    // unknown
		{"", "", false},      // empty
		{"nope", "", false},  // wrong length
	}
	for _, c := range cases {
		got, ok := NormalizeToPart3(c.code)
		if got != c.wantCode || ok != c.wantOK {
			t.Errorf("NormalizeToPart3(%q) = (%q, %v), want (%q, %v)", c.code, got, ok, c.wantCode, c.wantOK)
		}
	}
}
