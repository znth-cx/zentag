// Package naming builds RULES.md §3 directory and track file names
// from a metadata.Metadata object. Pure string-builder, no filesystem
// access; callers own creating directories and renaming files.
package naming

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// apaMinorWords are RULES.md §3's APA title-case minor words: articles, conjunctions, prepositions of 3 letters or fewer, lowercased unless first/last or after colon.
var apaMinorWords = map[string]bool{
	"a": true, "an": true, "the": true,
	"and": true, "but": true, "or": true, "nor": true, "for": true, "if": true, "so": true, "yet": true,
	"as": true, "at": true, "by": true, "in": true, "of": true, "off": true,
	"on": true, "per": true, "to": true, "up": true, "via": true,
}

// TitleCase renders title in APA title case (RULES.md §3) with language rules; acronyms and internal capitals are lowercased.
func TitleCase(title, lang string) string {
	tag, err := language.Parse(lang)
	if err != nil {
		tag = language.Und
	}
	titler := cases.Title(tag)
	lower := cases.Lower(tag)

	words := strings.Fields(title)
	if len(words) == 0 {
		return title
	}

	afterColon := false
	for i, w := range words {
		isFirst := i == 0
		isLast := i == len(words)-1
		bare := strings.Trim(w, ".,;:!?")
		minor := apaMinorWords[strings.ToLower(bare)]

		if isFirst || isLast || afterColon || !minor {
			words[i] = titler.String(w)
		} else {
			words[i] = lower.String(w)
		}

		afterColon = strings.HasSuffix(w, ":")
	}

	return strings.Join(words, " ")
}
