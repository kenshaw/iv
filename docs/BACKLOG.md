# Backlog

This document lists work that is known and not done. Each item names where
it came from.

A decision is not a backlog item. It goes in [`PLAN.md`](PLAN.md). When an
item here is done, delete it, and record in `PLAN.md` anything that was
decided on the way (D21).

On 2026-09-27, GitHub had no open issue and no open pull request for `iv`, and
the code had no `TODO` or `FIXME` comment. The items below come from the other
sources that each one names.

## Formats

### Support glTF and other 3D models

Source: `notes.txt`, a local file that is not tracked. Nothing decodes a 3D
model today.

### Play an animated WebP or APNG

Source: found while writing D19. `--animate` plays a gif, a lottie file and a
video. The `nativewebp` and `png` decoders register no animation, so an
animated WebP or APNG shows its first frame only, as the Formats section of
`README.md` now says.

### Keep the PNG bytes in an icon

Source: found while writing D22. An ico or icns file holds each large icon as
a PNG, and the `ico`, `icns` and `winres` decoders decode those through other
packages, which keep no bytes. The terminal encoder then sends raw pixels in
place of the PNG.

## Documentation

### Update the AUR sample output in the README

Source: found while reading `README.md`. The `makepkg` sample output shows
`iv-cli` 0.4.4 from 2023.

## Builds

### Try MINGW64 on Windows

Source: D28. Windows builds under UCRT64 because the mupdf of go-fitz called
`__intrinsic_setjmpex`, which only the UCRT runtime has. mupdf is gone, and
nothing has tested MINGW64 since. This is open question 1 in `PLAN.md`.

## Performance

### Benchmark the libreoffice and calibre decoders

Source: D26. `BenchmarkDecode` and `BenchmarkRender` leave them out, because
one run of `soffice` or `ebook-convert` takes a second or more. A benchmark of
their own, run with a smaller count, can cover them.

## Removed from notes.txt

`notes.txt` also lists three items that are done or that belong to another
repository. They are not backlog items here:

- "epub/mobi": `ca94f95` (2025-05-01) added the `fitz` decoder, which
  rendered EPUB, MOBI, XPS and FictionBook. D28 moved them to `calibre` and
  `gxps`.
- "support pdf decryption": `de977bc` (2026-08-05) added `--password` for a
  password protected pdf.
- "fix `ox`" and "migrate `usql` to `ox`": the work is in `xo/ox` and
  `xo/usql`.
