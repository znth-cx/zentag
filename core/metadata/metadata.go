// Package metadata defines the canonical Metadata struct: one audiobook item, consumed by every downstream package.
package metadata

import "time"

// MetadataOrigin identifies which source produced a Metadata object during gather/merge.
type MetadataOrigin string

const (
	OriginUserArgs     MetadataOrigin = "user_args"
	OriginAudnexus     MetadataOrigin = "audnexus"
	OriginFileMetadata MetadataOrigin = "file_metadata"
	// OriginSession tags metadata restored from a saved session file; ranks below user args, above audnexus in Merge's precedence.
	OriginSession MetadataOrigin = "session"
)

// ReleaseSource is RULES.md §3's naming "Source" field.
type ReleaseSource string

const (
	ReleaseSourceWEB      ReleaseSource = "WEB"
	ReleaseSourceCD       ReleaseSource = "CD"
	ReleaseSourceVinyl    ReleaseSource = "VINYL"
	ReleaseSourceCassette ReleaseSource = "CASSETTE"
)

// SeriesEntry pairs a series name with a part number; Part is a string since parts can be non-integer (e.g. "1.5").
type SeriesEntry struct {
	Name string
	Part string
}

// Chapter is a single chapter marker within one Track's timeline.
type Chapter struct {
	Title string
	Start time.Duration
	End   time.Duration
}

// Track is one file's technical facts and chapter timeline. Container/Codec are RULES.md §3 tokens, not raw mediainfo Format strings; see mediainfo.NormalizeAudioFormat.
type Track struct {
	Path       string
	PartNumber int // multi-file items only; parsed from filename
	Container  string
	Codec      string
	Bitrate    int // kbps
	Chapters   []Chapter
}

// MaxYear bounds Year at parse and write sites; keeps int32 tag atoms safe.
const MaxYear = 9999

// Metadata is the canonical model for one audiobook item (single file or multi-file directory).
type Metadata struct {
	OriginalPath   string
	MetadataOrigin MetadataOrigin

	Author      []string // index 0 = primary; artist tag values fold in here at gather time
	Title       string
	Subtitle    string
	Publisher   []string // index 0 = primary
	Year        int      // 0 = unset
	Narrator    []string // index 0 = primary; composer tag values fold in here at gather time
	Description string
	Genre       []string // index 0 = primary
	Series      []SeriesEntry
	Language    string // ISO-639-3 code, e.g. "eng"
	ISBN        string
	ASIN        string

	CoverImage []byte
	CoverMIME  string

	Edition string // e.g. "Abridged", "Full-Cast", "GraphicAudio", "UK"; "" = base edition
	Source  ReleaseSource

	Tracks []Track

	// AudnexusChapterCount: audnexus's chapter count for this ASIN (0 = not looked up, or no data). Set only by the audnexus source; compared by ruleset.CheckAudnexusChapters as an advisory check.
	AudnexusChapterCount int
}
