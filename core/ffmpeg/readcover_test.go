package ffmpeg

import (
	"context"
	"errors"
	"os"
	"testing"
)

// writingRunner simulates ffmpeg writing extracted image bytes to output path (last arg), for ReadCover tests needing real bytes.
type writingRunner struct {
	gotBinPath string
	gotArgs    []string
	writeBytes []byte // nil = simulate ffmpeg failure (no stream to extract)
}

func (w *writingRunner) Run(_ context.Context, binPath string, args []string) ([]byte, error) {
	w.gotBinPath = binPath
	w.gotArgs = args
	if w.writeBytes == nil {
		return []byte("ffmpeg: Output file does not contain any stream"), errors.New("exit status 1")
	}
	outPath := args[len(args)-1]
	if err := os.WriteFile(outPath, w.writeBytes, 0o644); err != nil {
		return nil, err
	}
	return nil, nil
}

func TestReadCover_HappyPath(t *testing.T) {
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	wr := &writingRunner{writeBytes: jpegBytes}
	w := &Wrapper{BinPath: "ffmpeg", Runner: wr}

	image, mime, err := w.ReadCover(context.Background(), "book.m4b")
	if err != nil {
		t.Fatalf("ReadCover() error = %v", err)
	}
	if string(image) != string(jpegBytes) {
		t.Errorf("image = %v, want %v", image, jpegBytes)
	}
	if mime != "image/jpeg" {
		t.Errorf("mime = %q, want %q", mime, "image/jpeg")
	}

	if wr.gotBinPath != "ffmpeg" {
		t.Errorf("binPath = %q, want %q", wr.gotBinPath, "ffmpeg")
	}
	if wr.gotArgs[0] != "-y" || wr.gotArgs[2] != "book.m4b" {
		t.Errorf("unexpected args = %q", wr.gotArgs)
	}

	// temp file must be cleaned up
	outPath := wr.gotArgs[len(wr.gotArgs)-1]
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Errorf("temp file %q not cleaned up", outPath)
	}
}

func TestReadCover_NoCoverIsNotAnError(t *testing.T) {
	wr := &writingRunner{writeBytes: nil}
	w := &Wrapper{BinPath: "ffmpeg", Runner: wr}

	image, mime, err := w.ReadCover(context.Background(), "book.mp3")
	if err != nil {
		t.Fatalf("ReadCover() error = %v, want nil (no cover is not an error)", err)
	}
	if image != nil {
		t.Errorf("image = %v, want nil", image)
	}
	if mime != "" {
		t.Errorf("mime = %q, want empty", mime)
	}
}
