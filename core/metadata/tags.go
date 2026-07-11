package metadata

import "strings"

// JoinTags joins values per RULES.md §4 with ';', escaping '\' as '\\' and ';' as '\;' (backslash first, so it isn't re-escaped).
func JoinTags(values []string) string {
	escaped := make([]string, len(values))
	for i, v := range values {
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `;`, `\;`)
		escaped[i] = v
	}
	return strings.Join(escaped, ";")
}

// SeriesNamesParts splits entries into parallel name/part slices for writers' multi-value tag fields.
func SeriesNamesParts(entries []SeriesEntry) (names, parts []string) {
	names = make([]string, len(entries))
	parts = make([]string, len(entries))
	for i, s := range entries {
		names[i] = s.Name
		parts[i] = s.Part
	}
	return names, parts
}

// SplitTags is JoinTags's inverse; returns nil for an empty string, so SplitTags(JoinTags(nil)) round-trips to nil.
func SplitTags(joined string) []string {
	if joined == "" {
		return nil
	}

	var values []string
	var current strings.Builder
	escaped := false
	for _, r := range joined {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case ';':
			values = append(values, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	values = append(values, current.String())

	return values
}
