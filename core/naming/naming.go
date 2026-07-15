package naming

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/znth-cx/zentag/core/lang"
	"github.com/znth-cx/zentag/core/metadata"
)

// chapterNumberPrefix matches leading digit-based chapter numbers like "01." or "1." to strip before use as ChapterName, avoiding duplication.
var chapterNumberPrefix = regexp.MustCompile(`^\d+[.)]\s*`)

// checkTracks verifies all tracks share Container/Codec/Bitrate per RULES.md §5.
func checkTracks(meta *metadata.Metadata) error {
	if len(meta.Tracks) == 0 {
		return errors.New("naming: metadata has no tracks")
	}
	first := meta.Tracks[0]
	for _, tr := range meta.Tracks[1:] {
		if tr.Container != first.Container || tr.Codec != first.Codec || tr.Bitrate != first.Bitrate {
			return fmt.Errorf(
				"naming: inconsistent Container/Codec/Bitrate across tracks (mixed file types not valid): track %q has %s/%s/%dkbps, expected %s/%s/%dkbps",
				tr.Path, tr.Container, tr.Codec, tr.Bitrate, first.Container, first.Codec, first.Bitrate,
			)
		}
	}
	return nil
}

// languageToken renders language as uppercase ISO-639-3 code, per RULES.md §3, falling back to uppercased input if unresolvable.
func languageToken(language string) string {
	if code, ok := lang.ResolveNameOrCode(language); ok {
		return strings.ToUpper(code)
	}
	return strings.ToUpper(language)
}

// directorySegments builds the text before and after the optional edition
// slot in a directory/track name, per RULES.md §3:
//
//	prefix = "Author - Title (Year) Language"
//	suffix = "{Narrator} [Source] Container Codec Bitrate"
//
// Edition, when present, is joined between the two (see baseName). Both
// segments share baseName's validation so callers that only need one stay
// consistent with DirectoryName.
func directorySegments(meta *metadata.Metadata) (prefix, suffix string, err error) {
	if len(meta.Author) == 0 || meta.Author[0] == "" {
		return "", "", errors.New("naming: metadata has no primary author")
	}
	if meta.Title == "" {
		return "", "", errors.New("naming: metadata has no title")
	}
	if len(meta.Narrator) == 0 || meta.Narrator[0] == "" {
		return "", "", errors.New("naming: metadata has no primary narrator")
	}
	if err := checkTracks(meta); err != nil {
		return "", "", err
	}

	track := meta.Tracks[0]
	title := TitleCase(meta.Title, meta.Language)

	prefix = fmt.Sprintf("%s - %s (%d) %s", meta.Author[0], title, meta.Year, languageToken(meta.Language))

	rest := []string{fmt.Sprintf("{%s} [%s]", meta.Narrator[0], meta.Source)}
	if track.Container != "" {
		rest = append(rest, track.Container)
	}
	rest = append(rest, fmt.Sprintf("%s %dkbps", track.Codec, track.Bitrate))
	suffix = strings.Join(rest, " ")
	return prefix, suffix, nil
}

// baseName builds shared prefix per RULES.md §3: "Author - Title (Year) Language Edition {Narrator} [Source] Container Codec Bitrate".
func baseName(meta *metadata.Metadata) (string, error) {
	prefix, suffix, err := directorySegments(meta)
	if err != nil {
		return "", err
	}
	parts := []string{prefix}
	if meta.Edition != "" {
		parts = append(parts, meta.Edition)
	}
	parts = append(parts, suffix)
	return strings.Join(parts, " "), nil
}

// DetectEdition checks whether actualDir is the directory name DirectoryName
// would produce for meta but with an extra edition token inserted between the
// language and the narrator — the case where the on-disk name carries an
// edition (e.g. "Love Lane", "Abridged") that the metadata does not. It
// returns the detected edition and true when the prefix and suffix match
// exactly; "" and false otherwise. Call only when meta.Edition is unset.
func DetectEdition(ctx context.Context, meta *metadata.Metadata, actualDir string) (string, bool) {
	prefix, suffix, err := directorySegments(meta)
	if err != nil {
		return "", false
	}
	prefix = sanitize(ctx, prefix)
	suffix = sanitize(ctx, suffix)

	pre := prefix + " "
	post := " " + suffix
	if !strings.HasPrefix(actualDir, pre) || !strings.HasSuffix(actualDir, post) {
		return "", false
	}
	if len(actualDir) < len(pre)+len(post) {
		return "", false
	}
	edition := strings.TrimSpace(actualDir[len(pre) : len(actualDir)-len(post)])
	if edition == "" {
		return "", false
	}
	return edition, true
}

// DirectoryName builds RULES.md §3's directory name for meta.
func DirectoryName(ctx context.Context, meta *metadata.Metadata) (string, error) {
	slog.DebugContext(ctx, "naming: building directory name", "path", meta.OriginalPath)

	name, err := baseName(meta)
	if err != nil {
		slog.ErrorContext(ctx, "naming: directory name build failed", "path", meta.OriginalPath, "error", err)
		return "", err
	}

	clean := sanitize(ctx, name)
	slog.DebugContext(ctx, "naming: directory name built", "name", clean)
	return clean, nil
}

// TrackName builds file name per RULES.md §3: DirectoryName for single-file, or "PartNumber. ChapterName - Title (Year)" for multi-file.
func TrackName(ctx context.Context, meta *metadata.Metadata, trackIndex int) (string, error) {
	slog.DebugContext(ctx, "naming: building track name", "path", meta.OriginalPath, "trackIndex", trackIndex)

	if trackIndex < 0 || trackIndex >= len(meta.Tracks) {
		err := fmt.Errorf("naming: trackIndex %d out of range (have %d tracks)", trackIndex, len(meta.Tracks))
		slog.ErrorContext(ctx, "naming: track name build failed", "path", meta.OriginalPath, "error", err)
		return "", err
	}

	if len(meta.Tracks) == 1 {
		return DirectoryName(ctx, meta)
	}

	if err := checkTracks(meta); err != nil {
		slog.ErrorContext(ctx, "naming: track name build failed", "path", meta.OriginalPath, "error", err)
		return "", err
	}
	if meta.Title == "" {
		err := errors.New("naming: metadata has no title")
		slog.ErrorContext(ctx, "naming: track name build failed", "path", meta.OriginalPath, "error", err)
		return "", err
	}

	track := meta.Tracks[trackIndex]
	title := TitleCase(meta.Title, meta.Language)

	maxPart := 0
	for _, tr := range meta.Tracks {
		if tr.PartNumber > maxPart {
			maxPart = tr.PartNumber
		}
	}
	width := len(strconv.Itoa(maxPart))
	partStr := fmt.Sprintf("%0*d", width, track.PartNumber)

	chapterName := fmt.Sprintf("Chapter %d", track.PartNumber)
	if len(track.Chapters) > 0 && track.Chapters[0].Title != "" {
		chapterName = chapterNumberPrefix.ReplaceAllString(track.Chapters[0].Title, "")
	}

	name := fmt.Sprintf("%s. %s - %s (%d)", partStr, chapterName, title, meta.Year)
	clean := sanitize(ctx, name)
	slog.DebugContext(ctx, "naming: track name built", "name", clean)
	return clean, nil
}
