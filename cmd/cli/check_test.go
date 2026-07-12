package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/znth-cx/zentag/core/ffmpeg"
	"github.com/znth-cx/zentag/core/mediainfo"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/naming"
	"github.com/znth-cx/zentag/core/ruleset"
	"github.com/znth-cx/zentag/internal/config"
)

func TestCheckCmd_RequiresExactlyOneArg(t *testing.T) {
	assert.Error(t, checkCmd.Args(checkCmd, []string{}))
	assert.Error(t, checkCmd.Args(checkCmd, []string{"one", "two"}))
	assert.NoError(t, checkCmd.Args(checkCmd, []string{"/path/to/item"}))
}

// fakeMediainfoRunner: mediainfo --Output=JSON stub; same response
// serves both ReadTags and Probe for a path.
type fakeMediainfoRunner struct {
	responses map[string]string
}

func (f *fakeMediainfoRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	path := args[len(args)-1]
	resp, ok := f.responses[path]
	if !ok {
		return nil, errFake("no mediainfo response for " + path)
	}
	return []byte(resp), nil
}

// fakeFfmpegRunner: stub for ffprobe (-show_chapters) and ffmpeg
// (cover extraction) calls via ffmpeg.Wrapper.
type fakeFfmpegRunner struct {
	chaptersJSON string
	coverBytes   map[string][]byte
}

func (f *fakeFfmpegRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	for _, a := range args {
		if a == "-show_chapters" {
			return []byte(f.chaptersJSON), nil
		}
	}
	inputPath := args[2]
	outPath := args[len(args)-1]
	data, ok := f.coverBytes[inputPath]
	if !ok || data == nil {
		return nil, errFake("no cover")
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return nil, err
	}
	return nil, nil
}

type errFake string

func (e errFake) Error() string { return string(e) }

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func mediainfoJSON(extra, format, profile string, bitrateKbps int) string {
	return `{"media":{"track":[
		{"@type":"General","Title":"Test Book","Genre":"Fantasy","extra":` + extra + `},
		{"@type":"Audio","Format":"` + format + `","Format_Profile":"` + profile + `","BitRate":"` + strconv.Itoa(bitrateKbps*1000) + `"}
	]}}`
}

// swapWrappers points check.go's wrapper seams at fakes for the test's
// duration, restoring real ffmpeg.New/mediainfo.New after.
func swapWrappers(t *testing.T, fw *ffmpeg.Wrapper, mi *mediainfo.Wrapper) {
	t.Helper()
	oldFF, oldMI := newFFmpegWrapper, newMediaInfoWrapper
	newFFmpegWrapper = func(string, string) *ffmpeg.Wrapper { return fw }
	newMediaInfoWrapper = func(string) *mediainfo.Wrapper { return mi }
	t.Cleanup(func() {
		newFFmpegWrapper = oldFF
		newMediaInfoWrapper = oldMI
	})
}

func setJSONOutput(t *testing.T, v bool) {
	t.Helper()
	old := jsonOutput
	jsonOutput = v
	t.Cleanup(func() { jsonOutput = old })
}

// setTestConfig sets cfg, normally set by PersistentPreRunE (bypassed
// when tests call RunE directly). Values don't matter for fake-wrapper
// tests, only non-nil matters: RunE derefs cfg before Gather runs.
func setTestConfig(t *testing.T) {
	t.Helper()
	old := cfg
	cfg = &config.Config{FFmpegPath: "ffmpeg", FFprobePath: "ffprobe", MediaInfoPath: "mediainfo"}
	t.Cleanup(func() { cfg = old })
}

func TestCheckCmd_RunE_CleanPassage(t *testing.T) {
	setTestConfig(t)
	setJSONOutput(t, false)

	meta := &metadata.Metadata{
		Author:   []string{"Test Author"},
		Title:    "Test Book",
		Year:     2020,
		Narrator: []string{"Test Narrator"},
		Language: "eng",
		Tracks: []metadata.Track{
			{PartNumber: 1, Container: "", Codec: "MP3", Bitrate: 128},
			{PartNumber: 2, Container: "", Codec: "MP3", Bitrate: 128},
		},
	}
	ctx := context.Background()
	dirName, err := naming.DirectoryName(ctx, meta)
	require.NoError(t, err)
	track1Name, err := naming.TrackName(ctx, meta, 0)
	require.NoError(t, err)
	track2Name, err := naming.TrackName(ctx, meta, 1)
	require.NoError(t, err)

	root := t.TempDir()
	bookDir := filepath.Join(root, dirName)
	require.NoError(t, os.MkdirAll(bookDir, 0o755))
	file1 := filepath.Join(bookDir, track1Name+".mp3")
	file2 := filepath.Join(bookDir, track2Name+".mp3")
	require.NoError(t, os.WriteFile(file1, nil, 0o644))
	require.NoError(t, os.WriteFile(file2, nil, 0o644))

	extra1 := `{"author":"Test Author","narrator":"Test Narrator","isbn":"9780765326355","year":"2020","language":"eng"}`
	cover := pngBytes(t, 100, 100)

	mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{
		responses: map[string]string{
			file1: mediainfoJSON(extra1, "MPEG Audio", "Layer 3", 128),
			file2: mediainfoJSON(`{}`, "MPEG Audio", "Layer 3", 128),
		},
	}}
	fw := &ffmpeg.Wrapper{Runner: &fakeFfmpegRunner{
		chaptersJSON: `{"chapters":[{"start_time":"0.000000","end_time":"60.000000"}]}`,
		coverBytes:   map[string][]byte{file1: cover},
	}}
	swapWrappers(t, fw, mi)

	var out bytes.Buffer
	checkCmd.SetContext(ctx)
	checkCmd.SetOut(&out)

	err = checkCmd.RunE(checkCmd, []string{bookDir})
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "✓ primary_keys")
	assert.Contains(t, out.String(), "✓ naming")
	assert.NotContains(t, out.String(), "✗")
}

func dirtyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.m4b")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Test Author","narrator":"Test Narrator","year":"2020"}`
	cover := pngBytes(t, 100, 100)

	mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSON(extra, "AAC", "", 64)},
	}}
	fw := &ffmpeg.Wrapper{Runner: &fakeFfmpegRunner{
		chaptersJSON: `{"chapters":[{"start_time":"0.000000","end_time":"60.000000"}]}`,
		coverBytes:   map[string][]byte{filePath: cover},
	}}
	swapWrappers(t, fw, mi)

	return filePath
}

func TestCheckCmd_RunE_ViolationsFound(t *testing.T) {
	setTestConfig(t)
	setJSONOutput(t, false)
	filePath := dirtyFixture(t)

	var out bytes.Buffer
	checkCmd.SetContext(context.Background())
	checkCmd.SetOut(&out)

	err := checkCmd.RunE(checkCmd, []string{filePath})
	assert.Error(t, err)
	assert.Contains(t, out.String(), "no ISBN or ASIN")
}

func TestCheckCmd_RunE_JSONFlag(t *testing.T) {
	setTestConfig(t)
	setJSONOutput(t, true)
	filePath := dirtyFixture(t)

	var out bytes.Buffer
	checkCmd.SetContext(context.Background())
	checkCmd.SetOut(&out)

	err := checkCmd.RunE(checkCmd, []string{filePath})
	assert.Error(t, err)

	var violations []ruleset.Violation
	require.NoError(t, json.Unmarshal(out.Bytes(), &violations))
	assert.NotEmpty(t, violations)
}

// Goes through real dispatch (Run -> rootCmd.ExecuteContext), unlike
// the RunE tests above which skip PersistentPreRunE/SilenceUsage.
// Catches SilenceUsage regressions (cobra dumping Usage: alongside
// errViolationsFound).
func TestRun_Check_RealDispatch_ViolationsFound(t *testing.T) {
	t.Chdir(t.TempDir())
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("APPDATA", homeDir)
	cfgFile = ""
	verbose = false
	cfg = nil
	logger = nil

	setTestConfig(t) // overwritten below; swapWrappers is what matters
	setJSONOutput(t, false)
	filePath := dirtyFixture(t)

	// Earlier tests' SetOut/SetErr are never reset, so checkCmd may
	// still be pinned to a prior buffer; clear so it falls through to
	// rootCmd's writers via OutOrStdout/ErrOrStderr.
	checkCmd.SetOut(nil)
	checkCmd.SetErr(nil)

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	rootCmd.SetArgs([]string{"check", filePath})
	defer rootCmd.SetArgs(nil)

	err := Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, outBuf.String(), "no ISBN or ASIN")
	assert.NotContains(t, errBuf.String(), "Usage:")
}

func TestCheckCmd_RunE_GatherErrorPropagates(t *testing.T) {
	setTestConfig(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist.m4b")

	var out bytes.Buffer
	checkCmd.SetContext(context.Background())
	checkCmd.SetOut(&out)

	err := checkCmd.RunE(checkCmd, []string{missing})
	assert.Error(t, err)
	assert.Empty(t, out.String())
}
