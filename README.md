# zentag

CLI tool for renaming, retagging, and linting audiobook metadata (M4B, MP3, FLAC).

> **You are responsible for metadata correctness.** zentag does a best effort fetch for metadata from external sources and the file's metadata to make suggestions. Correct information during in the edit form if it is wrong. It is not zentag's fault if incorrect information was suggested.

## Required dependencies

Install these before using zentag, and ensure they're on your `PATH` (or point to them in config, see below):

- [`ffmpeg`](https://ffmpeg.org/) / `ffprobe`: chapter and tag muxing (MP3/FLAC), chapter remuxing (M4B)
- [`mediainfo`](https://mediaarea.net/en/MediaInfo): reads back written tags for validation. Make sure to install the CLI version, not the GUI version.

## Install

### Linux
- Download a release from https://github.com/znth-cx/zentag/releases
- Extract the archive (ex. `tar -xzf zentag_0.1.0_linux_amd64.tar.gz`).
- Place the binary into your path. The local non-sudo path is `~/.local/bin`.
- You can now use the zentag command.

### Windows
- Download a release from https://github.com/znth-cx/zentag/releases
- Extract it and find the zentag.exe inside.
- Place the binary on your local or system PATH.
- Invoke the zentag command in your terminal, double clicking/opening the binary via file explorer will not work!

### MacOS
- you get the darwin award

### Post Install

On first run, zentag writes a default config to your user config dir (e.g. `~/.config/zentag/zentag.yaml` on Linux or `%APPDATA%\Roaming\zentag` on Windows) if none exists. Edit it to set:

- `ffmpeg_path`, `ffprobe_path`, `mediainfo_path`: binary locations (default: assume on `PATH`)
- `output_dir`: where transformed files go (default: `./zentag-output`)
- `session_dir`: where in-progress transform sessions are saved (default: user config dir)

## Commands

### `zentag check /path/to/item`

> zentag check is good to catch most blatant issues, but you are responsible for verifying. This is only a helper, not a source of truth.

Reads a single file or directory (of media files) and validates its existing metadata against zentag's ruleset. Prints a report of violations, doesn't change anything. Use `--json` for JSON output for scripting.

```
zentag check ./MyBook
```

### `zentag transform /path/to/item`

Fetches the best available metadata (from file tags and Audible/audnexus lookup by ASIN), merges it with anything you pass via flags, resolves conflicts, and writes a corrected copy of the files to `output_dir`. Source files are never modified.

Progress is saved after each phase, so a crash or Ctrl+C can resume where you left off on the next run. Use `--clean` to discard a saved session and start fresh.

```
zentag transform ./MyBook --asin B0XXXXXXX
```

Run `zentag transform --help` for the full list of field-override flags (`--author`, `--title`, `--series`, `--cover`, etc.).

## Help

Every command supports `--help`:

```
zentag --help
zentag transform --help
zentag check --help
```

Global flags: `--config <path>` (override config file location), `-v`/`--verbose` (debug logging).

## Build

Requires Go 1.26.4+.

```
go build -o zentag ./cmd/cli/
```

Produces a `zentag` (or `zentag.exe` on Windows) binary in the current directory.