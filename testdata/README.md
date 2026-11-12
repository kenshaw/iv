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

| Directory     | Decoder       | Notes                                                         |
| ------------- | ------------- | ------------------------------------------------------------- |
| `archives`    | `archives`    | The same two page comic as cbz, cbr, and cbt                  |
| `binwalk`     | `binwalk`     | Affinity Designer file with an embedded png                   |
| `blitz`       | `blitz`       | Markdown and html documents                                   |
| `bmp`         | `bmp`         |                                                               |
| `calibre`     | `calibre`     | One book as epub, mobi, azw3 and fb2; needs `ebook-convert`   |
| `ffmpeg`      | `ffmpeg`      | Same clip as mkv and mp4                                      |
| `fontimg`     | `fontimg`     | Static, variable, and monospace fonts                         |
| `gif`         | `gif`         | Still and animated                                            |
| `gxps`        | `gxps`        | XPS documents; needs `xpstopng`                               |
| `icns`        | `icns`        | Apple icon image, eight resolutions from 32 to 1024           |
| `ico`         | `ico`         | Single and multi image icons                                  |
| `jpeg`        | `jpeg`        |                                                               |
| `libreoffice` | `libreoffice` | Office documents; needs `soffice` in `$PATH`                  |
| `lottie`      | `lottie`      | Noto animated emoji, as json, `.lot`, and a dotLottie         |
| `nativewebp`  | `nativewebp`  | Lossy and lossless webp, each with its png reference decode   |
| `netpbm`      | `netpbm`      | pbm/pgm/ppm/pam, raw and plain                                |
| `okage`       | `okage`       | SVGs, plain and gzipped, trivial to a large choropleth        |
| `png`         | `png`         |                                                               |
| `psd`         | `psd`         | Photoshop, with alpha, CMYK, and 16-bit for the vips fallback |
| `strings`     | `data`, `qr`, | Command line arguments, not files -- see below                |
|               | `blitz-url`   |                                                               |
| `tag`         | `tag`         | Silent audio carrying embedded cover art                      |
| `tiff`        | `tiff`        | One file per compression the Go encoder supports              |
| `zuzu`        | `zuzu`        | DOT graphs and Mermaid diagrams, and ones zuzu cannot draw    |
| `vcard`       | `vcard`       | Contacts, drawn as a business card with a scannable code      |
| `vips`        | `vips`        | heic/heif/jxl, plus a plain and a password protected pdf      |
| `winres`      | `winres`      | Windows PE with embedded icons                                |

Every decoder has test data, and no test reaches the network. A test fails
when a command that it needs is missing, except for the commands that CI does
not install on an OS, which `ivcmd/decode_test.go` names with the reason
(D12 of `docs/PLAN.md`).

## Strings

Some arguments to `iv` are not paths at all -- a `data:` URL, a `WIFI:` code,
a uri -- so `strings/` holds one such argument per file, with the
extension `.iv_test_string` and a trailing newline that is not part of the
argument. The extension is deliberately not one any decoder claims, so these
files are never mistaken for something to render.

| File                                 | Decoder     | Decodes to           |
| ------------------------------------ | ----------- | -------------------- |
| `wifi.iv_test_string`                | `qr`        | 370x370 QR code      |
| `data-svg-base64.iv_test_string`     | `data`      | 100x100 svg          |
| `data-svg-urlencoded.iv_test_string` | `data`      | 64x64 svg            |
| `data-png-base64.iv_test_string`     | `data`      | 1x1 png              |

The uri files cover what the `qr` decoder claims: the schemes it knows by
name -- `mailto:`, `tel:`, `sms:`, `geo:`, `xmpp:`, `sip:`, `matrix:`,
`magnet:`, `ftp:`, `bitcoin:`, `ethereum:`, `lightning:`, `otpauth:`, `DPP:`
and `WIFI:` -- and, in `ssh.iv_test_string`, any other scheme with an
authority. `http:` and `https:` are deliberately not among them: those name a
document to fetch, and `blitz-url` renders it.

Each one is a different length, so each encodes to its own qr version and
comes out its own size. Every one of these round trips: `zbarimg --raw` reads
back exactly the string that went in.

`TestDecodeString` reads the directory and checks each one against a table
keyed by file name; a string added without an entry in that table fails, so
they cannot go quietly untested. `test.sh` passes the contents of each file as
an argument and skips them when walking for files to open.

No test reaches the network. `TestDecodeURL` in `ivcmd` serves
`blitz/sample.html` and `ico/1.ico` from an `httptest` server on this machine,
and covers the two things that `blitz-url` does with a url. It renders a page,
also after a redirect, against the url that the page was served from. It hands
an image that a url names back to the pipeline, which the mime type of the
result shows, because a rendered page comes back without one. Only the width of
a page is checked, which is the one part of it that `iv` decides. The four live
pages that the test fetched before were dropped by D12 of `docs/PLAN.md`.

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

The pre-2007 binary formats are all OLE compound files. libmagic reads the
directory inside one, so an `.xls` is reported as `application/vnd.ms-excel`,
but a `.doc` and a `.ppt` are only `application/x-ole-storage` -- the
`libreoffice` decoder claims both shapes rather than relying on the extension.
`TestLibreOfficeRouting` checks every one of these without needing `soffice`.

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
- `okage/test.svg` -- [SVG Repo][svg-repo]. `okage/rect.svgz` is
  `okage/rect.svg` gzipped: a `.svgz` sniffs as `application/gzip`, so only
  the extension identifies it.
- `icns/test.icns` -- generated from `okage/test.svg`. Each entry is rendered
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
- `calibre/` -- `blitz/sample.html`, converted with `ebook-convert` of calibre
  9.14 to epub, azw3 and fb2, with the title "iv sample". `sample.mobi` is the
  epub converted with `--mobi-file-type both`, because calibre cannot read
  back the MOBI 6 file that it writes from the html, which has no guide.
- `psd/` -- generated with ImageMagick: `alpha.psd` from `png/tux.png`,
  `cmyk.psd` from `png/rose.png` in the CMYK colorspace, and `deep16.psd` from
  `png/rose.png` at 16 bits. oov/psd cannot read `deep16.psd`, so it covers
  the fallback to libvips.

### Documents

`blitz/sample.md` and `blitz/sample.html` are the same idea in the two
languages the `blitz` decoder speaks. libmagic types plain text by what it
looks like, so the markdown comes back as `text/x-c` -- a guess at a language,
which `isGeneric` treats as saying nothing, leaving the extension to route it.
The html is read as `text/html`, which is a format rather than a guess. The html file is
deliberately not plain -- a gradient header, a flex row, a bordered table --
since what it is testing is a css engine rather than a markdown stylesheet.

`blitz/html-fragments.md` is Markdown that holds html, as the README of usql
does. It starts with a centered `<div>` that holds an image and a row of links,
and it uses `<sup>` in a table, `<br>`, `<hr/>` and a `<p>` of html. libmagic
types it as `text/html`, and the `.md` extension makes the `blitz` decoder read
it as Markdown. The image is `png/logo.png`, so the test reaches no network.

### Contacts

`vcard/john-doe.vcf` is a plain vCard 3.0. `vcard/folded.vcf` is the awkward
one, and everything in it is there to be awkward: a `TITLE` folded across two
lines, an `ORG` with an escaped comma, a `TYPE="voice,work"` parameter quoted
around its own separator, a lower case `TYPE=cell`, a structured `N` carrying
a prefix and a suffix, an `ADR`, and non-ascii throughout.

The card the decoder draws carries the record as it was written in a qr code,
so scanning it hands a phone the contact rather than a transcription of it.

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

### DOT graphs and Mermaid diagrams

`zuzu/` holds the graphs and diagrams that zuzu draws. `TestDecode` in
`decoder/zuzu` draws every file in `zuzu/`.

The first files are simple, and cover one kind each:

| File                   | Kind                         |
| ---------------------- | ---------------------------- |
| `booktest_sqlite3.dot` | DOT, with labels of HTML     |
| `flowchart.mmd`        | `flowchart`                  |
| `pie.mmd`              | `pie`                        |
| `gantt.mmd`            | `gantt`                      |
| `journey.mmd`          | `journey`                    |
| `sankey.mmd`           | `sankey-beta`                |
| `aws.mmd`              | `graph`, with many subgraphs |
| `emoji.mmd`            | `flowchart`, with emoji      |

`booktest_sqlite3.dot` is a schema graph in the form that dbtpl and usql
write. Each table is a label of HTML, and the edge of the foreign key runs
from the port of one row to the port of another. No libmagic type matches it,
so the decoder finds it by its content.

The five Mermaid files from `flowchart.mmd` to `sankey.mmd` come from the
corpus of [xo/mermaid][], which `iv` used before zuzu (D36). `sankey.mmd`
starts with front matter, a `config:` block. The decoder does not claim a file
by its content when it starts with front matter, so this file covers the route
by extension.

`aws.mmd` was written for the mermaid cli and its `--iconPacks`. Its labels
name icons of the fa pack, such as `fa:fa-globe`, and zuzu draws them.

`emoji.mmd` puts an emoji in each label. `TestDecodeEmoji` fails when zuzu
draws one of them as nothing (D40).

The other Mermaid files show the most complex features of each kind. They
come from the corpus of [mermaid-go-renderer][] at commit `9af8f11`, which
took them from the documentation and the end-to-end tests of Mermaid 12.2.0.
Both are under the MIT license, in `zuzu/LICENSE-mermaid.txt`.

| File                        | Kind and features                                             |
| --------------------------- | ------------------------------------------------------------- |
| `flowchart-large.mmd`       | A generated flowchart of 110 nodes, with subgraphs and styles |
| `flowchart-icons.mmd`       | A flowchart full of icons                                     |
| `flowchart-shapes.mmd`      | Every shape of `@{ shape: ... }`, each linked to the others   |
| `flowchart-markdown.mmd`    | Subgraphs and Markdown strings that span lines                |
| `gantt-sections.mmd`        | Four sections, critical tasks, and tasks that follow others   |
| `xychart-config.mmd`        | An xychart with all of its configuration in the front matter  |
| `quadrant-styles.mmd`       | A quadrant chart with points that have their own styles       |
| `treemap-nested.mmd`        | A treemap with many nested blocks                             |
| `wardley-tea-shop.mmd`      | The tea shop of a Wardley map                                 |
| `class-full.mmd`            | A class diagram with every kind of relation                   |
| `sequence-features.mmd`     | Actors made and destroyed, backgrounds, loops and notes       |
| `er-attributes.mmd`         | Attributes with keys and comments                             |
| `state-composite-fork.mmd`  | A fork and a join inside a composite state                    |
| `mindmap-shapes.mmd`        | Every node shape, with an icon of mdi                         |
| `architecture-nested.mmd`   | Groups nested deeply                                          |
| `gitgraph-cherry-pick.mmd`  | Branch order, a cherry-pick with a tag, bottom to top         |
| `timeline-sections.mmd`     | Sections, and long events with line breaks                    |
| `c4-context.mmd`            | The system context example of C4                              |
| `block-nested.mmd`          | Nested blocks that span columns                               |
| `requirement-relations.mmd` | Requirements and elements, with six kinds of relation         |
| `cynefin-domains.mmd`       | All four main domains, with two or three items in each        |
| `eventmodeling-data.mmd`    | Time frames of commands and events, with data blocks          |
| `ishikawa-deep.mmd`         | Twelve causes, a chain seven levels deep, a repeated label    |
| `kanban-metadata.mmd`       | Columns of cards with tickets, people and priorities          |
| `packet-tcp.mmd`            | The fields of a TCP header, with the bit numbers              |
| `railroad-grammar.mmd`      | The grammar of an expression, with loops and choices          |
| `treeview-config.mmd`       | A nested tree, with its layout and colors in the front matter |
| `venn-styles.mmd`           | Three sets, every union, text in each part, and fill styles   |
| `zenuml-returns.mmd`        | Sync and async messages that return values                    |

The `.gv` files come from the gallery of the Graphviz website, at commit
`ce9b383` of [graphviz.gitlab.io][]. They are under the Eclipse Public License
2.0, in `zuzu/LICENSE-graphviz.txt`. Each one uses features that the schema
graph does not: clusters, records with ports, labels of HTML in nested tables,
gradients, colors and fonts on each node, and `rank=same`.

Three of them also test how the decoder finds DOT. `psg.gv` starts with
comment lines that fill the first 512 bytes, and libmagic types it as
`text/html`. `grid.gv` and `Genetic_Programming.gv` put the `{` of the graph
on the line after `graph`. In all three, the extension decides.

### Derived files

`gif/`, `netpbm/`, and `tiff/` are generated from `png/tux.png` and
`okage/test.svg`. Tux carries an alpha channel, the document icon is line art,
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
  composited onto white first. libmagic types every one of them, though it
  spells the grey one `image/x-portable-greymap`, so the decoder registers
  both spellings. The description patterns in `decoder/netpbm` remain as a
  fallback for a libmagic build that only describes them.

[webp-gallery]: https://developers.google.com/speed/webp/gallery
[jxl-test]: https://jpegxl.info/resources/jpeg-xl-test-page
[svg-repo]: https://www.svgrepo.com
[go-winres]: https://github.com/tc-hib/go-winres
[xo/mermaid]: https://github.com/xo/mermaid
[mermaid-go-renderer]: https://github.com/bvolpato/mermaid-go-renderer
[graphviz.gitlab.io]: https://gitlab.com/graphviz/graphviz.gitlab.io
[noto-emoji]: https://github.com/googlefonts/noto-emoji
[xo/lottie]: https://github.com/xo/lottie
[cc-by-4]: https://creativecommons.org/licenses/by/4.0/
