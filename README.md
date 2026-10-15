# iv

`iv` is a command-line image viewer using terminal graphics (Sixel, iTerm,
Kitty).

<p align="center">
  <a href="#installing" title="Installing">Installing</a> |
  <a href="#building" title="Building">Building</a> |
  <a href="#using" title="Using">Using</a> |
  <a href="#sizing" title="Sizing">Sizing</a> |
  <a href="#formats" title="Formats">Formats</a> |
  <a href="#performance" title="Performance">Performance</a> |
  <a href="#documents" title="Documents">Documents</a> |
  <a href="https://github.com/kenshaw/iv/releases" title="Releases">Releases</a>
</p>

[![Releases][release-status]][Releases]
[![Discord Discussion][discord-status]][discord]

[releases]: https://github.com/kenshaw/iv/releases "Releases"
[release-status]: https://img.shields.io/github/v/release/kenshaw/iv?display_name=tag&sort=semver "Latest Release"
[discord]: https://discord.gg/WDWAgXwJqN "Discord Discussion"
[discord-status]: https://img.shields.io/discord/829150509658013727.svg?label=Discord&logo=Discord&colorB=7289da&style=flat-square "Discord Discussion"

## Overview

`iv` is a command-line image viewer using terminal graphics.

Uses [Sixel][sixel], [iTerm Inline Images][iterm], or [Kitty][kitty] graphics
protocols where available. See [Are We Sixel Yet?][arewesixelyet] for a list of
terminals known to work with this package.

[sixel]: https://saitoha.github.io/libsixel/
[iterm]: https://iterm2.com/documentation-images.html
[kitty]: https://sw.kovidgoyal.net/kitty/graphics-protocol/
[arewesixelyet]: https://www.arewesixelyet.com

## Installing

`iv` can be installed [via Release][], [via Homebrew][], [via AUR][], [via
Scoop][] or [via Go][]:

[via Release]: #installing-via-release
[via Homebrew]: #installing-via-homebrew-macos-and-linux
[via AUR]: #installing-via-aur-arch-linux
[via Scoop]: #installing-via-scoop-windows
[via Go]: #installing-via-go

### Installing via Release

1. [Download a release for your platform][releases]
2. Extract the `iv` or `iv.exe` file from the `.tar.bz2` or `.zip` file
3. Move the extracted executable to somewhere on your `$PATH` (Linux/macOS) or
   `%PATH%` (Windows)

### Installing via Homebrew (macOS and Linux)

Install `iv` from the [`kenshaw/iv` tap][iv-tap] in the usual way with the [`brew`
command][homebrew]:

```sh
# install
$ brew install kenshaw/iv/iv
```

### Installing via AUR (Arch Linux)

Install `iv` from the [Arch Linux AUR][aur] in the usual way with the [`yay`
command][yay]:

```sh
# install
$ yay -S iv-cli
```

Alternately, build and [install using `makepkg`][arch-makepkg]:

```sh
# clone package repo and make/install package
$ git clone https://aur.archlinux.org/iv-cli.git && cd iv-cli
$ makepkg -si
==> Making package: iv-cli 0.4.4-1 (Sat 11 Nov 2023 02:28:28 PM WIB)
==> Checking runtime dependencies...
==> Checking buildtime dependencies...
==> Retrieving sources...
...
```

### Installing via Scoop (Windows)

Install `iv` using [Scoop](https://scoop.sh):

```powershell
# Optional: Needed to run a remote script the first time
> Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# install scoop if not already installed
> irm get.scoop.sh | iex

# install iv with scoop
> scoop install iv
```

### Installing via Go

Install `iv` in the usual Go fashion:

```sh
# install latest iv version
$ go install github.com/kenshaw/iv@latest
```

Note that this builds from source, and so needs what [Building](#building)
does.

## Building

`iv` is cgo code. It links libvips for the image formats and pdfs the Go
decoders do not cover, and fontconfig and freetype for the document renderer,
so a Go toolchain on its own is not enough. Go 1.27 or later, plus the
following per platform.

### Linux

```sh
$ sudo apt-get install -y build-essential pkg-config libvips-dev \
    libfontconfig-dev libfreetype-dev
```

libvips must be 8.18 or later -- the bindings `iv` uses are generated against
it, and Ubuntu 24.04 and earlier ship 8.15.

Debian and Ubuntu build libheif without any codec, so HEIC and AVIF need the
plugins on top of that:

```sh
$ sudo apt-get install -y libheif-plugin-libde265 libheif-plugin-x265 \
    libheif-plugin-aomdec libheif-plugin-aomenc
```

### macOS

```sh
$ brew install vips pkgconf
```

Some libvips builds put `-Xpreprocessor` in their pkg-config cflags, which cgo
refuses to pass through. If the build stops on that, allow it:

```sh
$ export CGO_CFLAGS_ALLOW='-Xpreprocessor'
```

### Windows

Build inside [MSYS2][], in the UCRT64 environment. The CI builds there, and
a build in MINGW64 is not tested.

```sh
$ pacman -S mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-pkgconf \
    mingw-w64-ucrt-x86_64-libvips
```

### Building and testing

```sh
# build, vet, and test
$ go build ./...
$ go vet ./...
$ go test ./...

# render everything in testdata/ through the pipeline, as a smoke test --
# uses ./iv when it is there, and the iv on $PATH otherwise
$ go build -o ./iv . && ./test.sh

# a release build: versioned, stripped, and packed
$ ./build.sh -v v0.1.0
```

`build.sh` writes to `build/<os>/<arch>/<version>/`. `-r` takes the version
from the latest tag instead of `-v`, `-i` installs rather than packing, `-b`
builds without packing, and `-a` cross compiles to another arch. A release
is built for 64-bit targets only.

### Optional tools

Six decoders shell out. Without the command, the file fails to render with
`<command> not in path` rather than being quietly skipped:

| Command         | Used for                                             |
| --------------- | ---------------------------------------------------- |
| `soffice`       | Word, Excel, PowerPoint, and OpenDocument documents  |
| `ebook-convert` | EPUB, MOBI, AZW, and FictionBook books (calibre)     |
| `xpstopng`      | XPS documents (libgxps)                              |
| `pdftoppm`      | pdf pages, where libvips has no pdf loader (poppler) |
| `ffmpeg`        | video snapshots, and the waveform on an audio card   |
| `binwalk`       | images embedded in otherwise unrecognized files      |

An audio card is the exception: it is drawn without a waveform rather than not
at all.

## Using

```sh
$ iv /path/to/image_or_directory

# a data: URL or WIFI: code
$ iv 'WIFI:S:mynetwork;T:WPA;P:secret;;'

# a remote image
$ iv https://example.com/image.png

# a specific page of a pdf, epub, or comic archive
$ iv -p 4 /path/to/document.pdf

# play an animation -- a gif, a video, a lottie
$ iv --animate /path/to/clip.webm

# and again until control-c
$ iv --loop /path/to/animation.gif

# at a rate of your own, when the default flickers
$ iv --fps 8 /path/to/clip.webm

# convert instead of displaying -- the encoder follows the output extension
$ iv --out out.webp /path/to/image.heic

# the registered decoders, encoders, and file extensions
$ iv --list

# all command line options
$ iv --help
```

## Sizing

Drawing to a terminal, `iv` fits an image to it. Every measure is in pixels.

The display size (`-W`/`-H`) is a ceiling and the minimum size (`-w`/`-h`, 64
by default) a floor, and **between the two an image is shown at its own size**
-- so an ordinary image is never resampled, and only one too big to fit or too
small to make out is touched at all. An image above the ceiling is shrunk to
it; one below the floor is grown up to it, and no further.

Leave `-W`/`-H` at 0 and the ceiling comes from the terminal itself, which is
the usual case. `iv` asks the kernel for the window size and uses the pixel
geometry the terminal reports there; a terminal that reports only a character
grid has its pixel size estimated from it instead. Two rows are kept back for
the file name and the prompt that follow the image.

A file written with `--out` gets no ceiling from the terminal -- there is no
terminal to fit, and a conversion that quietly downscaled to whatever window
happened to be open would be a surprising thing for a conversion to do. An
explicit `-W`/`-H` still applies.

`-m`/`--mode` changes how the two are used:

| Mode       | What it does                                                     |
| ---------- | ---------------------------------------------------------------- |
| `best-fit` | the default, described above                                     |
| `none`     | no scaling at all; every image is shown at its own size          |
| `width`    | `best-fit` against the width alone, however tall the result runs |
| `height`   | `best-fit` against the height alone                              |
| `shrink`   | `best-fit` without the floor: a small image is left small        |
| `stretch`  | fill `-W` x `-H` exactly, disregarding the aspect ratio          |

Where the floor and the ceiling disagree -- a minimum larger than the room
there is to show it in -- the ceiling wins. An image that clears the floor in
one dimension is not grown to clear it in the other, so a 1000x2 banner stays
1000x2 rather than becoming 32000 wide.

An image small enough to be icon art is magnified by a whole number of pixels
and not resampled, so a 24x24 favicon is its own pixels drawn three times
larger rather than blurred up to 72. A vector -- svg, lottie, pdf -- has no
pixels of its own to preserve, so it is rasterized at the size it is displayed
at instead of being resampled to it, and comes out sharp at any size.

## Animating

`-a`/`--animate` plays an animated file instead of showing a still of it, and
`-l`/`--loop` plays it over and over until it is interrupted with control-c.
Either one takes **exactly one file** -- the frames are drawn over the top of
one another and there is one screen to draw them on -- and needs a terminal to
draw to, so neither goes with `--out`.

The screen is cleared, the file name is written at the top of it, and the
frames follow underneath, each one clearing the one before it.

A pass runs for **a minute at most**, or for the animation's own length when
that is shorter, and what it plays at depends on what it is playing:

| Format                      | What it plays                                              |
| --------------------------- | ---------------------------------------------------------- |
| Animated GIF                | every frame, at the delays the file records                |
| Lottie, including dotLottie | its timeline, at the frame rate the composition declares   |
| Video *(needs `ffmpeg`)*    | **12 frames a second**, sampled across its length -- below |

A video is *sampled* rather than played: `iv` reads its length, works out how
many frames a pass has room for, and takes them at even intervals across the
whole of it -- so a two hour film comes out as the shape of the film in a
minute, not its first minute at the right speed. What plays is a sampling
either way, and there is nothing in one that twelve frames a second does not
carry as well as thirty; what the terminal gets out of the difference is a
steadier picture, every frame being a clear and an encode of a whole image.
The frames are exported while the ones before them are on screen rather than
all up front, so the first frame is drawn straight away.

`--fps N` overrides all of that and shows every frame for `1/N` of a second. It
is the rate the terminal is redrawn at, so lowering it is the thing to reach
for when the picture flickers -- and because every frame is still shown, a rate
below an animation's own plays it slower than it was drawn for. Passing it is
enough to animate; `--animate` is implied.

Drawing a frame is not free: encoding an image for a terminal costs real time,
and how much varies with the protocol, the terminal, and the size of the image.
So the playback runs against the clock rather than against its own drawing --
each frame has a moment it belongs at, and one whose moment has passed while
the frame before it was still being drawn is dropped rather than played late.
An animation keeps its own speed on a terminal that cannot draw it that fast;
it just shows fewer of the frames.

A file with nothing to play -- a jpeg, a single frame gif -- is not animated,
and `--animate` says so rather than sitting on a still image.

## Formats

`iv --list` prints what the binary in front of you actually has. Everything
below is in a default build; the decoders marked *needs* are only as good as
the command they shell out to, and the libvips formats depend on how libvips
itself was compiled.

Anything holding more than one image is shown one at a time, and `-p N` picks
which: a pdf, epub or xps page, an icon size, a comic archive page, a lottie frame.
An animated `gif` is the exception, and always shows its first frame -- see
[Animating](#animating) for playing one instead. An animated `webp` or APNG
shows its first frame, and does not play.

### Images

| Format                                    | Extensions                            | Decoder      |
| ----------------------------------------- | ------------------------------------- | ------------ |
| Portable Network Graphics, including APNG | `png`                                 | `png`        |
| JPEG                                      | `jpg` `jpeg` `jpe` `jif` `jfif` `jfi` | `jpeg`       |
| GIF, first frame of an animation          | `gif`                                 | `gif`        |
| WebP, lossy and lossless                  | `webp`                                | `nativewebp` |
| TIFF                                      | `tif` `tiff`                          | `tiff`       |
| Windows Bitmap                            | `bmp` `dib`                           | `bmp`        |
| Netpbm, raw and plain                     | `pbm` `pgm` `ppm` `pnm` `pam`         | `netpbm`     |
| Windows icons and cursors                 | `ico` `cur`                           | `ico`        |
| Apple Icon Image                          | `icns`                                | `icns`       |
| Photoshop, the merged image               | `psd` `psb`                           | `psd`        |
| HEIC/HEIF and AVIF                        | `heic` `heif` `avif`                  | `vips`       |
| JPEG 2000 and JPEG XL                     | `jp2` `jpf` `j2k` `jxl` `jxs`         | `vips`       |
| OpenEXR, Radiance HDR, PFM                | `exr` `hdr` `pfm` `rad`               | `vips`       |
| FITS, MATLAB, NIfTI, native vips          | `fits` `mat` `nii` `v`                | `vips`       |

### Vector graphics and diagrams

| Format                           | Extensions            | Decoder    |
| -------------------------------- | --------------------- | ---------- |
| SVG, plain and gzipped           | `svg` `svgz`          | `resvg`    |
| Lottie animations, and dotLottie | `json` `lot` `lottie` | `lottie`   |
| Graphviz graph description       | `gv` `dot`            | `graphviz` |
| Mermaid diagrams                 | `mmd` `mermaid`       | `mermaid`  |

A `.json` is only taken as a lottie when the document itself says so, so an
ordinary json file is left alone.

Mermaid diagrams are rendered in process, with no browser and no Node -- the
renderer is WebAssembly, run on wazero. `--mermaid-theme` picks one of
`default`, `dark`, `forest`, `neutral` or `modern`. Compiling the module takes
over a second, so the compiled code is cached under the user cache directory
(`iv/wazero`) and every run after the first starts in tens of milliseconds.

It is a reimplementation of Mermaid rather than a port, so a diagram comes out
*Mermaid-like* rather than identical to what `mermaid.js` draws, and the
iconify icon packs the mermaid cli takes are not supported -- an `fa:fa-globe`
in a label renders as that text.

A `.mmd` the renderer cannot parse is not an error: it is plain text, so it
falls through to `blitz`, which renders the source as a document. Run with
`-v` to see what the renderer made of it.

### Documents

| Format                                                           | Extensions                                                                                 | Decoder       |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------- |
| PDF, including password protected                                | `pdf`                                                                                      | `vips-pdf`    |
| EPUB, MOBI, AZW, FictionBook *(needs `ebook-convert`)*           | `epub` `mobi` `azw` `azw3` `fb2`                                                           | `calibre`     |
| XPS *(needs `xpstopng`)*                                         | `xps` `oxps`                                                                               | `gxps`        |
| Markdown, HTML, and plain text                                   | `md` `markdown` `mkd` `mdown` `html` `htm` `xhtml` `txt`                                   | `blitz`       |
| Comic book archives                                              | `cbz` `cbr` `cbt` `cb7`                                                                    | `archives`    |
| Word, Excel, PowerPoint *(needs `soffice`)*                      | `doc` `docx` `dot` `dotx` `xls` `xlsx` `xlt` `xltx` `ppt` `pptx` `pot` `potx` `pps` `ppsx` | `libreoffice` |
| OpenDocument *(needs `soffice`)*                                 | `odt` `ods` `odp` `odg` `odf` `odc` `ott` `ots` `otp` `otg`                                | `libreoffice` |
| RTF, Publisher, Visio, WordPerfect, csv, tsv *(needs `soffice`)* | `rtf` `pub` `vsd` `wpd` `csv` `tsv`                                                        | `libreoffice` |

### Video and audio

| Format                                      | Extensions                                                                                                 | Decoder  |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | -------- |
| Video, as a single frame *(needs `ffmpeg`)* | `mp4` `m4v` `mkv` `mov` `avi` `webm` `mpeg` `mpeg2` `mpg` `mpg2` `flv` `asf` `wmv` `3gp` `3g2` `mj2` `ogv` | `ffmpeg` |
| Audio, as a card of its cover art and tags  | `mp3` `m4a` `m4b` `m4p` `flac` `ogg` `oga` `dsf` `aac`                                                     | `tag`    |

The audio card draws a waveform when `ffmpeg` is there, and is drawn without
one when it is not. `-t` picks the moment a video is snapshotted at, and
`--animate` plays it instead of snapshotting it at all.

### Fonts and executables

| Format                                                          | Extensions                                          | Decoder   |
| --------------------------------------------------------------- | --------------------------------------------------- | --------- |
| Font specimen previews                                          | `ttf` `ttc` `otf` `woff` `woff2` `sfnt` `eot` `pfb` | `fontimg` |
| Icons in a Windows PE                                           | `exe` `dll` `mui`                                   | `winres`  |
| vCard contacts, as a business card with a QR code of the record | `vcf` `vcard`                                       | `vcard`   |

### Arguments that are not files

| Argument                                               | Example                                                              | Decoder     |
| ------------------------------------------------------ | -------------------------------------------------------------------- | ----------- |
| `data:` URLs                                           | `data:image/png;base64,iVBOR...`                                     | `data`      |
| `WIFI:` codes, as a QR code                            | `WIFI:S:mynetwork;T:WPA;P:secret;;`                                  | `qr`        |
| Other URIs that name something to act on, as a QR code | `mailto:`, `tel:`, `sms:`, `geo:`, `otpauth:`, `bitcoin:`, `magnet:` | `qr`        |
| `http://` and `https://`                               | a page, or the image it names                                        | `blitz-url` |

### Anything else

A file nothing above recognizes is handed to `binwalk`, which renders an image
found inside it -- the Affinity formats (`afdesign`, `afphoto`, `afpub`) among
them. *Needs `binwalk`.*

### Output

Without `--out`, `iv` draws to the terminal with Kitty, iTerm, or Sixel
graphics. With it, the encoder follows the output extension, and `--encoder`
overrides that:

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

libvips also writes webp, and `--encoder vips-webp` selects it in place of
`nativewebp`. The `pdf` encoder puts the image on one page of its own size by
default. `--pdf-page a4` or `--pdf-page letter` lays it out on pages of that
size instead, with a margin of `--pdf-margin` points.

A png or a jpeg that is not scaled is written as the bytes of its file when it
goes out in its own format, with no encode.

## Performance

### How an image reaches the terminal

`iv` sends an image in the form that it already has, when the protocol of the
terminal can take that form:

| Image                                                                                        | Kitty                         | iTerm                        | Sixel   |
| -------------------------------------------------------------------------------------------- | ----------------------------- | ---------------------------- | ------- |
| A png that is not scaled, from a file, a `data:` URL, ffmpeg, a web page or a comic archive | the bytes of the file         | the bytes of the file        | encoded |
| A jpeg that is not scaled                                                                    | raw RGB                       | the bytes of the file        | encoded |
| An image that libvips decodes (heic, avif, jxl, pdf and others)                              | a PNG that libvips encodes    | a JPEG that libvips encodes  | encoded |
| An image that a renderer draws (svg, lottie, documents, fonts, psd)                          | raw RGBA                      | encoded                      | encoded |
| Any other image, such as a gif or a QR code                                                  | a PNG that rasterm encodes    | encoded                      | encoded |

A png or a jpeg decodes its pixels only when something asks for them, so a
file that is sent as its own bytes is never decoded. libvips scales and
composites an image that it decoded, and encodes it only when the terminal
encoder asks. An image that is scaled, or composited over `--bg`, is encoded
again, because its pixels changed. An animated png and a CMYK jpeg are
encoded too, because a terminal can show them differently from what `iv`
decoded, and a jpeg with an EXIF orientation is turned the right way up.

Raw pixels take the least time to encode, and the terminal must read more
bytes. rasterm compresses raw pixels for Kitty with zlib when they compress to
under a quarter of their size, so a page of markdown goes as 184 KiB in place
of 8 MiB, and a photo goes raw.

### Measures

`ivcmd/bench_test.go` holds two benchmarks. `BenchmarkDecode` decodes each of
30 files in `testdata/`, one or two for each decoder. `BenchmarkRender` also
scales the image to a display of 1920x1200 pixels, and encodes it with the
protocol that `TERM_GRAPHICS` selects. The output is discarded, so the times
do not include the time that a terminal takes to show the image. The
`out-B/op` metric is the size of the output.

The same benchmark file ran against five versions: v0.17.3, v0.18.1, v0.18.1
with only rasterm v0.1.18, this code with rasterm v0.1.18, and this code with
rasterm v0.1.20. They ran on 2026-10-15, on an AMD Ryzen 9 9950X, with Go
1.27.2 and libvips 8.18.7. Each benchmark ran 5 times in each of 6 rounds, and
each round ran the versions in turn, so that a change in the machine fell on
each version alike. Each value is the median of the 6 rounds.

Each value below is the geometric mean over the 28 files that every version
renders. v0.17.3 cannot render mermaid without `mmdc`, so mermaid is not in
the mean, and neither is `gxps/example.xps` (see below).

| Protocol | v0.17.3 | v0.18.1 | v0.18.1, rasterm v0.1.18 | This code, rasterm v0.1.18 | This code, rasterm v0.1.20 |
| -------- | ------- | ------- | ------------------------ | -------------------------- | -------------------------- |
| Kitty    | 14.7 ms | 14.6 ms | 11.8 ms, 0.81x           | 4.81 ms, 0.33x             | 5.14 ms, 0.35x             |
| iTerm    | 11.6 ms | 11.5 ms | 11.1 ms, 0.96x           | 5.20 ms, 0.45x             | 5.20 ms, 0.45x             |
| Sixel    | 21.3 ms | 21.3 ms | 21.1 ms, 0.99x           | 20.8 ms, 0.98x             | 20.8 ms, 0.98x             |

Over the 30 files, the output that Kitty must read is 13.0 MiB for v0.18.1,
56.0 MiB for this code with rasterm v0.1.18, which sends raw pixels as they
are, and 17.2 MiB with rasterm v0.1.20, which compresses them.

Some files, from v0.18.1 to this code with rasterm v0.1.20:

| File                        | Kitty            | Kitty output       | iTerm            | iTerm output      |
| --------------------------- | ---------------- | ------------------ | ---------------- | ----------------- |
| `png/dice.png`              | 18.2 ms, 0.16 ms | 326 KiB, 285 KiB   | 12.9 ms, 0.18 ms | 62 KiB, 285 KiB   |
| `png/precision.png`         | 70.1 ms, 0.65 ms | 1464 KiB, 1412 KiB | 33.2 ms, 0.65 ms | 242 KiB, 1409 KiB |
| `jpeg/precision.jpg`        | 116 ms, 19.3 ms  | 2706 KiB, 3788 KiB | 27.8 ms, 0.16 ms | 230 KiB, 269 KiB  |
| `vips/dome.heic`            | 838 ms, 498 ms   | 2497 KiB, 2715 KiB | 792 ms, 494 ms   | 463 KiB, 586 KiB  |
| `vips/precision.jxl`        | 59.6 ms, 38.1 ms | 1369 KiB, 1676 KiB | 34.0 ms, 13.4 ms | 234 KiB, 305 KiB  |
| `ffmpeg/sample_960x540.mp4` | 174 ms, 144 ms   | 601 KiB, 591 KiB   | 158 ms, 144 ms   | 135 KiB, 590 KiB  |
| `resvg/choropleth.svg`      | 29.9 ms, 18.2 ms | 300 KiB, 265 KiB   | 21.1 ms, 20.7 ms | 166 KiB, 166 KiB  |
| `lottie/rocket.json`        | 28.7 ms, 12.0 ms | 159 KiB, 148 KiB   | 26.1 ms, 25.5 ms | 121 KiB, 121 KiB  |
| `blitz/sample.md`           | 137 ms, 127 ms   | 192 KiB, 184 KiB   | 148 ms, 147 ms   | 149 KiB, 149 KiB  |
| `psd/alpha.psd`             | 76.1 ms, 4.30 ms | 380 KiB, 98 KiB    | 74.2 ms, 6.21 ms | 139 KiB, 31 KiB   |
| `gxps/example.xps`          | 147 ms, 802 ms   | 593 KiB, 648 KiB   | 147 ms, 815 ms   | 368 KiB, 369 KiB  |

An XPS document took 147 ms with mupdf, and takes about 800 ms with
`xpstopng`, which iv runs since mupdf was removed. That one file raises each
mean above by about 14%, so it is listed and not averaged.

A png that the terminal gets as its own bytes now sends larger output to
iTerm, which got a jpeg that rasterm encoded before. `png/dice.png` is 285 KiB
in place of 62 KiB, and takes 0.18 ms in place of 13 ms.

The decode benchmark is not shown, because a png, a jpeg and a file that
libvips loads now decode their pixels only when something asks for them, so
`BenchmarkDecode` measures only the header for them. Over the 18 other files,
the decode takes 3.13 ms against 3.24 ms for v0.18.1. Sixel encodes every
image from its pixels, so its row shows the cost of decoding, scaling and
compositing, which is close to unchanged.

### Running the benchmarks

Run the decode benchmark once, and the render benchmark once for each
protocol, in the `ivcmd` folder. Then compare two runs with `benchstat`:

```sh
cd ivcmd
go test -run '^$' -bench '^BenchmarkDecode$' -benchtime 5x -count 6 -benchmem > decode.txt
TERM_GRAPHICS=kitty go test -run '^$' -bench '^BenchmarkRender$' -benchtime 5x -count 6 -benchmem > kitty.txt
TERM_GRAPHICS=iterm go test -run '^$' -bench '^BenchmarkRender$' -benchtime 5x -count 6 -benchmem > iterm.txt
TERM_GRAPHICS=sixel go test -run '^$' -bench '^BenchmarkRender$' -benchtime 5x -count 6 -benchmem > sixel.txt
go run golang.org/x/perf/cmd/benchstat@latest old/kitty.txt kitty.txt
```

The tests and the benchmarks need the decoder commands, which the Optional
tools section names. To measure an earlier release, copy
`ivcmd/bench_test.go` into a checkout of it. The `render` method that the
benchmark calls has the same signature from v0.17.0.

## Documents

| Document                                   | For                                                                      |
| ------------------------------------------ | ------------------------------------------------------------------------ |
| [`README.md`](README.md)                   | A person who uses `iv`                                                   |
| [`CONTRIBUTING.md`](CONTRIBUTING.md)       | A person who changes `iv`                                                |
| [`AGENTS.md`](AGENTS.md)                   | A coding agent that changes `iv`. `CLAUDE.md` imports it for Claude Code |
| [`docs/PLAN.md`](docs/PLAN.md)             | The plan, every decision, and the open questions                         |
| [`docs/BACKLOG.md`](docs/BACKLOG.md)       | The work that is known and not done                                      |
| [`testdata/README.md`](testdata/README.md) | The test data, one folder per decoder                                    |

[homebrew]: https://brew.sh/
[msys2]: https://www.msys2.org/
[iv-tap]: https://github.com/kenshaw/homebrew-iv
[aur]: https://aur.archlinux.org/packages/iv-cli
[arch-makepkg]: https://wiki.archlinux.org/title/makepkg
[yay]: https://github.com/Jguer/yay
