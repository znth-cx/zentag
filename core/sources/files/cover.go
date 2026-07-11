package files

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg" // register JPEG decoder for image.DecodeConfig
	_ "image/png"  // register PNG decoder for image.DecodeConfig
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/znth-cx/zentag/core/ffmpeg"
)

var looseCoverNames = []string{"cover.jpg", "cover.jpeg", "cover.png"}

// findLooseCover looks for cover.jpg/jpeg/png (case-insensitive) in dir per RULES.md §8, returning nil if not found.
func findLooseCover(dir string) []byte {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		for _, candidate := range looseCoverNames {
			if name == candidate {
				data, err := os.ReadFile(filepath.Join(dir, e.Name()))
				if err != nil {
					return nil
				}
				return data
			}
		}
	}
	return nil
}

// coverPixelCount returns width*height, or 0 if undecidable.
func coverPixelCount(data []byte) int {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0
	}
	return cfg.Width * cfg.Height
}

// selectCover picks path's cover by comparing embedded (first track) vs loose file, keeping the larger; embedded wins ties. For single-file M4B, this fallback is essential.
func selectCover(ctx context.Context, fw *ffmpeg.Wrapper, dir string, isMultiFile bool, primaryTrackPath string) ([]byte, string, error) {
	embeddedImage, embeddedMIME, err := fw.ReadCover(ctx, primaryTrackPath)
	if err != nil {
		return nil, "", err
	}

	looseDir := dir
	if !isMultiFile {
		looseDir = filepath.Dir(primaryTrackPath)
	}

	looseImage := findLooseCover(looseDir)
	if looseImage == nil {
		return embeddedImage, embeddedMIME, nil
	}
	if embeddedImage == nil {
		return looseImage, http.DetectContentType(looseImage), nil
	}

	if coverPixelCount(looseImage) > coverPixelCount(embeddedImage) {
		return looseImage, http.DetectContentType(looseImage), nil
	}
	return embeddedImage, embeddedMIME, nil
}
