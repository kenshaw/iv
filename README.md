# iv

`iv` is a command-line image viewer using terminal graphics (Sixel, iTerm,
Kitty).

<p align="center">
  <a href="#installing" title="Installing">Installing</a> |
  <a href="#building" title="Building">Building</a> |
  <a href="#using" title="Using">Using</a> |
  <a href="#sizing" title="Sizing">Sizing</a> |
  <a href="#formats" title="Formats">Formats</a> |
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

Build inside [MSYS2][], in the UCRT64 environment rather than MINGW64: the
bundled mupdf calls `__intrinsic_setjmpex`, which only the UCRT runtime has.

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
builds without packing, `-a` cross compiles to another arch, and `-s` makes a
fully static linux binary.

### Optional tools

Four decoders shell out. Without the command, the file fails to render with
`<command> not in path` rather than being quietly skipped:

| Command   | Used for                                           |
| --------- | -------------------------------------------------- |
| `soffice` | Word, Excel, PowerPoint, and OpenDocument documents |
| `mmdc`    | Mermaid diagrams                                    |
| `ffmpeg`  | video snapshots, and the waveform on an audio card  |
| `binwalk` | images embedded in otherwise unrecognized files     |

A mermaid diagram is the exception: without `mmdc` it falls through to
`blitz`, which renders the source as a document. An audio card is another --
it is drawn without a waveform rather than not at all.

### 32-bit targets

On `386` and `arm`, `epub`, `xps`, `mobi`, `fb2` and `psd` do not decode:
go-fitz vendors its prebuilt mupdf for 64-bit targets only, and nothing
registers for those formats there.

## Using

```sh
$ iv /path/to/image_or_directory

# a data: URL or WIFI: code
$ iv 'WIFI:S:mynetwork;T:WPA;P:secret;;'

# a remote image
$ iv https://example.com/image.png

# a specific page of a pdf, epub, or comic archive
$ iv -p 4 /path/to/document.pdf

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

## Formats

`iv --list` prints what the binary in front of you actually has. Everything
below is in a default build; the decoders marked *needs* are only as good as
the command they shell out to, and the libvips formats depend on how libvips
itself was compiled.

Anything holding more than one image is shown one at a time, and `-p N` picks
which: a pdf or epub page, an icon size, a comic archive page, a lottie frame.
An animated `gif` or `webp` is the exception, and always shows its first frame.

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
| HEIC/HEIF and AVIF                        | `heic` `heif` `avif`                  | `vips`       |
| JPEG 2000 and JPEG XL                     | `jp2` `jpf` `j2k` `jxl` `jxs`         | `vips`       |
| OpenEXR, Radiance HDR, PFM                | `exr` `hdr` `pfm` `rad`               | `vips`       |
| FITS, MATLAB, NIfTI, native vips          | `fits` `mat` `nii` `v`                | `vips`       |

### Vector graphics and diagrams

| Format                            | Extensions            | Decoder    |
| --------------------------------- | --------------------- | ---------- |
| SVG, plain and gzipped            | `svg` `svgz`          | `resvg`    |
| Lottie animations, and dotLottie  | `json` `lot` `lottie` | `lottie`   |
| Graphviz graph description        | `gv` `dot`            | `graphviz` |
| Mermaid diagrams *(needs `mmdc`)* | `mmd` `mermaid`       | `mermaid`  |

A `.json` is only taken as a lottie when the document itself says so, so an
ordinary json file is left alone.

### Documents

| Format                                                           | Extensions                                                                                 | Decoder       |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------- |
| PDF, including password protected                                | `pdf`                                                                                      | `vips-pdf`    |
| EPUB, XPS, MOBI, FictionBook, Photoshop                          | `epub` `xps` `oxps` `mobi` `fb2` `psd`                                                     | `fitz`        |
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
one when it is not. `-t` picks the moment a video is snapshotted at.

### Fonts and executables

| Format                 | Extensions                                          | Decoder   |
| ---------------------- | --------------------------------------------------- | --------- |
| Font specimen previews | `ttf` `ttc` `otf` `woff` `woff2` `sfnt` `eot` `pfb` | `fontimg` |
| Icons in a Windows PE  | `exe` `dll` `mui`                                   | `winres`  |

### Arguments that are not files

| Argument                    | Example                             | Decoder     |
| --------------------------- | ----------------------------------- | ----------- |
| `data:` URLs                | `data:image/png;base64,iVBOR...`    | `data`      |
| `WIFI:` codes, as a QR code | `WIFI:S:mynetwork;T:WPA;P:secret;;` | `qr`        |
| `http://` and `https://`    | a page, or the image it names       | `blitz-url` |

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

libvips writes several of these itself, which `--encoder` reaches:
`vips-webp` instead of `nativewebp`, or `vips-tiff` for a file the Go encoder
has no compression for.

[homebrew]: https://brew.sh/
[msys2]: https://www.msys2.org/
[iv-tap]: https://github.com/kenshaw/homebrew-iv
[aur]: https://aur.archlinux.org/packages/iv-cli
[arch-makepkg]: https://wiki.archlinux.org/title/makepkg
[yay]: https://github.com/Jguer/yay
