package ffmpeg

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"codeberg.org/Ether/zentag/core/metadata"
)

// chapterMetadataContent builds ffmpeg's FFMETADATA1 chapter format (ms timebase).
func chapterMetadataContent(chapters []metadata.Chapter) string {
	var b strings.Builder
	b.WriteString(";FFMETADATA1\n")
	for i, c := range chapters {
		b.WriteString("[CHAPTER]\n")
		b.WriteString("TIMEBASE=1/1000\n")
		b.WriteString("START=" + strconv.FormatInt(c.Start.Milliseconds(), 10) + "\n")
		b.WriteString("END=" + strconv.FormatInt(c.End.Milliseconds(), 10) + "\n")
		b.WriteString("title=" + escapeFFMeta(c.Title) + "\n")
		if i != len(chapters)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// escapeFFMeta backslash-escapes FFMETADATA1 special chars in values (\ first).
func escapeFFMeta(s string) string {
	r := strings.NewReplacer(`\`, `\\`, "=", `\=`, ";", `\;`, "#", `\#`, "\n", "\\\n")
	return r.Replace(s)
}

func writeChapterFile(chapters []metadata.Chapter) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "zentag-chapters-*.txt")
	if err != nil {
		return "", nil, fmt.Errorf("create chapter temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(chapterMetadataContent(chapters)); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("write chapter temp file: %w", err)
	}

	return f.Name(), func() { os.Remove(f.Name()) }, nil
}
