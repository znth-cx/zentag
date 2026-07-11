// Package writers holds helpers shared by the per-format engine packages.
package writers

import (
	"context"
	"log/slog"
	"os"
)

// RemoveOutputs deletes outputs written before a failed write so no partial item remains. Remove errors warn, never fail.
func RemoveOutputs(ctx context.Context, engine string, paths []string) {
	for _, p := range paths {
		if err := os.Remove(p); err != nil {
			slog.WarnContext(ctx, engine+": cleanup of partial output failed", "path", p, "error", err)
		}
	}
}
