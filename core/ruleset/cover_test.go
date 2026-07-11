package ruleset

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"testing"

	"github.com/znth-cx/zentag/core/cover"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noiseJPEG returns a w x h JPEG of random noise. Noise is near-incompressible, so large dims reliably exceed cover.MaxBytes.
func noiseJPEG(t *testing.T, w, h, quality int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewSource(1))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(r.Intn(256)), G: uint8(r.Intn(256)), B: uint8(r.Intn(256)), A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}))
	return buf.Bytes()
}

func TestCheckCover_MissingCover(t *testing.T) {
	violations := CheckCover(context.Background(), &metadata.Metadata{})
	assert.Len(t, violations, 1)
	assert.Equal(t, "cover", violations[0].Rule)
	assert.Equal(t, "missing cover", violations[0].Message)
}

func TestCheckCover_OversizedRejected(t *testing.T) {
	img := noiseJPEG(t, 2000, 2000, 90)
	require.Greater(t, len(img), cover.MaxBytes, "fixture must exceed MaxBytes for this test to be meaningful")

	violations := CheckCover(context.Background(), &metadata.Metadata{CoverImage: img})
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, "exceeds")
}

func TestCheckCover_ValidSmallJPEGClean(t *testing.T) {
	img := noiseJPEG(t, 20, 20, 90)
	assert.Empty(t, CheckCover(context.Background(), &metadata.Metadata{CoverImage: img}))
}
