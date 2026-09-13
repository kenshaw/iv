# testdata

Test data for `iv`, one directory per decoder (`iv --list` names them all).
Files here are inputs for `go test ./...` and `./test.sh`; nothing is a golden
output.

Encoders have no test data of their own: `TestRoundTrip` encodes
`png/rose.png` with every one of them and decodes the result back. The `pdf`
encoder is the exception to what that checks -- a pdf carries a page rather
than a raster, so what comes back is whatever the reader rasterized it at, and
the test compares the shape rather than the pixel count. `encoder/pdf` covers
the pagination itself, which needs no file to exercise.

| Directory     | Decoder       | Notes                                                       |
| ------------- | ------------- | ----------------------------------------------------------- |
| `archives`    | `archives`    | The same two page comic as cbz, cbr, and cbt                |
| `binwalk`     | `binwalk`     | Affinity Designer file with an embedded png                 |
| `blitz`       | `blitz`       | A markdown and an html document                             |
| `bmp`         | `bmp`         |                                                             |
| `ffmpeg`      | `ffmpeg`      | Same clip as mkv and mp4                                    |
| `fitz`        | `fitz`        | XPS documents                                               |
| `fontimg`     | `fontimg`     | Static, variable, and monospace fonts                       |
| `gif`         | `gif`         | Still and animated                                          |
| `graphviz`    | `graphviz`    | DOT graph, sniffed by content rather than extension         |
| `icns`        | `icns`        | Apple icon image, eight resolutions from 32 to 1024         |
| `ico`         | `ico`         | Single and multi image icons                                |
| `jpeg`        | `jpeg`        |                                                             |
| `libreoffice` | `libreoffice` | Office documents; needs `soffice` in `$PATH`                |
| `lottie`      | `lottie`      | Noto animated emoji, as json, `.lot`, and a dotLottie       |
| `mermaid`     | `mermaid`     | Needs `mmdc` in `$PATH`                                     |
| `nativewebp`  | `nativewebp`  | Lossy and lossless webp, each with its png reference decode |
| `netpbm`      | `netpbm`      | pbm/pgm/ppm/pam, raw and plain                              |
| `png`         | `png`         |                                                             |
| `resvg`       | `resvg`       | SVGs, plain and gzipped, trivial to a large choropleth      |
| `strings`     | `data`, `qr`, | Command line arguments, not files -- see below              |
|               | `blitz-url`   |                                                             |
| `tag`         | `tag`         | Silent audio carrying embedded cover art                    |
| `tiff`        | `tiff`        | One file per compression the Go encoder supports            |
| `vips`        | `vips`        | heic/heif/jxl, plus a plain and a password protected pdf    |
| `winres`      | `winres`      | Windows PE with embedded icons                              |

Every decoder has test data. The two in `strings/` that reach the network are
the only ones that need it, and they skip themselves when it is not there.

## Strings

Some arguments to `iv` are not paths at all -- a `data:` URL, a `WIFI:` code,
a http URL -- so `strings/` holds one such argument per file, with the
extension `.iv_test_string` and a trailing newline that is not part of the
argument. The extension is deliberately not one any decoder claims, so these
files are never mistaken for something to render.

| File                                 | Decoder     | Decodes to           |
| ------------------------------------ | ----------- | -------------------- |
| `wifi.iv_test_string`                | `qr`        | 350x350 QR code      |
| `data-svg-base64.iv_test_string`     | `data`      | 100x100 svg          |
| `data-svg-urlencoded.iv_test_string` | `data`      | 64x64 svg            |
| `data-png-base64.iv_test_string`     | `data`      | 1x1 png              |
| `yahoo.iv_test_string`               | `blitz-url` | the page, 2400 wide  |
| `ifconfig-me.iv_test_string`         | `blitz-url` | the page, 2400 wide  |
| `finance-google.iv_test_string`      | `blitz-url` | the page, 2400 wide  |
| `microsoft-favicon.iv_test_string`   | `ico`       | 128x128 icon         |

`TestDecodeString` reads the directory and checks each one against a table
keyed by file name; a string added without an entry in that table fails, so
they cannot go quietly untested. `test.sh` passes the contents of each file as
an argument and skips them when walking for files to open.

The four URLs are the only test data that reaches the network. A fetch that
fails is reported as `blitz.ErrFetch`, which the test skips on, so an
unreachable site does not fail the suite while a blitz regression still does.

Only the width of a rendered page is checked: a page is as tall as whatever
the site served that minute, and the width is the one part of it `iv` decides.
The three pages are picked to differ -- `ifconfig.me` is a small static page,
`yahoo.com` a megabyte of markup, and `finance.google.com` a 302 to
`www.google.com/finance/` laid out in cards and tables. The redirect is the
point of that last one: the page has to be rendered against the url it was
served from rather than the one asked for, or everything relative in it
resolves against the wrong host.

`www.microsoft.com/favicon.ico` is the other half of what `blitz-url` does. A
url naming an image is still that image, so it is never rendered as a page --
`blitz-url` hands the bytes back to the pipeline and the `ico` decoder takes
them. The mime type is what the test asserts, because a page comes back
without one; the size is left alone, since it is Microsoft's icon to change.

`vips/file-sample_150kB.enc.pdf` is the plain pdf encrypted with the password
`password`, kept in step with `encPassword` in `ivcmd/decode_test.go` and
`ENC_PASSWORD` in `test.sh`. Both pass it with `--password`, since the decoder
prompts otherwise and neither harness has a terminal to prompt at.

`libreoffice/` covers the three document families and both generations of the
Microsoft formats, because they are detected differently:

| File                                                                             | Detected as                                                               |
| -------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `file-sample_100kB.docx`                                                         | `application/vnd.openxmlformats-officedocument.wordprocessingml.document` |
| `file_example_XLSX_50.xlsx`, `spreadsheet.xlsx`                                  | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`       |
| `file-sample_100kB.doc`, `file_example_XLS_50.xls`, `file_example_PPT_250kB.ppt` | `application/x-ole-storage`                                               |
| `file-sample_100kB.odt`                                                          | `application/vnd.oasis.opendocument.text`                                 |
| `file_example_ODS_100.ods`                                                       | `application/vnd.oasis.opendocument.spreadsheet`                          |
| `file_example_ODP_200kB.odp`                                                     | `application/vnd.oasis.opendocument.presentation`                         |
| `file-sample_100kB.rtf`                                                          | `text/rtf`                                                                |

The pre-2007 binary formats are all OLE compound files, so content sniffing
cannot tell a `.doc` from an `.xls` from a `.ppt` -- all three report
`application/x-ole-storage`, and the `libreoffice` decoder claims that type
outright rather than relying on the extension. `TestLibreOfficeRouting` checks
every one of these without needing `soffice`.

## Provenance

- `png/{card,dice,logo,rose,tux}.png` and all of `nativewebp/` -- the
  [Google WebP gallery][webp-gallery]. `-lossy` is the lossy-with-alpha
  encoding, `-lossless` the lossless one; each `.png` alongside is that
  file's reference decode.
- `png/precision.png`, `jpeg/precision.jpg`, `vips/precision.jxl`, and
  `nativewebp/precision.webp` -- the [JPEG XL test page][jxl-test], one image in
  four formats.
- `bmp/rose.bmp` -- the gallery rose, converted to bmp.
- `ico/Mathijssen-Tuxlets-Test-Dummy-Tux.ico` -- Mathijssen Tuxlets Test
  Dummy Tux.
- `resvg/test.svg` -- [SVG Repo][svg-repo]. `resvg/rect.svgz` is
  `resvg/rect.svg` gzipped: a `.svgz` sniffs as `application/gzip`, so only
  the extension identifies it.
- `icns/test.icns` -- generated from `resvg/test.svg`. Each entry is rendered
  from the svg at its own resolution rather than downsampled from a single
  raster, so every size is crisp. The eight entries, in file order, are the
  full set the `icns` encoder supports:

  | OSType | Pixels | OSType | Pixels |
  | ------ | ------ | ------ | ------ |
  | `ic10` | 1024   | `ic08` | 256    |
  | `ic14` | 512    | `ic07` | 128    |
  | `ic09` | 512    | `ic12` | 64     |
  | `ic13` | 256    | `ic11` | 32     |

  `ic14`/`ic13`/`ic12`/`ic11` are the @2x variants of 256/128/32/16, which is
  why 512 and 256 each appear twice. `iv -p N` selects between them.

- `lottie/` -- three of the [Noto animated emoji][noto-emoji], the set behind
  <https://googlefonts.github.io/noto-emoji-animation/>, retrieved with
  `scripts/fetch-noto-emoji.sh` in [xo/lottie][]. Copyright Google Inc.,
  licensed [CC BY 4.0][cc-by-4].
- `winres/go-winres.exe` -- built from [go-winres][].
- `fontimg/` -- Figtree, Noto Mono, and Ubuntu.

### Documents

`blitz/sample.md` and `blitz/sample.html` are the same idea in the two
languages the `blitz` decoder speaks. The markdown sniffs as `text/plain`,
which the decoder claims last so mermaid, graphviz and libreoffice get first
refusal; the html sniffs as `text/html`, which is its own. The html file is
deliberately not plain -- a gradient header, a flex row, a bordered table --
since what it is testing is a css engine rather than a markdown stylesheet.

### Comic archives

`archives/` holds one comic in three containers -- `science-preview.cbz`
(zip), `.cbr` (rar v5), and `.cbt` (tar) -- each with the same two pages,
`page_000.png` and `page_001.png`. None of these extensions is identified by
content: a cbz sniffs as `application/zip`, a cbt as `application/x-tar`, so
only the extension separates a comic from any other archive, and the
`archives` decoder pairs the two. `TestDecodeComicArchive` checks the routing,
checks each page comes out the same whichever container it was read from, and
checks the two pages actually differ so page selection cannot silently do
nothing. There is no `.cb7` file, though the decoder handles that too.

### Lottie animations

`lottie/` covers the three ways an animation reaches the decoder, since each
is identified differently:

| File           | Detected as         | Identified by                       |
| -------------- | ------------------- | ----------------------------------- |
| `rocket.json`  | `video/lottie+json` | the document, read by the sniffer   |
| `star.lot`     | `video/lottie+json` | the document, read by the sniffer   |
| `fire.lottie`  | `application/zip`   | the extension                       |

Nothing sniffs a lottie as anything but json, and `.json` is a name that says
nothing at all -- so the decoder reads the document instead, claiming it only
when the top level carries the frame rate, the in and out points, and the
composition size. `star.lot` is there to show the same sniff works whatever
the file is called, and to cover the `application/json` and `.lot` pairing
that catches a document the sniffer cannot read far enough into.

`fire.lottie` is a dotLottie: a zip holding `manifest.json` and
`animations/fire.json`. A zip sniffs as `application/zip` whatever is in it,
so only the extension separates this from any other archive. The decoder hands
the entries back to the pipeline, which sniffs the one it picks and comes
straight back here -- the same trip a comic archive's pages make.

### Derived files

`gif/`, `netpbm/`, and `tiff/` are generated from `png/tux.png` and
`resvg/test.svg`. Tux carries an alpha channel, the document icon is line art,
so between them they cover both what these formats need to represent.

- `gif/tux.gif`, `gif/test.gif` -- stills. `gif/animated.gif` is six frames
  alternating the two sources over white; `image.Decode` returns its first
  frame, which is what iv renders.
- `tiff/` -- `tux-uncompressed.tiff`, `tux-deflate.tiff`, and
  `test-deflate-predictor.tiff`. There is no LZW file: `x/image/tiff` decodes
  LZW but refuses to encode it.
- `tag/` -- ten seconds of silence in five containers, each with
  `png/tux.png` embedded as cover art.

  | File          | Detected as   | Metadata |
  | ------------- | ------------- | -------- |
  | `silent.mp3`  | `audio/mpeg`  | ID3v2.3  |
  | `silent.aac`  | `audio/mpeg`  | ID3v2.3  |
  | `silent.flac` | `audio/flac`  | Vorbis   |
  | `silent.ogg`  | `audio/ogg`   | Vorbis   |
  | `silent.m4a`  | `audio/x-m4a` | MP4      |

  The cover is embedded byte for byte, and the card the decoder draws embeds
  those bytes untouched, so `TestDecodeCard` in `decoder/tag` reads the card's
  svg before it is rasterized and checks the art against `png/tux.png` pixel
  for pixel. `TestDecodeTagCard` in `ivcmd` covers the other end, that the card
  comes out of the pipeline at the size it should. Note that `.aac` reports
  `audio/mpeg` too: it is ID3 wrapped ADTS, indistinguishable from mp3 by its
  leading bytes.

  These files carry no tags and no sound, which exercises the card at its
  emptiest: the title falls back to the file name, and the waveform is a flat
  line. The waveform itself needs `ffmpeg`, and the card is drawn without one
  when it is missing.

- `netpbm/` -- every format in the family, raw and plain (ASCII):

  | File             | Magic | Format           |
  | ---------------- | ----- | ---------------- |
  | `test.pbm`       | `P4`  | bilevel, raw     |
  | `test-plain.pbm` | `P1`  | bilevel, plain   |
  | `test.pgm`       | `P5`  | grayscale, raw   |
  | `test-plain.pgm` | `P2`  | grayscale, plain |
  | `tux.ppm`        | `P6`  | color, raw       |
  | `tux-plain.ppm`  | `P3`  | color, plain     |
  | `tux.pam`        | `P7`  | `RGB_ALPHA`, raw |

  PAM is the only one of these that keeps an alpha channel; the rest are
  composited onto white first. Content sniffing only recognizes `P1`/`P2`/`P3`
  and `P6`, so the `netpbm` decoder registers libmagic descriptions for the
  others -- see `RegisterMimeType` in `decoder/netpbm`.

[webp-gallery]: https://developers.google.com/speed/webp/gallery
[jxl-test]: https://jpegxl.info/resources/jpeg-xl-test-page
[svg-repo]: https://www.svgrepo.com
[go-winres]: https://github.com/tc-hib/go-winres
[noto-emoji]: https://github.com/googlefonts/noto-emoji
[xo/lottie]: https://github.com/xo/lottie
[cc-by-4]: https://creativecommons.org/licenses/by/4.0/
