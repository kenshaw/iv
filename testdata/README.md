# testdata

Test data for `iv`, one directory per decoder (`iv --list` names them all).
Files here are inputs for `go test ./...` and `./test.sh`; nothing is a golden
output.

| Directory  | Decoder       | Notes                                                      |
|------------|---------------|------------------------------------------------------------|
| `binwalk`  | `binwalk`     | Affinity Designer file with an embedded png                 |
| `bmp`      | `bmp`         |                                                             |
| `ffmpeg`   | `ffmpeg`      | Same clip as mkv and mp4                                    |
| `fitz`     | `fitz`        | XPS documents                                               |
| `fontimg`  | `fontimg`     | Static, variable, and monospace fonts                       |
| `graphviz` | `graphviz`    | DOT graph, sniffed by content rather than extension         |
| `icns`     | `icns`        | Apple icon image, eight resolutions from 32 to 1024         |
| `ico`      | `ico`         | Single and multi image icons                                |
| `jpeg`     | `jpeg`        |                                                             |
| `markdown` | `markdown`    |                                                             |
| `mermaid`  | `mermaid`     | Needs `mmdc` in `$PATH`                                     |
| `png`      | `png`         |                                                             |
| `resvg`    | `resvg`       | SVGs, from trivial to a large choropleth                    |
| `vips`     | `vips`        | heic/heif/jxl, plus a plain and a password protected pdf    |
| `webp`     | `webp`        | Lossy and lossless webp, each with its png reference decode |
| `winres`   | `winres`      | Windows PE with embedded icons                              |

No test data yet: `archives`, `data`, `gif`, `http`, `libreoffice`, `netpbm`,
`qr`, `tag`, `tiff`. Of those, `archives` is covered by a cbz built on the fly
in `TestDecodeComicArchive`, and `data`, `qr`, and `http` take strings rather
than files.

## Provenance

- `png/{card,dice,logo,rose,tux}.png` and all of `webp/` -- the
  [Google WebP gallery][webp-gallery]. `-lossy` is the lossy-with-alpha
  encoding, `-lossless` the lossless one; each `.png` alongside is that
  file's reference decode.
- `png/precision.png`, `jpeg/precision.jpg`, `vips/precision.jxl`, and
  `webp/precision.webp` -- the [JPEG XL test page][jxl-test], one image in
  four formats.
- `bmp/rose.bmp` -- the gallery rose, converted to bmp.
- `ico/Mathijssen-Tuxlets-Test-Dummy-Tux.ico` -- Mathijssen Tuxlets Test
  Dummy Tux.
- `resvg/test.svg` -- [SVG Repo][svg-repo].
- `icns/test.icns` -- generated from `resvg/test.svg`. Each entry is rendered
  from the svg at its own resolution rather than downsampled from a single
  raster, so every size is crisp. The eight entries, in file order, are the
  full set the `icns` encoder supports:

  | OSType | Pixels | OSType | Pixels |
  |--------|--------|--------|--------|
  | `ic10` | 1024   | `ic08` | 256    |
  | `ic14` | 512    | `ic07` | 128    |
  | `ic09` | 512    | `ic12` | 64     |
  | `ic13` | 256    | `ic11` | 32     |

  `ic14`/`ic13`/`ic12`/`ic11` are the @2x variants of 256/128/32/16, which is
  why 512 and 256 each appear twice. `iv -p N` selects between them.
- `winres/go-winres.exe` -- built from [go-winres][].
- `fontimg/` -- Figtree, Noto Mono, and Ubuntu.

[webp-gallery]: https://developers.google.com/speed/webp/gallery
[jxl-test]: https://jpegxl.info/resources/jpeg-xl-test-page
[svg-repo]: https://www.svgrepo.com
[go-winres]: https://github.com/tc-hib/go-winres
