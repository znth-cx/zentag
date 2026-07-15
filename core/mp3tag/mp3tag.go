package mp3tag

import (
	"context"
	"fmt"
	"log/slog"

	"go.senan.xyz/taglib"

	"github.com/znth-cx/zentag/core/metadata"
)

// WriteTags writes ID3v2.4-compliant tags to MP3 file at path using go-taglib.
// Uses merge mode (0 flag) to preserve existing tags not being written.
func WriteTags(ctx context.Context, path string, m *metadata.Metadata, track metadata.Track) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	slog.DebugContext(ctx, "mp3tag write starting", "path", path)

	tags := buildID3Tags(m, track)

	if err := taglib.WriteTags(path, tags, 0); err != nil {
		slog.ErrorContext(ctx, "mp3tag write failed", "path", path, "error", err)
		return fmt.Errorf("mp3tag write %q: %w", path, err)
	}

	slog.InfoContext(ctx, "mp3tag write succeeded", "path", path)
	return nil
}
