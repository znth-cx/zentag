package mediainfo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// TechnicalInfo holds track technical facts from mediainfo.
type TechnicalInfo struct {
	Container string
	Codec     string
	Profile   string
	Bitrate   int // kbps
}

// mediainfoTrack represents one track from mediainfo JSON output.
type mediainfoTrack struct {
	Type     string `json:"@type"`
	Format   string `json:"Format"`
	Profile  string `json:"Format_Profile"`
	BitRate  string `json:"BitRate"`
	Title    string `json:"Title"`
	Genre    string `json:"Genre"`
	Language string `json:"Language"`
	// Comment is a named General field (©cmt atom), not in Extra. Fallback needed for description round-trip.
	Comment string            `json:"Comment"`
	Extra   map[string]string `json:"extra"`
}

type mediainfoOutput struct {
	Media struct {
		Track []mediainfoTrack `json:"track"`
	} `json:"media"`
}

func (w *Wrapper) runAndFindTracks(ctx context.Context, path string) (general, audio *mediainfoTrack, err error) {
	out, err := w.Runner.Run(ctx, w.BinPath, []string{"--Output=JSON", path})
	if err != nil {
		slog.ErrorContext(ctx, "mediainfo run failed", "path", path, "error", err, "output", string(out))
		return nil, nil, fmt.Errorf("mediainfo %q: %w", path, err)
	}

	var parsed mediainfoOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		slog.ErrorContext(ctx, "mediainfo output parse failed", "path", path, "error", err)
		return nil, nil, fmt.Errorf("mediainfo %q: parse JSON: %w", path, err)
	}

	for i := range parsed.Media.Track {
		t := &parsed.Media.Track[i]
		switch t.Type {
		case "General":
			general = t
		case "Audio":
			audio = t
		}
	}
	if general == nil {
		return nil, nil, fmt.Errorf("mediainfo %q: no General track in output", path)
	}
	if audio == nil {
		return nil, nil, fmt.Errorf("mediainfo %q: no Audio track in output", path)
	}

	return general, audio, nil
}

// Dump runs mediainfo against path with no special output flags and
// returns its default human-readable text report verbatim, for display
// rather than parsing.
func (w *Wrapper) Dump(ctx context.Context, path string) (string, error) {
	slog.DebugContext(ctx, "mediainfo dump starting", "path", path)

	out, err := w.Runner.Run(ctx, w.BinPath, []string{path})
	if err != nil {
		slog.ErrorContext(ctx, "mediainfo dump failed", "path", path, "error", err, "output", string(out))
		return "", fmt.Errorf("mediainfo dump %q: %w", path, err)
	}

	slog.DebugContext(ctx, "mediainfo dump succeeded", "path", path)
	return string(out), nil
}

// Probe queries mediainfo for container, codec, and bitrate.
func (w *Wrapper) Probe(ctx context.Context, path string) (TechnicalInfo, error) {
	slog.DebugContext(ctx, "mediainfo probe starting", "path", path)

	general, audio, err := w.runAndFindTracks(ctx, path)
	if err != nil {
		return TechnicalInfo{}, fmt.Errorf("mediainfo probe: %w", err)
	}

	// Defensive parse: some mediainfo variants emit "128000.0" or stray whitespace.
	raw := strings.TrimSpace(audio.BitRate)
	bitrateBps, err := strconv.Atoi(raw)
	if err != nil {
		f, ferr := strconv.ParseFloat(raw, 64)
		if ferr != nil {
			return TechnicalInfo{}, fmt.Errorf("mediainfo probe %q: parse Audio BitRate %q: %w", path, audio.BitRate, err)
		}
		bitrateBps = int(f)
	}

	info := TechnicalInfo{
		Container: general.Format,
		Codec:     audio.Format,
		Profile:   audio.Profile,
		Bitrate:   (bitrateBps + 500) / 1000,
	}
	slog.DebugContext(ctx, "mediainfo probe succeeded", "path", path, "info", info)
	return info, nil
}
