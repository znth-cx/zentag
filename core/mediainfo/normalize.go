package mediainfo

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrUnsupportedAudioFormat returned when track's Codec/Profile doesn't match a supported container/codec combination.
var ErrUnsupportedAudioFormat = errors.New("mediainfo: unsupported audio format")

// NormalizeAudioFormat maps extension + codec to RULES.md container/codec tokens.
// M4B codecs: AAC, or Atmos-slot Dolby DD/DDP/TrueHD (E-AC-3 JOC folds into DDP).
func NormalizeAudioFormat(path string, info TechnicalInfo) (container, codec string, err error) {
	format := strings.ToLower(info.Codec)
	profile := strings.ToLower(info.Profile)
	isM4B := strings.EqualFold(filepath.Ext(path), ".m4b")

	switch {
	case isM4B && strings.Contains(format, "aac"):
		return "M4B", "AAC", nil
	// E-AC-3 before AC-3: "e-ac-3" contains "ac-3".
	case isM4B && strings.Contains(format, "e-ac-3"):
		return "M4B", "DDP", nil
	case isM4B && strings.Contains(format, "ac-3"):
		return "M4B", "DD", nil
	case isM4B && (strings.Contains(format, "truehd") || strings.Contains(format, "mlp")):
		return "M4B", "TrueHD", nil
	case strings.Contains(format, "mpeg audio") && strings.Contains(profile, "layer 3"):
		return "", "MP3", nil
	case strings.Contains(format, "flac"):
		return "", "FLAC", nil
	default:
		return "", "", fmt.Errorf("%w: path %q, format %q, profile %q (supported: AAC/DD/DDP/TrueHD in M4B, MP3, FLAC)", ErrUnsupportedAudioFormat, path, info.Codec, info.Profile)
	}
}
