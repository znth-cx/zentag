package main

import (
	"errors"
	"fmt"

	"codeberg.org/Ether/zentag/core/ffmpeg"
	"codeberg.org/Ether/zentag/core/mediainfo"
	"codeberg.org/Ether/zentag/core/ruleset"
	"codeberg.org/Ether/zentag/core/sources/audnexus"
	"codeberg.org/Ether/zentag/core/sources/files"
	"github.com/spf13/cobra"
)

// newFFmpegWrapper/newMediaInfoWrapper: package vars, not direct calls,
// so tests can swap in fake Runners.
var (
	newFFmpegWrapper    = ffmpeg.New
	newMediaInfoWrapper = mediainfo.New
)

var jsonOutput bool

// errViolationsFound signals nonzero exit; report already printed, avoid dup on stderr.
var errViolationsFound = errors.New("violations found")

var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Check an item's metadata for rule compliance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		path := args[0]

		fw := newFFmpegWrapper(cfg.FFmpegPath, cfg.FFprobePath)
		mi := newMediaInfoWrapper(cfg.MediaInfoPath)

		meta, err := files.Gather(ctx, fw, mi, path, nil)
		if err != nil {
			return err
		}

		// Best-effort audnexus lookup, file's ASIN tag only (check has
		// no --asin flag/prompt). Missing ASIN or failed lookup just
		// skips the chapter-count comparison.
		if meta.ASIN != "" {
			apiMeta, err := audnexus.Gather(ctx, newAudnexusWrapper(""), meta.ASIN, "")
			if err != nil {
				logger.Warn("check: audnexus lookup failed, continuing without it", "asin", meta.ASIN, "error", err)
			} else {
				meta.AudnexusChapterCount = apiMeta.AudnexusChapterCount
			}
		}

		violations := ruleset.Validate(ctx, meta)

		var report string
		if jsonOutput {
			report, err = formatReport(violations, true)
			if err != nil {
				return fmt.Errorf("check %q: %w", path, err)
			}
		} else {
			report = formatCheckReport(violations)
		}
		fmt.Fprintln(cmd.OutOrStdout(), report)

		if len(violations) > 0 {
			return errViolationsFound
		}
		return nil
	},
}

func init() {
	checkCmd.Flags().BoolVar(&jsonOutput, "json", false, "output violations as JSON")
	rootCmd.AddCommand(checkCmd)
}
