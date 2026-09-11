# testdata

Test data for `iv`, one directory per decoder (`iv --list` names them all).
Files here are inputs for `go test ./...` and `./test.sh`; nothing is a golden
output.

| Directory     | Decoder       | Notes                                                       |
| ------------- | ------------- | ----------------------------------------------------------- |
| `archives`    | `archives`    | The same two page comic as cbz, cbr, and cbt                |
| `binwalk`     | `binwalk`     | Affinity Designer file with an embedded png                 |
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
| `markdown`    | `markdown`    |                                                             |
| `mermaid`     | `mermaid`     | Needs `mmdc` in `$PATH`                                     |
| `nativewebp`  | `nativewebp`  | Lossy and lossless webp, each with its png reference decode |
| `netpbm`      | `netpbm`      | pbm/pgm/ppm/pam, raw and plain                              |
| `png`         | `png`         |                                                             |
| `resvg`       | `resvg`       | SVGs, plain and gzipped, trivial to a large choropleth      |
| `strings`     | `data`, `qr`  | Command line arguments, not files -- see below              |
| `tag`         | `tag`         | Silent audio carrying embedded cover art                    |
| `tiff`        | `tiff`        | One file per compression the Go encoder supports            |
| `vips`        | `vips`        | heic/heif/jxl, plus a plain and a password protected pdf    |
| `winres`      | `winres`      | Windows PE with embedded icons                              |

No test data yet: `http`, left out because exercising it would need the
network.

## Strings

Some arguments to `iv` are not paths at all -- a `data:` URL, a `WIFI:` code --
so `strings/` holds one such argument per file, with the extension
`.iv_test_string` and a trailing newline that is not part of the argument.
The extension is deliberately not one any decoder claims, so these files are
never mistaken for something to render.

| File                                 | Decoder | Decodes to      |
| ------------------------------------ | ------- | --------------- |
| `wifi.iv_test_string`                | `qr`    | 350x350 QR code |
| `data-svg-base64.iv_test_string`     | `data`  | 100x100 svg     |
| `data-svg-urlencoded.iv_test_string` | `data`  | 64x64 svg       |
| `data-png-base64.iv_test_string`     | `data`  | 1x1 png         |

`TestDecodeString` reads the directory and checks each one against a table
keyed by file name; a string added without an entry in that table fails, so
they cannot go quietly untested. `test.sh` passes the contents of each file as
an argument and skips them when walking for files to open.

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

- `winres/go-winres.exe` -- built from [go-winres][].
- `fontimg/` -- Figtree, Noto Mono, and Ubuntu.

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
- `tag/` -- five seconds of silence in five containers, each with
  `png/tux.png` embedded as cover art. The `tag` decoder only ever reads the
  metadata, so the audio itself is silent to keep the files small.

  | File          | Detected as   | Metadata |
  | ------------- | ------------- | -------- |
  | `silent.mp3`  | `audio/mpeg`  | ID3v2.3  |
  | `silent.aac`  | `audio/mpeg`  | ID3v2.3  |
  | `silent.flac` | `audio/flac`  | Vorbis   |
  | `silent.ogg`  | `audio/ogg`   | Vorbis   |
  | `silent.m4a`  | `audio/x-m4a` | MP4      |

  The cover is embedded byte for byte, so `TestDecodeTagArt` can check the
  extracted image against `png/tux.png` pixel for pixel. Note that `.aac`
  reports `audio/mpeg` too: it is ID3 wrapped ADTS, indistinguishable from mp3
  by its leading bytes.

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
