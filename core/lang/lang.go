// Package lang wraps github.com/barbashov/iso639-3 for validating and
// looking up ISO 639-3 language codes.
package lang

import (
	"strings"

	iso6393 "github.com/barbashov/iso639-3"
)

// bannedCodes are real ISO 639-3 codes rejected because they trap users.
// "enc" (Cameroonian "En") is indistinguishable from typo'd "en" (English ISO 639-1).
var bannedCodes = map[string]bool{
	"enc": true,
}

// ValidCode reports whether code is a known ISO 639-3 three-letter language code, excluding bannedCodes.
func ValidCode(code string) bool {
	code = strings.ToLower(code)
	if bannedCodes[code] {
		return false
	}
	return iso6393.FromPart3Code(code) != nil
}

// NormalizeToPart3 converts ISO 639-1/639-2 codes to ISO 639-3, returning unchanged if already valid.
// Used for best-effort normalization of existing file tags, not validation (use ValidCode for that).
func NormalizeToPart3(code string) (string, bool) {
	l := iso6393.FromAnyCode(strings.ToLower(code))
	if l == nil {
		return "", false
	}
	return l.Part3, true
}

// englishAlias accepts "en" as English despite RULES.md requiring ISO-639-3. No ambiguity unlike other two-letter codes.
const englishAlias = "en"

// ValidNameOrCode reports whether s is acceptable as Language per RULES.md §4: valid ISO-639-3 code, English name, or "en" alias.
func ValidNameOrCode(s string) bool {
	_, ok := ResolveNameOrCode(s)
	return ok
}

// ResolveNameOrCode returns the ISO-639-3 code s should map to per RULES.md §4: valid 639-3, English name, or "en" alias.
func ResolveNameOrCode(s string) (string, bool) {
	if strings.EqualFold(s, englishAlias) {
		return "eng", true
	}
	if ValidCode(s) {
		return strings.ToLower(s), true
	}
	return CodeForName(s)
}

// CodeForName looks up ISO 639-3 code for English language name ("English" -> "eng"), skipping bannedCodes.
func CodeForName(name string) (string, bool) {
	for _, l := range iso6393.LanguagesPart3 {
		if bannedCodes[l.Part3] {
			continue
		}
		if strings.EqualFold(l.Name, name) {
			return l.Part3, true
		}
	}
	return "", false
}
