// Package FLACEngine is a thin FLAC writer adapter over core/ffmpeg.
// FLAC items are multi-file. No cover or chapters embedded: loose cover.jpg
// required (RULES.md §8), chapter identity from file/part naming.
package FLACEngine

import (
	"context"

	"codeberg.org/Ether/zentag/core/ffmpeg"
	"codeberg.org/Ether/zentag/core/metadata"
	"codeberg.org/Ether/zentag/core/writers"
)

// Write tags each of meta's tracks into outputDir.
// On failure, removes outputs already written so no partial item remains.
func Write(ctx context.Context, w *ffmpeg.Wrapper, meta *metadata.Metadata, outputDir string, trackNames []string) error {
	return writers.WriteTracks(ctx, w, meta, outputDir, trackNames, "FLACEngine.Write")
}
