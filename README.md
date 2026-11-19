# iv

`iv` is a command-line image viewer. It draws images in the terminal with the
[Kitty][kitty], [iTerm][iterm] or [Sixel][sixel] graphics protocol. It also
draws vector graphics, diagrams, documents, ebooks, fonts, video frames and
the cover art of audio files, and it can write the result to a file.

<p align="center">
  <a href="#screenshots" title="Screenshots">Screenshots</a> |
  <a href="#installing" title="Installing">Installing</a> |
  <a href="#using" title="Using">Using</a> |
  <a href="#formats" title="Formats">Formats</a> |
  <a href="#building" title="Building">Building</a> |
  <a href="#performance" title="Performance">Performance</a> |
  <a href="#documents" title="Documents">Documents</a> |
  <a href="https://github.com/kenshaw/iv/releases" title="Releases">Releases</a>
</p>

[![Releases][release-status]][releases]
[![Discord Discussion][discord-status]][discord]

[releases]: https://github.com/kenshaw/iv/releases "Releases"
[release-status]: https://img.shields.io/github/v/release/kenshaw/iv?display_name=tag&sort=semver "Latest Release"
[discord]: https://discord.gg/WDWAgXwJqN "Discord Discussion"
[discord-status]: https://img.shields.io/discord/829150509658013727.svg?label=Discord&logo=Discord&colorB=7289da&style=flat-square "Discord Discussion"
[kitty]: https://sw.kovidgoyal.net/kitty/graphics-protocol/
[iterm]: https://iterm2.com/documentation-images.html
[sixel]: https://saitoha.github.io/libsixel/

[Are We Sixel Yet?][arewesixelyet] lists the terminals that draw graphics.

[arewesixelyet]: https://www.arewesixelyet.com

## Installing

You can install `iv` [from a release][via Release], [with Homebrew][via
Homebrew], [from the AUR][via AUR] or [with Go][via Go].

[via Release]: #installing-from-a-release
[via Homebrew]: #installing-with-homebrew-macos-and-linux
[via AUR]: #installing-from-the-aur-arch-linux
[via Go]: #installing-with-go

### Installing from a release

1. [Download the release for your platform][releases].
2. Extract `iv` or `iv.exe` from the `.tar.bz2` or `.zip` file.
3. Move the file to a directory in your `$PATH` (Linux and macOS) or
   `%PATH%` (Windows).

### Installing with Homebrew (macOS and Linux)

Install `iv` from the [`kenshaw/iv` tap][iv-tap] with [`brew`][homebrew]:

```sh
$ brew install kenshaw/iv/iv
```

### Installing from the AUR (Arch Linux)

Install the [`iv-cli`][aur] package with an AUR helper, such as
[`yay`][yay]. The `iv-cli-bin` package installs the release binary instead.
The AUR packages follow a release within about 12 hours.

```sh
$ yay -S iv-cli
```

You can also build the package with [`makepkg`][arch-makepkg]:

```sh
$ git clone https://aur.archlinux.org/iv-cli.git && cd iv-cli
$ makepkg -si
```

### Installing with Go

```sh
$ go install github.com/kenshaw/iv@latest
```

This builds `iv` from source, so it needs the C libraries that the
[Building](#building) section names.

[homebrew]: https://brew.sh/
[iv-tap]: https://github.com/kenshaw/homebrew-iv
[aur]: https://aur.archlinux.org/packages/iv-cli
[yay]: https://github.com/Jguer/yay
[arch-makepkg]: https://wiki.archlinux.org/title/makepkg

## Using

```sh
# draw a file, or each file in a directory
$ iv /path/to/image_or_directory

# draw a remote image or web page
$ iv https://example.com/image.png

# draw the QR code of a WIFI code, or the image of a data: URL
$ iv 'WIFI:S:mynetwork;T:WPA;P:secret;;'

# draw page 4 of a document, an ebook or a comic archive
$ iv -p 4 /path/to/document.pdf

# play an animated gif, a lottie or a video
$ iv --animate /path/to/clip.webm

# play it again and again, until control-c
$ iv --loop /path/to/animation.gif

# write a file instead, in the format of its extension
$ iv --out out.webp /path/to/image.heic

# list the decoders, the encoders and the file extensions
$ iv --list

# show every option
$ iv --help
```

`iv` finds the protocol of the terminal by asking the terminal, so it works
over SSH and in tmux. In tmux, the Kitty and iTerm protocols need the
`allow-passthrough` option of tmux. To choose the protocol yourself, set
`TERM_GRAPHICS` to `kitty`, `iterm` or `sixel`.

### Sizing

`iv` fits each image to the terminal. All sizes are in pixels.

- The display size, `-W` and `-H`, is the largest size. At 0, the default,
  `iv` takes it from the terminal, less two rows for the file name and the
  prompt.
- The minimum size, `-w` and `-h`, is 64 by default.
- An image between the two sizes is drawn at its own size, with no
  resampling.
- An image that is larger than the display size is made smaller to fit it.
  An image that is smaller than the minimum size is made larger, up to the
  minimum.

When the two sizes do not agree, the display size wins. With the default
minimum, an image that is at the minimum in one direction is not made larger
in the other. So a banner of 1000x2 stays 1000x2. A minimum that you give on
the command line always applies: `--min-width=1000` makes an image of 367x344
into 1000x937.

A small image, such as an icon, is made larger by a whole number, so each of
its pixels becomes a square block. A vector image, such as an svg, a lottie
or a pdf, is drawn at the size that it is shown at, so it is sharp at any
size.

A file that `--out` writes is not fitted to the terminal. Only `-W` and `-H`
limit its size.

`-m` or `--mode` changes the rules:

| Mode       | What it does                                                   |
| ---------- | -------------------------------------------------------------- |
| `best-fit` | The default, as above                                          |
| `none`     | Draws each image at its own size                               |
| `width`    | Fits the width only, so a tall image stays tall                |
| `height`   | Fits the height only                                           |
| `shrink`   | Makes a large image smaller, and leaves a small image as it is |
| `stretch`  | Fills `-W` by `-H` exactly, and ignores the aspect ratio       |

### Pages

A file with more than one image shows one of them, and `-p N` selects it.
That is a page of a document, an ebook or a comic archive, an icon in an icon
file, a face in a font collection, or a frame of a lottie. A Markdown or HTML
document that is taller than the terminal is cut into pages of the size of
the terminal. An animated gif, webp or png shows its first frame.

### Animating

`-a` or `--animate` plays an animated gif, lottie or video, and `-l` or
`--loop` plays it again and again until you press control-c. Each one takes
exactly one file, and draws to the terminal only, so neither goes with
`--out`. `iv` clears the screen, writes the file name at the top, and plays
the frames under it.

| Format                      | What it plays                                                       |
| --------------------------- | ------------------------------------------------------------------- |
| Animated GIF                | Each frame, at the delays of the file, for at most a minute         |
| Lottie, including dotLottie | The timeline, at the frame rate of the file, for at most a minute   |
| Video (needs `ffmpeg`)      | The whole video, at its own speed, at most 30 frames a second       |

`--fps N` shows each frame for `1/N` of a second, and implies `--animate`.

Each frame has a time at which it is due. A video frame that is not ready
when the next frame is due is dropped, so the video keeps its own length on
a slow terminal. A gif or lottie frame that is late is drawn late, so such an
animation can play more slowly, but it shows each frame.

`--animate-mode` selects how the frames go to the terminal:

| Mode     | What it does                                                                  |
| -------- | ----------------------------------------------------------------------------- |
| `auto`   | The default. `stream` on kitty and Ghostty 1.4.0 or later, and else `frames`  |
| `stream` | Draws the frames into one image, which keeps two frames in the terminal       |
| `kitty`  | Sends all the frames once, and the terminal plays them and keeps them all     |
| `frames` | Draws a new image for each frame                                              |

`stream` and `kitty` need the Kitty protocol. A mode other than `auto`
implies `--animate`. `kitty` sends nothing again for `--loop`, but a large
animation does not fit in the memory of the terminal.

A file with only one frame, such as a jpeg, is not animated, and
`--animate` says so.

## Formats

`iv --list` shows the decoders of the binary that you have. The tables below
are for a default build. The formats of libvips depend on how libvips was
built, and the decoders that need `ffmpeg` work only when it is installed.

### Images

| Format                                         | Extensions                                                                    | Decoder      |
| ---------------------------------------------- | ----------------------------------------------------------------------------- | ------------ |
| PNG, including APNG                            | `png`                                                                         | `png`        |
| JPEG                                           | `jpg` `jpeg` `jpe` `jif` `jfif` `jfi`                                         | `jpeg`       |
| GIF                                            | `gif`                                                                         | `gif`        |
| WebP, lossy and lossless                       | `webp`                                                                        | `nativewebp` |
| TIFF                                           | `tif` `tiff`                                                                  | `tiff`       |
| Windows Bitmap                                 | `bmp` `dib`                                                                   | `bmp`        |
| Netpbm, raw and plain                          | `pbm` `pgm` `ppm` `pnm` `pam`                                                 | `netpbm`     |
| Windows icons and cursors                      | `ico` `cur`                                                                   | `ico`        |
| Apple Icon Image                               | `icns`                                                                        | `icns`       |
| Photoshop, the merged image                    | `psd` `psb`                                                                   | `ivy-psd`    |
| OpenRaster and Krita, the merged image         | `ora` `kra`                                                                   | `ivy-ora`    |
| Camera RAW, the preview that the camera embeds | `cr2` `cr3` `nef` `nrw` `arw` `srf` `sr2` `dng` `orf` `rw2` `pef` `srw` `raf` | `ivy-camera` |
| HEIC, HEIF and AVIF                            | `heic` `heif` `avif`                                                          | `vips`       |
| JPEG 2000 and JPEG XL                          | `jp2` `jpf` `j2k` `jxl` `jxs`                                                 | `vips`       |
| OpenEXR, Radiance HDR and PFM                  | `exr` `hdr` `pfm` `rad`                                                       | `vips`       |
| FITS, MATLAB, NIfTI and the format of vips     | `fits` `mat` `nii` `v`                                                        | `vips`       |

### Vector graphics and diagrams

| Format                                | Extensions                     | Decoder   |
| ------------------------------------- | ------------------------------ | --------- |
| SVG, plain and gzipped                | `svg` `svgz`                   | `okage`   |
| Lottie and dotLottie                  | `json` `lot` `lottie`          | `lottie`  |
| Graphviz DOT                          | `gv` `dot`                     | `zuzu`    |
| Mermaid                               | `mmd` `mermaid`                | `zuzu`    |
| Vega                                  | `vg.json`                      | `zuzu`    |
| Vega-Lite                             | `vl.json`                      | `zuzu`    |
| EMF and WMF, plain and compressed     | `emf` `emz` `wmf` `wmz`        | `ivy-emf` |
| EPS, the preview that the file embeds | `eps` `epsi` `epsf` `ept` `ps` | `ivy-ps`  |

`iv` takes a `.json` file as a lottie only when its content is a lottie. A
file whose name ends in `.vg.json` is a Vega spec, and one whose name ends in
`.vl.json` is a Vega-Lite spec.

[zuzu][] draws DOT graphs, Mermaid diagrams, and Vega and Vega-Lite charts in
Go. It draws each kind of diagram of Mermaid 12.3, with the icon packs fa,
mdi, carbon and k8s, and draws emoji in color. `iv` finds a diagram by its
content, or by its extension when the diagram starts with front matter.

- `--mermaid-theme dark` draws a Mermaid diagram in dark colors, and any
  other theme draws it in light colors. `--mermaid-bg` sets its background.
- A Vega or Vega-Lite spec can read its data from a file, as in
  `"url": "data/sales.csv"`. `iv` reads such a file only from the directory
  of the spec and the directories under it. A path that leaves that
  directory, and a url with a scheme such as `https:`, are errors.
- `--zuzu-data DIR` reads the data from `DIR` instead, with the same rules.
  A spec that is not a file on disk, such as a file in an archive, reads data
  only from `DIR`.

[zuzu]: https://github.com/xo/zuzu

### Documents

| Format                                                  | Extensions                                                        | Decoder     |
| ------------------------------------------------------- | ----------------------------------------------------------------- | ----------- |
| PDF and Adobe Illustrator, including password protected | `pdf` `ai`                                                        | `ivy-pdf`   |
| EPUB and kepub                                          | `epub` `kepub`                                                    | `ivy-epub`  |
| MOBI, PRC, AZW, AZW3 and AZW4                           | `mobi` `prc` `azw` `azw3` `azw4`                                  | `ivy-mobi`  |
| FictionBook, plain and zipped                           | `fb2` `fbz`                                                       | `ivy-fb2`   |
| Compiled HTML Help                                      | `chm`                                                             | `ivy-chm`   |
| DjVu                                                    | `djvu` `djv`                                                      | `ivy-djvu`  |
| XPS and OpenXPS                                         | `xps` `oxps`                                                      | `ivy-xps`   |
| Word                                                    | `docx` `dotx` `docm` `dotm`                                       | `ivy-docx`  |
| Word 97 to 2003                                         | `doc` `dot`                                                       | `ivy-msdoc` |
| Excel                                                   | `xlsx` `xltx` `xlsm` `xltm`                                       | `ivy-xlsx`  |
| Excel 97 to 2003                                        | `xls` `xlt`                                                       | `ivy-msxls` |
| PowerPoint                                              | `pptx` `potx` `ppsx` `pptm` `ppsm`                                | `ivy-pptx`  |
| PowerPoint 97 to 2003                                   | `ppt` `pps` `pot`                                                 | `ivy-msppt` |
| OpenDocument                                            | `odt` `ott` `odm` `ods` `ots` `odp` `otp` `odg` `otg` `odc` `odf` | `ivy-odf`   |
| RTF                                                     | `rtf`                                                             | `ivy-rtf`   |
| CSV and TSV, as a table                                 | `csv` `tsv` `tab`                                                 | `ivy-csv`   |
| Pages, Keynote and Numbers, the preview                 | `pages` `key` `numbers`                                           | `ivy-iwork` |
| Visio, the preview                                      | `vsdx` `vsdm` `vstx` `vstm` `vssx` `vssm` `vsd` `vst` `vss`       | `ivy-vsdx`  |
| Publisher, the preview                                  | `pub`                                                             | `ivy-pub`   |
| Markdown, HTML and plain text                           | `md` `markdown` `mkd` `mdown` `html` `htm` `xhtml` `txt`          | `blitz`     |
| Comic book archives                                     | `cbz` `cbr` `cbt` `cb7`                                           | `archives`  |

[ivy][] draws documents, ebooks and page descriptions in Go.

- A page is drawn at `--dpi`, or at the size that it is shown at when that
  is smaller.
- A format with no pages, such as Visio, shows its preview.
- A document uses the fonts that it embeds, then the fonts of the system.
- If a document needs a password and `--password` does not give the right
  one, `iv` asks for it at the terminal, at most three times.
- A spreadsheet, such as a csv, xlsx, xls or ods file, is drawn in landscape
  on US Letter, unless the file gives its own orientation.

[blitz][] lays out Markdown and HTML. It colors the code blocks of a Markdown
document with [transit][], in the colors of GitHub, for Bash, C, C++, Go,
HTML, Java, JavaScript, JSON, Python, Ruby, Rust, SQL and TypeScript. A block
in a dialect of SQL, such as `postgres` or `sqlite3`, is colored as SQL.
`--blitz-dark` draws a document in the dark colors of GitHub.

[ivy]: https://github.com/xo/ivy
[blitz]: https://github.com/xo/blitz
[transit]: https://github.com/xo/transit

### Video and audio

| Format                                    | Extensions                                                                                                 | Decoder  |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------- | -------- |
| Video, one frame (needs `ffmpeg`)         | `mp4` `m4v` `mkv` `mov` `avi` `webm` `mpeg` `mpeg2` `mpg` `mpg2` `flv` `asf` `wmv` `3gp` `3g2` `mj2` `ogv` | `ffmpeg` |
| Audio, a card of its cover art and tags   | `mp3` `m4a` `m4b` `m4p` `flac` `ogg` `oga` `dsf` `aac`                                                     | `tag`    |

`-t` selects the time of the video frame, and `--animate` plays the video.
The audio card shows a waveform when `ffmpeg` is installed.

### Fonts, executables and contacts

| Format                                         | Extensions                                          | Decoder   |
| ---------------------------------------------- | --------------------------------------------------- | --------- |
| Fonts, as a preview                            | `ttf` `ttc` `otf` `otc` `woff` `woff2` `sfnt` `eot` | `fontimg` |
| The icons of a Windows executable              | `exe` `dll` `mui`                                   | `winres`  |
| vCard, as a business card with a QR code       | `vcf` `vcard`                                       | `vcard`   |

A font preview shows the name, the style and the version of the font, its
sample text, the alphabet, the digits, and pangrams at growing sizes. A font
of color emoji shows its emoji. [okage][] draws the preview, and reads every
format above, except an EOT that MicroType Express compresses. In a font
collection, `-p N` selects the face. `--font-weight N`, from 1 to 1000, and
`--font-italic` select another weight or italic, also on the axes of a
variable font.

[okage]: https://github.com/xo/okage

### Arguments that are not files

| Argument                                    | Example                                                              | Decoder     |
| ------------------------------------------- | -------------------------------------------------------------------- | ----------- |
| `data:` URLs                                | `data:image/png;base64,iVBOR...`                                     | `data`      |
| `WIFI:` codes, as a QR code                 | `WIFI:S:mynetwork;T:WPA;P:secret;;`                                  | `qr`        |
| Other URIs that do something, as a QR code  | `mailto:`, `tel:`, `sms:`, `geo:`, `otpauth:`, `bitcoin:`, `magnet:` | `qr`        |
| `http://` and `https://` URLs               | A web page, or the image that the URL names                          | `blitz-url` |

`--fg`, `--bg` and `--border` set the colors and the border of a QR code.

### Any other file

`iv` searches any other file for an image in it, as the `binwalk` command
does, with the walker of [magic][]. It also searches the files of an archive
that it finds, and `-p N` selects one of the images. This draws the preview
of an Affinity file (`afdesign`, `afphoto`, `afpub`).

[magic]: https://github.com/xo/magic

### Output

With `--out`, `iv` writes a file in the format of its extension.
`--encoder` selects another encoder.

| Extension | Encoder      |
| --------- | ------------ |
| `png`     | `png`        |
| `jpg`     | `jpeg`       |
| `webp`    | `nativewebp` |
| `avif`    | `vips-avif`  |
| `gif`     | `vips-gif`   |
| `heic`    | `vips-heif`  |
| `jp2`     | `vips-jp2k`  |
| `jxl`     | `vips-jxl`   |
| `tiff`    | `vips-tiff`  |
| `pdf`     | `pdf`        |

`--encoder vips-webp` writes webp with libvips. A png or a jpeg that is not
changed is written as the bytes of its file.

The `pdf` encoder writes a document that ivy opens, such as a docx, an epub
or a pdf, with its pages as vectors. It writes every page, or only the page
that `--page` selects. The text is text in the fonts of the document, so a
reader can search and copy it. The text of a pdf or a PostScript file is
drawn as paths:

```sh
$ iv -o report.pdf /path/to/report.docx
$ iv -p 2 -o page2.pdf /path/to/report.docx
```

The `pdf` encoder puts any other image on one page of its own size.
`--pdf-page a4` or `--pdf-page letter` puts it on pages of that size, with a
margin of `--pdf-margin` points.

## Building

`iv` is cgo code. It links libvips for the image formats that the Go
decoders do not read, and fontconfig and freetype for blitz. It needs Go
1.27 or later, libvips 8.18 or later, and the packages below.

### Linux

```sh
$ sudo apt-get install -y build-essential pkg-config libvips-dev \
    libfontconfig-dev libfreetype-dev
```

Ubuntu 24.04 and earlier have a libvips that is too old. Debian and Ubuntu
build libheif with no codec, so HEIC and AVIF also need these plugins:

```sh
$ sudo apt-get install -y libheif-plugin-libde265 libheif-plugin-x265 \
    libheif-plugin-aomdec libheif-plugin-aomenc
```

### macOS

```sh
$ brew install vips pkgconf
```

If the build stops on `-Xpreprocessor`, which some builds of libvips put in
their flags, allow it:

```sh
$ export CGO_CFLAGS_ALLOW='-Xpreprocessor'
```

### Windows

Build in the UCRT64 environment of [MSYS2][msys2]. CI builds there, and
nothing tests MINGW64.

```sh
$ pacman -S mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-pkgconf \
    mingw-w64-ucrt-x86_64-libvips
```

[msys2]: https://www.msys2.org/

### Build commands

```sh
# build iv in the current directory
$ go build -o ./iv .

# draw each file in testdata/ with ./iv, as a smoke test
$ ./test.sh

# make a release build, which is versioned, stripped and packed
$ ./build.sh -v v0.1.0
```

`build.sh` writes to `build/<os>/<arch>/<version>/`. `-r` takes the version
from the latest tag, `-i` installs `iv`, `-b` builds and does not pack, and
`-a` builds for another architecture. A release is built for 64-bit
platforms only. To test a change, see [`CONTRIBUTING.md`](CONTRIBUTING.md).

### Optional tools

The `ffmpeg` and `tag` decoders run `ffmpeg` and `ffprobe`, which come in
the same package. Without them, a video fails with `ffmpeg not in path`, and
an audio card has no waveform.

## Performance

### How an image reaches the terminal

`iv` sends an image in the form that it already has, when the protocol can
take that form:

| Image                                                                                     | Kitty                      | iTerm                       | Sixel   |
| ----------------------------------------------------------------------------------------- | -------------------------- | --------------------------- | ------- |
| A png that is not scaled, from a file, a `data:` URL, ffmpeg, a web page or an archive   | The bytes of the file      | The bytes of the file       | Encoded |
| A jpeg that is not scaled                                                                 | Raw RGB                    | The bytes of the file       | Encoded |
| An image that libvips decodes, such as heic, avif or jxl                                  | A PNG that libvips encodes | A JPEG that libvips encodes | Encoded |
| An image that a renderer draws, such as svg, lottie, a document, a font or a psd          | Raw RGBA                   | Encoded                     | Encoded |
| Any other image, such as a gif or a QR code                                               | A PNG that rasterm encodes | Encoded                     | Encoded |

A png or a jpeg decodes its pixels only when something asks for them, so a
file that goes out as its own bytes is never decoded. An image that is
scaled, or drawn over `--bg`, is encoded again, because its pixels changed.
An animated png and a CMYK jpeg are encoded again too, because a terminal
can show them differently. A jpeg with an EXIF orientation is turned the
right way up.

Raw pixels take the least time to encode, but the terminal must read more
bytes. For Kitty, rasterm compresses raw pixels with zlib when they compress
to less than a quarter of their size. So a page of Markdown goes as 184 KiB
in place of 8 MiB, and a photo goes raw.

### Measures

On 2026-10-15, the benchmarks rendered 28 files with five versions of `iv`,
on an AMD Ryzen 9 9950X with Go 1.27.2 and libvips 8.18.7. Each value is the
geometric mean of the time to decode, scale and encode a file, against
v0.17.3:

| Protocol | v0.17.3 | v0.18.1 | v0.18.1, rasterm v0.1.18 | Later, rasterm v0.1.18 | Later, rasterm v0.1.20 |
| -------- | ------- | ------- | ------------------------ | ---------------------- | ---------------------- |
| Kitty    | 14.7 ms | 14.6 ms | 11.8 ms, 0.81x           | 4.81 ms, 0.33x         | 5.14 ms, 0.35x         |
| iTerm    | 11.6 ms | 11.5 ms | 11.1 ms, 0.96x           | 5.20 ms, 0.45x         | 5.20 ms, 0.45x         |
| Sixel    | 21.3 ms | 21.3 ms | 21.1 ms, 0.99x           | 20.8 ms, 0.98x         | 20.8 ms, 0.98x         |

Sixel encodes each image from its pixels, so its row shows the cost of
decoding and scaling, which did not change. D26 in
[`docs/PLAN.md`](docs/PLAN.md) has the method.

### Running the benchmarks

`ivcmd/bench_test.go` holds two benchmarks. `BenchmarkDecode` decodes one or
two files for each decoder. `BenchmarkRender` also scales each image to
1920x1200 and encodes it with the protocol of `TERM_GRAPHICS`. The output is
discarded, and the `out-B/op` metric is its size.

Run the decode benchmark once, and the render benchmark once for each
protocol. Then compare two runs with `benchstat`:

```sh
cd ivcmd
go test -run '^$' -bench '^BenchmarkDecode$' -benchtime 5x -count 6 -benchmem > decode.txt
TERM_GRAPHICS=kitty go test -run '^$' -bench '^BenchmarkRender$' -benchtime 5x -count 6 -benchmem > kitty.txt
TERM_GRAPHICS=iterm go test -run '^$' -bench '^BenchmarkRender$' -benchtime 5x -count 6 -benchmem > iterm.txt
TERM_GRAPHICS=sixel go test -run '^$' -bench '^BenchmarkRender$' -benchtime 5x -count 6 -benchmem > sixel.txt
go run golang.org/x/perf/cmd/benchstat@latest old/kitty.txt kitty.txt
```

The benchmarks need `ffmpeg`. To measure an earlier release, copy
`ivcmd/bench_test.go` into a checkout of it. The benchmark works with
v0.17.0 and later.

## Documents

| Document                                   | For                                                                      |
| ------------------------------------------ | ------------------------------------------------------------------------ |
| [`README.md`](README.md)                   | A person who uses `iv`                                                   |
| [`CONTRIBUTING.md`](CONTRIBUTING.md)       | A person who changes `iv`                                                |
| [`AGENTS.md`](AGENTS.md)                   | A coding agent that changes `iv`. `CLAUDE.md` imports it for Claude Code |
| [`docs/PLAN.md`](docs/PLAN.md)             | Each decision, and the open questions                                    |
| [`docs/BACKLOG.md`](docs/BACKLOG.md)       | The work that is known and not done                                      |
| [`testdata/README.md`](testdata/README.md) | The test data, one folder for each decoder                               |
