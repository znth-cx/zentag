package cover

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log/slog"

	"golang.org/x/image/draw"
)

const jpegQuality = 90

// minLongestEdge is the floor dimension (pixels) for downscale binary search. Var not const so tests can force floor scale errors.
var minLongestEdge = 200

// Resize returns cover passing Validate, re-encoding as JPEG and downscaling if needed via binary search.
func Resize(ctx context.Context, img []byte) ([]byte, error) {
	if ok, _ := Validate(ctx, img); ok {
		slog.DebugContext(ctx, "cover resize: input already valid, passing through", "bytes", len(img))
		return img, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	decoded, _, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		slog.DebugContext(ctx, "cover resize: decode failed", "error", err)
		return nil, fmt.Errorf("cover: decode: %w", err)
	}

	full, err := encodeJPEG(decoded, jpegQuality)
	if err != nil {
		return nil, err
	}
	if len(full) <= MaxBytes {
		slog.DebugContext(ctx, "cover resize: fit at original size", "bytes", len(full))
		return full, nil
	}

	bounds := decoded.Bounds()
	longest := bounds.Dx()
	if bounds.Dy() > longest {
		longest = bounds.Dy()
	}
	floorScale := float64(minLongestEdge) / float64(longest)
	if floorScale >= 1 {
		slog.DebugContext(ctx, "cover resize: image too small to downscale under cap", "longest_edge", longest, "min_longest_edge", minLongestEdge)
		return nil, fmt.Errorf("cover: image too small to downscale further and still exceeds %d bytes", MaxBytes)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	floorEncoded, err := encodeJPEG(scaleImage(decoded, floorScale), jpegQuality)
	if err != nil {
		return nil, err
	}
	if len(floorEncoded) > MaxBytes {
		slog.DebugContext(ctx, "cover resize: still over cap at floor scale", "scale", floorScale, "bytes", len(floorEncoded))
		return nil, fmt.Errorf("cover: cannot shrink under %d bytes even at floor scale", MaxBytes)
	}

	lo, hi := floorScale, 1.0
	best := floorEncoded
	for i := 0; i < 10; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		mid := (lo + hi) / 2
		enc, err := encodeJPEG(scaleImage(decoded, mid), jpegQuality)
		if err != nil {
			return nil, err
		}
		slog.DebugContext(ctx, "cover resize: downscale trial", "scale", mid, "bytes", len(enc))
		if len(enc) <= MaxBytes {
			best = enc
			lo = mid
		} else {
			hi = mid
		}
	}
	slog.DebugContext(ctx, "cover resize: downscale done", "scale", lo, "bytes", len(best))
	return best, nil
}

func scaleImage(img image.Image, scale float64) image.Image {
	b := img.Bounds()
	w := int(float64(b.Dx()) * scale)
	if w < 1 {
		w = 1
	}
	h := int(float64(b.Dy()) * scale)
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("cover: jpeg encode: %w", err)
	}
	return buf.Bytes(), nil
}
