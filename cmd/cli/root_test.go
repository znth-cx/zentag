package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistentPreRunE_SetsConfigAndLogger(t *testing.T) {
	t.Chdir(t.TempDir())
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("APPDATA", homeDir)
	cfgFile = ""
	verbose = false
	cfg = nil
	logger = nil

	err := rootCmd.PersistentPreRunE(rootCmd, nil)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, logger)
	assert.Equal(t, "ffmpeg", cfg.FFmpegPath)
}

func TestRun_UnknownCommandReturnsError(t *testing.T) {
	t.Chdir(t.TempDir())
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("APPDATA", homeDir)
	cfgFile = ""
	verbose = false
	cfg = nil
	logger = nil

	rootCmd.SetArgs([]string{"totally-not-a-real-subcommand"})
	defer rootCmd.SetArgs(nil)

	err := Run(context.Background())
	assert.Error(t, err)
}
