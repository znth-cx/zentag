package cover

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_SmallJPEG(t *testing.T) {
	img := encodeJPEGBytes(t, noiseImage(t, 20, 20), 90)
	ok, reason := Validate(context.Background(), img)
	assert.True(t, ok, "reason: %s", reason)
}

func TestValidate_SmallPNG(t *testing.T) {
	img := encodePNGBytes(t, noiseImage(t, 20, 20))
	ok, reason := Validate(context.Background(), img)
	assert.True(t, ok, "reason: %s", reason)
}

func TestValidate_OversizedJPEG(t *testing.T) {
	img := encodeJPEGBytes(t, noiseImage(t, 2000, 2000), 90)
	require.Greater(t, len(img), MaxBytes, "fixture must exceed MaxBytes for this test to be meaningful")

	ok, reason := Validate(context.Background(), img)
	assert.False(t, ok)
	assert.Contains(t, reason, "exceeds")
}

func TestValidate_WrongFormat_GIF(t *testing.T) {
	img := encodeGIFBytes(t, noiseImage(t, 20, 20))
	ok, reason := Validate(context.Background(), img)
	assert.False(t, ok)
	assert.Contains(t, reason, "not jpeg or png")
}

func TestValidate_CorruptBytes(t *testing.T) {
	ok, reason := Validate(context.Background(), []byte("not an image"))
	assert.False(t, ok)
	assert.Contains(t, reason, "not jpeg or png")
}
