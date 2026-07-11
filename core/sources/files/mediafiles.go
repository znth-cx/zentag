package files

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// mediaFile is one resolved file with its part number (0 for single-file).
type mediaFile struct {
	Path       string
	PartNumber int
}

var mediaExtensions = map[string]bool{
	".m4b":  true,
	".mp3":  true,
	".flac": true,
}

var leadingDigits = regexp.MustCompile(`^\d+`)

var bracketedSource = regexp.MustCompile(`\[([^\[\]]+)\]`)

// PartNumberError reports filename part-number extraction failure. Transform's TUI uses errors.As to offer custom pattern prompt.
type PartNumberError struct {
	File string
	Err  error
}

func (e *PartNumberError) Error() string {
	return fmt.Sprintf("file %q: part number: %v", e.File, e.Err)
}

func (e *PartNumberError) Unwrap() error { return e.Err }

// matchPartNumber extracts name's part number: leading digits if re is nil, otherwise re's first capture group.
func matchPartNumber(name string, re *regexp.Regexp) (int, error) {
	if re == nil {
		match := leadingDigits.FindString(name)
		if match == "" {
			return 0, fmt.Errorf("no leading part number")
		}
		return strconv.Atoi(match)
	}

	sub := re.FindStringSubmatch(name)
	if sub == nil {
		return 0, fmt.Errorf("pattern does not match")
	}
	if len(sub) < 2 {
		return 0, fmt.Errorf("pattern has no capture group")
	}
	n, err := strconv.Atoi(sub[1])
	if err != nil {
		return 0, fmt.Errorf("captured %q is not a number", sub[1])
	}
	return n, nil
}

// mediaFilenames lists path's media files (extension-filtered), ignoring directories and other files.
func mediaFilenames(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("gather %q: %w", path, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !mediaExtensions[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		names = append(names, e.Name())
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("gather %q: no media files found", path)
	}
	return names, nil
}

// PartNumberPreview is one filename's outcome for a candidate part-number pattern; Part is only valid when Err is nil.
type PartNumberPreview struct {
	Filename string
	Part     int
	Err      error
}

// PreviewPartNumbers matches re's first capture group against all media filenames in path, showing what part numbers result. Per-file errors in each preview.
func PreviewPartNumbers(path string, re *regexp.Regexp) ([]PartNumberPreview, error) {
	names, err := mediaFilenames(path)
	if err != nil {
		return nil, err
	}

	results := make([]PartNumberPreview, len(names))
	for i, name := range names {
		part, matchErr := matchPartNumber(name, re)
		results[i] = PartNumberPreview{Filename: name, Part: part, Err: matchErr}
	}
	return results, nil
}

// sourceFromName extracts RULES.md §3's "[Source]" token from name, returning "" if absent.
func sourceFromName(name string) string {
	m := bracketedSource.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	return m[1]
}

// resolveMediaFiles resolves path into media files: single file returns PartNumber 0; multi-file requires part numbers for ordering. partNumberRe is nil for default leading-digits or a caller-validated pattern.
func resolveMediaFiles(path string, info os.FileInfo, partNumberRe *regexp.Regexp) ([]mediaFile, error) {
	if !info.IsDir() {
		return []mediaFile{{Path: path, PartNumber: 0}}, nil
	}

	names, err := mediaFilenames(path)
	if err != nil {
		return nil, err
	}

	if len(names) == 1 {
		return []mediaFile{{Path: filepath.Join(path, names[0]), PartNumber: 0}}, nil
	}

	var files []mediaFile
	seen := make(map[int]string, len(names)) // part number -> filename, for duplicate detection
	for _, name := range names {
		part, matchErr := matchPartNumber(name, partNumberRe)
		if matchErr != nil {
			return nil, fmt.Errorf("gather %q: %w", path, &PartNumberError{File: name, Err: matchErr})
		}
		if prev, ok := seen[part]; ok {
			return nil, fmt.Errorf("gather %q: duplicate part number %d: %q and %q", path, part, prev, name)
		}
		seen[part] = name
		files = append(files, mediaFile{Path: filepath.Join(path, name), PartNumber: part})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].PartNumber < files[j].PartNumber })

	return files, nil
}
