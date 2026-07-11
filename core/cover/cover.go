// Package cover validates and fixes audiobook cover images per RULES.md §8 (JPEG/PNG, <3MB).
package cover

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// MaxBytes is RULES.md §8's hard cap: covers must be under 3MB.
const MaxBytes = 3 * 1024 * 1024

// Validate reports whether img is JPEG/PNG under MaxBytes (RULES.md §8).
func Validate(ctx context.Context, img []byte) (ok bool, reason string) {
	_, format, err := image.DecodeConfig(bytes.NewReader(img))

	var problems []string
	if err != nil || (format != "jpeg" && format != "png") {
		problems = append(problems, fmt.Sprintf("format %q is not jpeg or png", format))
	}
	if len(img) > MaxBytes {
		problems = append(problems, fmt.Sprintf("size %d bytes exceeds %d byte limit", len(img), MaxBytes))
	}

	if len(problems) > 0 {
		reason = strings.Join(problems, "; ")
		slog.DebugContext(ctx, "cover validate failed", "reason", reason)
		return false, reason
	}
	slog.DebugContext(ctx, "cover validate passed", "bytes", len(img))
	return true, ""
}
