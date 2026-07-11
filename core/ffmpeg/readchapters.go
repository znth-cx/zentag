package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"codeberg.org/Ether/zentag/core/metadata"
)

type ffprobeChaptersOutput struct {
	Chapters []struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Tags      struct {
			Title string `json:"title"`
		} `json:"tags"`
	} `json:"chapters"`
}

// ReadChapters runs ffprobe on path, returns its chapters; no chapters returns empty slice, not error (RULES.md §9).
func (w *Wrapper) ReadChapters(ctx context.Context, path string) ([]metadata.Chapter, error) {
	slog.DebugContext(ctx, "ffprobe read chapters starting", "path", path)

	// -v quiet suppresses ffprobe's stderr banner; realRunner merges stdout+stderr, so the banner would break JSON parsing below.
	out, err := w.Runner.Run(ctx, w.ProbeBinPath, []string{"-v", "quiet", "-show_chapters", "-print_format", "json", path})
	if err != nil {
		slog.ErrorContext(ctx, "ffprobe run failed", "path", path, "error", err, "output", string(out))
		return nil, fmt.Errorf("ffprobe read chapters %q: %w", path, err)
	}

	var parsed ffprobeChaptersOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		slog.ErrorContext(ctx, "ffprobe output parse failed", "path", path, "error", err)
		return nil, fmt.Errorf("ffprobe read chapters %q: parse JSON: %w", path, err)
	}

	chapters := make([]metadata.Chapter, 0, len(parsed.Chapters))
	for _, c := range parsed.Chapters {
		start, err := parseSeconds(c.StartTime)
		if err != nil {
			return nil, fmt.Errorf("ffprobe read chapters %q: parse start_time %q: %w", path, c.StartTime, err)
		}
		end, err := parseSeconds(c.EndTime)
		if err != nil {
			return nil, fmt.Errorf("ffprobe read chapters %q: parse end_time %q: %w", path, c.EndTime, err)
		}
		chapters = append(chapters, metadata.Chapter{
			Title: c.Tags.Title,
			Start: start,
			End:   end,
		})
	}

	slog.DebugContext(ctx, "ffprobe read chapters succeeded", "path", path, "count", len(chapters))
	return chapters, nil
}

// ProbeDump runs ffprobe on path, returns its raw report for display, not parsing.
func (w *Wrapper) ProbeDump(ctx context.Context, path string) (string, error) {
	slog.DebugContext(ctx, "ffprobe dump starting", "path", path)

	out, err := w.Runner.Run(ctx, w.ProbeBinPath, []string{"-hide_banner", "-show_format", "-show_streams", "-show_chapters", path})
	if err != nil {
		slog.ErrorContext(ctx, "ffprobe dump failed", "path", path, "error", err, "output", string(out))
		return "", fmt.Errorf("ffprobe dump %q: %w", path, err)
	}

	slog.DebugContext(ctx, "ffprobe dump succeeded", "path", path)
	return string(out), nil
}

func parseSeconds(s string) (time.Duration, error) {
	seconds, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds * float64(time.Second)), nil
}
