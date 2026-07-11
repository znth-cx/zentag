package cover

import (
	"bytes"
	"context"
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeDims(t *testing.T, img []byte) (w, h int) {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(img))
	require.NoError(t, err)
	return cfg.Width, cfg.Height
}

func TestResize_AlreadyValidJPEG_PassThrough(t *testing.T) {
	in := encodeJPEGBytes(t, noiseImage(t, 20, 20), 90)
	out, err := Resize(context.Background(), in)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(in, out), "expected byte-identical pass-through")
}

func TestResize_AlreadyValidPNG_PassThrough(t *testing.T) {
	in := encodePNGBytes(t, noiseImage(t, 20, 20))
	out, err := Resize(context.Background(), in)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(in, out), "expected byte-identical pass-through")
}

func TestResize_OversizedJPEG_DownscalesAndStaysJPEG(t *testing.T) {
	in := encodeJPEGBytes(t, noiseImage(t, 2000, 2000), 90)
	require.Greater(t, len(in), MaxBytes)

	out, err := Resize(context.Background(), in)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(out), MaxBytes)

	_, format, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)

	w, h := decodeDims(t, out)
	assert.Less(t, w*h, 2000*2000, "expected dimensions to shrink")
}

func TestResize_OversizedPNG_ConvertsToJPEG(t *testing.T) {
	in := encodePNGBytes(t, noiseImage(t, 2000, 2000))
	require.Greater(t, len(in), MaxBytes)

	out, err := Resize(context.Background(), in)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(out), MaxBytes)

	_, format, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
}

func TestResize_SmallGIF_ConvertsToJPEG_SameDimensions(t *testing.T) {
	in := encodeGIFBytes(t, noiseImage(t, 30, 30))

	out, err := Resize(context.Background(), in)
	require.NoError(t, err)

	_, format, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)

	w, h := decodeDims(t, out)
	assert.Equal(t, 30, w)
	assert.Equal(t, 30, h)
}

func TestResize_SmallBMP_ConvertsToJPEG(t *testing.T) {
	in := encodeBMPBytes(t, noiseImage(t, 30, 30))

	out, err := Resize(context.Background(), in)
	require.NoError(t, err)

	_, format, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
}

func TestResize_OversizedGIF_DownscalesToJPEG(t *testing.T) {
	in := encodeGIFBytes(t, noiseImage(t, 2000, 2000))
	require.Greater(t, len(in), MaxBytes, "fixture must exceed MaxBytes for this test to be meaningful")

	out, err := Resize(context.Background(), in)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(out), MaxBytes)

	w, h := decodeDims(t, out)
	assert.Less(t, w*h, 2000*2000)
}

func TestResize_CorruptInput_Errors(t *testing.T) {
	_, err := Resize(context.Background(), []byte("not an image"))
	assert.Error(t, err)
}

func TestResize_CancelledContext_Errors(t *testing.T) {
	in := encodeJPEGBytes(t, noiseImage(t, 2000, 2000), 90)
	require.Greater(t, len(in), MaxBytes)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Resize(ctx, in)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestResize_UnreachableCap_Errors(t *testing.T) {
	old := minLongestEdge
	minLongestEdge = 10000 // bigger than the fixture, forces the "too small to downscale" branch
	defer func() { minLongestEdge = old }()

	in := encodeJPEGBytes(t, noiseImage(t, 2000, 2000), 90)
	require.Greater(t, len(in), MaxBytes)

	_, err := Resize(context.Background(), in)
	assert.Error(t, err)
}
