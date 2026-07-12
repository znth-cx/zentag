package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/znth-cx/zentag/core/ebookmeta"
	"github.com/znth-cx/zentag/core/isbn"
	"github.com/znth-cx/zentag/core/lang"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/naming"
	ebooksource "github.com/znth-cx/zentag/core/sources/ebook"
	"github.com/znth-cx/zentag/core/writers/EbookEngine"
)

// test seam
var newEbookmetaWrapper = ebookmeta.New

// var so tests can stub; gates edit form to real terminals
var stdinIsInteractive = func() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

var ebookCmd = &cobra.Command{
	Use:   "ebook [path]",
	Short: "Organize, name and tag a single ebook",
	Args:  cobra.ExactArgs(1),
	RunE:  runEbook,
}

func init() {
	ebookCmd.Flags().String("author", "", "comma-separated author(s), primary first")
	ebookCmd.Flags().String("title", "", "book title")
	ebookCmd.Flags().Int("year", 0, "publication year")
	ebookCmd.Flags().String("isbn", "", "ISBN")
	ebookCmd.Flags().String("series", "", "series name")
	ebookCmd.Flags().String("series-part", "", "part number within --series")
	ebookCmd.Flags().String("edition", "", "edition, omit for first edition")
	ebookCmd.Flags().Bool("retail", false, "mark as unmodified digital retail release")
	ebookCmd.Flags().String("publisher", "", "comma-separated publisher(s)")
	ebookCmd.Flags().String("language", "", "ISO-639-3 language code")
	ebookCmd.Flags().String("description", "", "description")
	ebookCmd.Flags().String("tags", "", "comma-separated tags/genres")
	ebookCmd.Flags().String("asin", "", "ASIN")
	rootCmd.AddCommand(ebookCmd)
}

func runEbook(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	inputPath := args[0]

	// validate path/format before calibre provisioning prompt
	info, err := os.Stat(inputPath)
	if err != nil || info.IsDir() {
		return fmt.Errorf("input %q is not a readable file", inputPath)
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(inputPath), "."))
	format, ok := naming.FormatFromExtension(ext)
	if !ok {
		return fmt.Errorf("unsupported ebook format %q (allowed: epub, pdf, djvu, mobi, azw3)", ext)
	}

	w := newEbookmetaWrapper(cfg.EbookMetaPath)
	if err := w.Validate(ctx); err != nil {
		return fmt.Errorf("ebook-meta not usable; install calibre (https://calibre-ebook.com) or set ebook_meta_path in config: %w", err)
	}

	fileMeta := &metadata.Metadata{MetadataOrigin: metadata.OriginFileMetadata}
	if fm, rerr := ebooksource.Gather(ctx, w, inputPath); rerr != nil {
		logger.Warn("could not read existing ebook metadata", "path", inputPath, "err", rerr)
	} else {
		fileMeta = fm
	}

	userMeta, retail := ebookUserArgs(cmd)
	merged, conflicts := metadata.Merge(ctx, userMeta, fileMeta)
	merged = metadata.ApplyResolutions(merged, conflicts, nil)

	if stdinIsInteractive() {
		edited, accepted, eerr := runEbookEditForm(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), merged, retail)
		if eerr != nil {
			return eerr
		}
		if !accepted {
			return fmt.Errorf("aborted")
		}
		merged, retail = edited.meta, edited.retail
	}

	if verr := validateEbook(merged, retail); verr != nil {
		return verr
	}

	nameParams := naming.EbookNameParams{
		Author:     strings.Join(merged.Author, ", "),
		Series:     seriesName(merged),
		SeriesPart: seriesPart(merged),
		Title:      merged.Title,
		Year:       merged.Year,
		Language:   merged.Language,
		Edition:    merged.Edition,
		Format:     format,
		ISBN:       merged.ISBN,
		ASIN:       merged.ASIN,
		Retail:     retail,
	}
	folderName := naming.EbookFolderName(ctx, nameParams)
	fileName := naming.EbookFileName(ctx, nameParams)
	outDir := filepath.Join(cfg.OutputDir, folderName)
	if _, serr := os.Stat(outDir); serr == nil {
		return fmt.Errorf("output directory already exists, refusing to overwrite: %s", outDir)
	}
	if merr := os.MkdirAll(outDir, 0o755); merr != nil {
		return fmt.Errorf("create output dir: %w", merr)
	}
	outFile := filepath.Join(outDir, fileName+"."+ext)
	if cerr := copyFile(inputPath, outFile); cerr != nil {
		_ = os.RemoveAll(outDir)
		return fmt.Errorf("copy ebook into output: %w", cerr)
	}
	if format.Writable() {
		if werr := EbookEngine.Write(ctx, w, merged, outFile); werr != nil {
			_ = os.RemoveAll(outDir)
			return fmt.Errorf("write ebook metadata: %w", werr)
		}
	} else {
		logger.Warn("format cannot embed metadata; wrote name-only release", "format", format.String())
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", outFile)
	return nil
}

// ebookUserArgs builds Metadata from explicitly-set flags, plus retail bool.
func ebookUserArgs(cmd *cobra.Command) (*metadata.Metadata, bool) {
	f := cmd.Flags()
	m := &metadata.Metadata{MetadataOrigin: metadata.OriginUserArgs}
	if f.Changed("author") {
		s, _ := f.GetString("author")
		m.Author = splitCSV(s)
	}
	if f.Changed("title") {
		m.Title, _ = f.GetString("title")
	}
	if f.Changed("year") {
		m.Year, _ = f.GetInt("year")
	}
	if f.Changed("isbn") {
		m.ISBN, _ = f.GetString("isbn")
	}
	if f.Changed("series") {
		name, _ := f.GetString("series")
		part, _ := f.GetString("series-part")
		m.Series = []metadata.SeriesEntry{{Name: name, Part: part}}
	}
	if f.Changed("edition") {
		m.Edition, _ = f.GetString("edition")
	}
	if f.Changed("publisher") {
		s, _ := f.GetString("publisher")
		m.Publisher = splitCSV(s)
	}
	if f.Changed("language") {
		m.Language, _ = f.GetString("language")
	}
	if f.Changed("description") {
		m.Description, _ = f.GetString("description")
	}
	if f.Changed("tags") {
		s, _ := f.GetString("tags")
		m.Genre = splitCSV(s)
	}
	if f.Changed("asin") {
		m.ASIN, _ = f.GetString("asin")
	}
	retail, _ := f.GetBool("retail")
	return m, retail
}

// ISBN required unless retail; language normalized to ISO-639-3
func validateEbook(m *metadata.Metadata, retail bool) error {
	if m.Language != "" {
		if code, ok := resolveLanguage(m.Language); ok {
			m.Language = code
		}
	}
	var missing []string
	if len(m.Author) == 0 {
		missing = append(missing, "author")
	}
	if m.Title == "" {
		missing = append(missing, "title")
	}
	if m.Year == 0 {
		missing = append(missing, "year")
	}
	if !retail && m.ISBN == "" {
		missing = append(missing, "isbn")
	}
	if m.Language == "" {
		missing = append(missing, "language")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	if len(m.Series) > 0 && m.Series[0].Name != "" && m.Series[0].Part == "" {
		return fmt.Errorf("--series-part is required when --series is set")
	}
	if m.ISBN != "" {
		valid, err := isbn.Validate(m.ISBN)
		if err != nil {
			return fmt.Errorf("invalid isbn: %w", err)
		}
		if !valid {
			return fmt.Errorf("invalid isbn: check digit is wrong")
		}
	}
	if !lang.ValidCode(m.Language) {
		return fmt.Errorf("invalid language %q: want an ISO-639-3 code such as eng", m.Language)
	}
	return nil
}

func seriesName(m *metadata.Metadata) string {
	if len(m.Series) > 0 {
		return m.Series[0].Name
	}
	return ""
}

func seriesPart(m *metadata.Metadata) string {
	if len(m.Series) > 0 {
		return m.Series[0].Part
	}
	return ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
