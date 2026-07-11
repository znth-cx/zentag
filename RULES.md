# Audiobook Upload Ruleset

---

## 1 — Allowed Content

### General prohibitions
- No DRM content.
- Do not upload works from the **Banned Authors** list or the **Banned Works** list (see below).
- No public domain audiobooks (Librivox, etc.).
- No pre-retail or leaked material.
- No podcasts.
- No video.
- Unofficial audiobooks not produced professionally (e.g. you can't record an audiobook yourself and upload it without staff permission).
- No AI-generated audiobooks unless they are from the original author or an institution associated with the original author.

### Banned Authors
- J.R.R. Tolkien
- Anne Perry
- Simon Scarrow
- Sara Gruen
- Joan Elliott
- Alan Dart
- Chris Mead
- Paul Moore & Gavin Jones
- Noah K Sturdevant
- Benedict Brown
- Erika T Wurth
- Randolph Lalonde
- Unpublished works of J. D. Salinger
- Andrea Sfiligoi
- Ana-Maria Babanica

### Banned Works
- Four Against Darkness Expanded Edition, and all associated content

---

## 2 — Primary Keys

Uploads must link one of the following, and both if possible:
- ISBN-13 or ISBN-10
- ASIN

Audiobooks without an ISBN or ASIN require staff approval to upload.

---

## 3 — Naming

### Name formats

**Directory Name:**
```
Author - Title (Year) Language Edition {Narrator} [Source] Container Codec Bitrate
```

**Track Name (Single file):**
Same name as the directory.

**Track Name (Multi-file):**
```
PartNumber. ChapterName - Title (Year)
```

### Field rules

**Author + Narrator**
- Only the primary author / narrator is allowed. Do not include multiple authors / narrators.
- The first author and narrator listed is the primary one.

**Title**
- Capitalization format is [title case](https://apastyle.apa.org/style-grammar-guidelines/capitalization/title-case).
- GraphicAudio titles must preserve the `(1 of 5)` part naming if the title is split into parts.

**Language**
- Uses the ISO-639-3 three-letter language codes — https://en.wikipedia.org/wiki/List_of_ISO_639_language_codes
- Example: English is `ENG`.

**Edition**
- Omitted if it is the base edition.
- Use your best judgement to determine if this release is special compared to the "base edition".
- Possible values:
  - Abridged
  - Full-Cast
  - GraphicAudio
  - UK (if the regular release was US)

**Source**
- WEB
- CD
- VINYL

**Container**
- M4B
- If the torrent is FLAC or MP3, it should be omitted from the name. These are not container formats.

**Codec**
- AAC
- MP3
- FLAC

**Bitrate**
- Formatted as `xxxkbps`.
- Examples: `192kbps`, `128kbps`, `64kbps`, etc.

**PartNumber**
- Padded with zeros so that it has the same number of digits as the highest Part number.
- Example: if the highest part is 100, then Part 1 should be padded as `001`.

**ChapterName**
- If there are no chapter names, use the format `Chapter N`, where `N` is the part number (without padded zeros).

---

## 4 — File Metadata

For these tags to be valid, they must show up as named in the table below when viewed in MediaInfo. If they do not, the upload may be rejected or marked trumpable for missing metadata.

FFmpeg is recommended. For M4B uses `udata` > `meta` with a handler of type `mdta` with `keys` + `ilist` atoms.

### Multi-field tags
- If a tag can contain multiple elements, they must be separated by `;`.
- If the name itself contains `;`, escape it with `\`: `\;`.
- If the name contains `\`, escape it with `\`: `\\`.

### Tag table

| Tag Name | Required | Multiple | Comments |
|---|---|---|---|
| author | Yes | Yes | |
| artist | No | Yes | Some software reads the author name from this field, so it is preferred to set this to the author value. |
| title | Yes | — | |
| subtitle | Yes | — | Required if there is a subtitle. |
| publisher | No | Yes | Name of the publisher. |
| year | Yes | — | |
| narrator | Yes | Yes | |
| composer | No | Yes | Some software reads narrator from this. You are encouraged to set this to the same value as the narrator field. |
| description | No | — | Synopsis of the book. Removing advertisements that do not describe the plot from the synopsis is recommended, even if it is present on the source website. |
| genre | No | Yes | |
| series | Yes | Yes | Can be omitted if the book is not in a series. |
| series-part | Yes | Yes | Can be omitted if the book is not in a series. |
| language | Yes | — | ISO-639-3 code (e.g. `eng`) or the full English language name (e.g. `English`). `en` (ISO-639-1) is also accepted as an alias for English. May be set on the General track or the Audio track's own Language field — MP3/FLAC taggers commonly only expose the latter. |
| isbn | Yes | — | ISBN-13 is preferred, else ISBN-10. |
| asin | Conditional | — | Preferred, however not required if ISBN is available. |
| Chapter Metadata | Required if in source | Yes | Refer to the Chapters section. |

---

## 5 — Slots

Every unique edition has its own separate set of slots. A unique edition is when an audiobook has (non-exhaustive list):
- different narrators
- different language
- a unique source (WEB, CD, Vinyl, Cassette)
- abridged vs unabridged

Note that purchasing an audiobook from different digital retailers does **not** grant its own slots.
- Example: purchasing from Audible and from the publisher directly are both WEB formats, and if they do not have other differences they cannot co-exist.
- However, if one was from Audible and the other was from a CD sold by the publisher, they are unique editions.

Each edition has three slots:
- **Atmos** — in the highest quality available (Formats: TrueHD, DD+, or DD)
- **Lossless** — (Formats: FLAC)
- **Lossy** — (Formats: M4B, MP3)

---

## 6 — Trumping

There are two tags:
- **Trumpable**: the release can be downloaded, modified, and re-uploaded to fix the issue.
- **Upgradable**: the release does not fit our standards, but the release can't be modified to fix the issue and needs a new/different rip (bitrate too low, lossy transcode, etc.).

Being marked trumpable means you should attempt to fix the release and re-upload it, otherwise another user may do it. Being marked upgradable is fine if there is no better quality available to you.

> **Note:** Exceptions can be granted if your specific edition cannot meet one of these requirements.

### Trumpable
- File names that do not meet requirements are marked trumpable.
- Improper/missing MD5 hash in a FLAC is trumpable.
- No cover as metadata or loose file is trumpable.
- No chapters are trumpable when the source includes them (chapters in metadata for M4B, and in the filename for multi-file).
- M4B uploads as split files are trumpable. They should be a single file.
- Covers as separate files are trumpable (e.g. `cover.jpg`) for M4B uploads. They should be embedded as metadata.
- Covers over 3MB are trumpable.
- Missing required metadata fields are trumpable unless they do not exist.
- Metadata or filenames with spelling mistakes that are more than one or two characters affecting legibility are trumpable.
- Uploads that are not inside a directory are trumpable. Even single file uploads are required to be within a directory.
- Extra files not specified in the rules.
- Personal advertisements in metadata.

### Upgradable
- Bitrates below 64kbps are Upgradable unless it is the best quality source.
- Non-M4B releases in the lossy slot are marked Upgradable unless it is the original source and no better quality exists.
- Lossy → Lossy transcodes are Upgradable.
- Large artifacts or issues with the recording are Upgradable.

---

## 7 — Upload Description

Spectrals are required for lossless audiobooks for at least one track. Spectrals must be present for at least one track. Make sure to adjust the input flac and the `-S` flag, which sets the starting time to grab the spectrals from. It should ideally start somewhere in the middle of the track.

Adjust the **bolded** parts below (`example.flac` input and the `-S` start time):

- Large sample spectral:
```
sox example.flac -n remix 1 spectrogram -x 3000 -y 513 -z 120 -w Kaiser -S 15:00 -d 5:00 -o spectral-large.png
```

- Small sample spectral:
```
sox example.flac -n remix 1 spectrogram -X 800 -y 1025 -z 120 -w Kaiser -S 10:00 -d 0:02 -o spectral-small.png
```

---

## 8 — Covers

Covers are required.

- Must be digital covers. No photographs taken of physical media.
- A well-made digital scan is allowed if a digital cover cannot be found.
- If you cannot find a cover, use Helpdesk to ask for an exception.
- M4B uploads require a cover, either embedded within the file or as a `cover.jpg`/`cover.png` alongside it. zentag always embeds the cover in-container for M4B.
- Multi-file audiobooks such as MP3 and FLAC require covers as a `cover.jpg` in the root of the folder.
- Covers must be under 3MB.

---

## 9 — Chapters

- Chapters are required if the original source included them. The original source is the original retail source, not where the uploader downloaded it from.
- Custom chapters that follow book order are allowed as long as they do not contain defects. Please note custom chapters in the description.
- Chapters in M4B must be in the QuickTime format. Nero is optional for compatibility.