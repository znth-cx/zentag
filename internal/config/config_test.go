package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultsWhenNoConfigFile(t *testing.T) {
	t.Chdir(t.TempDir())
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("APPDATA", homeDir)

	wd, err := os.Getwd()
	require.NoError(t, err)

	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "ffmpeg", cfg.FFmpegPath)
	assert.Equal(t, "ffprobe", cfg.FFprobePath)
	assert.Equal(t, "mediainfo", cfg.MediaInfoPath)
	// Dirs resolved to absolute at load time.
	assert.Equal(t, filepath.Join(wd, "zentag-output"), cfg.OutputDir)
	assert.True(t, filepath.IsAbs(cfg.SessionDir))
}

func TestLoad_ExplicitConfigFileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "custom.yaml")
	outDir := filepath.Join(dir, "out")
	content := []byte("ffmpeg_path: /usr/local/bin/ffmpeg\nffprobe_path: /usr/local/bin/ffprobe\nmediainfo_path: /usr/local/bin/mediainfo\noutput_dir: " + filepath.ToSlash(outDir) + "\n")
	require.NoError(t, os.WriteFile(cfgPath, content, 0o644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "/usr/local/bin/ffmpeg", cfg.FFmpegPath)
	assert.Equal(t, "/usr/local/bin/ffprobe", cfg.FFprobePath)
	assert.Equal(t, "/usr/local/bin/mediainfo", cfg.MediaInfoPath)
	assert.Equal(t, outDir, cfg.OutputDir)
}

func TestLoad_ExplicitConfigFileMissingErrors(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
	assert.Nil(t, cfg)
}
