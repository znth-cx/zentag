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
	Type           string `json:"@type"`
	Format         string `json:"Format"`
	Profile        string `json:"Format_Profile"`
	BitRate        string `json:"BitRate"`
	BitRateNominal string `json:"BitRate_Nominal"`
	StreamSize     string `json:"StreamSize"`
	Duration       string `json:"Duration"`
	Title          string `json:"Title"`
	Genre          string `json:"Genre"`
	Language       string `json:"Language"`
	// Comment is a named General field (©cmt atom), not in Extra. Fallback needed for description round-trip.
	Comment string   `json:"Comment"`
	Extra   extraMap `json:"extra"`
}

// extraMap decodes mediainfo's "extra" object, coercing array values (a
// repeated atom emits an array) to strings so lookups stay simple.
type extraMap map[string]string

func (m *extraMap) UnmarshalJSON(b []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = coerceExtraValue(v)
	}
	*m = out
	return nil
}

// coerceExtraValue renders an extra value as a string; arrays are deduped
// and joined with ";", non-strings fall back to raw JSON unquoted.
func coerceExtraValue(v json.RawMessage) string {
	var s string
	if json.Unmarshal(v, &s) == nil {
		return s
	}
	var arr []string
	if json.Unmarshal(v, &arr) == nil {
		seen := make(map[string]bool, len(arr))
		var uniq []string
		for _, a := range arr {
			if !seen[a] {
				seen[a] = true
				uniq = append(uniq, a)
			}
		}
		return strings.Join(uniq, ";")
	}
	return strings.Trim(string(v), `"`)
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

	// BitRate absent on VBR encodes; fall back to nominal, then stream size/duration, else 0.
	bitrateBps := parseBitrateBps(audio.BitRate, audio.BitRateNominal)
	if bitrateBps == 0 {
		bitrateBps = bitrateFromStreamSize(audio.StreamSize, audio.Duration)
	}
	if bitrateBps == 0 {
		slog.WarnContext(ctx, "mediainfo probe: no bitrate reported, defaulting to 0", "path", path)
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

// parseBitrateBps returns the first parseable bps value (int or float), or 0.
func parseBitrateBps(vals ...string) int {
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return int(f)
		}
	}
	return 0
}

// bitrateFromStreamSize derives bps from byte count and duration; 0 if either is unusable.
func bitrateFromStreamSize(streamSize, duration string) int {
	bytes, err := strconv.ParseFloat(strings.TrimSpace(streamSize), 64)
	if err != nil || bytes <= 0 {
		return 0
	}
	secs, err := strconv.ParseFloat(strings.TrimSpace(duration), 64)
	if err != nil || secs <= 0 {
		return 0
	}
	return int(bytes * 8 / secs)
}
