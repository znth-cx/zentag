// Package session persists working Metadata to enable resuming interrupted transforms.
package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/znth-cx/zentag/core/metadata"
)

// file is the on-disk envelope with Metadata and provenance (slugified filename is lossy).
type file struct {
	Path     string             `json:"path"`
	SavedAt  time.Time          `json:"saved_at"`
	Metadata *metadata.Metadata `json:"metadata"`
}

// Path returns the session-file path for itemPath, resolving to absolute first.
func Path(sessionDir, itemPath string) (string, error) {
	abs, err := filepath.Abs(itemPath)
	if err != nil {
		return "", fmt.Errorf("session: resolve %q: %w", itemPath, err)
	}
	return filepath.Join(sessionDir, fileName(abs)), nil
}

func fileName(abs string) string {
	var b strings.Builder
	b.Grow(len(abs) + len(".json"))
	for _, r := range abs {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	slug := b.String()
	const maxSlug = 200 // leave headroom under the common 255-byte limit
	if len(slug) > maxSlug {
		// hash full path so truncated slugs sharing a prefix stay unique
		sum := sha256.Sum256([]byte(abs))
		slug = slug[:maxSlug-9] + "_" + hex.EncodeToString(sum[:4])
	}
	return slug + ".json"
}

// Save writes metadata to itemPath's session file, creating sessionDir if needed.
func Save(ctx context.Context, sessionDir, itemPath string, m *metadata.Metadata) error {
	p, err := Path(sessionDir, itemPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("session: create %q: %w", sessionDir, err)
	}

	data, err := json.MarshalIndent(file{Path: itemPath, SavedAt: time.Now(), Metadata: m}, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal %q: %w", itemPath, err)
	}
	// atomic: temp file in same dir, then rename over target
	tmp, err := os.CreateTemp(sessionDir, ".zentag-*.tmp")
	if err != nil {
		return fmt.Errorf("session: temp for %q: %w", p, err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("session: write %q: %w", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("session: close %q: %w", tmp.Name(), err)
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("session: rename %q: %w", p, err)
	}
	slog.DebugContext(ctx, "session: saved", "item", itemPath, "file", p)
	return nil
}

// Load reads itemPath's saved session, returning (nil, false, nil) if not found.
func Load(ctx context.Context, sessionDir, itemPath string) (m *metadata.Metadata, found bool, err error) {
	p, err := Path(sessionDir, itemPath)
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("session: read %q: %w", p, err)
	}

	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, false, fmt.Errorf("session: parse %q: %w", p, err)
	}
	if f.Metadata == nil {
		return nil, false, fmt.Errorf("session: %q has no metadata", p)
	}
	slog.DebugContext(ctx, "session: loaded", "item", itemPath, "file", p, "saved_at", f.SavedAt)
	return f.Metadata, true, nil
}

// Clean removes itemPath's session file. Missing files are not an error.
func Clean(ctx context.Context, sessionDir, itemPath string) error {
	p, err := Path(sessionDir, itemPath)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("session: remove %q: %w", p, err)
	}
	slog.DebugContext(ctx, "session: cleaned", "item", itemPath, "file", p)
	return nil
}
