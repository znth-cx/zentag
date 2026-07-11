package mp4tag

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/znth-cx/zentag/core/metadata"
)

// maxSize caps WriteTags' input: go-mp4tag seeks/rewrites atoms directly rather than streaming, so huge files risk pathological memory/time use.
var maxSize int64 = 2 << 30 // 2 GiB

// WriteTags writes m's tags/cover onto path in place; M4B/M4A only, MP3/FLAC must go through core/ffmpeg instead.
func WriteTags(ctx context.Context, path string, m *metadata.Metadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	slog.DebugContext(ctx, "mp4tag write starting", "path", path, "cover", len(m.CoverImage) > 0)

	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("mp4tag %q: %w", path, err)
	}
	if fi.Size() > maxSize {
		return fmt.Errorf("mp4tag %q: %d bytes exceeds the %d-byte limit the in-process tagger can handle", path, fi.Size(), maxSize)
	}

	if err := write(path, buildTags(m)); err != nil {
		slog.ErrorContext(ctx, "mp4tag write failed", "path", path, "error", err)
		return fmt.Errorf("mp4tag write %q: %w", path, err)
	}

	slog.InfoContext(ctx, "mp4tag write succeeded", "path", path)
	return nil
}
