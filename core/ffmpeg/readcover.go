package ffmpeg

import (
	"context"
	"log/slog"
	"net/http"
	"os"
)

// ReadCover extracts path's embedded cover via ffmpeg; failure (no video stream) means no cover, returns nil/"", not error.
func (w *Wrapper) ReadCover(ctx context.Context, path string) (image []byte, mime string, err error) {
	slog.DebugContext(ctx, "ffmpeg read cover starting", "path", path)

	f, err := os.CreateTemp("", "zentag-cover-read-*.img")
	if err != nil {
		return nil, "", err
	}
	tempPath := f.Name()
	f.Close()
	defer os.Remove(tempPath)

	args := []string{"-y", "-i", path, "-an", "-c:v", "copy", "-f", "image2", tempPath}
	out, runErr := w.Runner.Run(ctx, w.BinPath, args)
	if runErr != nil {
		slog.DebugContext(ctx, "ffmpeg read cover found no cover", "path", path, "output", string(out))
		return nil, "", nil
	}

	data, err := os.ReadFile(tempPath)
	if err != nil {
		return nil, "", err
	}

	slog.DebugContext(ctx, "ffmpeg read cover succeeded", "path", path, "bytes", len(data))
	return data, http.DetectContentType(data), nil
}
