package writers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/znth-cx/zentag/core/ffmpeg"
	"github.com/znth-cx/zentag/core/metadata"
)

// WriteTracks tags each of meta's tracks into outputDir, the shared body of the
// multi-file engines (MP3/FLAC). On failure, removes outputs already written so
// no partial item remains. engine names the caller in errors and logs.
func WriteTracks(ctx context.Context, w *ffmpeg.Wrapper, meta *metadata.Metadata, outputDir string, trackNames []string, engine string) error {
	if len(trackNames) != len(meta.Tracks) {
		return fmt.Errorf("%s: got %d track names for %d tracks", engine, len(trackNames), len(meta.Tracks))
	}
	var written []string
	for i, track := range meta.Tracks {
		outPath := filepath.Join(outputDir, trackNames[i]+filepath.Ext(track.Path))
		err := w.WriteTrack(ctx, ffmpeg.WriteOpts{
			InputPath:     track.Path,
			OutputPath:    outPath,
			Metadata:      meta,
			Track:         track,
			EmbedChapters: false,
		})
		if err != nil {
			RemoveOutputs(ctx, engine, written)
			return err
		}
		written = append(written, outPath)
	}
	return nil
}
