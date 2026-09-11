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
| `ico`      | `ico`         | Single and multi image icons                                |
| `jpeg`     | `jpeg`        |                                                             |
| `markdown` | `markdown`    |                                                             |
| `mermaid`  | `mermaid`     | Needs `mmdc` in `$PATH`                                     |
| `png`      | `png`         |                                                             |
| `resvg`    | `resvg`       | SVGs, from trivial to a large choropleth                    |
| `vips`     | `vips`        | heic/heif/jxl, plus a plain and a password protected pdf    |
| `webp`     | `webp`        | Lossy and lossless webp, each with its png reference decode |
| `winres`   | `winres`      | Windows PE with embedded icons                              |

No test data yet: `archives`, `data`, `gif`, `http`, `icns`, `libreoffice`,
`netpbm`, `qr`, `tag`, `tiff`. Of those, `archives` is covered by a cbz built
on the fly in `TestDecodeComicArchive`, and `data`, `qr`, and `http` take
strings rather than files.

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
- `winres/go-winres.exe` -- built from [go-winres][].
- `fontimg/` -- Figtree, Noto Mono, and Ubuntu.

[webp-gallery]: https://developers.google.com/speed/webp/gallery
[jxl-test]: https://jpegxl.info/resources/jpeg-xl-test-page
[svg-repo]: https://www.svgrepo.com
[go-winres]: https://github.com/tc-hib/go-winres
