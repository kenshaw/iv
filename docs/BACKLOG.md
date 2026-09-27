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
animated WebP or APNG shows its first frame only. The Formats section of
`README.md` says that an animated `gif` or `webp` can play, and points at the
Animating section. Either add the animation or correct the README.

## Documentation

### Correct the package comment of internal/cmd

Source: found while writing D18. The package comment in
`internal/cmd/cmd.go` names `mmdc` as a command that a decoder runs. `2a02ec6`
removed the last use of `mmdc`.

### Add the vcard decoder and the QR schemes to the README

Source: found while writing the current state in `PLAN.md`. `3a1e7b4` added
the `vcard` decoder for `.vcf` files and more URI schemes to the `qr` decoder.
The Formats section of `README.md` names neither. The "Arguments that are
not files" table names only `data:`, `WIFI:`, `http://` and `https://`.

### Add the pdf encoder to the README

Source: found while writing the current state in `PLAN.md`. `encoder/pdf`
registers the `pdf` encoder, and the Output table of `README.md` does not name
it.

### Update the AUR sample output in the README

Source: found while reading `README.md`. The `makepkg` sample output shows
`iv-cli` 0.4.4 from 2023.

## Builds

### Decide the 32-bit targets

Source: D15. `13ec599` removed the 32-bit ARM release build. `README.md` still
has a "32-bit targets" section, and `decoder/fitz/fitz_unsupported.go` still
builds for `386` and `arm`. This is open question 5 in `PLAN.md`.

### Decide the static Linux release build

Source: D15. The static build step in `.github/workflows/release.yml` is
commented out, and `build.sh` still takes `-s`. This is open question 6 in
`PLAN.md`.

### Decide whether iv has a lint configuration

Source: D21. `iv` has no `.golangci.yml`, and CI runs no linter. This is open
question 3 in `PLAN.md`.

## Code

### Wrap the soffice error in the decode test with %w

Source: found while reading the error handling. `ivcmd/decode_test.go` writes
`fmt.Errorf("soffice cannot run here: %v: %s", err, ...)`, which drops the
error chain. Every other `fmt.Errorf` in the code wraps with `%w`.

## Removed from notes.txt

`notes.txt` also lists three items that are done or that belong to another
repository. They are not backlog items here:

- "epub/mobi": `ca94f95` (2025-05-01) added the `fitz` decoder, which renders
  EPUB, MOBI, XPS and FictionBook.
- "support pdf decryption": `de977bc` (2026-08-05) added `--password` for a
  password protected pdf.
- "fix `ox`" and "migrate `usql` to `ox`": the work is in `xo/ox` and
  `xo/usql`.
