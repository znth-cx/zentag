# Overview
zentag CLI tool in Go. Rename, retag metadata (chapters, general, covers), lint files for rule compliance.

# Tech Stack
- language: Go 1.26.4 (available in path)
- cobra: CLI
- viper: config
- log/slog: logging
- per fileformat writer engines:
  - M4B: ffmpeg remuxes chapters, then go-mp4tag writes tags+cover in-place (github.com/Sorrow446/go-mp4tag, in-process atom editing)
  - MP3: two-pass approach: ffmpeg copies audio stream, then go-taglib writes ID3v2.4-compliant tags (go.senan.xyz/taglib, embedded WebAssembly binary)
  - FLAC: ffmpeg writes metadata directly (Vorbis comments)
- MediaInfo for validating metadata
- audnexus wrapper for fetching from API
- https://github.com/barbashov/iso639-3 for language to ISO 639-3 codes

# Structure
App logic in core directory, room to support multiple UI formats (CLI, TUI, WebUI). For now, CLI only.

/core
  metadata/    canonical model, gather, merge (+ provenance)
  ruleset/     single source of truth for rule compliance. pure validation func for metadata (check in RULES.md for rules)
  writers/	   Translates final metadata object into final file output. All take single metadata object.
	M4BEngine/		ffmpeg remuxes chapters onto the output, then mp4tag writes tags + cover in place (in-container, no loose cover file)
	FLACEngine/		Uses ffmpeg
	MP3Engine/		two-pass approach: ffmpeg copies audio stream, then go-taglib writes ID3v2.4-compliant tags
  sources/	   various sources to construct metadata object from
	files/			Construct metadata object from files passed in. Single file or multi-file directory
	audnexus/		Requires ASIN to lookup. (spec in audnexus_openapi.json)
  mediainfo/   mediainfo CLI wrapper — source of truth for technical fields (case insensitive)
  isbn/        checksum validation
  lang/        iso639-3 library wrapper
  naming/      build file name for directory and multi-file using metadata object
  cover/       validate/resize provided cover. filepath only.
   ffmpeg/	   lightweight ffmpeg wrapper handling chapters (all formats) and metadata (FLAC only — M4B's tags/cover go through mp4tag, MP3's tags go through go-taglib).
   mp4tag/	   in-place M4B/M4A tag + cover writer (github.com/Sorrow446/go-mp4tag), no exec/shell-out — edits atoms directly instead of remuxing.
   mp3tag/	   in-place MP3 tag writer using go-taglib (go.senan.xyz/taglib), writes ID3v2.4-compliant frames, uses embedded WebAssembly binary.
/cmd/cli       cobra commands
/internal      config (viper), logging (slog) setup

## Metadata
When merging two metadata objects, emit conflicts first � caller asks user input for resolve, returns object w/ conflicts resolved. Conflict struct includes field defaulting to recommended choice, or -1 if hard tie. Caller submits conflict object, picks winner by choice. -1 means omit field.
All conflicts presented together, single return value.

## Ruleset
Big main check function calls many smaller-scope functions. Lets callers check specific part (e.g. ISBN validity) from ruleset instead of isbn library direct.

# Guidelines
- Always use /caveman:caveman ultra mode skill. Sub-agents too.
- All core library actions wrapped with ctx for cancels/timeouts.
- Every action logged via slog at appropriate level.
- App depends on user picking highest quality sources. Zentag may recommend, final say = user.
- Source files never modified.
- Don't parse filenames of input files/dirs. Two allowed exceptions: the part number from the start of a file in multi-file inputs, and the `[Source]` token (RULES.md §3) from the input directory/file base name, since that release-source fact lives nowhere else on disk.
- Always prompt user for design/architecture choices before making them.

# Supported media files
Currently M4B (AAC codec), MP3, FLAC only.

# Cobra Commands
zentag operates on single file or directory (may hold one+ media files).
- Source folder may contain extra non-media files � ignore those.

## transform
`zentag transform /path/to/item`

Not just fixes issues � fetches best info, overwrites fields where possible.

args: optional args for every metadata field user can specify
pipeline: if a saved session exists (and no --clean) resume straight from it — skip file/API gather AND conflict resolution (re-merging would repopulate cleared fields); user args still override on top. Otherwise gather (file metadata, API), merge with precedence user args > API > file, resolve conflicts. Then transform files into output dir after user confirm.
On a TTY: huh TUI drives it — phase 1 resolves merge conflicts (skipped on resume), phase 2 is a multi-tab edit form (Book data / Chapters / Submit) before accept. Keys: pgup/pgdn hop whole tabs (via a pagedForm wrapping the huh form; huh has no native group jump), tab/shift+tab move fields, alt+←/→ (or ctrl+←/→) jump a word, ctrl+w or alt+backspace delete a word. After Submit a spinner shows write progress. Non-TTY falls back to text prompts — a human must still answer them; there is no unattended/auto-yes mode.
Session: working metadata is dumped to a per-item JSON under session_dir (slugified abs path) after each phase, so a crash resumes and a re-run repopulates. Aborting (ctrl+c) or declining at Submit still saves in-progress edits before cancelling. Never deleted except `--clean`, which discards only that item's session.

## check
`zentag check /path/to/item`

Gathers max metadata from item, builds metadata object, validates via ruleset library. Prints report.

# Viper Config
Config file needs:
- ffmpeg binary location, defaults to ffmpeg (in path)
- mediainfo binary location, defaults to mediainfo (in path)
- output directory: where transformed files go
- session_dir: where per-item session JSON files live, defaults to <user config dir>/zentag/sessions
