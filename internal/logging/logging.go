package logging

import (
	"log/slog"
	"os"
)

// New returns a text-handler slog.Logger writing to stderr.
// Passing verbose=true lowers the level to Debug; otherwise it is Info.
func New(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
