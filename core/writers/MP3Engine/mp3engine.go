// Package MP3Engine is a thin MP3 writer adapter over core/ffmpeg and core/mp3tag.
// MP3 items are multi-file. No cover or chapters embedded: loose cover.jpg
// required (RULES.md §8), chapter identity from file/part naming.
// Uses two-pass approach: ffmpeg copies audio stream, go-taglib writes ID3v2.4-compliant tags.
package MP3Engine

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/znth-cx/zentag/core/ffmpeg"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/mp3tag"
	"github.com/znth-cx/zentag/core/writers"
)

// writeMP3Tags holds mp3tag.WriteTags for test injection.
var writeMP3Tags = mp3tag.WriteTags

// Write tags each of meta's tracks into outputDir using two-pass approach:
// 1. ffmpeg copies audio stream (no metadata)
// 2. go-taglib writes ID3v2.4-compliant tags
// On failure, removes outputs already written so no partial item remains.
func Write(ctx context.Context, w *ffmpeg.Wrapper, meta *metadata.Metadata, outputDir string, trackNames []string) error {
	if len(trackNames) != len(meta.Tracks) {
		return fmt.Errorf("MP3Engine.Write: got %d track names for %d tracks", len(trackNames), len(meta.Tracks))
	}

	var written []string
	for i, track := range meta.Tracks {
		outPath := filepath.Join(outputDir, trackNames[i]+filepath.Ext(track.Path))
		// Record before writing so a partial file is cleaned up on failure.
		written = append(written, outPath)

		if err := w.WriteTrack(ctx, ffmpeg.WriteOpts{
			InputPath:     track.Path,
			OutputPath:    outPath,
			Metadata:      meta,
			Track:         track,
			SkipMetadata:  true,
			EmbedChapters: false,
		}); err != nil {
			writers.RemoveOutputs(ctx, "MP3Engine.Write", written)
			return err
		}

		if err := writeMP3Tags(ctx, outPath, meta, track); err != nil {
			writers.RemoveOutputs(ctx, "MP3Engine.Write", written)
			return err
		}
	}
	return nil
}
