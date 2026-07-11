// Package files constructs Metadata from files/dirs: tags (mediainfo), chapters (ffprobe), cover (ffmpeg/loose). Gather is strict (RULES.md §4), GatherGreedy tries fallbacks.
package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"codeberg.org/Ether/zentag/core/ffmpeg"
	"codeberg.org/Ether/zentag/core/lang"
	"codeberg.org/Ether/zentag/core/mediainfo"
	"codeberg.org/Ether/zentag/core/metadata"
)

// Gather constructs Metadata from path (single file or directory), reading tags only from RULES.md §4 locations for check validation. partNumberRe is nil for default leading-digits or a caller-validated pattern.
func Gather(ctx context.Context, fw *ffmpeg.Wrapper, mi *mediainfo.Wrapper, path string, partNumberRe *regexp.Regexp) (*metadata.Metadata, error) {
	return gather(ctx, fw, mi, path, mi.ReadTags, false, partNumberRe)
}

// GatherGreedy constructs Metadata like Gather but tries fallback tag locations and normalizes Language to ISO 639-3 for transform's best-effort completion.
func GatherGreedy(ctx context.Context, fw *ffmpeg.Wrapper, mi *mediainfo.Wrapper, path string, partNumberRe *regexp.Regexp) (*metadata.Metadata, error) {
	return gather(ctx, fw, mi, path, mi.ReadTagsGreedy, true, partNumberRe)
}

// tagReader reads book-wide tags; satisfied by both ReadTags and ReadTagsGreedy, letting gather share other steps between variants.
type tagReader func(ctx context.Context, path string) (mediainfo.TagSet, error)

func gather(ctx context.Context, fw *ffmpeg.Wrapper, mi *mediainfo.Wrapper, path string, readTags tagReader, normalizeLanguage bool, partNumberRe *regexp.Regexp) (*metadata.Metadata, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("gather %q: %w", path, err)
	}

	mediaFiles, err := resolveMediaFiles(path, info, partNumberRe)
	if err != nil {
		return nil, err
	}

	meta := &metadata.Metadata{
		OriginalPath:   path,
		MetadataOrigin: metadata.OriginFileMetadata,
		Source:         metadata.ReleaseSource(sourceFromName(filepath.Base(path))),
	}

	tags, err := readTags(ctx, mediaFiles[0].Path)
	if err != nil {
		return nil, fmt.Errorf("gather %q: read tags: %w", path, err)
	}
	meta.Author = tags.Author
	meta.Title = tags.Title
	meta.Subtitle = tags.Subtitle
	meta.Publisher = tags.Publisher
	meta.Year = tags.Year
	meta.Narrator = tags.Narrator
	meta.Description = tags.Description
	meta.Genre = tags.Genre
	meta.Series = tags.Series
	meta.Language = tags.Language
	if normalizeLanguage {
		if code, ok := lang.NormalizeToPart3(tags.Language); ok {
			meta.Language = code
		} else if code, ok := lang.CodeForName(tags.Language); ok {
			meta.Language = code
		}
	}
	meta.ISBN = tags.ISBN
	meta.ASIN = tags.ASIN

	for _, mf := range mediaFiles {
		tech, err := mi.Probe(ctx, mf.Path)
		if err != nil {
			return nil, fmt.Errorf("gather %q: probe %q: %w", path, mf.Path, err)
		}
		container, codec, err := mediainfo.NormalizeAudioFormat(mf.Path, tech)
		if err != nil {
			return nil, fmt.Errorf("gather %q: normalize format %q: %w", path, mf.Path, err)
		}
		chapters, err := fw.ReadChapters(ctx, mf.Path)
		if err != nil {
			return nil, fmt.Errorf("gather %q: read chapters %q: %w", path, mf.Path, err)
		}
		meta.Tracks = append(meta.Tracks, metadata.Track{
			Path:       mf.Path,
			PartNumber: mf.PartNumber,
			Container:  container,
			Codec:      codec,
			Bitrate:    tech.Bitrate,
			Chapters:   chapters,
		})
	}

	coverImage, coverMIME, err := selectCover(ctx, fw, path, info.IsDir(), mediaFiles[0].Path)
	if err != nil {
		return nil, fmt.Errorf("gather %q: read cover: %w", path, err)
	}
	meta.CoverImage = coverImage
	meta.CoverMIME = coverMIME

	return meta, nil
}
