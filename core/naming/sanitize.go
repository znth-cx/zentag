package naming

import (
	"context"
	"log/slog"
	"strings"
)

// illegalCharReplacements maps filesystem-illegal characters (Windows/exFAT) to safe replacements, or "" to strip entirely.
var illegalCharReplacements = map[rune]string{
	':':  " -",
	'\\': "",
	'/':  "-",
	'*':  "",
	'?':  "",
	'"':  "'",
	'<':  "",
	'>':  "",
	'|':  "-",
}

// windowsReservedNames are device names Windows forbids as file/dir names, with or without extension.
var windowsReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// sanitize replaces filesystem-illegal characters in name with safe substitutes, logging a warning if any change occurs.
func sanitize(ctx context.Context, name string) string {
	var b strings.Builder
	changed := false
	for _, r := range name {
		if repl, illegal := illegalCharReplacements[r]; illegal {
			b.WriteString(repl)
			changed = true
			continue
		}
		b.WriteRune(r)
	}

	clean := b.String()

	// Windows: trailing dots/spaces invalid in file/dir names.
	if trimmed := strings.TrimRight(clean, ". "); trimmed != clean {
		clean = trimmed
		changed = true
	}

	// Empty after trimming: fall back.
	if clean == "" {
		clean = "_"
		changed = true
	}

	// Windows reserved device names, case-insensitive, extension ignored.
	base := clean
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	if windowsReservedNames[strings.ToUpper(base)] {
		clean = "_" + clean
		changed = true
	}

	if changed {
		slog.WarnContext(ctx, "naming: sanitized illegal characters", "original", name, "sanitized", clean)
	}
	return clean
}
