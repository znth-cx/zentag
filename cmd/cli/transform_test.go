package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/Ether/zentag/core/ffmpeg"
	"codeberg.org/Ether/zentag/core/mediainfo"
	"codeberg.org/Ether/zentag/core/metadata"
	"codeberg.org/Ether/zentag/core/naming"
	"codeberg.org/Ether/zentag/core/ruleset"
	"codeberg.org/Ether/zentag/core/session"
	"codeberg.org/Ether/zentag/core/sources/audnexus"
	"codeberg.org/Ether/zentag/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformCmd_RequiresExactlyOneArg(t *testing.T) {
	assert.Error(t, transformCmd.Args(transformCmd, []string{}))
	assert.Error(t, transformCmd.Args(transformCmd, []string{"one", "two"}))
	assert.NoError(t, transformCmd.Args(transformCmd, []string{"/path/to/item"}))
}

// stubWriteM4B: fake writeM4B drops a placeholder file, so routing/
// session/naming tests don't need a real mp4tag-openable fixture
// (M4BEngine has its own tests for that pipeline).
func stubWriteM4B(t *testing.T) {
	t.Helper()
	orig := writeM4B
	writeM4B = func(_ context.Context, _ *ffmpeg.Wrapper, _ *metadata.Metadata, outputDir string, trackNames []string) error {
		return os.WriteFile(filepath.Join(outputDir, trackNames[0]+".m4b"), []byte("fake-m4b"), 0o644)
	}
	t.Cleanup(func() { writeM4B = orig })
}

func setTransformFlag(t *testing.T, name, value string) {
	t.Helper()
	require.NoError(t, transformCmd.Flags().Set(name, value))
	t.Cleanup(func() {
		f := transformCmd.Flags().Lookup(name)
		require.NoError(t, f.Value.Set(f.DefValue))
		f.Changed = false
	})
}

func TestUserArgsFromFlags_OnlyChangedFlagsPopulateFields(t *testing.T) {
	setTransformFlag(t, "title", "Custom Title")
	setTransformFlag(t, "year", "2019")

	got, err := userArgsFromFlags(context.Background(), transformCmd)
	require.NoError(t, err)
	assert.Equal(t, metadata.OriginUserArgs, got.MetadataOrigin)
	assert.Equal(t, "Custom Title", got.Title)
	assert.Equal(t, 2019, got.Year)
	assert.Empty(t, got.Author)
	assert.Empty(t, got.Narrator)
}

func TestUserArgsFromFlags_CSVFieldsSplitAndTrimmed(t *testing.T) {
	setTransformFlag(t, "author", "Brandon Sanderson, Co-Author")

	got, err := userArgsFromFlags(context.Background(), transformCmd)
	require.NoError(t, err)
	assert.Equal(t, []string{"Brandon Sanderson", "Co-Author"}, got.Author)
}

func TestUserArgsFromFlags_SeriesPairsNameAndPart(t *testing.T) {
	setTransformFlag(t, "series", "The Stormlight Archive")
	setTransformFlag(t, "series-part", "1")

	got, err := userArgsFromFlags(context.Background(), transformCmd)
	require.NoError(t, err)
	require.Len(t, got.Series, 1)
	assert.Equal(t, metadata.SeriesEntry{Name: "The Stormlight Archive", Part: "1"}, got.Series[0])
}

func TestUserArgsFromFlags_SeriesPartWithoutSeriesErrors(t *testing.T) {
	setTransformFlag(t, "series-part", "1")

	_, err := userArgsFromFlags(context.Background(), transformCmd)
	require.EqualError(t, err, "--series-part requires --series")
}

func TestUserArgsFromFlags_NoFlagsChangedReturnsEmptyMetadata(t *testing.T) {
	got, err := userArgsFromFlags(context.Background(), transformCmd)
	require.NoError(t, err)
	assert.Equal(t, metadata.OriginUserArgs, got.MetadataOrigin)
	assert.Equal(t, "", got.Title)
	assert.Nil(t, got.Author)
	assert.Equal(t, 0, got.Year)
}

func TestResolveASIN_FlagTakesPrecedenceOverFileMetadata(t *testing.T) {
	setTransformFlag(t, "asin", "B0FLAGASIN")
	fileMeta := &metadata.Metadata{ASIN: "B0FILEASIN"}

	assert.Equal(t, "B0FLAGASIN", resolveASIN(transformCmd, fileMeta))
}

func TestResolveASIN_FallsBackToFileMetadata(t *testing.T) {
	fileMeta := &metadata.Metadata{ASIN: "B0FILEASIN"}
	assert.Equal(t, "B0FILEASIN", resolveASIN(transformCmd, fileMeta))
}

func TestResolveASIN_EmptyWhenNeitherPresent(t *testing.T) {
	fileMeta := &metadata.Metadata{}
	assert.Equal(t, "", resolveASIN(transformCmd, fileMeta))
}

func TestSplitCSV_TrimsAndDropsEmpty(t *testing.T) {
	assert.Equal(t, []string{"A", "B"}, splitCSV(" A ,  , B"))
	assert.Nil(t, splitCSV(""))
}

func TestPromptConflicts_BlankInputUsesRecommended(t *testing.T) {
	conflicts := []metadata.Conflict{
		{Field: "Title", Values: []string{"User Title", "File Title"}, Origins: []metadata.MetadataOrigin{metadata.OriginUserArgs, metadata.OriginFileMetadata}, Recommended: 0},
	}
	in := bufio.NewScanner(strings.NewReader("\n"))
	var out bytes.Buffer

	choices, err := promptConflicts(in, &out, conflicts)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Title": 0}, choices)
	assert.Contains(t, out.String(), "Field: Title")
	assert.Contains(t, out.String(), "(recommended)")
}

func TestPromptConflicts_ExplicitIndexChoice(t *testing.T) {
	conflicts := []metadata.Conflict{
		{Field: "Title", Values: []string{"User Title", "File Title"}, Origins: []metadata.MetadataOrigin{metadata.OriginUserArgs, metadata.OriginFileMetadata}, Recommended: 0},
	}
	in := bufio.NewScanner(strings.NewReader("1\n"))
	var out bytes.Buffer

	choices, err := promptConflicts(in, &out, conflicts)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Title": 1}, choices)
}

func TestPromptConflicts_NegativeOneOmits(t *testing.T) {
	conflicts := []metadata.Conflict{
		{Field: "Title", Values: []string{"User Title", "File Title"}, Origins: []metadata.MetadataOrigin{metadata.OriginUserArgs, metadata.OriginFileMetadata}, Recommended: 0},
	}
	in := bufio.NewScanner(strings.NewReader("-1\n"))
	var out bytes.Buffer

	choices, err := promptConflicts(in, &out, conflicts)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Title": -1}, choices)
}

func TestPromptConflicts_InvalidInputReprompts(t *testing.T) {
	conflicts := []metadata.Conflict{
		{Field: "Title", Values: []string{"User Title", "File Title"}, Origins: []metadata.MetadataOrigin{metadata.OriginUserArgs, metadata.OriginFileMetadata}, Recommended: 0},
	}
	in := bufio.NewScanner(strings.NewReader("not-a-number\n5\n1\n"))
	var out bytes.Buffer

	choices, err := promptConflicts(in, &out, conflicts)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Title": 1}, choices)
	assert.Contains(t, out.String(), "invalid choice")
}

func TestPromptConflicts_MultipleConflictsEachPrompted(t *testing.T) {
	conflicts := []metadata.Conflict{
		{Field: "Title", Values: []string{"A", "B"}, Origins: []metadata.MetadataOrigin{metadata.OriginUserArgs, metadata.OriginFileMetadata}, Recommended: 0},
		{Field: "Year", Values: []string{"2010", "2011"}, Origins: []metadata.MetadataOrigin{metadata.OriginUserArgs, metadata.OriginFileMetadata}, Recommended: 0},
	}
	in := bufio.NewScanner(strings.NewReader("0\n1\n"))
	var out bytes.Buffer

	choices, err := promptConflicts(in, &out, conflicts)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Title": 0, "Year": 1}, choices)
}

func TestConfirmProceed_YAnswersTrue(t *testing.T) {
	in := bufio.NewScanner(strings.NewReader("y\n"))
	var out bytes.Buffer

	proceed, err := confirmProceed(in, &out, "summary text")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.Contains(t, out.String(), "summary text")
	assert.Contains(t, out.String(), "Proceed?")
}

func TestConfirmProceed_YesAnswersTrue(t *testing.T) {
	in := bufio.NewScanner(strings.NewReader("yes\n"))
	var out bytes.Buffer

	proceed, err := confirmProceed(in, &out, "summary text")
	require.NoError(t, err)
	assert.True(t, proceed)
}

func TestConfirmProceed_NAnswersFalse(t *testing.T) {
	in := bufio.NewScanner(strings.NewReader("n\n"))
	var out bytes.Buffer

	proceed, err := confirmProceed(in, &out, "summary text")
	require.NoError(t, err)
	assert.False(t, proceed)
}

func TestConfirmProceed_BlankAnswersFalse(t *testing.T) {
	in := bufio.NewScanner(strings.NewReader("\n"))
	var out bytes.Buffer

	proceed, err := confirmProceed(in, &out, "summary text")
	require.NoError(t, err)
	assert.False(t, proceed)
}

func TestSummarizeMetadata_IncludesKeyFieldsAndViolations(t *testing.T) {
	meta := &metadata.Metadata{
		Author:   []string{"Brandon Sanderson"},
		Title:    "The Way of Kings",
		Year:     2010,
		Narrator: []string{"Michael Kramer"},
		Language: "en",
		Source:   metadata.ReleaseSourceWEB,
	}
	violations := []ruleset.Violation{{Rule: "primary_keys", Severity: ruleset.SeverityProhibited, Message: "no ISBN or ASIN"}}

	summary, err := summarizeMetadata(meta, violations)
	require.NoError(t, err)
	assert.Contains(t, summary, "Brandon Sanderson")
	assert.Contains(t, summary, "The Way of Kings")
	assert.Contains(t, summary, "2010")
	assert.Contains(t, summary, "Michael Kramer")
	assert.Contains(t, summary, "no ISBN or ASIN")
	assert.Contains(t, summary, "Cover: none")
}

func TestSummarizeMetadata_NoViolationsSaysSo(t *testing.T) {
	meta := &metadata.Metadata{Author: []string{"A"}, Title: "T", Narrator: []string{"N"}}
	summary, err := summarizeMetadata(meta, nil)
	require.NoError(t, err)
	assert.Contains(t, summary, "No rule violations found.")
}

func TestSummarizeMetadata_CoverPresentShowsMIMEAndSize(t *testing.T) {
	meta := &metadata.Metadata{
		Author:     []string{"A"},
		Title:      "T",
		Narrator:   []string{"N"},
		CoverImage: []byte{1, 2, 3, 4, 5},
		CoverMIME:  "image/jpeg",
	}
	summary, err := summarizeMetadata(meta, nil)
	require.NoError(t, err)
	assert.Contains(t, summary, "Cover: found (image/jpeg, 5 bytes)")
}

func TestDispatchWrite_M4BUsesM4BEngine(t *testing.T) {
	stubWriteM4B(t)
	fr := &fakeTransformFfmpegRunner{}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}
	meta := &metadata.Metadata{
		Title:  "Some Book",
		Tracks: []metadata.Track{{Path: "book.m4b", Container: "M4B", Codec: "AAC", Bitrate: 64}},
	}
	dir := t.TempDir()

	err := dispatchWrite(context.Background(), fw, meta, dir, []string{"Some Book"})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "Some Book.m4b"))
}

func TestDispatchWrite_M4BWithCoverWritesNoLooseCoverFile(t *testing.T) {
	stubWriteM4B(t)
	fr := &fakeTransformFfmpegRunner{}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}
	meta := &metadata.Metadata{
		Title:      "Some Book",
		CoverImage: []byte{1, 2, 3},
		CoverMIME:  "image/jpeg",
		Tracks:     []metadata.Track{{Path: "book.m4b", Container: "M4B", Codec: "AAC", Bitrate: 64}},
	}
	dir := t.TempDir()

	err := dispatchWrite(context.Background(), fw, meta, dir, []string{"Some Book"})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "Some Book.m4b"))
	assert.NoFileExists(t, filepath.Join(dir, "cover.jpg"), "M4B embeds its cover, dispatchWrite must not also write a loose one")
}

func TestDispatchWrite_MP3WritesLooseCoverWhenPresent(t *testing.T) {
	fr := &fakeTransformFfmpegRunner{}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}
	meta := &metadata.Metadata{
		Title:      "Some Book",
		CoverImage: []byte{1, 2, 3},
		CoverMIME:  "image/jpeg",
		Tracks: []metadata.Track{
			{Path: "part1.mp3", Container: "", Codec: "MP3", Bitrate: 128},
			{Path: "part2.mp3", Container: "", Codec: "MP3", Bitrate: 128},
		},
	}
	dir := t.TempDir()

	err := dispatchWrite(context.Background(), fw, meta, dir, []string{"001. Chapter One", "002. Chapter Two"})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "001. Chapter One.mp3"))
	assert.FileExists(t, filepath.Join(dir, "002. Chapter Two.mp3"))
	assert.FileExists(t, filepath.Join(dir, "cover.jpg"))
}

func TestDispatchWrite_FLACNoCoverWritesNoLooseCoverFile(t *testing.T) {
	fr := &fakeTransformFfmpegRunner{}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}
	meta := &metadata.Metadata{
		Title:  "Some Book",
		Tracks: []metadata.Track{{Path: "part1.flac", Container: "", Codec: "FLAC", Bitrate: 1000}},
	}
	dir := t.TempDir()

	err := dispatchWrite(context.Background(), fw, meta, dir, []string{"001. Chapter One"})
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(dir, "cover.jpg"))
	assert.NoFileExists(t, filepath.Join(dir, "cover.png"))
}

func TestDispatchWrite_UnsupportedContainerCodecErrors(t *testing.T) {
	fr := &fakeTransformFfmpegRunner{}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", Runner: fr}
	meta := &metadata.Metadata{
		Tracks: []metadata.Track{{Path: "part1.wav", Container: "", Codec: "PCM", Bitrate: 1000}},
	}

	err := dispatchWrite(context.Background(), fw, meta, t.TempDir(), []string{"001. Chapter One"})
	assert.Error(t, err)
}

// fakeTransformFfmpegRunner: chapter calls return chaptersJSON; a
// cover-extraction output path (image ext, or ".img") serves from
// coverBytes keyed by input path; every other call writes placeholder
// bytes so callers can assert the file exists without real ffmpeg.
type fakeTransformFfmpegRunner struct {
	chaptersJSON string
	coverBytes   map[string][]byte
}

func (f *fakeTransformFfmpegRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	for _, a := range args {
		if a == "-show_chapters" {
			return []byte(f.chaptersJSON), nil
		}
	}
	outPath := args[len(args)-1]
	switch strings.ToLower(filepath.Ext(outPath)) {
	case ".jpg", ".jpeg", ".png", ".img":
		inputPath := args[2]
		data, ok := f.coverBytes[inputPath]
		if !ok || data == nil {
			return nil, errFake("no cover")
		}
		return nil, os.WriteFile(outPath, data, 0o644)
	default:
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return nil, err
		}
		return nil, os.WriteFile(outPath, []byte("fake-audio"), 0o644)
	}
}

func setTransformTestConfig(t *testing.T) {
	t.Helper()
	setTestConfig(t)
	cfg.OutputDir = t.TempDir()
	// isolate sessions per test: no cwd litter, no stray resume trigger.
	cfg.SessionDir = t.TempDir()
}

func swapAudnexusWrapper(t *testing.T, w *audnexus.Wrapper) {
	t.Helper()
	old := newAudnexusWrapper
	newAudnexusWrapper = func(string) *audnexus.Wrapper { return w }
	t.Cleanup(func() { newAudnexusWrapper = old })
}

// transformM4BFixture: clean single-file m4b fixture, mirrors
// check_test.go's dirtyFixture but with a clean tag set and a runner
// that writes real output files.
func transformM4BFixture(t *testing.T) (filePath string, cover []byte) {
	t.Helper()
	dir := t.TempDir()
	filePath = filepath.Join(dir, "book.m4b")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Test Author","narrator":"Test Narrator","isbn":"9780765326355","year":"2020","language":"eng"}`
	cover = pngBytes(t, 100, 100)

	mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSON(extra, "AAC", "", 64)},
	}}
	fw := &ffmpeg.Wrapper{Runner: &fakeTransformFfmpegRunner{
		chaptersJSON: `{"chapters":[{"start_time":"0.000000","end_time":"60.000000"}]}`,
		coverBytes:   map[string][]byte{filePath: cover},
	}}
	swapWrappers(t, fw, mi)
	stubWriteM4B(t)
	return filePath, cover
}

func expectedM4BMeta() *metadata.Metadata {
	return &metadata.Metadata{
		Author:   []string{"Test Author"},
		Title:    "Test Book",
		Year:     2020,
		Narrator: []string{"Test Narrator"},
		Language: "eng",
		Source:   metadata.ReleaseSourceWEB,
		Tracks:   []metadata.Track{{Container: "M4B", Codec: "AAC", Bitrate: 64}},
	}
}

func TestTransformCmd_RunE_HappyPath_ConfirmedWritesFile(t *testing.T) {
	logger = logging.New(false)
	setTransformTestConfig(t)
	filePath, _ := transformM4BFixture(t)

	var out bytes.Buffer
	transformCmd.SetContext(context.Background())
	transformCmd.SetOut(&out)
	transformCmd.SetIn(strings.NewReader("y\n"))

	err := transformCmd.RunE(transformCmd, []string{filePath})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Wrote")

	dirName, err := naming.DirectoryName(context.Background(), expectedM4BMeta())
	require.NoError(t, err)
	trackName, err := naming.TrackName(context.Background(), expectedM4BMeta(), 0)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(cfg.OutputDir, dirName, trackName+".m4b"))
}

func TestTransformCmd_RunE_ResumesFromSession(t *testing.T) {
	logger = logging.New(false)
	setTransformTestConfig(t)
	filePath, _ := transformM4BFixture(t)

	// session must drive the run, not the file's own title "Test Book"
	sessMeta := expectedM4BMeta()
	sessMeta.Title = "Resumed Book"
	sessMeta.Tracks[0].Path = filePath
	require.NoError(t, session.Save(context.Background(), cfg.SessionDir, filePath, sessMeta))

	var out bytes.Buffer
	transformCmd.SetContext(context.Background())
	transformCmd.SetOut(&out)
	transformCmd.SetIn(strings.NewReader("y\n"))

	err := transformCmd.RunE(transformCmd, []string{filePath})
	require.NoError(t, err)

	resumedDir, err := naming.DirectoryName(context.Background(), sessMeta)
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(cfg.OutputDir, resumedDir))

	fileDir, err := naming.DirectoryName(context.Background(), expectedM4BMeta())
	require.NoError(t, err)
	assert.NoDirExists(t, filepath.Join(cfg.OutputDir, fileDir), "must not use the file's gathered title")
}

func TestTransformCmd_RunE_CleanIgnoresSession(t *testing.T) {
	logger = logging.New(false)
	setTransformTestConfig(t)
	filePath, _ := transformM4BFixture(t)

	sessMeta := expectedM4BMeta()
	sessMeta.Title = "Resumed Book"
	sessMeta.Tracks[0].Path = filePath
	require.NoError(t, session.Save(context.Background(), cfg.SessionDir, filePath, sessMeta))

	cleanFlag = true
	t.Cleanup(func() { cleanFlag = false })

	var out bytes.Buffer
	transformCmd.SetContext(context.Background())
	transformCmd.SetOut(&out)
	transformCmd.SetIn(strings.NewReader("y\n"))

	err := transformCmd.RunE(transformCmd, []string{filePath})
	require.NoError(t, err)

	// --clean discards the session, so the run re-gathers the file.
	fileDir, err := naming.DirectoryName(context.Background(), expectedM4BMeta())
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(cfg.OutputDir, fileDir))
}

func TestTransformCmd_RunE_ConflictPromptThenConfirm(t *testing.T) {
	logger = logging.New(false)
	setTransformTestConfig(t)
	filePath, _ := transformM4BFixture(t)
	setTransformFlag(t, "title", "Custom Title")

	var out bytes.Buffer
	transformCmd.SetContext(context.Background())
	transformCmd.SetOut(&out)
	transformCmd.SetIn(strings.NewReader("\ny\n")) // accept recommended title, then confirm

	err := transformCmd.RunE(transformCmd, []string{filePath})
	require.NoError(t, err)

	want := expectedM4BMeta()
	want.Title = "Custom Title"
	dirName, err := naming.DirectoryName(context.Background(), want)
	require.NoError(t, err)
	trackName, err := naming.TrackName(context.Background(), want, 0)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(cfg.OutputDir, dirName, trackName+".m4b"))
}

func TestTransformCmd_RunE_DeclinedConfirmation_NoWrite(t *testing.T) {
	logger = logging.New(false)
	setTransformTestConfig(t)
	filePath, _ := transformM4BFixture(t)

	var out bytes.Buffer
	transformCmd.SetContext(context.Background())
	transformCmd.SetOut(&out)
	transformCmd.SetIn(strings.NewReader("n\n"))

	err := transformCmd.RunE(transformCmd, []string{filePath})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Aborted")

	entries, err := os.ReadDir(cfg.OutputDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTransformCmd_RunE_GatherErrorPropagates(t *testing.T) {
	logger = logging.New(false)
	setTransformTestConfig(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist.m4b")

	var out bytes.Buffer
	transformCmd.SetContext(context.Background())
	transformCmd.SetOut(&out)

	err := transformCmd.RunE(transformCmd, []string{missing})
	assert.Error(t, err)
}

func TestTransformCmd_RunE_AudnexusMergeApplied(t *testing.T) {
	logger = logging.New(false)
	setTransformTestConfig(t)
	filePath, _ := transformM4BFixture(t)
	setTransformFlag(t, "asin", "B0TESTASIN")

	bookJSON := `{"asin":"B0TESTASIN","title":"Audnexus Title","language":"english","formatType":"unabridged"}`
	swapAudnexusWrapper(t, &audnexus.Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeAudnexusHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(bookJSON))}, nil
		},
	}})

	var out bytes.Buffer
	transformCmd.SetContext(context.Background())
	transformCmd.SetOut(&out)
	transformCmd.SetIn(strings.NewReader("\ny\n")) // Title conflict (audnexus vs file): accept recommended (audnexus, higher precedence), then confirm

	err := transformCmd.RunE(transformCmd, []string{filePath})
	require.NoError(t, err)

	want := expectedM4BMeta()
	want.Title = "Audnexus Title"
	want.ASIN = "B0TESTASIN"
	dirName, err := naming.DirectoryName(context.Background(), want)
	require.NoError(t, err)
	trackName, err := naming.TrackName(context.Background(), want, 0)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(cfg.OutputDir, dirName, trackName+".m4b"))
}

func TestTransformCmd_RunE_AudnexusLookupFailure_ContinuesWithFileOnly(t *testing.T) {
	logger = logging.New(false)
	setTransformTestConfig(t)
	filePath, _ := transformM4BFixture(t)
	setTransformFlag(t, "asin", "B0TESTASIN")

	swapAudnexusWrapper(t, &audnexus.Wrapper{BaseURL: "https://api.audnex.us", Client: &fakeAudnexusHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		},
	}})

	var out bytes.Buffer
	transformCmd.SetContext(context.Background())
	transformCmd.SetOut(&out)
	transformCmd.SetIn(strings.NewReader("y\n"))

	err := transformCmd.RunE(transformCmd, []string{filePath})
	require.NoError(t, err)

	want := expectedM4BMeta()
	want.ASIN = "B0TESTASIN"
	dirName, err := naming.DirectoryName(context.Background(), want)
	require.NoError(t, err)
	trackName, err := naming.TrackName(context.Background(), want, 0)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(cfg.OutputDir, dirName, trackName+".m4b"))
}

// fakeAudnexusHTTPClient: mirrors audnexus/gather_test.go's unexported
// fakeHTTPClient, redeclared since that one is package-private.
type fakeAudnexusHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (f *fakeAudnexusHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f.do(req)
}
