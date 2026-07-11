package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"codeberg.org/Ether/zentag/core/cover"
	"codeberg.org/Ether/zentag/core/ffmpeg"
	"codeberg.org/Ether/zentag/core/mediainfo"
	"codeberg.org/Ether/zentag/core/metadata"
	"codeberg.org/Ether/zentag/core/naming"
	"codeberg.org/Ether/zentag/core/ruleset"
	"codeberg.org/Ether/zentag/core/session"
	"codeberg.org/Ether/zentag/core/sources/audnexus"
	"codeberg.org/Ether/zentag/core/sources/files"
	"codeberg.org/Ether/zentag/core/writers/FLACEngine"
	"codeberg.org/Ether/zentag/core/writers/M4BEngine"
	"codeberg.org/Ether/zentag/core/writers/MP3Engine"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var cleanFlag bool

// newAudnexusWrapper: package var, not a direct call, so tests can
// swap in a fake HTTPClient.
var newAudnexusWrapper = audnexus.New

// writeM4B: M4BEngine.Write as a var so tests can fake it instead of
// running the real ffmpeg-remux-then-mp4tag pipeline.
var writeM4B = M4BEngine.Write

var transformCmd = &cobra.Command{
	Use:   "transform [path]",
	Short: "Fetch the best metadata and rewrite an item's files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		path := args[0]

		fw := newFFmpegWrapper(cfg.FFmpegPath, cfg.FFprobePath)
		mi := newMediaInfoWrapper(cfg.MediaInfoPath)

		userArgs, err := userArgsFromFlags(ctx, cmd)
		if err != nil {
			return fmt.Errorf("transform %q: %w", path, err)
		}

		// --clean discards only this item's session.
		if cleanFlag {
			if err := session.Clean(ctx, cfg.SessionDir, path); err != nil {
				return fmt.Errorf("transform %q: %w", path, err)
			}
		}

		var merged *metadata.Metadata
		var conflicts []metadata.Conflict
		var filesCfg *filesTabConfig // built when interactive; feeds ASIN/conflicts/edit tabs' Files tab

		if sm, found := loadSession(ctx, path); found {
			// Resume: skip re-gathering and conflict resolution, re-merging
			// would repopulate fields the user deliberately cleared. User
			// args still override on top; --clean forces a fresh gather.
			merged = sm
			merged.MetadataOrigin = metadata.OriginSession
			applyUserOverrides(merged, userArgs)
			logger.InfoContext(ctx, "transform: resuming from saved session", "path", path)
		} else {
			fileMeta, err := gatherFileMeta(ctx, cmd, fw, mi, path)
			if err != nil {
				return err
			}
			sources := []*metadata.Metadata{userArgs}
			flagASIN, _ := cmd.Flags().GetString("asin")
			asin := resolveASIN(cmd, fileMeta)
			region, _ := cmd.Flags().GetString("region")
			if isInteractive(cmd) {
				filesCfg = newFilesTabConfig(ctx, fw, mi, fileMeta.Tracks)
			}
			switch {
			case flagASIN == "" && isInteractive(cmd):
				// ASIN tab always shows unless --asin was passed, even with a
				// file-tag ASIN prefill: user still confirms/changes it.
				// Runs audnexus.Gather itself (runASINForm), no separate
				// lookup needed below.
				var apiMeta *metadata.Metadata
				asin, apiMeta, err = runASINForm(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), newAudnexusWrapper(""), region, asin, filesCfg)
				if err != nil {
					return err
				}
				if asin != "" {
					userArgs.ASIN = asin // fold in so Merge picks it up
				}
				if apiMeta != nil {
					sources = append(sources, apiMeta)
				}
			case asin != "":
				apiMeta, err := audnexus.Gather(ctx, newAudnexusWrapper(""), asin, region)
				if err != nil {
					logger.Warn("transform: audnexus lookup failed, continuing without it", "asin", asin, "error", err)
				} else {
					sources = append(sources, apiMeta)
				}
			}
			sources = append(sources, fileMeta)
			merged, conflicts = metadata.Merge(ctx, sources...)
		}

		if merged.Source == "" {
			merged.Source = metadata.ReleaseSourceWEB
		}

		in := bufio.NewScanner(cmd.InOrStdin())
		out := cmd.OutOrStdout()

		// dumps working metadata so a crash can resume; save failure is warn-only.
		saveSession := func(m *metadata.Metadata) {
			if err := session.Save(ctx, cfg.SessionDir, path, m); err != nil {
				logger.Warn("transform: could not save session", "error", err)
			}
		}

		if isInteractive(cmd) {
			// Phase 1: resolve conflicts. Abort still applies/saves current
			// selections so a re-run resumes where the user left off.
			if len(conflicts) > 0 {
				choices, accepted, err := runConflictForm(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), conflicts, filesCfg)
				if err != nil {
					return err
				}
				merged = metadata.ApplyResolutions(merged, conflicts, choices)
				if !accepted {
					saveSession(merged)
					fmt.Fprintln(out, "Aborted, progress saved. Re-run to resume (or --clean to discard).")
					return nil
				}
			}
			saveSession(merged)

			fixCover(ctx, merged)

			// Phase 2: edit all metadata. runEditForm always returns the
			// edited copy, even on abort/decline, so edits persist.
			edited, accepted, err := runEditForm(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), merged, fw, mi, filesCfg)
			if err != nil {
				return err
			}
			merged = edited
			fixCover(ctx, merged) // edit form's cover may need resizing
			saveSession(merged)
			if !accepted {
				fmt.Fprintln(out, "Aborted, progress saved. Re-run to resume (or --clean to discard).")
				return nil
			}
		} else {
			// Non-interactive fallback: numeric conflict picks + y/N.
			if len(conflicts) > 0 {
				choices, err := promptConflicts(in, out, conflicts)
				if err != nil {
					return err
				}
				merged = metadata.ApplyResolutions(merged, conflicts, choices)
			}
			fixCover(ctx, merged)
			saveSession(merged) // after fixCover so declining still keeps its changes

			violations := ruleset.Validate(ctx, merged)
			summary, err := summarizeMetadata(merged, violations)
			if err != nil {
				return fmt.Errorf("transform %q: %w", path, err)
			}

			proceed, err := confirmProceed(in, out, summary)
			if err != nil {
				return err
			}
			if !proceed {
				fmt.Fprintln(out, "Aborted, no files written.")
				return nil
			}
		}

		outputDir, err := writeOutput(ctx, cmd, fw, merged, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "Wrote %q\n", outputDir)
		return nil
	},
}

func writeOutput(ctx context.Context, cmd *cobra.Command, fw *ffmpeg.Wrapper, merged *metadata.Metadata, path string) (string, error) {
	out := cmd.OutOrStdout()
	spin := isInteractive(cmd)
	step := func(title string, fn func(context.Context) error) error {
		if spin {
			return runSpinner(ctx, out, title, fn)
		}
		return fn(ctx)
	}

	var dirName string
	if err := step("Building output name…", func(context.Context) error {
		var e error
		dirName, e = naming.DirectoryName(ctx, merged)
		return e
	}); err != nil {
		return "", fmt.Errorf("transform %q: %w", path, err)
	}

	outputDir := filepath.Join(cfg.OutputDir, dirName)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("transform %q: create output dir: %w", path, err)
	}

	trackNames := make([]string, len(merged.Tracks))
	for i := range merged.Tracks {
		name, err := naming.TrackName(ctx, merged, i)
		if err != nil {
			return "", fmt.Errorf("transform %q: %w", path, err)
		}
		trackNames[i] = name
	}

	title := fmt.Sprintf("Writing %d file(s) to %s…", len(merged.Tracks), dirName)
	if err := step(title, func(context.Context) error {
		return dispatchWrite(ctx, fw, merged, outputDir, trackNames)
	}); err != nil {
		return "", fmt.Errorf("transform %q: %w", path, err)
	}

	return outputDir, nil
}

func init() {
	transformCmd.Flags().String("author", "", "comma-separated author(s), primary first")
	transformCmd.Flags().String("title", "", "book title")
	transformCmd.Flags().String("subtitle", "", "book subtitle")
	transformCmd.Flags().String("publisher", "", "comma-separated publisher(s), primary first")
	transformCmd.Flags().Int("year", 0, "publication year")
	transformCmd.Flags().String("narrator", "", "comma-separated narrator(s), primary first")
	transformCmd.Flags().String("genre", "", "comma-separated genre(s), primary first")
	transformCmd.Flags().String("series", "", "series name")
	transformCmd.Flags().String("series-part", "", "part number within --series")
	transformCmd.Flags().String("language", "", "ISO-639-3 language code")
	transformCmd.Flags().String("isbn", "", "ISBN")
	transformCmd.Flags().String("asin", "", "Audible ASIN, also used for the audnexus lookup")
	transformCmd.Flags().String("edition", "", `edition, e.g. "Abridged", "Full-Cast"`)
	transformCmd.Flags().String("cover", "", "cover image, as a URL or local filepath")
	transformCmd.Flags().String("source", "", "release source: WEB, CD, VINYL, or CASSETTE")
	transformCmd.Flags().String("region", "", "audnexus region code (default us)")
	transformCmd.Flags().BoolVar(&cleanFlag, "clean", false, "ignore and discard this item's saved session before running")
	rootCmd.AddCommand(transformCmd)
}

// isInteractive: true only when both in/out are real terminals. Tests
// and piped/CI usage wire non-*os.File streams, so they take the text
// fallback path, which still requires explicit y/N; never unattended.
func isInteractive(cmd *cobra.Command) bool {
	inFile, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		return false
	}
	outFile, ok := cmd.OutOrStdout().(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(inFile.Fd())) && term.IsTerminal(int(outFile.Fd()))
}

// fixCover resizes merged's cover in place if it fails RULES.md §8;
// resize failure keeps original (warn only). No-op if no cover.
func fixCover(ctx context.Context, merged *metadata.Metadata) {
	if len(merged.CoverImage) == 0 {
		return
	}
	if ok, reason := cover.Validate(ctx, merged.CoverImage); !ok {
		fixed, fixErr := cover.Resize(ctx, merged.CoverImage)
		if fixErr != nil {
			logger.Warn("transform: cover resize failed, keeping original", "reason", reason, "error", fixErr)
			return
		}
		merged.CoverImage = fixed
		merged.CoverMIME = "image/jpeg"
	}
}

// loadSession returns (session, true), or (nil, false) if missing or
// --clean. An unreadable session warns and is treated as absent, a
// corrupt file must never block a run.
func loadSession(ctx context.Context, path string) (*metadata.Metadata, bool) {
	if cleanFlag {
		return nil, false
	}
	sm, found, err := session.Load(ctx, cfg.SessionDir, path)
	if err != nil {
		logger.Warn("transform: ignoring unreadable session", "error", err)
		return nil, false
	}
	return sm, found
}

// applyUserOverrides copies src's set fields (from userArgsFromFlags)
// over dst, so --flags win when resuming a session without a re-merge.
func applyUserOverrides(dst, src *metadata.Metadata) {
	if len(src.Author) > 0 {
		dst.Author = src.Author
	}
	if src.Title != "" {
		dst.Title = src.Title
	}
	if src.Subtitle != "" {
		dst.Subtitle = src.Subtitle
	}
	if len(src.Publisher) > 0 {
		dst.Publisher = src.Publisher
	}
	if src.Year != 0 {
		dst.Year = src.Year
	}
	if len(src.Narrator) > 0 {
		dst.Narrator = src.Narrator
	}
	if len(src.Genre) > 0 {
		dst.Genre = src.Genre
	}
	if len(src.Series) > 0 {
		dst.Series = src.Series
	}
	if src.Language != "" {
		dst.Language = src.Language
	}
	if src.ISBN != "" {
		dst.ISBN = src.ISBN
	}
	if src.ASIN != "" {
		dst.ASIN = src.ASIN
	}
	if src.Edition != "" {
		dst.Edition = src.Edition
	}
	if src.Source != "" {
		dst.Source = src.Source
	}
	if len(src.CoverImage) > 0 {
		dst.CoverImage = src.CoverImage
		dst.CoverMIME = src.CoverMIME
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// userArgsFromFlags builds the user-args Metadata source (highest
// merge precedence). Only flags actually passed populate a field, so
// unset flags never shadow another source with a zero value.
func userArgsFromFlags(ctx context.Context, cmd *cobra.Command) (*metadata.Metadata, error) {
	f := cmd.Flags()
	meta := &metadata.Metadata{MetadataOrigin: metadata.OriginUserArgs}

	// series-part only pairs inside a --series entry, alone it is lost.
	if f.Changed("series-part") && !f.Changed("series") {
		return nil, errors.New("--series-part requires --series")
	}

	if f.Changed("author") {
		s, _ := f.GetString("author")
		meta.Author = splitCSV(s)
	}
	if f.Changed("title") {
		meta.Title, _ = f.GetString("title")
	}
	if f.Changed("subtitle") {
		meta.Subtitle, _ = f.GetString("subtitle")
	}
	if f.Changed("publisher") {
		s, _ := f.GetString("publisher")
		meta.Publisher = splitCSV(s)
	}
	if f.Changed("year") {
		meta.Year, _ = f.GetInt("year")
	}
	if f.Changed("narrator") {
		s, _ := f.GetString("narrator")
		meta.Narrator = splitCSV(s)
	}
	if f.Changed("genre") {
		s, _ := f.GetString("genre")
		meta.Genre = splitCSV(s)
	}
	if f.Changed("series") {
		name, _ := f.GetString("series")
		part, _ := f.GetString("series-part")
		meta.Series = []metadata.SeriesEntry{{Name: name, Part: part}}
	}
	if f.Changed("language") {
		meta.Language, _ = f.GetString("language")
	}
	if f.Changed("isbn") {
		meta.ISBN, _ = f.GetString("isbn")
	}
	if f.Changed("asin") {
		meta.ASIN, _ = f.GetString("asin")
	}
	if f.Changed("edition") {
		meta.Edition, _ = f.GetString("edition")
	}
	if f.Changed("source") {
		s, _ := f.GetString("source")
		meta.Source = metadata.ReleaseSource(s)
	}
	if f.Changed("cover") {
		s, _ := f.GetString("cover")
		img, mime, err := cover.Load(ctx, s)
		if err != nil {
			return nil, fmt.Errorf("transform: load cover %q: %w", s, err)
		}
		meta.CoverImage = img
		meta.CoverMIME = mime
	}
	return meta, nil
}

// gatherFileMeta runs files.GatherGreedy, retrying with a user-supplied
// part-number regexp on *files.PartNumberError. A declined/aborted
// prompt (or non-interactive run) surfaces the original gather error
// instead of looping forever.
func gatherFileMeta(ctx context.Context, cmd *cobra.Command, fw *ffmpeg.Wrapper, mi *mediainfo.Wrapper, path string) (*metadata.Metadata, error) {
	var partNumberRe *regexp.Regexp
	for {
		fileMeta, err := files.GatherGreedy(ctx, fw, mi, path, partNumberRe)
		if err == nil {
			return fileMeta, nil
		}

		var partErr *files.PartNumberError
		if !errors.As(err, &partErr) || !isInteractive(cmd) {
			return nil, err
		}
		re, promptErr := promptPartNumberRegex(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), path)
		if promptErr != nil {
			return nil, promptErr
		}
		if re == nil {
			return nil, err // declined/aborted: surface original gather error
		}
		partNumberRe = re
	}
}

// resolveASIN: --asin flag if set, else fileMeta's ASIN, else "" (no lookup).
func resolveASIN(cmd *cobra.Command, fileMeta *metadata.Metadata) string {
	if asin, _ := cmd.Flags().GetString("asin"); asin != "" {
		return asin
	}
	return fileMeta.ASIN
}

// promptConflicts: blank keeps recommended, "-1" omits the field, any
// other in-range index picks that candidate. Invalid input reprompts.
func promptConflicts(in *bufio.Scanner, out io.Writer, conflicts []metadata.Conflict) (map[string]int, error) {
	choices := make(map[string]int, len(conflicts))
	for _, c := range conflicts {
		fmt.Fprintf(out, "Field: %s\n", c.Field)
		for i, v := range c.Values {
			marker := ""
			if i == c.Recommended {
				marker = " (recommended)"
			}
			fmt.Fprintf(out, "  [%d]%s %q (%s)\n", i, marker, v, c.Origins[i])
		}
		fmt.Fprintf(out, "Choice [%d]: ", c.Recommended)

		choice := c.Recommended
		for {
			if !in.Scan() {
				if err := in.Err(); err != nil {
					return nil, fmt.Errorf("transform: reading conflict choice: %w", err)
				}
				break // EOF: keep recommended default
			}
			line := strings.TrimSpace(in.Text())
			if line == "" {
				break // blank: keep recommended default
			}
			n, err := strconv.Atoi(line)
			if err != nil || n < -1 || n >= len(c.Values) {
				fmt.Fprintf(out, "invalid choice %q, enter -1 or 0-%d: ", line, len(c.Values)-1)
				continue
			}
			choice = n
			break
		}
		choices[c.Field] = choice
	}
	return choices, nil
}

func summarizeMetadata(meta *metadata.Metadata, violations []ruleset.Violation) (string, error) {
	report, err := formatReport(violations, false)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Author: %s\n", strings.Join(meta.Author, ", "))
	fmt.Fprintf(&b, "Title: %s\n", meta.Title)
	if meta.Subtitle != "" {
		fmt.Fprintf(&b, "Subtitle: %s\n", meta.Subtitle)
	}
	fmt.Fprintf(&b, "Year: %d\n", meta.Year)
	fmt.Fprintf(&b, "Narrator: %s\n", strings.Join(meta.Narrator, ", "))
	fmt.Fprintf(&b, "Language: %s\n", meta.Language)
	if meta.Edition != "" {
		fmt.Fprintf(&b, "Edition: %s\n", meta.Edition)
	}
	fmt.Fprintf(&b, "Source: %s\n", meta.Source)
	if len(meta.CoverImage) > 0 {
		fmt.Fprintf(&b, "Cover: found (%s, %d bytes)\n", meta.CoverMIME, len(meta.CoverImage))
	} else {
		fmt.Fprintf(&b, "Cover: none\n")
	}
	fmt.Fprintf(&b, "Tracks: %d\n", len(meta.Tracks))
	b.WriteString(report)
	return b.String(), nil
}

// confirmProceed: true only for "y"/"yes" (case-insensitive); any
// other answer or EOF declines.
func confirmProceed(in *bufio.Scanner, out io.Writer, summary string) (bool, error) {
	fmt.Fprintln(out, summary)
	fmt.Fprint(out, "Proceed? [y/N]: ")

	if !in.Scan() {
		if err := in.Err(); err != nil {
			return false, fmt.Errorf("transform: reading confirmation: %w", err)
		}
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(in.Text()))
	return answer == "y" || answer == "yes", nil
}

// dispatchWrite picks the engine by meta.Tracks[0]'s Container/Codec
// (RULES.md §3). M4B embeds its cover in-container; MP3/FLAC get a
// loose cover.jpg/png per RULES.md §8.
func dispatchWrite(ctx context.Context, fw *ffmpeg.Wrapper, meta *metadata.Metadata, outputDir string, trackNames []string) error {
	if len(meta.Tracks) == 0 {
		return fmt.Errorf("transform: dispatchWrite: metadata has no tracks")
	}
	track := meta.Tracks[0]

	var err error
	switch {
	case track.Container == "M4B":
		err = writeM4B(ctx, fw, meta, outputDir, trackNames)
	case track.Codec == "MP3":
		err = MP3Engine.Write(ctx, fw, meta, outputDir, trackNames)
	case track.Codec == "FLAC":
		err = FLACEngine.Write(ctx, fw, meta, outputDir, trackNames)
	default:
		return fmt.Errorf("transform: unsupported container/codec %q/%q", track.Container, track.Codec)
	}
	if err != nil {
		return err
	}

	if track.Container != "M4B" && len(meta.CoverImage) > 0 {
		name := "cover.jpg"
		if meta.CoverMIME == "image/png" {
			name = "cover.png"
		}
		if err := os.WriteFile(filepath.Join(outputDir, name), meta.CoverImage, 0o644); err != nil {
			return fmt.Errorf("transform: write loose cover: %w", err)
		}
	}
	return nil
}
