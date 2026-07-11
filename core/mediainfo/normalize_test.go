package mediainfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAudioFormat(t *testing.T) {
	cases := []struct {
		name          string
		path          string
		info          TechnicalInfo
		wantContainer string
		wantCodec     string
	}{
		{
			name:          "m4b aac",
			path:          "book.m4b",
			info:          TechnicalInfo{Codec: "AAC"},
			wantContainer: "M4B",
			wantCodec:     "AAC",
		},
		{
			name:          "mp3",
			path:          "001. Chapter One.mp3",
			info:          TechnicalInfo{Codec: "MPEG Audio", Profile: "Layer 3"},
			wantContainer: "",
			wantCodec:     "MP3",
		},
		{
			name:          "flac",
			path:          "001. Chapter One.flac",
			info:          TechnicalInfo{Codec: "FLAC"},
			wantContainer: "",
			wantCodec:     "FLAC",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			container, codec, err := NormalizeAudioFormat(tc.path, tc.info)
			require.NoError(t, err)
			assert.Equal(t, tc.wantContainer, container)
			assert.Equal(t, tc.wantCodec, codec)
		})
	}
}

func TestNormalizeAudioFormat_UnknownFormatErrors(t *testing.T) {
	_, _, err := NormalizeAudioFormat("book.wav", TechnicalInfo{Codec: "PCM"})
	assert.ErrorIs(t, err, ErrUnknownAudioFormat)
}

func TestNormalizeAudioFormat_AACWithoutM4BExtensionErrors(t *testing.T) {
	_, _, err := NormalizeAudioFormat("book.mp4", TechnicalInfo{Codec: "AAC"})
	assert.ErrorIs(t, err, ErrUnknownAudioFormat)
}
