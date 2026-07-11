package naming

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize_ReplacesColonAndStripsIllegalChars(t *testing.T) {
	got := sanitize(context.Background(), `Book: Name/With*Illegal?Chars"<>|`)
	assert.Equal(t, "Book - Name-WithIllegalChars'-", got)
}

func TestSanitize_NoChangeForCleanName(t *testing.T) {
	got := sanitize(context.Background(), "Clean Name (2019)")
	assert.Equal(t, "Clean Name (2019)", got)
}

func TestSanitize_LogsWarningOnChange(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(old)

	sanitize(context.Background(), "Illegal/Name")

	assert.Contains(t, buf.String(), "sanitized illegal characters")
}

func TestSanitize_TrimsTrailingDots(t *testing.T) {
	got := sanitize(context.Background(), "Book Title.")
	assert.Equal(t, "Book Title", got)
}

func TestSanitize_TrimsTrailingSpaces(t *testing.T) {
	got := sanitize(context.Background(), "  spaced  ")
	assert.Equal(t, "  spaced", got)
}

func TestSanitize_ReservedName(t *testing.T) {
	got := sanitize(context.Background(), "CON")
	assert.Equal(t, "_CON", got)
}

func TestSanitize_ReservedNameWithExtension(t *testing.T) {
	got := sanitize(context.Background(), "con.mp3")
	assert.Equal(t, "_con.mp3", got)
}

func TestSanitize_ReservedNameCaseInsensitive(t *testing.T) {
	got := sanitize(context.Background(), "lpt1")
	assert.Equal(t, "_lpt1", got)
}

func TestSanitize_AllIllegalFallsBackToUnderscore(t *testing.T) {
	got := sanitize(context.Background(), `*?<>`)
	assert.Equal(t, "_", got)
}

func TestSanitize_NoWarningForCleanName(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(old)

	sanitize(context.Background(), "Clean Name")

	assert.Empty(t, buf.String())
}
