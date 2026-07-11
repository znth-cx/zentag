package mediainfo

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrUnknownAudioFormat returned when track's Codec/Profile doesn't match supported container/codec combinations.
var ErrUnknownAudioFormat = errors.New("mediainfo: unknown audio format")

// NormalizeAudioFormat maps path extension and audio codec to format tokens: container (M4B or empty) and codec (AAC, MP3, or FLAC).
func NormalizeAudioFormat(path string, info TechnicalInfo) (container, codec string, err error) {
	format := strings.ToLower(info.Codec)
	profile := strings.ToLower(info.Profile)

	switch {
	case strings.Contains(format, "aac") && strings.EqualFold(filepath.Ext(path), ".m4b"):
		return "M4B", "AAC", nil
	case strings.Contains(format, "mpeg audio") && strings.Contains(profile, "layer 3"):
		return "", "MP3", nil
	case strings.Contains(format, "flac"):
		return "", "FLAC", nil
	default:
		return "", "", fmt.Errorf("%w: path %q, format %q, profile %q", ErrUnknownAudioFormat, path, info.Codec, info.Profile)
	}
}
