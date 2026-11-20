# Backlog

This document lists work that is known and not done. Each item names where
it came from.

A decision is not a backlog item. It goes in [`PLAN.md`](PLAN.md). When an
item here is done, delete it, and record in `PLAN.md` anything that was
decided on the way (D21).

On 2026-11-19, GitHub had no open issue and no open pull request for `iv`, and
the code had no `TODO` or `FIXME` comment. The items below come from the other
sources that each one names.

## Formats

### Convert colors with ICC profiles

Source: D56. libvips converted a CMYK image, and the color space that an
image names, to sRGB. The pure Go decoders do not. An image with an ICC
profile, a wide gamut such as Display P3 or BT.2020, or HDR, shows its
colors as if they were sRGB. ivy has a port of moxcms that converts colors,
in `internal/moxcms`, which `iv` cannot import. A maintainer asked the ivy
session to make it public. When it is, apply the profile, or the CICP values
of an AVIF or a HEIC, in the decoders.

### Read a MATLAB file of version 7.3

Source: D56. scigolib/matlab v0.3.19 opens a MAT-file of version 7.3, but
returns each array with one dimension, even an array that it wrote itself
with two. The `matlab` decoder then finds no image. Report it to
scigolib/matlab, and add a test file of version 7.3 when it is fixed.

### Write alpha in a JPEG XL, and more than 8 bits in an AVIF

Source: D56. The `jxl` encoder of gen2brain/jxl drops alpha, and treats each
image as sRGB. The `avif` encoder of gen2brain/gav1d writes 8 bits.
KarpelesLab/goavif writes 10 and 12 bits, but is young. Use a newer release
of either when it closes the gap.

### Support glTF and other 3D models

Source: `notes.txt`, a local file that is not tracked. Nothing decodes a 3D
model today.

### Play an animated WebP or APNG

Source: found while writing D19. `--animate` plays a gif, a lottie file and a
video. The `nativewebp` and `png` decoders register no animation, so an
animated WebP or APNG shows its first frame only, as the Formats section of
`README.md` now says.

### Add test files for ten ivy formats

Source: a review of the documents on 2026-11-19. `testdata/ivy` holds no
file for the formats camera, chm, djvu, emf, iwork, ora, pptx, ps, pub and
vsdx, so no test draws them. Add a small file for each, with its source and
license in `testdata/README.md`.

### Keep the PNG bytes in an icon

Source: found while writing D22. An ico or icns file holds each large icon as
a PNG, and the `ico`, `icns` and `winres` decoders decode those through other
packages, which keep no bytes. The terminal encoder then sends raw pixels in
place of the PNG.

## Builds

### Try Ubuntu 24.04 in CI and for the release

Source: D56. CI and the release build on Ubuntu 26.04, because libvips 8.18
was not in 24.04. libvips is gone. A build on 24.04 links an older glibc, so
the release binary runs on more systems. Make sure that the archives of blitz
link there first.

### Try MINGW64 on Windows

Source: D28. Windows builds under UCRT64 because the mupdf of go-fitz called
`__intrinsic_setjmpex`, which only the UCRT runtime has. mupdf is gone, and
nothing has tested MINGW64 since. This is open question 1 in `PLAN.md`.

### Remove the flag that allows a duplicate symbol on Windows

Source: D35. `decoder/blitz/link_windows.go` passes
`-Wl,--allow-multiple-definition`, because the rust archives of blitz and
resvg each held the runtime of rust. resvg is gone, so blitz is the only rust
archive. Remove the flag, and make sure that the Windows job of CI still
links.
