// Package M4BEngine is a thin M4B writer adapter over core/ffmpeg and core/mp4tag.
// M4B items are always single-file and embed chapters and cover art in-container (RULES.md §8/§9).
// LANDMINE: ffmpeg's mov/mp4 muxer corrupts attached-picture when zentag tags written in same pass.
// Solution: ffmpeg remuxes chapters only (SkipMetadata: true), then mp4tag writes tags+cover in-place.
package M4BEngine

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/znth-cx/zentag/core/ffmpeg"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/mp4tag"
	"github.com/znth-cx/zentag/core/writers"
)

// writeMP4Tags holds mp4tag.WriteTags for test injection.
var writeMP4Tags = mp4tag.WriteTags

// Write embeds chapters, tags, and cover for meta's single track into outputDir.
func Write(ctx context.Context, w *ffmpeg.Wrapper, meta *metadata.Metadata, outputDir string, trackNames []string) error {
	if len(meta.Tracks) != 1 {
		return fmt.Errorf("M4BEngine.Write: expected exactly 1 track, got %d", len(meta.Tracks))
	}
	if len(trackNames) != 1 {
		return fmt.Errorf("M4BEngine.Write: expected exactly 1 track name, got %d", len(trackNames))
	}

	track := meta.Tracks[0]
	outputPath := filepath.Join(outputDir, trackNames[0]+filepath.Ext(track.Path))

	if err := w.WriteTrack(ctx, ffmpeg.WriteOpts{
		InputPath:     track.Path,
		OutputPath:    outputPath,
		Metadata:      meta,
		Track:         track,
		SkipMetadata:  true,
		EmbedChapters: true,
	}); err != nil {
		return err
	}

	if err := writeMP4Tags(ctx, outputPath, meta); err != nil {
		// Remove the remuxed output too, matching FLAC/MP3: no partial item remains.
		writers.RemoveOutputs(ctx, "M4BEngine.Write", []string{outputPath})
		return err
	}
	return nil
}
