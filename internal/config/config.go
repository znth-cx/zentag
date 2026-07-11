package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the zentag runtime configuration.
type Config struct {
	FFmpegPath    string `mapstructure:"ffmpeg_path"`
	FFprobePath   string `mapstructure:"ffprobe_path"`
	MediaInfoPath string `mapstructure:"mediainfo_path"`
	OutputDir     string `mapstructure:"output_dir"`
	// SessionDir holds per-item session JSON for resuming after crashes. Removed only via --clean.
	SessionDir string `mapstructure:"session_dir"`
}

// Load reads config from cfgFile or searches "." and user config dir for "zentag.yaml".
// If not found and cfgFile is empty, writes a default to user config dir as a starting point.
func Load(cfgFile string) (*Config, error) {
	v := viper.New()
	v.SetDefault("ffmpeg_path", "ffmpeg")
	v.SetDefault("ffprobe_path", "ffprobe")
	v.SetDefault("mediainfo_path", "mediainfo")
	v.SetDefault("output_dir", "./zentag-output")
	// Prefer user config dir for session_dir so resume works from any cwd. Fallback is Abs'd after unmarshal.
	if configDir, err := os.UserConfigDir(); err == nil {
		v.SetDefault("session_dir", filepath.Join(configDir, "zentag", "sessions"))
	} else {
		v.SetDefault("session_dir", "./zentag-sessions")
	}

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("zentag")
		// LANDMINE: SetConfigType matches extensionless files like "./zentag" binary.
		// Leaving unset forces ".yaml" extension match, avoiding false positives.
		v.AddConfigPath(".")
		if configDir, err := os.UserConfigDir(); err == nil {
			v.AddConfigPath(filepath.Join(configDir, "zentag"))
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		// Explicit path must exist and parse. Auto-discovery may miss, defaults OK.
		if cfgFile != "" || !errors.As(err, &notFound) {
			return nil, fmt.Errorf("reading config: %w", err)
		}

		if err := writeDefaultConfig(v); err != nil {
			return nil, fmt.Errorf("writing default config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Resolve dirs to absolute so behavior is cwd-independent across runs.
	for _, dir := range []*string{&cfg.OutputDir, &cfg.SessionDir} {
		abs, err := filepath.Abs(*dir)
		if err != nil {
			return nil, fmt.Errorf("resolving path %q: %w", *dir, err)
		}
		*dir = abs
	}
	return &cfg, nil
}

// writeDefaultConfig writes default config to user config dir so user has an editable starting point. No-op if user config dir unavailable.
func writeDefaultConfig(v *viper.Viper) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		slog.Warn("skipping default config write, user config dir unavailable", "error", err)
		return nil
	}

	dir := filepath.Join(configDir, "zentag")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return v.SafeWriteConfigAs(filepath.Join(dir, "zentag.yaml"))
}
