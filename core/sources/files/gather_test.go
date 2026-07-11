package files

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"codeberg.org/Ether/zentag/core/ffmpeg"
	"codeberg.org/Ether/zentag/core/mediainfo"
	"codeberg.org/Ether/zentag/core/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeFfmpegRunner serves ffprobe (-show_chapters) and ffmpeg (cover extraction) calls via ffmpeg.Wrapper.
type fakeFfmpegRunner struct {
	chaptersJSON string            // returned for ffprobe -show_chapters calls
	coverBytes   map[string][]byte // path -> cover bytes; missing/nil = no cover (simulated ffmpeg failure)
}

func (f *fakeFfmpegRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	if slices.Contains(args, "-show_chapters") {
		return []byte(f.chaptersJSON), nil
	}
	// cover extraction call: args = [-y -i <path> -an -c:v copy -f image2 <tempfile>]
	inputPath := args[2]
	outPath := args[len(args)-1]
	data, ok := f.coverBytes[inputPath]
	if !ok || data == nil {
		return nil, assertError{"no cover"}
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return nil, err
	}
	return nil, nil
}

type assertError struct{ msg string }

func (e assertError) Error() string { return e.msg }

// fakeMediainfoRunner serves mediainfo --Output=JSON calls keyed by file path.
type fakeMediainfoRunner struct {
	responses map[string]string
}

func (f *fakeMediainfoRunner) Run(_ context.Context, _ string, args []string) ([]byte, error) {
	path := args[len(args)-1]
	resp, ok := f.responses[path]
	if !ok {
		return nil, assertError{"no mediainfo response for " + path}
	}
	return []byte(resp), nil
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

const noChaptersJSON = `{"chapters": []}`

func mediainfoJSON(title, extra string, bitrateKbps int) string {
	return mediainfoJSONWithFormat(title, extra, bitrateKbps, "MPEG Audio", "Layer 3")
}

func mediainfoJSONWithFormat(title, extra string, bitrateKbps int, format, profile string) string {
	return `{"media":{"track":[
		{"@type":"General","Title":"` + title + `","Genre":"Fantasy","extra":` + extra + `},
		{"@type":"Audio","Format":"` + format + `","Format_Profile":"` + profile + `","BitRate":"` + itoa(bitrateKbps*1000) + `"}
	]}}`
}

func mediainfoJSONWithLanguage(title, extra string, bitrateKbps int, format, profile, audioLanguage string) string {
	return `{"media":{"track":[
		{"@type":"General","Title":"` + title + `","Genre":"Fantasy","extra":` + extra + `},
		{"@type":"Audio","Format":"` + format + `","Format_Profile":"` + profile + `","BitRate":"` + itoa(bitrateKbps*1000) + `","Language":"` + audioLanguage + `"}
	]}}`
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func TestGather_SingleFileHappyPath(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.m4b")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Brandon Sanderson","narrator":"Michael Kramer","isbn":"9780765326355","year":"2010"}`
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSONWithFormat("The Way of Kings", extra, 128, "AAC", "")},
	}}
	cover := pngBytes(t, 100, 100)
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{
		chaptersJSON: `{"chapters":[{"start_time":"0.000000","end_time":"60.000000","tags":{"title":"Chapter One"}}]}`,
		coverBytes:   map[string][]byte{filePath: cover},
	}}

	got, err := Gather(context.Background(), fw, mi, filePath, nil)
	require.NoError(t, err)

	assert.Equal(t, filePath, got.OriginalPath)
	assert.Equal(t, metadata.OriginFileMetadata, got.MetadataOrigin)
	assert.Equal(t, "The Way of Kings", got.Title)
	assert.Equal(t, []string{"Brandon Sanderson"}, got.Author)
	assert.Equal(t, []string{"Michael Kramer"}, got.Narrator)
	assert.Equal(t, "9780765326355", got.ISBN)
	assert.Equal(t, 2010, got.Year)
	assert.Equal(t, cover, got.CoverImage)

	require.Len(t, got.Tracks, 1)
	track := got.Tracks[0]
	assert.Equal(t, filePath, track.Path)
	assert.Equal(t, 0, track.PartNumber)
	assert.Equal(t, "AAC", track.Codec)
	assert.Equal(t, "M4B", track.Container)
	assert.Equal(t, 128, track.Bitrate)
	require.Len(t, track.Chapters, 1)
	assert.Equal(t, "Chapter One", track.Chapters[0].Title)
}

func TestGather_MultiFileSortsByPartNumberAndIgnoresExtras(t *testing.T) {
	dir := t.TempDir()
	part2 := filepath.Join(dir, "002. Chapter Two.mp3")
	part1 := filepath.Join(dir, "001. Chapter One.mp3")
	require.NoError(t, os.WriteFile(part1, nil, 0o644))
	require.NoError(t, os.WriteFile(part2, nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cover.jpg"), pngBytes(t, 1, 1), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore me"), 0o644))

	part1Extra := `{"author":"Book Author"}`
	part2Extra := `{"author":"Should Not Be Used"}`
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{
		responses: map[string]string{
			part1: mediainfoJSON("Some Book", part1Extra, 128),
			part2: mediainfoJSON("Some Book", part2Extra, 64),
		},
	}}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{
		chaptersJSON: noChaptersJSON,
		coverBytes:   map[string][]byte{},
	}}

	got, err := Gather(context.Background(), fw, mi, dir, nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"Book Author"}, got.Author, "book-wide tags must come from part 1 only")

	require.Len(t, got.Tracks, 2)
	assert.Equal(t, part1, got.Tracks[0].Path)
	assert.Equal(t, 1, got.Tracks[0].PartNumber)
	assert.Equal(t, 128, got.Tracks[0].Bitrate)
	assert.Equal(t, part2, got.Tracks[1].Path)
	assert.Equal(t, 2, got.Tracks[1].PartNumber)
	assert.Equal(t, 64, got.Tracks[1].Bitrate)
}

func TestGather_MultiFileCoverSelection(t *testing.T) {
	part1Extra := `{"author":"Book Author"}`

	newDir := func(t *testing.T) (dir, part1 string) {
		t.Helper()
		dir = t.TempDir()
		part1 = filepath.Join(dir, "001. Chapter One.mp3")
		require.NoError(t, os.WriteFile(part1, nil, 0o644))
		return dir, part1
	}

	t.Run("embedded bigger wins", func(t *testing.T) {
		dir, part1 := newDir(t)
		embedded := pngBytes(t, 200, 200)
		loose := pngBytes(t, 50, 50)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cover.png"), loose, 0o644))

		mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{responses: map[string]string{part1: mediainfoJSON("Book", part1Extra, 64)}}}
		fw := &ffmpeg.Wrapper{Runner: &fakeFfmpegRunner{chaptersJSON: noChaptersJSON, coverBytes: map[string][]byte{part1: embedded}}}

		got, err := Gather(context.Background(), fw, mi, dir, nil)
		require.NoError(t, err)
		assert.Equal(t, embedded, got.CoverImage)
	})

	t.Run("loose bigger wins", func(t *testing.T) {
		dir, part1 := newDir(t)
		embedded := pngBytes(t, 50, 50)
		loose := pngBytes(t, 200, 200)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cover.png"), loose, 0o644))

		mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{responses: map[string]string{part1: mediainfoJSON("Book", part1Extra, 64)}}}
		fw := &ffmpeg.Wrapper{Runner: &fakeFfmpegRunner{chaptersJSON: noChaptersJSON, coverBytes: map[string][]byte{part1: embedded}}}

		got, err := Gather(context.Background(), fw, mi, dir, nil)
		require.NoError(t, err)
		assert.Equal(t, loose, got.CoverImage)
	})

	t.Run("equal size embedded wins tie", func(t *testing.T) {
		dir, part1 := newDir(t)
		embedded := pngBytes(t, 100, 100)
		loose := pngBytes(t, 100, 100)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cover.png"), loose, 0o644))

		mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{responses: map[string]string{part1: mediainfoJSON("Book", part1Extra, 64)}}}
		fw := &ffmpeg.Wrapper{Runner: &fakeFfmpegRunner{chaptersJSON: noChaptersJSON, coverBytes: map[string][]byte{part1: embedded}}}

		got, err := Gather(context.Background(), fw, mi, dir, nil)
		require.NoError(t, err)
		assert.Equal(t, embedded, got.CoverImage)
	})

	t.Run("loose only", func(t *testing.T) {
		dir, part1 := newDir(t)
		loose := pngBytes(t, 100, 100)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cover.png"), loose, 0o644))

		mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{responses: map[string]string{part1: mediainfoJSON("Book", part1Extra, 64)}}}
		fw := &ffmpeg.Wrapper{Runner: &fakeFfmpegRunner{chaptersJSON: noChaptersJSON, coverBytes: map[string][]byte{}}}

		got, err := Gather(context.Background(), fw, mi, dir, nil)
		require.NoError(t, err)
		assert.Equal(t, loose, got.CoverImage)
	})

	t.Run("neither present", func(t *testing.T) {
		dir, part1 := newDir(t)

		mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{responses: map[string]string{part1: mediainfoJSON("Book", part1Extra, 64)}}}
		fw := &ffmpeg.Wrapper{Runner: &fakeFfmpegRunner{chaptersJSON: noChaptersJSON, coverBytes: map[string][]byte{}}}

		got, err := Gather(context.Background(), fw, mi, dir, nil)
		require.NoError(t, err)
		assert.Nil(t, got.CoverImage)
	})
}

func TestGather_PerFileProbeErrorAbortsGather(t *testing.T) {
	dir := t.TempDir()
	part1 := filepath.Join(dir, "001. Chapter One.mp3")
	part2 := filepath.Join(dir, "002. Chapter Two.mp3")
	require.NoError(t, os.WriteFile(part1, nil, 0o644))
	require.NoError(t, os.WriteFile(part2, nil, 0o644))

	mi := &mediainfo.Wrapper{Runner: &fakeMediainfoRunner{
		responses: map[string]string{
			part1: mediainfoJSON("Book", `{"author":"Author"}`, 64),
			// part2 intentionally has no response, simulating a probe failure
		},
	}}
	fw := &ffmpeg.Wrapper{Runner: &fakeFfmpegRunner{chaptersJSON: noChaptersJSON, coverBytes: map[string][]byte{}}}

	_, err := Gather(context.Background(), fw, mi, dir, nil)
	assert.Error(t, err)
}

func TestGather_DoesNotUseCDEKFallback(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.m4b")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Brandon Sanderson","CDEK":"B0FALLBACKASIN"}`
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSONWithFormat("Some Book", extra, 128, "AAC", "")},
	}}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{
		chaptersJSON: noChaptersJSON,
		coverBytes:   map[string][]byte{},
	}}

	got, err := Gather(context.Background(), fw, mi, filePath, nil)
	require.NoError(t, err)
	assert.Empty(t, got.ASIN, "Gather (strict) must not fall back to the CDEK tag")
}

// TestGather_UsesAudioTrackLanguageWhenGeneralTagAbsent: mediainfo doesn't surface MP3/FLAC language tags in General's extra, only in Audio's Language field. Strict Gather must read from there.
func TestGather_UsesAudioTrackLanguageWhenGeneralTagAbsent(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.mp3")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Brandon Sanderson"}`
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSONWithLanguage("Some Book", extra, 128, "MPEG Audio", "Layer 3", "fr")},
	}}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{
		chaptersJSON: noChaptersJSON,
		coverBytes:   map[string][]byte{},
	}}

	got, err := Gather(context.Background(), fw, mi, filePath, nil)
	require.NoError(t, err)
	assert.Equal(t, "fr", got.Language, "Gather (strict) must read language from the Audio track when General has none")
}

func TestGatherGreedy_FallsBackToCDEKTagAndNormalizesAudioLanguage(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.m4b")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Brandon Sanderson","CDEK":"B0FALLBACKASIN"}`
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSONWithLanguage("Some Book", extra, 128, "AAC", "", "fr")},
	}}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{
		chaptersJSON: noChaptersJSON,
		coverBytes:   map[string][]byte{},
	}}

	got, err := GatherGreedy(context.Background(), fw, mi, filePath, nil)
	require.NoError(t, err)
	assert.Equal(t, "B0FALLBACKASIN", got.ASIN)
	assert.Equal(t, "fra", got.Language, "GatherGreedy must normalize a fallback ISO-639-1 code to ISO 639-3")
}

func TestGatherGreedy_NormalizesFullLanguageNameToPart3(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.m4b")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Brandon Sanderson","language":"English"}`
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSONWithFormat("Some Book", extra, 128, "AAC", "")},
	}}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{
		chaptersJSON: noChaptersJSON,
		coverBytes:   map[string][]byte{},
	}}

	got, err := GatherGreedy(context.Background(), fw, mi, filePath, nil)
	require.NoError(t, err)
	assert.Equal(t, "eng", got.Language, "GatherGreedy must normalize a full language name to its ISO 639-3 code")
}

func TestGather_KeepsRawLanguageCodeUnnormalized(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.m4b")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Brandon Sanderson","language":"en"}`
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSONWithFormat("Some Book", extra, 128, "AAC", "")},
	}}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{
		chaptersJSON: noChaptersJSON,
		coverBytes:   map[string][]byte{},
	}}

	got, err := Gather(context.Background(), fw, mi, filePath, nil)
	require.NoError(t, err)
	assert.Equal(t, "en", got.Language, "Gather (strict) must validate the file's actual tag, not a normalized one")
}

func TestGatherGreedy_NormalizesDirectLanguageTagToPart3(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.m4b")
	require.NoError(t, os.WriteFile(filePath, nil, 0o644))

	extra := `{"author":"Brandon Sanderson","language":"en"}`
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{
		responses: map[string]string{filePath: mediainfoJSONWithFormat("Some Book", extra, 128, "AAC", "")},
	}}
	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{
		chaptersJSON: noChaptersJSON,
		coverBytes:   map[string][]byte{},
	}}

	got, err := GatherGreedy(context.Background(), fw, mi, filePath, nil)
	require.NoError(t, err)
	assert.Equal(t, "eng", got.Language)
}

func TestGather_MultiFileDuplicatePartNumberErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "01_a.mp3"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "1_b.mp3"), nil, 0o644))

	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{}}
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{responses: map[string]string{}}}

	_, err := Gather(context.Background(), fw, mi, dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate part number 1")
	assert.Contains(t, err.Error(), `"01_a.mp3"`)
	assert.Contains(t, err.Error(), `"1_b.mp3"`)
}

func TestGather_MultiFileMissingPartNumberErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "no_part_number.mp3"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "001. Chapter One.mp3"), nil, 0o644))

	fw := &ffmpeg.Wrapper{BinPath: "ffmpeg", ProbeBinPath: "ffprobe", Runner: &fakeFfmpegRunner{}}
	mi := &mediainfo.Wrapper{BinPath: "mediainfo", Runner: &fakeMediainfoRunner{responses: map[string]string{}}}

	_, err := Gather(context.Background(), fw, mi, dir, nil)
	assert.Error(t, err)
}
