# testdata

This directory holds the test data of `iv`, with one directory for each
decoder. `iv --list` names the decoders. The files are inputs for
`go test ./...` and `./test.sh`. None of them is an expected output.

The encoders have no test data of their own. `TestRoundTrip` encodes
`png/rose.png` with each encoder and decodes the result. A pdf holds a page,
not pixels, so for the `pdf` encoder the test compares the shape of the
result and not its size. `encoder/pdf` tests the pages, and needs no file.

| Directory    | Decoder      | Contents                                                      |
| ------------ | ------------ | ------------------------------------------------------------- |
| `archives`   | `archives`   | One comic of two pages, as cbz, cbr and cbt                   |
| `avif`       | `avif`       | 8, 10 and 12 bits, alpha, a grid, a turned image, BT.2020     |
| `binwalk`    | `binwalk`    | An Affinity Designer file with a png in it                    |
| `blitz`      | `blitz`      | Markdown and HTML documents                                   |
| `bmp`        | `bmp`        |                                                               |
| `exr`        | `exr`        | OpenEXR, half and float, in five compressions                 |
| `ffmpeg`     | `ffmpeg`     | One clip as mkv and mp4, and a test pattern, plain and turned |
| `fontimg`    | `fontimg`    | Static, variable and monospace fonts, WOFF, WOFF2, collection |
| `gif`        | `gif`        | Still and animated                                            |
| `hdr`        | `hdr`        | Radiance HDR and Portable Float Map                           |
| `heic`       | `heic`       | heic and heif, 10 bits, 4:4:4, alpha and grids                |
| `icns`       | `icns`       | An Apple icon image with eight sizes, from 32 to 1024         |
| `ico`        | `ico`        | Icons with one image and with many                            |
| `ivy`        | `ivy-*`      | Documents, ebooks, XPS and Photoshop files that xo/ivy reads  |
| `jpeg`       | `jpeg`       |                                                               |
| `jpeg2000`   | `jpeg2000`   | jp2 files and a raw codestream                                |
| `jxl`        | `jxl`        | Color, gray, squeeze and Display P3                           |
| `lottie`     | `lottie`     | Noto animated emoji, as json, `.lot` and dotLottie            |
| `matlab`     | `matlab`     | MAT-files of version 5                                        |
| `nativewebp` | `nativewebp` | Lossy and lossless webp, each with its png reference          |
| `netpbm`     | `netpbm`     | pbm, pgm, ppm and pam, raw and plain                          |
| `okage`      | `okage`      | SVGs, plain and gzipped, from a rectangle to a large map      |
| `png`        | `png`        |                                                               |
| `strings`    | `data`, `qr` | Command line arguments that are not files                     |
| `tag`        | `tag`        | Silent audio with cover art                                   |
| `tiff`       | `tiff`       | One file for each compression that the Go encoder writes      |
| `vcard`      | `vcard`      | Contacts, drawn as a business card with a QR code             |
| `vipsfile`   | `vipsfile`   | A file in the format of libvips                               |
| `winres`     | `winres`     | A Windows executable with icons                               |
| `zuzu`       | `zuzu`       | DOT graphs, Mermaid diagrams, and Vega charts with their data |

Each decoder has test data, except ten formats of ivy, which
[`docs/BACKLOG.md`](../docs/BACKLOG.md) lists. No test reaches the network.
A test fails when a command that it needs is missing, unless
`ivcmd/decode_test.go` names the command with its reason (D27 of
[`docs/PLAN.md`](../docs/PLAN.md)).

## Strings

Some arguments of `iv` are not files, such as a `data:` URL, a `WIFI:` code
or another uri. `strings/` holds one such argument in each file. The
extension is `.iv_test_string`, which no decoder takes, and the newline at
the end is not part of the argument.

The uri files cover the schemes that the `qr` decoder takes by name:
`mailto:`, `tel:`, `sms:`, `geo:`, `xmpp:`, `sip:`, `matrix:`, `magnet:`,
`ftp:`, `bitcoin:`, `ethereum:`, `lightning:`, `otpauth:`, `DPP:` and `WIFI:`.
`ssh.iv_test_string` covers any other scheme with an authority. `http:` and
`https:` are not among them, because they name a page that `blitz-url`
draws. A longer uri gives a larger QR code. `zbarimg --raw` reads each QR
code back as the string that went in.

`TestDecodeString` in `ivcmd` reads each file, and checks the size of its
image and its mime type against a table. A file that is not in the table fails
the test. `test.sh` passes the content of each file as an argument.

## URLs

`TestDecodeURL` in `ivcmd` serves `blitz/sample.html` and `ico/1.ico` from an
`httptest` server. It checks the two things that `blitz-url` does with a url.
It draws a page, also after a redirect. It gives an image that a url names
back to the pipeline, which the mime type of the result shows. The test
checks only the width of a page, which is the part that `iv` sets. D27
removed the live pages that the test fetched before.

## Documents of ivy

`ivy/` holds the files of the `ivy-*` decoders (D49):

- the office documents in the table below
- one book as epub, mobi, azw3 and fb2
- two XPS documents and three Photoshop files
- a pdf, and the same pdf with a password
- a csv table

`TestDecode` in `decoder/ivy` draws each one, and checks that an ivy decoder
takes it.

The office documents cover the three families and both generations of the
Microsoft formats, because libmagic detects each one differently:

| File                                                  | Detected as                                                               |
| ----------------------------------------------------- | ------------------------------------------------------------------------- |
| `file-sample_100kB.docx`                              | `application/vnd.openxmlformats-officedocument.wordprocessingml.document` |
| `file_example_XLSX_50.xlsx`, `spreadsheet.xlsx`       | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`       |
| `file-sample_100kB.doc`, `file_example_PPT_250kB.ppt` | `application/x-ole-storage`                                               |
| `file_example_XLS_50.xls`                             | `application/vnd.ms-excel`                                                |
| `file-sample_100kB.odt`                               | `application/vnd.oasis.opendocument.text`                                 |
| `file_example_ODS_100.ods`                            | `application/vnd.oasis.opendocument.spreadsheet`                          |
| `file_example_ODP_200kB.odp`                          | `application/vnd.oasis.opendocument.presentation`                         |
| `file-sample_100kB.rtf`                               | `text/rtf`                                                                |

The binary formats of Office 97 to 2003 are OLE compound files. libmagic
reads the directory in the file, and finds an `.xls`, but it gives only
`application/x-ole-storage` for a `.doc` and a `.ppt`. Their extension then
selects the decoder. `TestRouting` in `decoder/ivy` checks that rule.

`ivy/file-sample_150kB.enc.pdf` is `ivy/file-sample_150kB.pdf` with the
password `password`. `encPassword` in `ivcmd/decode_test.go` and
`decoder/ivy/ivy_test.go`, and `ENC_PASSWORD` in `test.sh`, give it with
`--password`, because the decoder asks at the terminal otherwise (D50).

## Sources

- `png/{card,dice,logo,rose,tux}.png` and all of `nativewebp/`: the
  [Google WebP gallery][webp-gallery]. A `-lossy` file is lossy with alpha,
  and a `-lossless` file is lossless. The `.png` next to each one is its
  reference decode.
- `png/precision.png`, `jpeg/precision.jpg`, `nativewebp/precision.webp`,
  `avif/precision.avif` and `jxl/precision.jxl`: one image of the
  [JPEG XL test page][jxl-test] in five formats.
- The other files of `heic/`: the test data of [gen2brain/h265][h265], under
  the MIT license.
- The other files of `avif/`: the test data of [gen2brain/gav1d][gav1d],
  under the BSD 2-Clause license.
- The other files of `jxl/`: the test streams of [gen2brain/jxl][jxl], under
  the BSD 3-Clause license.
- `jpeg2000/8bit_GRAY_DWT.jp2`: the test data of
  [mrjoshuak/go-jpeg2000][go-jpeg2000], under the Apache License 2.0.
- `exr/` except `rose.exr`: the test data and the conformance files of
  [mrjoshuak/go-openexr][go-openexr], under the Apache License 2.0.
  `sample.exr` has only a G and a Z channel, so it is drawn in green.
  `tiled.exr` holds noise, with NaN, negative and very large values.
- `matlab/testdouble_7.4_GLNX86.mat`: the test data of
  [scigolib/matlab][scigolib-matlab], which took it from SciPy, under the MIT
  license. It is one row of nine numbers.
- `bmp/rose.bmp`: the rose of the gallery, as a bmp.
- `ico/Mathijssen-Tuxlets-Test-Dummy-Tux.ico`: Mathijssen Tuxlets Test Dummy
  Tux.
- `okage/test.svg`: [SVG Repo][svg-repo]. `okage/rect.svgz` is
  `okage/rect.svg` compressed with gzip. A `.svgz` is detected as
  `application/gzip`, so only its extension tells that it is an svg.
- `icns/test.icns`: made from `okage/test.svg`, with each size drawn from
  the svg. It holds `ic10` (1024), `ic14` (512), `ic09` (512), `ic13` (256),
  `ic08` (256), `ic07` (128), `ic12` (64) and `ic11` (32), in that order.
  `ic14`, `ic13`, `ic12` and `ic11` are the double resolution forms of 256,
  128, 32 and 16. `iv -p N` selects one.
- `lottie/`: three of the [Noto animated emoji][noto-emoji], fetched with
  `scripts/fetch-noto-emoji.sh` of [xo/lottie][]. Copyright Google Inc.,
  under [CC BY 4.0][cc-by-4].
- `winres/go-winres.exe`: built from [go-winres][].
- `fontimg/`: Figtree, Noto Mono and Ubuntu. `NotoMono-Regular.woff` and
  `NotoMono-Regular.woff2` are `NotoMono-Regular.ttf` saved by fontTools
  4.68.0 (`TTFont.flavor`), as the SIL Open Font License of Noto Mono
  allows. `TestDecode` in `decoder/fontimg` checks that each one draws the
  same pixels as the TrueType file.
- `fontimg/IBMPlexSans.woff2`: the six faces of IBM Plex Sans 3.005 as one
  collection in WOFF2, from `text/fonts` of [zuzu][], as its D34 says.
  Copyright IBM Corp., under the SIL Open Font License 1.1, in
  `fontimg/LICENSE-ibm-plex.txt`. `TestCollection` in `decoder/fontimg`
  checks that `-p` selects each face.
- `ivy/sample.*`: `blitz/sample.html`, converted by `ebook-convert` of
  calibre 9.14 to epub, azw3 and fb2, with the title "iv sample".
  `sample.mobi` is the epub converted with `--mobi-file-type both`, because
  calibre cannot read back the MOBI 6 file that it makes from the html.
- `ivy/sales.csv`: written for `iv`, a table of eight columns with no
  orientation. `TestPaper` in `decoder/ivy` checks that it is drawn in
  landscape (D51).
- `ivy/*.psd`: made with ImageMagick. `alpha.psd` is `png/tux.png`,
  `cmyk.psd` is `png/rose.png` in CMYK, and `deep16.psd` is `png/rose.png` at
  16 bits.
- `ffmpeg/testsrc2.mp4`: the `testsrc2` pattern of ffmpeg n9.0.2, 10 seconds
  at 640x360 and 24 frames a second. `testsrc2-rotated.mp4` is its first 2
  seconds, turned 90 degrees in the display matrix, as a phone stores a video
  that it recorded upright. `TestAnimateRotated` in `decoder/ffmpeg` checks
  that it plays upright at 360x640. These commands made them:

  ```sh
  ffmpeg -f lavfi -i testsrc2=size=640x360:rate=24 -t 10 -c:v libx264 \
    -crf 28 -preset slow -pix_fmt yuv420p -movflags +faststart testsrc2.mp4
  ffmpeg -display_rotation 90 -i testsrc2.mp4 -t 2 -c copy \
    -movflags +faststart testsrc2-rotated.mp4
  ```

The source of `okage/choropleth.svg`, `okage/folder.svg`,
`okage/sunburst-book.svg`, `ffmpeg/sample_960x540.*`, `heic/cyberpunk.heic`,
`heic/dome.heic` and `heic/sample1.heif` is not recorded.

## Markdown and HTML

`blitz/sample.md` and `blitz/sample.html` hold the same content in the two
languages that `blitz` reads. libmagic gives `text/x-c` for the Markdown,
which is a guess at a language, so its extension selects the decoder. It
gives `text/html` for the html. The html uses a gradient, a flex row and a
table with borders, because it tests a css engine.

`blitz/html-fragments.md` is Markdown that holds html, as the README of usql
does: a centered `<div>` with an image and a row of links, `<sup>` in a
table, `<br>`, `<hr/>` and a `<p>`. libmagic gives `text/html`, and the `.md`
extension makes `blitz` read it as Markdown. The image is `png/logo.png`.

## Contacts

`vcard/john-doe.vcf` is a plain vCard 3.0. `vcard/folded.vcf` holds the hard
cases:

- a `TITLE` folded over two lines
- an `ORG` with an escaped comma
- a `TYPE="voice,work"` parameter, and a lower case `TYPE=cell`
- an `N` with a prefix and a suffix
- an `ADR`, and text that is not ASCII

The card holds the record in a QR code, so a phone that scans it gets the
contact as it was written.

## Comic archives

`archives/` holds one comic in three containers: `science-preview.cbz`
(zip), `.cbr` (rar 5) and `.cbt` (tar), each with the pages `page_000.png`
and `page_001.png`. A cbz is detected as `application/zip` and a cbt as
`application/x-tar`, so only the extension tells a comic from another
archive. `TestDecodeComicArchive` checks the routing, checks that each
container gives the same pages, and checks that the two pages differ. There
is no `.cb7` file.

## Lottie animations

The `lottie` decoder takes a lottie in three ways:

| File          | Detected as         | Taken by      |
| ------------- | ------------------- | ------------- |
| `rocket.json` | `video/lottie+json` | Its content   |
| `star.lot`    | `video/lottie+json` | Its content   |
| `fire.lottie` | `application/zip`   | Its extension |

The decoder takes a json document only when it has the frame rate, the in
and out points and the size of a lottie. `star.lot` shows that the name of
the file does not matter. `fire.lottie` is a dotLottie, a zip that holds
`manifest.json` and `animations/fire.json`. The decoder gives the animation
back to the pipeline, as the `archives` decoder does with a page.

## Diagrams and charts

`zuzu/` holds the files that zuzu draws, and `TestDecode` in `decoder/zuzu`
draws each one.

- `booktest_sqlite3.dot` is a schema graph, as dbtpl and usql write it, with
  a label of HTML for each table. No libmagic type matches it, so the decoder
  finds it by its content.
- `flowchart.mmd`, `pie.mmd`, `gantt.mmd`, `journey.mmd` and `sankey.mmd`
  come from [xo/mermaid][], which `iv` used before zuzu (D36). `sankey.mmd`
  starts with front matter, so its extension selects the decoder.
- `aws.mmd` uses icons of the fa pack, such as `fa:fa-globe`.
- `emoji.mmd` puts an emoji in each label. `TestDecodeEmoji` fails when zuzu
  draws one as nothing (D40).
- The other `.mmd` files show the hardest features of each kind of Mermaid
  diagram. They come from [mermaid-go-renderer][] at commit `9af8f11`, which
  took them from Mermaid 12.2.0. Both are under the MIT license, in
  `zuzu/LICENSE-mermaid.txt`.
- The `.gv` files come from the gallery of Graphviz, at commit `ce9b383` of
  [graphviz.gitlab.io][], under the Eclipse Public License 2.0, in
  `zuzu/LICENSE-graphviz.txt`. libmagic gives `text/html` for `psg.gv`, and
  `grid.gv` and `Genetic_Programming.gv` put the `{` on the next line, so
  their extension selects the decoder.
- `sales.vg.json` is a Vega spec of a stacked bar chart, and `sales.vl.json`
  is the same chart in Vega-Lite. Each reads `data/sales.csv` by url (D44,
  D46). `TestDecodeVega` in `decoder/zuzu` draws both, and `TestVegaData`
  tests what the loader refuses.

## Made from other files

- `gif/tux.gif` and `gif/test.gif` are stills of `png/tux.png` and
  `okage/test.svg`. `gif/animated.gif` has six frames of the two on white.
  `iv` draws its first frame.
- `tiff/` holds `tux-uncompressed.tiff`, `tux-deflate.tiff` and
  `test-deflate-predictor.tiff`. There is no LZW file, because
  `x/image/tiff` reads LZW but does not write it.
- `netpbm/` holds each format of the family, raw and plain:

  | File             | Magic | Format           |
  | ---------------- | ----- | ---------------- |
  | `test.pbm`       | `P4`  | bilevel, raw     |
  | `test-plain.pbm` | `P1`  | bilevel, plain   |
  | `test.pgm`       | `P5`  | grayscale, raw   |
  | `test-plain.pgm` | `P2`  | grayscale, plain |
  | `tux.ppm`        | `P6`  | color, raw       |
  | `tux-plain.ppm`  | `P3`  | color, plain     |
  | `tux.pam`        | `P7`  | `RGB_ALPHA`, raw |

  Only PAM keeps an alpha channel. libmagic spells the grayscale type
  `image/x-portable-greymap`, so the decoder registers both spellings.

- `jpeg2000/rose.jp2` is `png/rose.png`, saved by libvips 8.18.7.
  `rose.j2k` is the codestream of its `jp2c` box, with nothing around it.
- The rose files of `hdr/`, `exr/`, `matlab/` and `vipsfile/` are
  `png/rose.png`, made 128 pixels wide with `vips thumbnail`:

  | File                 | Made by                                                   |
  | -------------------- | --------------------------------------------------------- |
  | `hdr/rose.hdr`       | mdouchement/hdr, from linear light, over black            |
  | `hdr/rose.pfm`       | libvips, from linear light (scRGB), over black            |
  | `exr/rose.exr`       | go-openexr, from linear light, premultiplied, with alpha  |
  | `matlab/rose-v5.mat` | scigolib/matlab, as an array of uint8 of 96 by 128 by 3   |
  | `vipsfile/rose.v`    | `vips vipssave`, as 8-bit sRGB with alpha                 |

  Each one draws the image that it was made from, so a decoder that turns,
  darkens or flips it fails to look like `png/rose.png`. libvips writes a
  Radiance HDR file 255 times too bright, so `rose.hdr` is made in Go.
  libvips writes a comment line in the header of a PFM, which the format does
  not have, and the `hdr` decoder skips it. scigolib/matlab writes a header
  that starts with `MATLAB MAT-file`, and libvips reads only one that starts
  with `MATLAB 5.0 MAT-file`, so the text of the header was changed to that.
- `tag/` holds ten seconds of silence in five containers, each with
  `png/tux.png` as its cover art:

  | File          | Detected as   | Metadata |
  | ------------- | ------------- | -------- |
  | `silent.mp3`  | `audio/mpeg`  | ID3v2.3  |
  | `silent.aac`  | `audio/mpeg`  | ID3v2.3  |
  | `silent.flac` | `audio/flac`  | Vorbis   |
  | `silent.ogg`  | `audio/ogg`   | Vorbis   |
  | `silent.m4a`  | `audio/x-m4a` | MP4      |

  The card keeps the bytes of the cover, so `TestDecodeCard` in
  `decoder/tag` compares its art with `png/tux.png`, pixel for pixel.
  `TestDecodeTagCard` in `ivcmd` checks the size of the card. The files have
  no tags and no sound, so the card shows the file name as its title, and a
  flat waveform.

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
[zuzu]: https://github.com/xo/zuzu
[h265]: https://github.com/gen2brain/h265
[gav1d]: https://github.com/gen2brain/gav1d
[jxl]: https://github.com/gen2brain/jxl
[go-jpeg2000]: https://github.com/mrjoshuak/go-jpeg2000
[go-openexr]: https://github.com/mrjoshuak/go-openexr
[scigolib-matlab]: https://github.com/scigolib/matlab
