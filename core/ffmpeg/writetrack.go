package ffmpeg

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/znth-cx/zentag/core/metadata"
)

// WriteOpts holds inputs WriteTrack needs to write one track's tags and chapters.
type WriteOpts struct {
	InputPath  string
	OutputPath string
	Metadata   *metadata.Metadata
	Track      metadata.Track

	// SkipMetadata drops metadata during remux (-map_metadata -1), skips metadataArgs. Only M4BEngine sets this: mp4tag writes M4B tags/cover in a later in-place pass, so ffmpeg's only job for M4B is chapters.
	SkipMetadata bool

	// EmbedChapters: M4B-only. M4B embeds chapters in-container; MP3/FLAC chapter identity comes from file/part naming instead.
	EmbedChapters bool
}

// buildArgs assembles ffmpeg args for opts (stream copy, no re-encode). Cleanup removes chapter temp files; callers must call it after run, success or failure.
func buildArgs(opts WriteOpts) (args []string, cleanup func(), err error) {
	var cleanups []func()
	cleanup = func() {
		for _, c := range cleanups {
			c()
		}
	}

	args = []string{"-y", "-i", opts.InputPath}
	chapterInputIdx := -1

	if opts.EmbedChapters && len(opts.Track.Chapters) > 0 {
		path, cf, err := writeChapterFile(opts.Track.Chapters)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("build chapter args: %w", err)
		}
		cleanups = append(cleanups, cf)
		args = append(args, "-i", path)
		chapterInputIdx = 1 // second -i input
	}

	args = append(args, "-map", "0:a")
	if opts.SkipMetadata {
		if chapterInputIdx != -1 {
			// -map_chapters (mov muxer, stream-copy) only writes an orphaned legacy chpl atom, no real QT chapter track: mediainfo/ffprobe can't resolve it. -map_metadata on the chapter temp file (implicit carry-over) works instead. Safe: that file (see writeChapterFile) holds only [CHAPTER] blocks, no global tags, so nothing leaks into mp4tag's clean slate.
			args = append(args, "-map_metadata", strconv.Itoa(chapterInputIdx))
		} else {
			// M4BEngine's chapters-only remux: mp4tag writes tags afterward, nothing from source worth carrying over here.
			args = append(args, "-map_metadata", "-1")
		}
	} else {
		// Global metadata always comes from input 0, never the chapter temp file: that file holds only chapter markers, so -map_metadata on it would drop every tag ffmpeg doesn't itself write. Chapters are pulled separately via -map_chapters, so replacing them doesn't touch global metadata.
		args = append(args, "-map_metadata", "0")
		if chapterInputIdx != -1 {
			args = append(args, "-map_chapters", strconv.Itoa(chapterInputIdx))
		}
	}

	if opts.Metadata.Language != "" {
		// Stream-level (unlike metadataArgs' global -metadata language=): mov/mp4 muxer packs this into the audio track's mdhd language field. mp4tag can't touch mdhd, so this always runs here, even for M4B under SkipMetadata.
		args = append(args, "-metadata:s:a:0", "language="+opts.Metadata.Language)
	}

	args = append(args, "-c", "copy")
	if !opts.SkipMetadata {
		args = append(args, metadataArgs(opts.Metadata)...)
	}

	args = append(args, opts.OutputPath)

	return args, cleanup, nil
}

// samePath reports whether two absolute paths address the same file. Windows and macOS default filesystems are case-insensitive, so fold there; Linux compares exact.
func samePath(a, b string) bool {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// WriteTrack writes opts.Metadata's tags and chapters onto opts.InputPath into opts.OutputPath; audio always stream-copied, never re-encoded.
func (w *Wrapper) WriteTrack(ctx context.Context, opts WriteOpts) error {
	slog.DebugContext(ctx, "ffmpeg write track starting", "input", opts.InputPath, "output", opts.OutputPath)

	// -y would clobber the source in place; sources are never modified.
	inAbs, err := filepath.Abs(opts.InputPath)
	if err != nil {
		return fmt.Errorf("writetrack: resolve input path %q: %w", opts.InputPath, err)
	}
	outAbs, err := filepath.Abs(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("writetrack: resolve output path %q: %w", opts.OutputPath, err)
	}
	if samePath(inAbs, outAbs) {
		return fmt.Errorf("writetrack: output path equals input path %q, refusing to overwrite source", opts.InputPath)
	}
	// Windows MAX_PATH: long author/title names can push output past 260 chars; fail early with a clear error instead of a cryptic ffmpeg one.
	if runtime.GOOS == "windows" && len(outAbs) >= 260 {
		return fmt.Errorf("writetrack: output path %d chars exceeds Windows 260-char limit, use a shorter output directory", len(outAbs))
	}

	args, cleanup, err := buildArgs(opts)
	if err != nil {
		slog.ErrorContext(ctx, "ffmpeg build args failed", "input", opts.InputPath, "error", err)
		return err
	}
	defer cleanup()

	slog.DebugContext(ctx, "ffmpeg args built", "args", args)

	out, err := w.Runner.Run(ctx, w.BinPath, args)
	if err != nil {
		slog.ErrorContext(ctx, "ffmpeg run failed", "input", opts.InputPath, "error", err, "output", string(out))
		return fmt.Errorf("ffmpeg write track %q: %w", opts.InputPath, err)
	}

	slog.InfoContext(ctx, "ffmpeg write track succeeded", "input", opts.InputPath, "output", opts.OutputPath)
	return nil
}
