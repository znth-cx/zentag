# RULES.md

## 1 — Overview

These rules define the standards for audiobook uploads, covering naming conventions, file metadata, chapters, covers, and trumping/upgradable status.

## 2 — Primary Keys: ISBN/ASIN

- Audiobooks without an ISBN or ASIN require staff approval to upload.
- The ISBN/ASIN must be for the audiobook version, not the ebook/paperback/hardcover/etc.
- Uploads must link one of the following (both if possible):
  - ISBN-13 / ISBN-10
  - ASIN

## 3 — Naming

### File Naming Convention

| Directory Name | `Author - Title (Year) Language Edition {Narrator} [Source] Container Codec Bitrate` |
| Track Name (Single file) | Same name as directory. |
| Track Name (Multi-file) | `PartNumber. ChapterName - Title (Year)` |

**Author + Narrator:**
- Only the primary author / narrator is allowed. Do not include multiple authors / narrators.
- First author and narrator listed is primary.

**Title:**
- Capitalization format is [title case](https://apastyle.apa.org/style-grammar-guidelines/capitalization/title-case)
- GraphicAudio titles must preserve the (1 of 5) part naming if it is split into parts
- Do not include the subtitle

**Year:**
- Must be the year the audiobook was released, not the date the original book was published.

**Language:**
- Uses the ISO-639-3 3 letter language codes — https://iso639-3.sil.org/code_tables/639/data
- ex. English is ENG

**Edition:**
- Omitted if the base edition
- Use your best judgement to determine if this release is special from the "base edition":
- Non-exhaustive list of examples below.
  - Abridged
  - Full-Cast
  - GraphicAudio
  - UK if the regular release was US

**Source:**
- WEB
- CD
- VINYL

**Container:**
- M4B
- If the torrent is FLAC or MP3 it should be omitted from the name. These are not container formats.

**Codec:**
- AAC
- MP3
- FLAC

**Bitrate:**
- Formatted as "xxxkbps"
- ex. 192kbps, 128kbps, 64kbps etc...

**PartNumber:**
- PartNumber should be padded with zeros such that it has the same number of digits as the highest Part number.
- ex. If the highest part is 100, then Part 1 should be padded as "001"

**ChapterName:**
- If there are no chapter names, use the format "Chapter N" where N is the part number (without padded zeros)

**Disc:**
- If the directory is separated by disc parts in the original source, they must use the "Disc x" naming scheme. Where x is the disc number.

### Torrent Title

Same as the directory name.

## 4 — File Metadata

For these tags to be valid they must show up in MediaInfo. If they do not, the upload may be rejected or marked trumpable for missing metadata.

**Multi-field tags:**
- If a tag can contain multiple elements, they must be separated by `;`.
- If the name itself contains `;`, escape it with `\`: `\;`.
- If the name contains `\`, escape it with `\`: `\\`.

### M4B

| Tag Name | Required | Multiple | Comments |
|----------|----------|----------|----------|
| `.ART` | Yes | Yes | The author name(s). |
| `.nam` | Yes | No | Title of the book. Do not include the subtitle here. |
| `.day` | Yes | No | Year released. |
| `.wrt` | Yes | Yes | Narrator(s) |
| `----com.apple.iTunes:SERIES` | Yes | Yes | Can be omitted if the book is not in a series. |
| `----com.apple.iTunes:SERIES-PART` | Yes | Yes | Can be omitted if the book is not in a series. |
| `Language` | Yes | No | Must be present on the audio track as a mdhd tag. |
| `----com.apple.iTunes:ISBN` | Yes | No | Either the ISBN or ASIN must be present as a tag. ISBN-13 is preferred, else ISBN-10. |
| `----com.apple.iTunes:ASIN` | Yes | No | Either the ISBN or ASIN must be present as a tag. |
| `covr` | Yes | No | Cover for the audiobook. See Cover section. |
| `----com.apple.iTunes:PUBLISHER` | No | Yes | Name of the publisher. |
| `----com.apple.iTunes:YEAR` | No | No | Custom tag to be more explicit. Not required. |
| `desc` | No | No | Synopsis of the book. Removing advertisements that do not describe the plot from the synopsis is recommended, even if it is present on the source website. |
| `.gen` | No | Yes | Genres and tags |
| `Chapter Metadata` | Required if in retail source | Yes | Refer to Chapter section. |

### MP3

| Tag Name | Required | Multiple | Comments |
|----------|----------|----------|----------|
| `TIT2` (Title) | Yes | No | Title of the book. Do not include the subtitle here. |
| `TPE1` (Artist) | Yes | Yes | The author name(s). |
| `TDRC` (Recording Time / Year) | Yes | No | Year released as a 4-digit number (YYYY). |
| `TLAN` (Language) | Yes | No | Language code (e.g., "eng", "spa"). |
| `TCOM` (Composer) | Yes | Yes | Narrator(s). |
| `TXXX:SERIES` | Yes | Yes | Series name(s). Can be omitted if the book is not in a series. |
| `TXXX:SERIES-PART` | Yes | Yes | Series part(s). Can be omitted if the book is not in a series. |
| `TXXX:ISBN` | Yes | No | ISBN of the book. Either ISBN or ASIN must be present. ISBN-13 is preferred, else ISBN-10. |
| `TXXX:ASIN` | Yes | No | ASIN of the book. Either ISBN or ASIN must be present. |
| `TALB` (Album) | No | No | Set to the same value as Title. |
| `TPUB` (Publisher) | No | Yes | The publisher(s). |
| `TXXX:YEAR` | No | No | Custom tag for year, set to the same value as Date. |
| `COMM` (Comment) | No | No | Description or summary of the book. |
| `TCON` (Genre) | No | Yes | Genre(s) of the book. |
| `TIT3` (Subtitle) | No | No | Subtitle of the book (if present). |
| `TXXX:NARRATOR` | No | Yes | Custom tag for narrator(s). |

### FLAC

| Tag Name | Required | Multiple | Comments |
|----------|----------|----------|----------|
| `author` | Yes | Yes | |
| `title` | Yes | No | |
| `year` | Yes | No | |
| `narrator` | Yes | Yes | |
| `series` | Yes | Yes | Can be omitted if the book is not in a series. |
| `series-part` | Yes | Yes | Can be omitted if the book is not in a series. |
| `language` | Yes | No | |
| `isbn` | Yes | No | Required if the ASIN not present. ISBN-13 preferred to ISBN-10. |
| `asin` | Yes | No | Required if the ISBN is not added. |
| `artist` | No | Yes | Some software reads the author name from this field so it is preferred to set this to the author value. |
| `subtitle` | No | No | |
| `publisher` | No | Yes | Name of the publisher. |
| `composer` | No | Yes | Some software reads narrator from this. You are encouraged to set this to the same value as the narrator field. |
| `description` | No | No | Synopsis of the book. Removing advertisements that do not describe the plot from the synopsis is recommended, even if it is present on the source website. |
| `genre` | No | Yes | |

## 5 — Slots

Every unique edition has its own separate set of slots. A unique edition is when an audiobook has (non-exhaustive list):
- different narrators
- different language
- a unique source (WEB, CD, Vinyl)
- abridged vs unabridged

Note that purchasing an audiobook from different digital retailers does not grant its own slots. Example: purchasing from Audible and the publisher directly are both WEB formats and if they do not have other differences they cannot co-exist. However if one was from Audible and the other was from a CD sold by the publisher they are unique editions.

Each edition has three slots:
- Atmos in the highest quality available. (Formats: TrueHD, DD+, or DD)
- Lossless (Formats: FLAC)
- Lossy (Formats: M4B, MP3)

## 6 — Trumping

We have two tags:
- **Trumpable**: the release can be downloaded, modified, and re-uploaded to fix the issue.
- **Upgradable**: the release does not fit our standards but the release cant be modified to fix the issue and needs a new/different rip (bitrate too low, lossy transcode..)

Being marked trumpable means you should attempt to fix the release and re-upload otherwise another user may do it. However, being marked upgradable is fine if there is no better quality available to you. Exceptions can be granted if your specific edition cannot meet one of these requirements.

**Note:** In order to trump another release, your upload must not have any quality issues.

### Trumpable

- File names that do not meet requirements marked trumpable.
- Improper/missing MD5 hash in a FLAC is trumpable.
- No cover as metadata or loose file is trumpable.
- No chapters are trumpable when the source includes them (chapters in metadata for M4B and in the filename for multi-file).
- M4B uploads as split files are trumpable. They should be a single file unless they are from a multi-part Disc release (Disc 1, Disc 2...).
- Covers as separate files are trumpable (ex. cover.jpg) for M4B uploads. They should be embedded as metadata.
- Covers over 3MB are trumpable.
- Missing required metadata fields are trumpable unless they do not exist.
- Metadata or filenames with spelling mistakes that are more than one or two characters affecting legibility are trumpable.
- Uploads that are not inside a directory are trumpable. Even single file uploads are required to be within a directory.
- Extra files not specified in the rules.
- Personal advertisements in metadata.

### Upgradable

- Bitrates below 64kbps are Upgradable unless it is the best quality source.
- non-M4B releases in the lossy slot are marked Upgradable unless original source and no better quality exists.
- Lossy -> Lossy transcodes are Upgradable.
- Large artifacts or issues with the recording are Upgradable.

## 7 — Upload Page Metadata

### MediaInfo

- MediaInfo is required; Use the MediaInfo field on the upload page.

## 8 — Covers

Covers are required.

- Must be digital covers. No photographs taken of physical media.
- A well made digital scan is allowed if a digital cover cannot be found.
- If you cannot find a cover, use Helpdesk to ask for an exception.
- M4B uploads require covers embedded within the file.
- Multi-file audiobooks such as MP3 and FLAC require covers as a cover.jpg in the root of the folder.
- Covers must be under 3MB

## 9 — Chapters

- Chapters are required if the original source included it. Original source is the original retail source, not where the uploader downloaded it from.
- Custom chapters that follow book order are allowed as long as they do not contain defects. Please note custom chapters in the description.
- Chapters in M4B must be in the QuickTime format. Nero is optional for compatibility.