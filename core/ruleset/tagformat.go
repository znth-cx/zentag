package ruleset

import (
	"fmt"
	"strings"

	"github.com/znth-cx/zentag/core/metadata"
)

// CheckTagSeparators validates multi-field tag separator format per RULES.md §4.
// Multi-field tags must use `;` as separator with proper escaping (`\` for `;` and `\`).
func CheckTagSeparators(meta *metadata.Metadata) []Violation {
	if meta == nil || len(meta.Tracks) == 0 {
		return nil
	}

	container := meta.Tracks[0].Container
	multiFields, ok := MultiValueTags[container]
	if !ok {
		return nil
	}

	var violations []Violation

	for _, field := range multiFields {
		switch field {
		case "author":
			for i, value := range meta.Author {
				violations = append(violations, validateSeparatorFormat("author", value, i)...)
			}
		case "narrator":
			for i, value := range meta.Narrator {
				violations = append(violations, validateSeparatorFormat("narrator", value, i)...)
			}
		case "genre":
			for i, value := range meta.Genre {
				violations = append(violations, validateSeparatorFormat("genre", value, i)...)
			}
		case "publisher":
			for i, value := range meta.Publisher {
				violations = append(violations, validateSeparatorFormat("publisher", value, i)...)
			}
		case "series":
			for i, s := range meta.Series {
				violations = append(violations, validateSeparatorFormat("series", s.Name, i)...)
			}
		}
	}

	return violations
}

// validateSeparatorFormat validates a tag value uses proper separator format per RULES.md §4.
// - Multi-field tags must use `;` as separator
// - If tag value contains `;`, it must be escaped as `\;`
// - If tag value contains `\`, it must be escaped as `\\`
func validateSeparatorFormat(fieldName string, value string, index int) []Violation {
	var violations []Violation

	if value == "" {
		return nil
	}

	i := 0
	for i < len(value) {
		if value[i] == '\\' {
			if i+1 >= len(value) {
				violations = append(violations, Violation{
					Rule:     "tag_separator_format",
					Severity: SeverityTrumpable,
					Message:  fmt.Sprintf("%s[%d]: backslash at end of string without escape", fieldName, index),
				})
				break
			}

			nextChar := value[i+1]
			if nextChar != '\\' && nextChar != ';' {
				violations = append(violations, Violation{
					Rule:     "tag_separator_format",
					Severity: SeverityTrumpable,
					Message:  fmt.Sprintf("%s[%d]: unescaped backslash before '%c' (should be \\\\ or \\;)", fieldName, index, nextChar),
				})
			}
			i += 2
		} else if value[i] == ';' {
			violations = append(violations, Violation{
				Rule:     "tag_separator_format",
				Severity: SeverityTrumpable,
				Message:  fmt.Sprintf("%s[%d]: unescaped semicolon (should be \\; for literal semicolon)", fieldName, index),
			})
			i++
		} else {
			i++
		}
	}

	return violations
}

// parseMultiFieldTag parses a multi-field tag value with proper escaping per RULES.md §4.
// Returns individual field values after unescaping.
// Examples:
//   - "Author One;Author Two" -> ["Author One", "Author Two"]
//   - "Author\\;With;Semicolon" -> ["Author;With", "Semicolon"]
//   - "Author\\\\Backslash" -> ["Author\\Backslash"]
//   - "Author\\;One;Author\\\\Two" -> ["Author;One", "Author\\Two"]
func parseMultiFieldTag(value string) []string {
	if value == "" {
		return []string{}
	}

	var parts []string
	var current strings.Builder

	i := 0
	for i < len(value) {
		if value[i] == '\\' && i+1 < len(value) {
			nextChar := value[i+1]
			if nextChar == '\\' || nextChar == ';' {
				current.WriteByte(nextChar)
				i += 2
				continue
			}
		}

		if value[i] == ';' {
			parts = append(parts, current.String())
			current.Reset()
			i++
			continue
		}

		current.WriteByte(value[i])
		i++
	}

	parts = append(parts, current.String())
	return parts
}
