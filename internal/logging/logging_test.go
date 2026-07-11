package logging

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_DefaultLevelIsInfo(t *testing.T) {
	logger := New(false)
	assert.True(t, logger.Enabled(context.Background(), slog.LevelInfo))
	assert.False(t, logger.Enabled(context.Background(), slog.LevelDebug))
}

func TestNew_VerboseEnablesDebug(t *testing.T) {
	logger := New(true)
	assert.True(t, logger.Enabled(context.Background(), slog.LevelDebug))
}
