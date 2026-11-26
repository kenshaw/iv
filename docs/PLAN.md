# Plan

This document holds the plan for `iv`, then every decision that shapes it,
then the open questions for the maintainers.

## Purpose

`iv` is a command-line image viewer. It draws an image in the terminal with
Sixel, iTerm or Kitty graphics. It opens most things that hold a picture:
images, vector graphics, diagrams, documents, video, audio cover art, fonts,
the icons in a Windows executable, and command line arguments such as a
`data:` URL or a `WIFI:` code. It can also write the result to a file instead
of the terminal.

`iv` is for viewing. It is not an image editor and it is not a text-mode
renderer (D1). Speed is not a goal (D2).

## Current state

This is the state of 2026-11-27, after D66.

- One package for each format registers a decoder or an encoder from `init`,
  and a pipeline in `decoder` runs them (D10). `iv --list` names each one.
- `iv` selects a decoder by the mime type that xo/magic, a port of libmagic
  to Go, finds in the content (D6, D17).
- An image reaches the encoder in the form that it already has: the bytes of
  its PNG or JPEG file, until a step changes its pixels (D22, D30). A large
  JPEG is decoded at a fraction of its size when it is scaled down (D62). `ivimg.Prepare` scales and composites stills and
  animation frames alike.
- Pure Go codecs decode HEIC, AVIF, JPEG XL, JPEG 2000, OpenEXR, Radiance
  HDR, PFM and MATLAB files, and encode AVIF, HEIC, JPEG XL, JPEG 2000, TIFF,
  WebP and GIF. libvips is gone, and blitz is the only C code (D56). The
  `icc` package of ivy converts their colors, and those of a CMYK JPEG, to
  sRGB (D57). `--page` selects an image of a heic or avif file, and `-v`
  says which image of a dynamic wallpaper of macOS it is (D58, D59). `-v`
  writes all of the metadata of an image file (D60).
  blitz draws Markdown, HTML and web pages (D16). zuzu draws
  DOT graphs, Mermaid diagrams and Vega charts (D36), with the math of KaTeX
  (D64). xo/ivy draws office documents, ebooks, PDF, XPS, Photoshop files,
  LaTeX and other formats, and asks
  for a missing password (D49, D50, D51). A document that ivy opens is
  written to a pdf as vectors (D55).
- Two decoders run an external command from `$PATH` (D7): `ffmpeg` and `tag`
  run `ffmpeg` and `ffprobe`.
- `iv` plays an animated gif, png, webp or jpeg xl, a heic or avif sequence,
  a lottie or a video, and `--page` selects a frame (D19, D42, D43, D61).
- CI tests and lints each push on Linux amd64 and arm64, macOS and Windows
  (D14, D29). A version tag builds a release for the same 64-bit platforms
  (D15, D32, D53). Scheduled workflows keep the AUR packages and the Homebrew formula
  current (D52, D66).
- `go test -race -timeout 30m ./...` passes, and fails when a command or a
  file that a test needs is missing (D27). `./test.sh` draws each file in
  `testdata/` (D11). The benchmarks in `ivcmd` compare releases (D26).

## Direction

No document states a direction beyond the work in [`BACKLOG.md`](BACKLOG.md).
Ask a maintainer before you start a new format or a new feature that is not in
that list.

## Decisions

Each decision that shapes `iv`, in the order that it was made. The entries are
append only. A decision is never edited to change its conclusion. When a later
decision changes one, the older entry keeps its text, and its heading names
the decision that changed it.

Each heading carries a status after the title. `Decided` means that the history
shows that a maintainer chose it. `Proposed` means that the code follows it and
nothing shows that a maintainer chose it, so ask a maintainer. After the
status, `Amends Dn` or `Amended by Dn` names a decision that this one changes
or that changes it.

D1 to D20 were written on 2026-09-27 from the history of the repository, from
`3b94bb5` (2024-02-02) to `d2151b1` (2026-09-18), and from the closed issues
and pull requests on GitHub. Most commit messages are short, so a reason
usually comes from a code comment, a workflow comment, the README or a comment
by a maintainer on GitHub. If none of these gives a reason, the entry says
"Reason not recorded".

A reference to a decision of another repository names that repository, as in
dbmeta D110. A bare number, as in D3, names a decision in this document.

| Decision | Title | Status |
| --- | --- | --- |
| [D1](#d1-iv-draws-images-with-terminal-graphics-and-never-as-text-decided) | iv draws images with terminal graphics and never as text | Decided |
| [D2](#d2-speed-is-not-a-goal-decided) | Speed is not a goal | Decided |
| [D3](#d3-svg-renders-with-xoresvg-decided-amended-by-d35) | SVG renders with xo/resvg | Decided. Amended by D35 |
| [D4](#d4-the-command-line-is-parsed-with-xoox-decided) | The command line is parsed with xo/ox | Decided |
| [D5](#d5-libvips-decodes-the-formats-that-no-go-package-covers-decided-amended-by-d8-amended-by-d56) | libvips decodes the formats that no Go package covers | Decided. Amended by D8. Amended by D56 |
| [D6](#d6-iv-selects-a-decoder-by-the-mime-type-of-the-content-decided-amended-by-d17) | iv selects a decoder by the mime type of the content | Decided. Amended by D17 |
| [D7](#d7-an-external-command-is-found-in-path-when-a-decoder-first-uses-it-decided-amended-by-d18-amended-by-d47) | An external command is found in $PATH when a decoder first uses it | Decided. Amended by D18. Amended by D47 |
| [D8](#d8-the-libvips-binding-is-cshumvipsgen-decided-amends-d5-amended-by-d56) | The libvips binding is cshum/vipsgen | Decided. Amends D5. Amended by D56 |
| [D9](#d9-iv-is-cgo-code-decided-amended-by-d28-amended-by-d56) | iv is cgo code | Decided. Amended by D28. Amended by D56 |
| [D10](#d10-decoders-and-encoders-register-themselves-and-one-pipeline-runs-them-decided) | Decoders and encoders register themselves, and one pipeline runs them | Decided |
| [D11](#d11-test-data-has-one-folder-per-decoder-and-testsh-renders-all-of-it-decided) | Test data has one folder per decoder, and test.sh renders all of it | Decided |
| [D12](#d12-a-test-skips-when-its-tool-its-library-support-or-its-file-is-missing-proposed-amended-by-d27) | A test skips when its tool, its library support or its file is missing | Proposed. Amended by D27 |
| [D13](#d13-test-data-is-checked-out-with-no-line-ending-conversion-decided) | Test data is checked out with no line ending conversion | Decided |
| [D14](#d14-ci-tests-on-linux-macos-and-windows-and-linux-runs-ubuntu-2604-decided) | CI tests on Linux, macOS and Windows, and Linux runs ubuntu-26.04 | Decided |
| [D15](#d15-a-release-builds-for-linux-macos-and-windows-decided) | A release builds for Linux, macOS and Windows | Decided |
| [D16](#d16-markdown-html-and-web-pages-render-with-xoblitz-decided-amended-by-d38) | Markdown, HTML and web pages render with xo/blitz | Decided. Amended by D38 |
| [D17](#d17-mime-detection-uses-libmagic-through-xomagic-decided-amends-d6) | Mime detection uses libmagic through xo/magic | Decided. Amends D6 |
| [D18](#d18-mermaid-renders-in-process-with-xomermaid-on-wazero-decided-amends-d7-amended-by-d36) | Mermaid renders in process with xo/mermaid on wazero | Decided. Amends D7. Amended by D36 |
| [D19](#d19-iv-plays-one-animation-to-the-terminal-for-at-most-one-minute-decided-amended-by-d42-amended-by-d43) | iv plays one animation to the terminal, for at most one minute | Decided. Amended by D42. Amended by D43 |
| [D20](#d20-the-go-directive-follows-the-newest-release-of-go-decided) | The go directive follows the newest release of Go | Decided |
| [D21](#d21-iv-is-set-up-for-coding-agents-as-every-xo-repository-is-decided) | iv is set up for coding agents as every xo repository is | Decided |
| [D22](#d22-an-image-reaches-the-encoder-in-the-form-that-it-already-has-decided-amended-by-d28-amended-by-d33-amended-by-d42-amended-by-d43-amended-by-d56-amended-by-d62) | An image reaches the encoder in the form that it already has | Decided. Amended by D28. Amended by D33. Amended by D42. Amended by D43. Amended by D56. Amended by D62 |
| [D23](#d23-a-jpeg-with-an-exif-orientation-a-cmyk-jpeg-and-an-animated-png-keep-no-bytes-decided) | A JPEG with an EXIF orientation, a CMYK JPEG and an animated PNG keep no bytes | Decided |
| [D24](#d24---fg-and---border-set-the-colors-and-the-border-of-a-qr-code-decided) | --fg and --border set the colors and the border of a QR code | Decided |
| [D25](#d25---mermaid-bg-sets-the-background-in-the-mermaid-renderer-decided) | --mermaid-bg sets the background in the mermaid renderer | Decided |
| [D26](#d26-benchmarks-compare-the-releases-and-the-readme-holds-the-numbers-decided) | Benchmarks compare the releases, and the README holds the numbers | Decided |
| [D27](#d27-a-test-fails-when-what-it-needs-is-missing-decided-amends-d12-amended-by-d34-amended-by-d47-amended-by-d49-amended-by-d56) | A test fails when what it needs is missing | Decided. Amends D12. Amended by D34. Amended by D47. Amended by D49. Amended by D56 |
| [D28](#d28-mupdf-is-removed-and-calibre-libgxps-oovpsd-and-poppler-take-its-formats-decided-amends-d9-amends-d22-amended-by-d49) | mupdf is removed, and calibre, libgxps, oov/psd and poppler take its formats | Decided. Amends D9. Amends D22. Amended by D49 |
| [D29](#d29-golangci-lint-checks-the-code-in-ci-with-the-rules-of-the-xo-repositories-decided) | golangci-lint checks the code in CI, with the rules of the xo repositories | Decided |
| [D30](#d30-a-png-or-a-jpeg-decodes-its-pixels-only-when-they-are-needed-decided) | A PNG or a JPEG decodes its pixels only when they are needed | Decided |
| [D31](#d31-the-code-follows-go-pedantry-where-the-two-disagreed-decided) | The code follows go-pedantry where the two disagreed | Decided |
| [D32](#d32-iv-is-built-for-64-bit-targets-only-and-has-no-static-build-decided) | iv is built for 64-bit targets only, and has no static build | Decided |
| [D33](#d33-kitty-gets-raw-pixels-compressed-with-zlib-when-they-compress-well-decided-amends-d22) | Kitty gets raw pixels compressed with zlib when they compress well | Decided. Amends D22 |
| [D34](#d34-a-test-file-that-crashes-a-command-on-one-system-is-not-decoded-there-decided-amends-d27-amended-by-d49) | A test file that crashes a command on one system is not decoded there | Decided. Amends D27. Amended by D49 |
| [D35](#d35-svg-renders-with-okage-a-port-of-resvg-to-go-decided-amends-d3) | SVG renders with okage, a port of resvg to Go | Decided. Amends D3 |
| [D36](#d36-one-decoder-named-zuzu-draws-dot-and-mermaid-decided-amends-d18) | One decoder named zuzu draws DOT and Mermaid | Decided. Amends D18 |
| [D37](#d37-a-minimum-size-given-on-the-command-line-is-a-hard-floor-decided) | A minimum size given on the command line is a hard floor | Decided |
| [D38](#d38-a-tall-document-is-drawn-whole-and-cut-into-pages-decided-amends-d16) | A tall document is drawn whole and cut into pages | Decided. Amends D16 |
| [D39](#d39-blitz-colors-the-code-blocks-of-markdown-with-transit-decided) | blitz colors the code blocks of Markdown with transit | Decided |
| [D40](#d40-zuzu-draws-an-emoji-in-color-decided) | zuzu draws an emoji in color | Decided |
| [D41](#d41-a-lottie-draws-with-the-lottie-package-of-okage-decided) | A lottie draws with the lottie package of okage | Decided |
| [D42](#d42-a-kitty-terminal-that-can-play-an-animation-plays-it-itself-decided-amends-d19-amends-d22-amended-by-d43) | A Kitty terminal that can play an animation plays it itself | Decided. Amends D19. Amends D22. Amended by D43 |
| [D43](#d43-a-video-plays-at-its-own-speed-and-a-kitty-image-shows-each-frame-decided-amends-d19-amends-d22-amends-d42) | A video plays at its own speed, and a Kitty image shows each frame | Decided. Amends D19. Amends D22. Amends D42 |
| [D44](#d44-zuzu-draws-a-vega-spec-and-reads-its-data-only-under-its-directory-decided-amended-by-d45-amended-by-d46) | zuzu draws a Vega spec, and reads its data only under its directory | Decided. Amended by D45. Amended by D46 |
| [D45](#d45---zuzu-data-sets-the-directory-that-a-vega-spec-reads-its-data-from-decided-amends-d44) | --zuzu-data sets the directory that a Vega spec reads its data from | Decided. Amends D44 |
| [D46](#d46-zuzu-draws-a-vega-lite-spec-as-it-draws-a-vega-spec-decided-amends-d44) | zuzu draws a Vega-Lite spec as it draws a Vega spec | Decided. Amends D44 |
| [D47](#d47-the-binwalk-decoder-uses-the-walker-of-xomagic-decided-amends-d7-amends-d27) | The binwalk decoder uses the walker of xo/magic | Decided. Amends D7. Amends D27 |
| [D48](#d48-okage-draws-font-previews-and-measures-the-text-of-the-cards-decided) | okage draws font previews, and measures the text of the cards | Decided |
| [D49](#d49-xoivy-draws-documents-ebooks-and-page-descriptions-in-go-decided-amends-d27-amends-d28-amends-d34-amended-by-d50) | xo/ivy draws documents, ebooks and page descriptions in Go | Decided. Amends D27. Amends D28. Amends D34. Amended by D50 |
| [D50](#d50-ivy-draws-every-pdf-and-asks-for-a-missing-password-decided-amends-d49) | ivy draws every pdf, and asks for a missing password | Decided. Amends D49 |
| [D51](#d51-a-spreadsheet-is-drawn-in-landscape-by-default-decided) | A spreadsheet is drawn in landscape by default | Decided |
| [D52](#d52-a-scheduled-workflow-keeps-the-aur-packages-current-decided-amended-by-d66) | A scheduled workflow keeps the AUR packages current | Decided. Amended by D66 |
| [D53](#d53-the-release-workflow-runs-only-for-a-version-tag-decided) | The Release workflow runs only for a version tag | Decided |
| [D54](#d54-a-script-makes-the-screenshots-of-the-readme-in-ghostty-decided) | A script makes the screenshots of the README in Ghostty | Decided |
| [D55](#d55-a-document-is-written-to-a-pdf-with-ivy-as-vectors-decided) | A document is written to a pdf with ivy, as vectors | Decided |
| [D56](#d56-pure-go-codecs-replace-libvips-decided-amends-d5-amends-d8-amends-d9-amends-d22-amends-d27-amended-by-d57) | Pure Go codecs replace libvips | Decided. Amends D5. Amends D8. Amends D9. Amends D22. Amends D27. Amended by D57 |
| [D57](#d57-the-decoders-convert-colors-to-srgb-with-the-icc-package-of-ivy-decided-amends-d56) | The decoders convert colors to sRGB with the icc package of ivy | Decided. Amends D56 |
| [D58](#d58---page-selects-a-top-level-image-of-a-heic-or-avif-file-decided) | --page selects a top-level image of a heic or avif file | Decided |
| [D59](#d59-the-verbose-output-says-which-image-of-a-dynamic-wallpaper-is-shown-decided) | The verbose output says which image of a dynamic wallpaper is shown | Decided |
| [D60](#d60-the-verbose-output-holds-all-of-the-metadata-of-an-image-file-decided) | The verbose output holds all of the metadata of an image file | Decided |
| [D61](#d61-every-animated-format-plays-and---page-selects-a-frame-decided-amended-by-d63) | Every animated format plays, and --page selects a frame | Decided. Amended by D63 |
| [D62](#d62-jpegn-decodes-a-jpeg-and-a-large-one-at-a-fraction-of-its-size-decided-amends-d22) | jpegn decodes a JPEG, and a large one at a fraction of its size | Decided. Amends D22 |
| [D63](#d63-vpx-encodes-webp-and-nativewebp-is-gone-decided-amends-d61) | vpx encodes webp, and nativewebp is gone | Decided. Amends D61 |
| [D64](#d64-ivy-draws-latex-and-plain-tex-and-zuzu-draws-the-math-of-katex-decided) | ivy draws LaTeX and plain TeX, and zuzu draws the math of KaTeX | Decided |
| [D65](#d65-zuzu-takes-the-emoji-font-and-the-time-when-it-parses-decided) | zuzu takes the emoji font and the time when it parses | Decided |
| [D66](#d66-a-scheduled-workflow-keeps-the-homebrew-formula-current-decided-amends-d52) | A scheduled workflow keeps the Homebrew formula current | Decided. Amends D52 |

### D1. iv draws images with terminal graphics and never as text. Decided.

The first commit, `3b94bb5` (2024-02-02), draws with `github.com/kenshaw/rasterm`,
which writes Sixel, iTerm and Kitty graphics. `encoder/rasterm` does the same
today. If the terminal has none of the three, `iv` stops with the error that
`rasterm` returns.

A maintainer closed pull request 4, "Added ascii support and auto width", on
2025-08-23, with the comment that `iv` is an image viewer for terminals that
support terminal graphics, and that `iv` will not support terminals that do
not. Issue 3 says that Kitty detection belongs in `rasterm`, and that the
`TERM_GRAPHICS` environment variable selects a protocol.

### D2. Speed is not a goal. Decided.

Issue 1, "iv is very slow", was closed on 2024-02-18, with the comment that
there is no intention to make `iv` run faster. The comment names two causes:
Go is slow to decode and encode images, and Sixel is slower than the other
protocols.

### D3. SVG renders with xo/resvg. Decided. Amended by D35.

`542deaf` (2024-02-02), the first release, added `github.com/xo/resvg`.
`decoder/resvg` registers it for plain and gzipped SVG. Reason not recorded.

### D4. The command line is parsed with xo/ox. Decided.

`3451594` (2024-12-04) replaced `spf13/cobra` and `spf13/pflag` with
`github.com/xo/ox`. `edb8b3d` added back the short flags that the change lost,
and `be6b7fe` (2025-05-27) moved the version to `ox.VersionString`. The flags
are the `ox` struct tags on `ivcmd.Args`, and `ivctx` registers its own types
with `ox`. Reason not recorded.

### D5. libvips decodes the formats that no Go package covers. Decided. Amended by D8. Amended by D56.

`b7f68c3` (2024-02-08) added `github.com/xo/heif`, "not yet fully working", and
`34d692f` (2024-03-30) removed it. `ff44958` (2024-04-11) added libvips
through `github.com/davidbyttow/govips`. From then on, HEIC, HEIF, AVIF, JPEG
XL and the other formats that the Go decoders do not cover go to libvips.
Reason not recorded.

### D6. iv selects a decoder by the mime type of the content. Decided. Amended by D17.

`6cece88` (2025-01-30), "Rewrite", changed the dispatch from a switch on the
file extension to a switch on the mime type that
`github.com/gabriel-vasile/mimetype` sniffs from the content. `61c78a5`
(2026-06-23) added `--force-mime`, and `ce5809d` treats
`application/octet-stream` as a font when the extension is a font extension.
Both came from issue 5, where `mimetype` did not detect a TTF file.

### D7. An external command is found in $PATH when a decoder first uses it. Decided. Amended by D18. Amended by D47.

`4298105` (2025-01-30) runs `soffice` for office documents. `930d808`
(2025-01-30), "Switching to ffmpeg command for videos", replaced the
`go-astiav` library with the `ffmpeg` command. `dbb6c69` (2025-08-28) ran
`mmdc` for mermaid. Each looked up its command once, on first use.

`3890a24` (2026-09-11) moved the lookup to `internal/cmd` and added `binwalk`
as the decoder of last resort. The README gives the rule for a missing
command: the file fails to render with `<command> not in path`, and is not
skipped without a message. The audio card is the exception. Without
`ffmpeg`, it is drawn with no waveform.

### D8. The libvips binding is cshum/vipsgen. Decided. Amends D5. Amended by D56.

`ad71d8f` (2026-02-01), "Change vips package", replaced `davidbyttow/govips`
with `github.com/cshum/vipsgen`. The commit gives no reason. The README gives
one result: `vipsgen` generates its bindings against libvips 8.18, so `iv`
needs libvips 8.18 or later. `1b24400` (2026-06-23) later set the `Unlimited`
and `Memory` load options of `vipsgen`. A maintainer wrote on issue 5 that the
new `vipsgen` can override the memory security limits of libvips.

A maintainer asked on 2026-11-09 to move to `github.com/cshum/vipsgen/v2`. In
v2, an option is a pointer when libvips takes a zero that differs from its
default, and a nil pointer means that the option is not set. v1 dropped an
option of zero. The nearest kernel is 0, so `Image.Resize` in `vips/` scaled
small images with lanczos3 where `ivimg.Nearest` asked for nearest neighbor.
`TestResizeNearest` fails when that happens again.

### D9. iv is cgo code. Decided. Amended by D28. Amended by D56.

libvips arrived in `ff44958` (2024-04-11), and from then on `iv` links C
libraries: libvips, mupdf through go-fitz, and fontconfig and freetype for
blitz. The workflows and `build.sh` set `CGO_ENABLED=1`. The README says that
a Go toolchain on its own cannot build `iv`.

### D10. Decoders and encoders register themselves, and one pipeline runs them. Decided.

`3890a24` (2026-09-11), "Massive rewrite", split `main.go` into these parts:

- `decoder` and `encoder` hold a registry each. A package registers an
  `Entry` with `Register` and a list of options.
- One package per format registers itself from `init`. `decoder/all` and
  `encoder/all` import every one of them, and `ivcmd` imports the two.
- `decoder` holds the pipeline. A decoder returns an image, a set of images,
  or a new input for the pipeline. The pipeline stops after 8 hand-offs, to
  bound a cycle between decoders (`maxDepth`).
- `ivctx` holds the configuration, which travels on the context.
- `ivcmd` holds the command, and `main.go` only runs it.

`5abc5a3` added `RegisterBuiltin` on the same day. It marks a format that is in
the `image.Decode` registry of Go, so that a container such as a comic archive
can render it. `bc0c9bf` (2026-09-12) reworked the pipeline for scaling, which
closed issue 2, "Add resize feature". A maintainer wrote on issue 2 that a
Claude agent did this work.

### D11. Test data has one folder per decoder, and test.sh renders all of it. Decided.

`7eb4ce8` (2026-09-11) reorganized `testdata/` into one folder per decoder and
added `testdata/README.md`. The files are inputs for `go test ./...` and
`./test.sh`, and none is a golden output. `5abc5a3` renamed `webp` to
`nativewebp` to keep the folder name equal to the decoder name.
`testdata/strings/` holds the command line arguments that are not files.

`test.sh` began in `b909ed8` (2026-08-04). It now renders every file and every
string in `testdata/` through a built `iv`.

### D12. A test skips when its tool, its library support or its file is missing. Proposed. Amended by D27.

The tests call `t.Skip` when an external command is not in `$PATH`, when
libmagic is not available, when a test file is missing, when the network is not
available (`blitz.ErrFetch`), and when this build of libvips cannot read or
write a format (`ErrUnsupportedFormat`). `19c8f46` gives one case: the libvips
of MSYS2 has no pdf loader. The rule holds in every test that needs one of
these. Nothing shows that a maintainer chose it as a rule, so ask a maintainer.

A maintainer did not make it a rule. D27 replaces it: a test fails.

### D13. Test data is checked out with no line ending conversion. Decided.

`864434a` (2026-09-12) added `/testdata/** -text` to `.gitattributes`. The
comment gives the reason: if git converts the line endings, the plain netpbm
files stop being detected as images once their newlines become CRLF.
`19c8f46` added `!/testdata/**` to `.gitignore`, because the patterns that
ignore stray output also match the test files.

### D14. CI tests on Linux, macOS and Windows, and Linux runs ubuntu-26.04. Decided.

The test workflow runs on `ubuntu-26.04`, `ubuntu-26.04-arm`, `macos-latest`
and `windows-latest`. `8e060d0` (2026-06-23) first moved Linux to 26.04, and
`00872cc` (2026-09-12) moved it back after the rewrite, with the reason:
`vipsgen` generates its bindings against libvips 8.18, and Ubuntu 24.04 ships
8.15.

Windows builds under MSYS2 in the UCRT64 environment (`2aaf573`).
The reason was in the workflow and the README: the mupdf of go-fitz called
`__intrinsic_setjmpex`, and only the UCRT runtime has it. D28 removed mupdf.

### D15. A release builds for Linux, macOS and Windows. Decided.

`5bdd98e` (2025-01-02) cut the release to Linux amd64 and turned off macOS,
Windows and ARM, with no reason. `3890a24` put back the ARM cross builds,
`2aaf573` (2026-09-12) builds arm64 on a native runner, and `13ec599` removed
the 32-bit ARM build, with no reason. `6c72a47` (2026-09-12), "Add macos +
windows release builds", builds macOS on a native runner for each
architecture and joins the two into a universal binary with `lipo`. The
comment gives the reason: a cross build needs a libvips for the other
architecture, and Homebrew does not install one. Windows builds under MSYS2,
as D14 says.

### D16. Markdown, HTML and web pages render with xo/blitz. Decided. Amended by D38.

`b9b637e` (2026-09-12), "blitz: add decoder, replacing markdown", removed the
Markdown decoder, the HTTP decoder and `goldmark`. Before it, Markdown became
a pdf through `goldmark-pdf`, and libvips rendered the pdf. `blitz` renders
Markdown and HTML documents, and `blitz-url` renders an `http` or `https`
argument.

### D17. Mime detection uses libmagic through xo/magic. Decided. Amends D6.

`3890a24` (2026-09-11) added `github.com/xo/magic` behind `mimetype`.
`3a1e7b4` (2026-09-13) removed `mimetype` from `decoder/mime.go`,
`decoder/winres` and `go.mod`. The commit that says it does this, `19c8f46`,
changed only tests and `.gitignore`. `62ded31` moved to magic v0.2.1. A
comment in `decoder/mime.go` gave one limit: the text analysis of libmagic
read out of bounds and stopped the process on the mermaid and dot test files.
The port to Go in magic v0.5.0, below, removed that limit and the comment.

On 2026-11-15, the database of libmagic was 10.8 MB of the 117.5 MB release
build of `iv`. A maintainer asked the maintainers of xo/magic to compress
it. magic v0.4.0 embeds it compressed with zstd, in 345,182 bytes, and
expands it into C memory in `magic.New`. The release build went from 117.48
MB to 107.04 MB. `magic.New` takes about 1.7 ms in place of 0.27 ms, and `iv`
calls it once.

magic v0.4.1 fixed a crash on Linux in every earlier release. Its Linux
archives were compiled against the `regex.h` of musl, so a program built with
glibc could crash in libmagic, or get a wrong description. v0.4.1 bundles its
own regex engine. `iv` moved to it the same day.

magic v0.5.0 is a port of libmagic to pure Go, with a new API (magic D39).
It needs no cgo, no C compiler and no libmagic, and gives the output of
libmagic 5.48. One call of `magic.Detect` returns the mime type and the
description, and it is safe from many goroutines, so `decoder/mime.go` keeps
no handle and no lock. Each of the 156 test files got the same mime type as
with v0.4.1. The release build went from 107.10 MB to 106.64 MB. A test can no
longer fail because libmagic is missing.

### D18. Mermaid renders in process with xo/mermaid on wazero. Decided. Amends D7. Amended by D36.

`dbb6c69` (2025-08-28) rendered mermaid with the `mmdc` command, and `cab5f7e`
added its default icon pack and a white background. `2a02ec6` (2026-09-13)
replaced `mmdc` with `github.com/xo/mermaid`, which runs on
`github.com/tetratelabs/wazero`. It replaced `--mermaid-icons` with
`--mermaid-theme`, and it keeps the compiled code in the user cache folder,
under `iv/wazero`. The README gives the reason: the renderer needs no browser
and no Node. It also gives the cost: the renderer is a reimplementation, so
the output is like mermaid and not the same, and icon packs do not work.

### D19. iv plays one animation to the terminal, for at most one minute. Decided. Amended by D42. Amended by D43.

`e3d2ea5` (2026-09-18) added `--animate` for a gif, a lottie file and a video,
and `d2151b1` changed how the frame rate is chosen. The README gives the
rules:

- `--animate` takes exactly one file, and it does not work with `--out`.
- One pass plays for at most one minute.
- A video is sampled at 12 frames per second across its length.
- Playback follows the clock and drops a frame that is late, unless the
  decoder made it late (see below).
- `--loop` and `--fps` imply `--animate`.

An animated WebP or APNG does not play. No text gives a reason, and
[`BACKLOG.md`](BACKLOG.md) holds the item.

On 2026-11-13, a maintainer found that `--animate` showed only the first frame
of a lottie. A frame of okage v0.6.0 took longer to draw than its time on
screen, so every frame after the first came late, and the player dropped each
one. A dropped frame saves time only when the drawing is slow and the next
frame is ready. The decoder makes each frame in turn, so a frame that the
decoder made late is followed by another late frame. The player now measures
how long it waited for each frame. When it waited 1 ms or more, it draws the
late frame and moves the rest of the pass back by the time lost. `fire.lottie`
at 1024x1024 showed 1 of its 65 frames, and now shows all 65 in 2.3 s, where
the animation is 1.08 s long. `TestPlaySlowProducer` covers it.

### D20. The go directive follows the newest release of Go. Decided.

The `go` directive in `go.mod` went from 1.21.6 in `3b94bb5` to 1.27.1 in
`2a02ec6`, and it moved with each release of Go in the commits that update
the dependencies. The README asks for Go 1.27 or later. The pattern holds
across the whole history. A maintainer made it a rule on 2026-10-15.

### D21. iv is set up for coding agents as every xo repository is. Decided.

A maintainer decided on 2026-09-27 that every `xo` repository is set up for
coding agents the same way (dbmeta D110), and asked for the same setup here.
`iv` is a small library in this sense, so its decisions stay in this document
and are not split into one file each (dbmeta D111).

On 2026-10-15 a maintainer said that every `kenshaw` repository follows dbmeta
D110 too, and not only `iv`.

The setup is this:

- `AGENTS.md` holds the rules for a coding agent, and opens with the three
  standing rules. `CLAUDE.md` holds the one line `@AGENTS.md`, which Claude
  Code reads as an import.
- `CONTRIBUTING.md` has an Agent skills section with the two commands that
  install the skills.
- `simple-english` and `go-pedantry` are ordinary folders in
  `.agents/skills/<name>` and `.claude/skills/<name>`, and `skills-lock.json`
  names their source.
- `docs/` holds this document and [`BACKLOG.md`](BACKLOG.md). Only
  `README.md`, `AGENTS.md`, `CLAUDE.md` and `CONTRIBUTING.md` are in the root.
- `.gitignore` ignores `.claude/settings.local.json`. It ignored all of
  `.claude/` before, so it now ignores everything in `.claude/` except
  `.claude/skills/`.
- `.gitattributes` starts with `* text=auto eol=lf`. The rule for `testdata/`
  in D13 comes after it, so it still wins for the test files.

`docs_test.go` holds the tests. `TestSkillsAreCopies` fails on a symbolic
link, on a missing copy, on two copies that differ, and on a skill that
`skills-lock.json` does not name. `TestClaudeImportsAgents` fails unless
`CLAUDE.md` is an ordinary file that holds only `@AGENTS.md`.
`TestTheRootHoldsFourDocuments` fails on any other `.md` file in the root.
`TestTheDecisionIndexIsComplete` compares the index above with the headings,
`TestAnAmendmentPointsBothWays` fails unless both decisions name each other,
and `TestEveryDecisionReferenceExists` fails on a bare reference to a decision
that this document does not hold.

### D22. An image reaches the encoder in the form that it already has. Decided. Amended by D28. Amended by D33. Amended by D42. Amended by D43. Amended by D56. Amended by D62.

A maintainer asked on 2026-10-15 for this pipeline, after `rasterm` v0.1.18
began to send the bytes that an image holds without an encode. The `ivimg`
package holds it, and every still and every animation frame goes through it the
same way:

- A PNG or a JPEG keeps the bytes of its file. The `png` and `jpeg` decoders
  return a `rasterm.PNG` or a `rasterm.JPEG`. Every path that hands bytes back
  to the pipeline reaches these two decoders: a `data:` URL, an ffmpeg
  snapshot, an image from `blitz-url`, a page of a comic archive and a file
  that `binwalk` extracts. An ffmpeg animation keeps the bytes of each frame.
- libvips keeps the image that it decoded, as a `vips.Image` of iv. It scales
  and composites itself in libvips, and encodes itself as a PNG only when the
  encoder asks for one. The `vips-*` encoders save it from libvips with
  nothing copied. libvips decodes the pixels only then, so it loads from the
  bytes of the input, which the image keeps, and not from a stream of the
  file, which the pipeline closes first. An error that libvips returns then
  is kept by the image, and the encoder entry returns it. Both were found
  when a JPEG XL failed in the encoder, with a warning and a blank image.
  `TestEncodeAfterDecode` covers them.
- `ivimg.Prepare` scales an image to the display size and composites it over
  the background. A step that does not change the pixels returns the image as
  it was. A step that changes the pixels returns an image that holds no
  bytes, so stale bytes cannot reach an encoder. The background is skipped
  for an image that the format says is opaque.
- The encoder decides last, by the protocol. `ivimg.Raw` holds the rules:
  - Kitty gets the bytes of a PNG file, or the PNG of libvips. A JPEG goes as
    RGB, because Kitty cannot take a JPEG. A tight `image.NRGBA` goes as RGBA
    with nothing copied, and an `image.RGBA` goes as RGBA. rasterm encodes
    any other image, such as a paletted gif or a QR code, as a PNG.
  - iTerm gets the bytes of a PNG or a JPEG file. An opaque image that
    libvips holds hides its PNG, and rasterm encodes its pixels as a JPEG.
  - Sixel and half blocks encode the pixels.
  - The `png` and `jpeg` file encoders write the bytes of a file in their
    own format as they are.

  The first form of these rules sent raw pixels for every image without bytes,
  and the PNG of libvips to iTerm. The benchmarks of D26 showed three costs.
  Raw pixels of a paletted gif or a QR code took more time than a PNG, and
  made an output up to 700 times larger. The PNG of libvips took more time for
  iTerm than the JPEG before it, and made an output 8 to 11 times larger. A
  JPEG from libvips was faster still, but the JPEG encoder of libvips fails in
  iv with "Wrong JPEG library version: library is 100, caller expects 80",
  because another libjpeg is linked into the binary. The PNG of libvips uses
  the sub filter, which took less time than no filter for a photo and made a
  smaller file.

  D28 removed the second libjpeg. iTerm now gets an opaque image that libvips
  holds as the JPEG that libvips encodes, which took 3.5 ms for a 896x1080
  photo, where the Go encoder of rasterm takes about 30 ms.

The design went to Gemini and DeepSeek for review. Both found the same risks,
and the code answers each: a raw buffer must be tight before it is sent, a
libvips image needs a close, an alpha scan costs as much as the compositing
that it saves, and some JPEG and PNG files are wrong to send as they are (see
D23).

Raw pixels take less time to encode than a PNG, and the terminal must read 3
to 5 times the bytes, and up to 40 times for a flat image. D33 compresses the
raw pixels that compress well.

### D23. A JPEG with an EXIF orientation, a CMYK JPEG and an animated PNG keep no bytes. Decided.

The Go decoders do not apply an EXIF orientation, and a terminal can, so a
JPEG whose orientation is not 1 shows a different image from the one that
iv sized. A terminal can convert the colors of a CMYK or YCCK JPEG
differently from the Go decoder, and some show them inverted. A terminal that
reads an animated PNG can play it, where iv shows its first frame. Each of
these is decoded and encoded again, as before D22. The reviews of D22 named
the three cases.

A maintainer decided on 2026-10-15, and asked for the orientation to be
applied. A JPEG whose orientation is not 1 is turned the right way up when it
is decoded (`ivimg.orient`), so it shows as the camera meant, and the turned
image is new pixels with no bytes.

ffmpeg wrote its snapshots and frames with the `apng` muxer, which writes an
animated PNG of one frame. They are now plain PNGs, written with the `png`
codec and the `image2pipe` muxer, so that they keep their bytes.

### D24. --fg and --border set the colors and the border of a QR code. Decided.

`--fg` and `--border` set the colors and the quiet zone of a QR code until
`3a1e7b4` (2026-09-13), which changed the `qr` decoder to draw black on white
with the quiet zone of `go-qrcode`, and gave no reason. From then on nothing
read the two settings. The first form of this decision removed the flags, and
a maintainer asked on 2026-10-15 for them to be restored for QR codes.

`--fg` is the color of the modules, and black by default. `--bg` is the
background, and a transparent background, the default, draws white, because a
code with no background is hard to read on a dark terminal. `--border` is the
quiet zone in pixels, 40 by default, which is the 4 modules that the standard
asks for at 10 pixels a module. The defaults draw the same code as v0.18.1,
byte for byte. `TestDecodeColors` covers them.

### D25. --mermaid-bg sets the background in the mermaid renderer. Decided.

`--mermaid-bg` did nothing. The background was composited when the mime type
was `text/plain`, and the pipeline only saw the mime type of the svg that the
mermaid decoder hands back. The renderer draws its own background, so the color
now goes to it as the `background` theme variable. The flag has no default, so
each theme keeps its own background, and the default theme is white as before.
By hand, `--mermaid-bg red` gave a red background and `--mermaid-theme dark`
the dark background of the theme. A maintainer decided on 2026-10-15.

### D26. Benchmarks compare the releases, and the README holds the numbers. Decided.

A maintainer asked on 2026-10-15 for benchmarks of the decoding, compared with
the earlier releases, in the README. `ivcmd/bench_test.go` holds
`BenchmarkDecode` and `BenchmarkRender`. The second runs the `render` method of
the command, from the file to the terminal graphics in `io.Discard`, with the
protocol that `TERM_GRAPHICS` selects. The method has the same signature from
v0.17.0, so the same file runs against each release. The Performance section of
`README.md` holds the results and the commands that made them.

### D27. A test fails when what it needs is missing. Decided. Amends D12. Amended by D34. Amended by D47. Amended by D49. Amended by D56.

A maintainer decided on 2026-10-15 that a test fails, and does not skip, when
an external command, libmagic, a test file or a libvips format is missing. A
test that skips says nothing about the decoder that it was meant to test.

Two tests could not pass that way, and both changed:

- Four strings in `testdata/strings/` fetched live web pages. `TestDecodeURL`
  replaces them. It serves `blitz/sample.html` and `ico/1.ico` from an
  `httptest` server, and covers a page, a redirect and an image, as the four
  did.
- The libvips of MSYS2 has no pdf loader, so the pdf tests skipped on
  Windows. The `poppler` decoder renders a pdf with `pdftoppm` when libvips
  cannot, and CI installs poppler on Windows.

The one exception is a command that CI does not install on an operating
system. `untested` in `ivcmd/decode_test.go` names each one with its reason:
on Windows, LibreOffice and calibre have no MSYS2 package, and binwalk has no
Windows package. CI installs every other command on every runner.

### D28. mupdf is removed, and calibre, libgxps, oov/psd and poppler take its formats. Decided. Amends D9. Amends D22. Amended by D49.

A maintainer decided on 2026-10-15 to remove `github.com/gen2brain/go-fitz`,
which linked a static mupdf, from a module of 258 MB. It decoded EPUB, MOBI,
XPS, FictionBook and Photoshop files. Its libjpeg was the second libjpeg in the
binary that broke the JPEG encoder of libvips (D22): with fitz linked, a save
failed with "Wrong JPEG library version", and without it the save works. It was
also why Windows builds under UCRT64 (D14), and why 32-bit targets needed a
stub.

Each format went to the replacement that a maintainer chose:

- EPUB, MOBI, AZW and FictionBook go to the `calibre` decoder, which converts
  a book to pdf with `ebook-convert`. The pdf then renders with libvips, so
  `-p` selects a page. A conversion takes about 1.5 seconds. mupdf rendered
  the XPS test file in about 17 ms. calibre cannot read a MOBI 6 file that has no guide, which is
  why the test file is a MOBI of both types.
- XPS and OXPS go to the `gxps` decoder, which renders a page with
  `xpstopng` of libgxps, in about 300 ms a page at 150 dpi.
- Photoshop files go to the `psd` decoder, which decodes the merged image with
  `github.com/oov/psd`, and hands a file that it cannot read to libvips. The
  two were compared side by side on 2026-10-15. oov/psd matched the source
  image pixel for pixel on an RGB, an RGB with alpha and a CMYK file, in 3.1
  to 18 ms, where the ImageMagick loader of libvips took 3.4 to 65 ms, shifted
  the colors of the CMYK file by 17 to 24 levels and lost the soft edge of the
  alpha. oov/psd failed on a 16-bit file, which libvips then decodes.

### D29. golangci-lint checks the code in CI, with the rules of the xo repositories. Decided.

A maintainer decided on 2026-10-15 to add the configuration of the xo
repositories and run it in CI. `.golangci.yml` sets `default: all`, and
disables each linter that makes idiomatic Go worse here, with its reason beside
it. A real defect got a code change: a context stored in `decoder.Sink`, a
`context.Background` in the command, an error left unchecked, and a
hand-written filter that `slices.DeleteFunc` replaces. The Lint job pins a
release of golangci-lint. v2.14.0 found 0 issues, and v2.15.0 found 0 issues on
2026-10-20.

### D30. A PNG or a JPEG decodes its pixels only when they are needed. Decided.

A maintainer decided on 2026-10-15. `ivimg.DecodePNG` and `ivimg.DecodeJPEG`
read the size and the color model from the header, and decode the pixels when a
step or an encoder asks for them. A terminal that gets the bytes of the file
never decodes them: the render of `png/precision.png` for Kitty went from 20.8
ms to 2.2 ms. A file with a good header and bad data now fails when the pixels
are asked for. The image keeps the error, and the scaler, the compositor and
the encoder entries return it.

### D31. The code follows go-pedantry where the two disagreed. Decided.

A maintainer decided on 2026-10-15 that the skill wins. Each receiver has one
or two letters, and an error message does not start with "unable to". A context
is passed, and never stored in a struct.

### D32. iv is built for 64-bit targets only, and has no static build. Decided.

A maintainer decided on 2026-10-15. `13ec599` had removed the 32-bit ARM
release, and the README still described 32-bit targets. The README section and
the 32-bit ARM path of `build.sh` are removed, and the stub of go-fitz went
with D28. The static Linux build was commented out of the release, and `-s` of
`build.sh` is removed.

### D33. Kitty gets raw pixels compressed with zlib when they compress well. Decided. Amends D22.

A maintainer chose the design on 2026-10-15, from measurements of each file
that Kitty gets as raw pixels in iv, and the rasterm session built it as
rasterm D18 and D19. iv requires rasterm v0.1.20. For raw pixels of over 64
KiB, the Kitty encoder writes one zlib stream at level 1, and checks after each
64 KiB that the output so far is under 25% of the input so far. At the first
check that fails, it sends the pixels raw. A PNG is still sent as its bytes.

The first form checked only the first 64 KiB, which is the top rows of an
image. A photo with a flat top then compressed whole, at 11 ms where raw took
1.35 ms. A maintainer chose on 2026-10-15 to check after each 64 KiB. In iv, a
page of markdown now goes to Kitty as 186 KB in place of 8 MB of raw RGBA, and
a photo stays raw.

rasterm v0.1.19 flushed the zlib stream at each check, and Ghostty 1.3.1 then
closed itself on `gxps/example.xps` at 1390x1966, over ssh. Each flush wrote a
stored block of 1 byte and an empty one after each block of 65535 bytes.
Python's zlib and Zig 0.16 inflate that stream, so the fault is in the inflater
of that Ghostty, likely an older Zig. The same pixels in one stream with no
flush, raw, and as a PNG all worked. rasterm v0.1.20 checks without a flush, so
the stream is one plain stream of all the pixels, and a maintainer confirmed
that it works in Ghostty.

### D34. A test file that crashes a command on one system is not decoded there. Decided. Amends D27. Amended by D49.

On macOS, `ebook-convert` of calibre 9.16 sometimes crashes with a
segmentation fault when it converts `calibre/sample.fb2`. It crashed in 2 of
about 5 CI runs, on 476d042 and on the tag v0.19.3, and passed when the job
ran again. It never crashed on Linux. The epub, mobi and azw3 files convert on
macOS every time.

A maintainer decided on 2026-10-28 that CI does not decode that file on macOS.
`untestedFiles` in `ivcmd/decode_test.go` names each such file with its
reason. The test still checks the mime type and the decoder of the file, and
skips only the decode.

A test can now skip for three reasons, and a map in `ivcmd/decode_test.go`
names each case for each system:

- `untested` names a command that CI does not install (D27).
- `unsupported` names a libvips operation that the libvips of CI is built
  without. On Windows, that is `gifsave_buffer`, because the libvips of MSYS2
  has no cgif.
- `untestedFiles` names a test file that crashes a command.

### D35. SVG renders with okage, a port of resvg to Go. Decided. Amends D3.

okage v0.1.0 is a port of resvg 0.49.0 and of tiny-skia to Go, with no cgo. On
2026-11-10, the svg decoder moved from xo/resvg to okage, at the request of a
maintainer, to see whether the slowdown is acceptable. A maintainer decided the
same day to keep okage. The decoder took the name `okage`, in `decoder/okage`,
and its test files moved to `testdata/okage` (D11).

Three things changed:

- okage reads no configuration of fontconfig. Its `svg.Renderer` loads the
  fonts of the system folders and of the home folder on its first render. It
  sets each css generic family, sans-serif too, to the first of its candidate
  families that is installed, and uses the family of serif for a font that is
  not installed. The lists of candidates were in `iv`, and moved to okage
  v0.2.0 at the request of a maintainer, so that the decoder only makes a
  renderer and calls it.
- xo/resvg loaded every font of the system when the package started, for
  `resvg.Default`. Each run of `iv` paid about 30 ms for it, whatever the
  file. okage loads the fonts only when an svg is decoded, in about 78 ms for
  3202 faces.
- okage rounds the size of an svg, as the resvg command does. xo/resvg
  truncated it, so some images are one pixel wider or taller.

The renders of the test files look the same. Of the 10 that were compared,
four have the same pixels, and the others differ in less than 1% of their
pixels.

The measurements ran on 2026-11-10, on an AMD Ryzen 9 9950X, with Go 1.27.3.
`BenchmarkDecode` loads the fonts once, so it measures only the drawing. Each
value is the median of 6 rounds, as the Performance section of `README.md`
says:

| File                            | xo/resvg | okage   | Change |
| ------------------------------- | -------- | ------- | ------ |
| `graphviz/booktest_sqlite3.dot` | 3.24 ms  | 3.99 ms | +23%   |
| `mermaid/flowchart.mmd`         | 38.5 ms  | 39.5 ms | +3%    |
| `okage/choropleth.svg`          | 15.3 ms  | 31.4 ms | +105%  |

A run of the command includes the start of `iv` and the fonts. Each value is
the mean of 15 runs that write a PNG to `/dev/null`:

| File                            | xo/resvg | okage   |
| ------------------------------- | -------- | ------- |
| `png/dice.png`                  | 58 ms    | 30 ms   |
| `okage/rect.svg`                | 108 ms   | 102 ms  |
| `okage/choropleth.svg`          | 137 ms   | 160 ms  |
| `okage/sunburst-book.svg`       | 1242 ms  | 619 ms  |
| `graphviz/booktest_sqlite3.dot` | 111 ms   | 112 ms  |

okage draws paths more slowly than resvg, and a file of many paths, such as
the choropleth, shows it. A build with `GOEXPERIMENT=simd` gave the same bytes
and the same times. A file that is not an svg starts faster, and a file with
much text, such as the sunburst with its Chinese labels, draws in half the
time.

### D36. One decoder named zuzu draws DOT and Mermaid. Decided. Amends D18.

zuzu (`github.com/xo/zuzu`) is a chart library in Go, with no cgo and no
WebAssembly. It reads DOT and Mermaid, and draws them to an `*image.RGBA`
through okage. On 2026-11-11, a maintainer decided that one decoder named
`zuzu` replaces the `graphviz` and `mermaid` decoders, as `resvg` became
`okage` (D35). `testdata/graphviz` and `testdata/mermaid` merged into
`testdata/zuzu`. The decoder came with zuzu v0.2.0 and okage v0.4.0. zuzu
v0.2.1 fixed a flowchart that it cut off at the bottom, and centered the labels
of a pie.

A maintainer made two more decisions on the same day:

- `iv` bundles the four icon packs of zuzu: fa, mdi, carbon and k8s. The
  binary grew by 4 MB, from 111.9 MB to 115.9 MB.
- `iv` keeps no fallback to xo/mermaid. If zuzu does not draw a Mermaid file,
  the work is an issue for zuzu. zuzu v0.2.0 does not draw an architecture,
  class, entity relationship, mindmap, sequence or state diagram. For these,
  the decoder returns the error of zuzu. Their test files left `testdata`.

The mime detector of the decoder is `zuzu.Sniff`. It returns
`text/vnd.graphviz` for DOT and `text/vnd.mermaid` for Mermaid. The detector
does not claim a file whose first line is `---`. zuzu reads that line as the
front matter of a Mermaid diagram, but a Markdown or a YAML file starts the
same way. A claim would keep such a file from libmagic and from `blitz`. A
Mermaid diagram with front matter routes by its extension.

The extensions `gv`, `dot`, `mmd` and `mermaid` route a file to zuzu when
libmagic types it as any kind of text. libmagic typed a DOT graph that starts
with long comments as `text/html`, and the `blitz` decoder took it. A `.dot`
that is not text is a template of Microsoft Word, and does not match. The
extension also sets the dialect, and `zuzu.Sniff` decides only for a source
with no extension, such as one read from standard input.

On 2026-11-11, a maintainer asked for the test data to hold the most complex
examples of Mermaid, and also examples that zuzu does not draw, as targets for
fixes in zuzu. `testdata/zuzu/broken` holds those. `TestDecodeBroken` fails
when zuzu draws one of them, so that the file moves to `testdata/zuzu`.

`--mermaid-theme dark` sets the dark mode of zuzu, and each other theme sets
the light mode. An unknown theme is an error that names it, as before.
`--mermaid-bg` sets the background through `zuzu.WithBackground` (D25). Both
flags change a Mermaid diagram only, and not a DOT graph.

A maintainer decided that the output of zuzu does not have to match the output
of graphviz or xo/mermaid. A difference in layout, size, color or font is not a
defect. A defect is a part of a diagram that zuzu draws wrongly or does not
draw, such as a node outside the image or two labels that overlap. Such a
defect is an issue for zuzu.

The time of one run of `iv`, the median of 21 runs, on 2026-11-11:

| File                   | Before | zuzu  |
| ---------------------- | ------ | ----- |
| `booktest_sqlite3.dot` | 104 ms | 38 ms |
| `flowchart.mmd`        | 171 ms | 37 ms |
| `pie.mmd`              | 127 ms | 37 ms |
| `gantt.mmd`            | 123 ms | 38 ms |
| `journey.mmd`          | 122 ms | 41 ms |
| `aws.mmd`              | 230 ms | 41 ms |
| `sankey.mmd`           | 148 ms | 56 ms |

On 2026-11-12, `iv` moved to zuzu v0.4.0, which also needs okage v0.4.0.
zuzu now draws the architecture, class, entity relationship, mindmap,
sequence and state diagrams, and seven files moved out of
`testdata/zuzu/broken`. It does not draw a block, C4, gitGraph, requirement or
timeline diagram. The decoder passes `zuzu.WithScale(2)`, so the image is
twice as wide and twice as high as the layout of zuzu. At a scale of 1, the
lines and the text of a diagram are blurred in a terminal.

On the same day, `iv` moved to zuzu v0.5.0, which reads more of the syntax
of Mermaid. Of the files in `testdata/zuzu`, only `gantt-sections.mmd` drew
differently: it gained the bar of a short task that v0.4.0 left out. The
five files in `testdata/zuzu/broken` still fail.

On the same day, `iv` moved to zuzu v0.6.0, which needs okage v0.5.0. zuzu now
draws the block, C4, gitGraph, requirement and timeline diagrams, and the five
files moved out of `testdata/zuzu/broken`. Nine new files took their place,
one for each kind that zuzu does not draw: cynefin, eventmodeling, ishikawa,
kanban, packet, railroad, treeView, venn and zenuml. They come from the same
corpus as the other files. zuzu v0.6.0 also draws edges darker and wider, with
larger arrowheads, and moves apart the labels of edges that end at the same
port. Both changes came from a report on the schema graphs of dbtpl.

On the same day, `iv` moved to zuzu v0.7.0, which draws every kind of diagram
of Mermaid 12.3. The nine files moved out of `testdata/zuzu/broken`, and the
folder was empty. `TestDecodeBroken` and the folder were removed. If a later
version of Mermaid adds a kind that zuzu does not draw, they can come back.

zuzu v0.7.1 fixed the one defect that the nine files showed. In
`venn-styles.mmd`, the texts "REST" and "GraphQL" of one area drew over each
other. zuzu now stacks the texts of an area in one column when one of them is
wider than its cell.

zuzu v0.7.2 came with okage v0.8.0 on 2026-11-15. It embeds IBM Plex Sans as
one collection in WOFF2 (zuzu D34), 958 KB smaller, and fixes marks, axes
and colors of Vega and Vega-Lite. Every test of `iv` passed with it.

zuzu v0.7.3 came on 2026-11-18. It changes no drawing, and requires okage
v0.12.0, the version that `iv` uses with ivy (D49).

zuzu v0.7.5 came on 2026-11-21, with okage v0.20.0 and the ivy commit
f96aae4. It draws the math of KaTeX in the labels of a Mermaid diagram
(D64).
okage now reads fonts with its own copy of go-text, so the module
go-text/typesetting left the `go.mod` of `iv`.

zuzu v0.10.0 came on 2026-11-26, with okage v0.22.1 and ivy v0.3.0. zuzu
lays out a Mermaid graph with a copy of dagre, draws Mermaid in new colors,
and takes the emoji font and the time in its parse (D65).

### D37. A minimum size given on the command line is a hard floor. Decided.

`--min-width` and `--min-height` default to 64. With the defaults, `iv` grows
an image only when it is below both minimums, and only until it reaches one of
them. A 1000x2 banner therefore stays 1000x2, and does not grow to 32000x64.

That rule ignored a minimum that the user gave. `iv --min-width=1000` left a
pie chart of 367x344 at its own size, because it was already taller than the
default minimum height of 64. On 2026-11-11, a maintainer decided that a
minimum given on the command line is forced, whatever the other minimum is.
`iv` grows the image until it reaches each minimum that was given. The default
of a minimum that was not given does not count.

The `set:` option of the `ox` struct tag records whether a flag was given. It
sets `MinWidthSet` and `MinHeightSet` of `ivctx.Config`.

Two rules did not change. The display size is still the hard bound, so an
image does not grow past the terminal. The modes `shrink`, `width` and
`height` still drop the floor that they drop, even a forced one.

### D38. A tall document is drawn whole and cut into pages. Decided. Amends D16.

Before, `blitz` drew a Markdown or HTML document up to 4000 CSS pixels and cut
off the rest. `iv` then shrank the whole image to fit the terminal, so a long
document was too small to read. On 2026-11-12, a maintainer decided that `iv`
draws the whole document and cuts the image into pages itself. A page is as
tall as the display height allows at the scale that fits the display width, so
each page fills the width of the terminal. `iv` cuts at that fixed height, and
does not look for a gap between lines. A document that fits the display is not
cut. `-p N` selects a page, as it does for a pdf.

`ivctx.PageHeight` computes the height of a page. It returns 0, and nothing
is cut, when no display height is set. This is the case for `--out` without
`--height`, and for the modes `none` and `width`.

The renderer of blitz draws at most 65532 pixel rows. It keeps the height in
16 bits and rounds it up to a tile of 4 rows. Before blitz v0.4.5, a taller
render failed with a wrong size or panicked, and since then it is an error
that names the limit. At scale 2, that is 32766 CSS pixels and 600 MB of
pixels. A maintainer chose
to draw a document at scale 2, and to draw it again at scale 1 when it is
taller than that. At scale 1, the same rows hold 65532 CSS pixels. Only a
document taller than that is cut off, with a warning. The README of usql is
36257 CSS pixels tall. It takes two renders, 1.5 seconds, and 41 pages on a
display of 1600x900.

### D39. blitz colors the code blocks of Markdown with transit. Decided.

blitz v0.5.0 added a hook for syntax highlighting. It calls a function for
each fenced code block of a Markdown render, and the function returns the
code as html. On 2026-11-12, a maintainer decided that `iv` fills the hook with
transit (`github.com/xo/transit`), a port of tree-sitter to Go with no cgo.
`decoder/blitz/highlight.go` parses each block with the grammar of its
language, runs the highlight query of the grammar, and colors each capture
with an inline style. The light style is `github` and the dark style, for
`--blitz-dark`, is `github-dark`, both from `github.com/xo/transit/styles`.

`iv` bundles 13 grammars of transit v0.7.8: Bash, C, C++, Go, HTML, Java,
JavaScript, JSON, Python, Ruby, Rust, SQL and TypeScript. They cover the code
blocks of most READMEs. The binary grew by 17.7 MB, from 117.9 MB to
135.6 MB. A block in another language keeps its plain look.

transit also has a grammar for each of 13 dialects of SQL. All 33 grammars with
a highlight query make the binary 163.1 MB, and the dialects are most of the
difference: neo4j alone is 4.4 MB. A maintainer decided that `iv` bundles only
the generic `sql` grammar, and that a block of a dialect is colored with it.
The names of cql, duckdb, mysql, oracle, postgres, sqlite3, sqlserver,
surrealdb, trino and usql, and their common spellings, such as `postgresql` and
`tsql`, map to `sql`. clickhouse, neo4j and ydb are not mapped, and their
blocks keep the plain look. An HTML document is not highlighted, because the
hook works on Markdown only.

The highlight query of C++ joins the queries of C and C++, and that of
TypeScript joins the queries of TypeScript and JavaScript, in the order of
the `tree-sitter.json` of each grammar. A query compiles on the first use of
its language. One highlighter serves both renders of a document (D38), and it
keeps the html of each block, so that the second render does not parse the
blocks again.

### D40. zuzu draws an emoji in color. Decided.

Without a font for emoji, zuzu draws nothing for an emoji, and reports the
character with `zuzu.ErrNoGlyph`. zuzu v0.6.0 added the package
`github.com/xo/zuzu/text/emoji`, which holds the font Noto Color Emoji. A
render that gets the font through `zuzu.WithEmoji` draws an emoji in color.

On 2026-11-12, a maintainer decided that `iv` passes the font. The binary grew
by 10.7 MB, from 138.0 MB to 148.7 MB. `testdata/zuzu/emoji.mmd` is a flowchart
with an emoji in each label, and `TestDecodeEmoji` fails when zuzu reports an
emoji that no font has.

### D41. A lottie draws with the lottie package of okage. Decided.

`iv` drew a lottie with `github.com/xo/lottie`, which runs the ThorVG engine as
WebAssembly. okage v0.6.0 added the package `github.com/xo/okage/lottie`, a
port of the crate velato to Go, which draws as ThorVG does where velato
differs. On 2026-11-12, a maintainer decided that `iv` draws a lottie with it.
The decoder parses the document once, and draws each frame on a new pixmap of
okage, at the size that the frame is shown at.

The pictures match those of ThorVG. Frames 1, 10 and 20 of the three files
in `testdata/lottie`, at 300x300, differ by an RMSE of 0.27% to 0.59%. The
binary is 4.2 MB smaller, because the WebAssembly runtime is gone.

ThorVG took about 230 ms to start in each run of `iv`, and okage starts at
once. One run of `iv` on a file at 1024x1024, the median of 7 runs:

| File          | ThorVG | okage v0.6.0 | okage v0.6.1 |
| ------------- | ------ | ------------ | ------------ |
| `rocket.json` | 251 ms | 236 ms       | 55 ms        |
| `fire.lottie` | 260 ms | 138 ms       | 76 ms        |
| `star.lot`    | 240 ms | 57 ms        | 36 ms        |

One frame took longer to draw. `BenchmarkDecode` drew `rocket.json` at
1024x1024 in 8.4 ms with ThorVG, after ThorVG started, and in 195 ms with
okage v0.6.0. Almost all of the time was in the pixel pipeline of okage,
which composited each layer over the whole canvas. okage v0.6.1 composites
only the part of each layer that has pixels, and the frames did not change.
One frame at 1024x1024, as `iv -v` logs it:

| File          | okage v0.6.0 | okage v0.6.1 |
| ------------- | ------------ | ------------ |
| `rocket.json` | 200 ms       | 18 ms        |
| `fire.lottie` | 100 ms       | 40 ms        |
| `star.lot`    | 26 ms        | 4.5 ms       |

`fire.lottie` spends most of its time on gradients of more than two stops,
which okage draws in its slower pipeline. A frame at 60 fps is on screen for
16.7 ms, so at full size the fire plays slower than its own speed (D19).

On 2026-11-13, the maintainer of okage moved the general code of the decoder
into okage v0.7.0 (okage D44). okage now detects a lottie, lists the
animations of a dotLottie archive, and draws a frame at a size. `iv` keeps
the mime types, the frame that a still shows, `--page`, `--fps` and the loop
that plays the frames. The frames did not change.

### D42. A Kitty terminal that can play an animation plays it itself. Decided. Amends D19. Amends D22. Amended by D43.

rasterm v0.2.0 changed its API (rasterm D26, rasterm D27), and added Kitty
animation (rasterm D25, rasterm D28). On 2026-11-14, a maintainer asked to move
`iv` to it, and to change the encoding and the animation to match.

`iv` calls `rasterm.Detect` once, before the first image, when it draws to the
terminal. Detect asks the terminal which protocols it supports, so the
protocol is right over SSH too. `ivctx.Config.Terminal` keeps the answer, and
the terminal encoder reads it from there. These rules apply:

- `iv` reads no input, so it drops the keys that the user types while Detect
  waits for the answers.
- When `TERM_GRAPHICS=halfblock` selects half blocks, `iv` stops with
  `rasterm.ErrTermGraphicsNotAvailable`. Half blocks draw an image as text,
  and `iv` does not do that (D1).
- When the ioctl of the terminal gives no size in pixels, the size of a cell
  that the terminal reports to Detect sets the display size.

On a terminal that plays a Kitty animation, `--animate` sends the frames as
one animation. `iv` sends each frame once, as the decoder makes it, and the
terminal keeps the time. No frame is dropped, and `--loop` sends nothing
after the first pass. kitty 0.50.0 and later report that they play
animations. Older kitty and Ghostty 1.4.0 and later play them and do not
report it, so `iv` also takes the name kitty, and the name ghostty with
version 1.4 or later. Another terminal with the Kitty protocol, such as
WezTerm or Konsole, shows only the first frame. There, and for Sixel and
iTerm, `iv` draws one image for each frame, as D19 describes.

The terminal stores every frame of an animation. Ghostty keeps 320 MB of images
for a screen by default, and kitty 320 MiB. The other images on the screen
count too, and a frame that does not fit fails with no message, because rasterm
sends `q=2`. When the frames need more than 256 MiB, at 4 bytes for each pixel
of the first frame, `iv` draws one image for each frame. The first limit was
320 MiB, which is 335.5 MB. In Ghostty, the last frames of
`testdata/ffmpeg/sample_960x540.mp4`, 331.8 MB in all, did not play. rasterm
D25 says that video must stay one image for each frame, because a stream has no
end. A video in `iv` is sampled to at most 720 frames (D19), so it has an end,
and the same limit applies to it.

A pass ends as it did before: at the start of the last frame, or after one
minute. With `--loop`, the terminal plays the frames until the playback is
interrupted. An interrupt stops the animation on the frame that shows.

In kitty 0.50.0, with a display of 2619x1220, `iv` sent the 6 frames of
`animated.gif` in 0.8 ms. It sent the 65 frames of `fire.lottie` at 1024x1024
in 2.2 s, as fast as okage drew them (D41). The first pass plays at that
speed. Each pass after it plays at the speed of the animation, 1.08 s, where
the player of D19 drew each pass in about 2.3 s. Ghostty and tmux were not
tested.

`iv` draws one image for each frame on the other terminals. On Kitty, it
deletes the images of the terminal before each frame. In tmux, the delete now
goes through the passthrough of tmux, which dropped it before.

On 2026-11-14, a maintainer found that `rocket.json` plays differently in
Ghostty 1.4.0. Its 40 frames draw the cycle of the rocket twice, and one image
for each frame showed both cycles. As one Kitty animation in Ghostty, it showed
one. The frames are the same in both cases, because okage draws frames 0 to 39
for each. The cause in Ghostty is not known. A maintainer asked for
`--animate-mode` to compare the two in one build: `auto` chooses as above,
`kitty` always sends one Kitty animation with no check of the name or of the
size, and `frames` always draws one image for each frame.

A maintainer then asked that `iv` use one Kitty image for any animation, and
make a frame only when the player needs it. The protocol does not report which
frame the terminal shows, so an animation that the terminal plays must store
every frame. rasterm v0.3.0 added `KittyStream` for this (rasterm D29). The
image stays stopped, and it holds two frames. For each frame, rasterm writes
the picture into the frame that is not shown, and then shows it. The player of
D19 keeps the time, and drops a late frame as before. So `--animate-mode` has
four values:

- `auto` sends one Kitty animation to a terminal that plays it, when the
  frames fit, and uses the stream when they do not. Any other terminal gets
  one image for each frame.
- `kitty` always sends one Kitty animation, with no check of the size.
- `stream` always uses the stream.
- `frames` always draws one image for each frame.

In kitty 0.50.0, the stream drew the 40 frames of `rocket.json` in 0.65 s.
With `auto`, `fire.lottie` at 2619x1220 used the stream and drew all 65
frames, and `sample_960x540.mp4` at 2000x1125 drew 156 of 160 frames. The
stream was not tested in Ghostty.

### D43. A video plays at its own speed, and a Kitty image shows each frame. Decided. Amends D19. Amends D22. Amends D42.

On 2026-11-14, a maintainer found that a long video played too fast. That was
the sampling of D19: a video longer than a minute was cut into 720 frames,
spread over its length, and played in one minute. A 3 minute test video played
one frame for each 250 ms of video, in 59.9 s, in v0.21.1 and with D42 alike. A
maintainer asked that a video play at its own speed and length, and said that
dropped frames are fine.

The `ffmpeg` decoder now decodes a video once, from the start. It uses the
frame rate that ffprobe reports, up to 30 frames a second, or `--fps`.
ffmpeg scales each frame to the size that it is shown at, and writes it as
raw RGBA pixels. A raw frame needs no encode in ffmpeg and no decode in
`iv`, and Kitty sends it as it is. ffmpeg turns a rotated video upright, so
`iv` reads the rotation from ffprobe and swaps the width and the height for
90 and 270 degrees. The seek mode of D19 for long videos is gone.

`decoder.Anim` has a `Video` field. For a video, the player of D19 has two
changes. A pass is as long as the video, with no limit of one minute. The
player drops each late frame, also one that the decoder made late, so that
the video keeps its length. For any animation, the clock now starts at the
first frame, so the time that ffmpeg takes to start makes no frame late.

A maintainer also found that `rocket.json` plays too fast in Ghostty 1.4.0 when
the terminal plays it (D42), and asked that the stream be the default. So
`--animate-mode auto` now uses the `KittyStream` of rasterm v0.3.0 on a
terminal that plays Kitty animations, and one image for each frame on any other
terminal. The terminal never stores more than two frames, so the size limit of
D42 is gone. `--animate-mode kitty` still sends one animation that the terminal
plays. The cause in Ghostty is not known, and rasterm keeps it in its backlog.

In kitty 0.50.0, `sample_960x540.mp4`, which is 13.3 s at 29.97 frames a
second, played its 400 frames in 13.36 s with none dropped, at 960x540 and
at 2000x1125. A test video with a rotation of 90 degrees played upright at
360x640, 49 frames in 2.0 s. `testdata/ffmpeg` holds two videos for these
tests, `testsrc2.mp4` and `testsrc2-rotated.mp4`.

### D44. zuzu draws a Vega spec, and reads its data only under its directory. Decided. Amended by D45. Amended by D46.

On 2026-11-15, a maintainer asked that `iv` draw a Vega spec with zuzu, and
that the spec read its data only from the directory of the spec and the
directories under it.

The `zuzu` decoder claims a file whose name ends in `.vg.json`, in any case,
with the mime type `application/vnd.vega+json`. The extension of such a file
is `json`, which says nothing, so the whole ending of the name decides. A
json file with another name is not claimed, even with the `$schema` of Vega.
`--force-mime application/vnd.vega+json` draws a spec with another name.

zuzu reads no file of its own (zuzu D25). A spec names its data by url, and
the program gives zuzu a loader. `iv` opens the directory of the spec with
`os.Root`, and gives zuzu `zuzu.FSLoader(root.FS())`. The root refuses a path
that leaves the directory, through `..` or through a symbolic link that
points out of it. `FSLoader` refuses a url with a scheme, such as `https:`,
and reads a leading `/` as the directory of the spec.

Only a file on disk gets a loader. The path name of a file in an archive is
a name in the archive, and the directory of that name could be any directory
on disk. So the pipeline marks the file that `DecodeFile` opens with
`ivctx.WithFile`, and clears the mark for a file of a set. A spec that is not
a file on disk reads no data by url, and zuzu returns `zuzu.ErrNoLoader`.

`testdata/zuzu/sales.vg.json` reads `testdata/zuzu/data/sales.csv`.
`TestVegaData` tests a subdirectory, a leading `/`, `..`, a symbolic link
that points out, a url with a scheme, and a spec that is not a file on disk.

### D45. --zuzu-data sets the directory that a Vega spec reads its data from. Decided. Amends D44.

On 2026-11-15, a maintainer asked for an option that sets another directory
as the root of the data of a Vega spec. `--zuzu-data DIR` does that. Empty,
the default, keeps D44: the root is the directory of the spec.

With `--zuzu-data`, `iv` opens `DIR` with `os.Root`, and a url is a path
under `DIR`. The rules of D44 still hold: a path that leaves `DIR`, through
`..` or a symbolic link, and a url with a scheme are refused. A spec that is
not a file on disk, such as a file in an archive, also reads its data from
`DIR`, because the user named the directory. A `DIR` that does not exist is
an error.

### D46. zuzu draws a Vega-Lite spec as it draws a Vega spec. Decided. Amends D44.

On 2026-11-15, a maintainer asked that `iv` draw a Vega-Lite spec too. A
file `group.vl.json` stopped with `mime type "application/json": not
supported`. The `zuzu` decoder now claims a file whose name ends in
`.vl.json`, in any case, with the mime type `application/vnd.vegalite+json`,
and draws it with the Vega-Lite dialect of zuzu. A Vega-Lite spec reads its
data by url with the same loader as a Vega spec (D44), and `--zuzu-data`
sets its root too (D45). `testdata/zuzu/sales.vl.json` is the chart of
`sales.vg.json` in Vega-Lite, and reads the same data.

### D47. The binwalk decoder uses the walker of xo/magic. Decided. Amends D7. Amends D27.

The `binwalk` decoder ran the `binwalk` command with `--extract` into a
temporary folder, and kept the extracted files that a builtin decoder reads
(D7). Linux CI built binwalk 3 with `cargo`, because binwalk 2 of Ubuntu
could not extract the image in `testdata/binwalk/icon.afdesign`, and Windows
had no package of it.

On 2026-11-15, a maintainer asked to move to the walker of xo/magic, a port
of binwalk 3 to Go. `iv` first required its untagged commit `665c775`, then
magic v0.6.0, the first release with the walker, and then v0.6.1. The decoder now walks
the file in memory with `walk.WalkBytes`, and needs no command and no
temporary folder:

- It keeps a match of png, jpeg, gif or bmp, and a RIFF that holds a WebP,
  when a builtin decoder reads its extension. A WAVE is a RIFF too, so the
  description decides.
- It walks to depth 1, so it also finds an image in a file of an archive
  that it finds, as `--extract` does. It does not walk into an image match,
  because the walker carves an image as a file of its own, and would find
  it again. It unpacks at most 256 MiB from one match.
- Each image is named by its offset in hex, with 8 digits, and by the name
  of the file that holds it, so the names sort in the order of the file. The
  name of a member comes from the file, so it is cleaned into a path that
  stays in the file set.
- It imports `walk/format/all`, and the stream decoders of
  `compress/xz` and `compress/zstd`, which are modules of their own. The
  walker unpacks xz, lzma and zstd only with them, as the binwalk command
  does.

`icon.afdesign` gives the same png, at offset 0x1D279, and the same output,
byte for byte. The walker and its stream decoders add 1.03 MB to the release
build, from 106.64 MB to 107.67 MB. CI no longer installs binwalk.

magic v0.6.0 missed an xz stream when the bytes before it were at least the
bytes from its start to the end of the input, where binwalk 3.1.0 finds it.
The walker had copied a fault of binwalk after 3.1.0, which is reported as
ReFirmLabs/binwalk#982. magic v0.6.1 fixed it, and `TestDecodeXZ` puts its
stream after 256 bytes.

### D48. okage draws font previews, and measures the text of the cards. Decided.

The `fontimg` decoder drew a specimen of a font with
`github.com/kenshaw/fontimg`, on tdewolff/canvas and tdewolff/font.
`internal/font` used canvas too, to find the font of the contact and audio
cards and to measure their text. On
2026-11-15, a maintainer asked to move to okage. The maintainers of okage
agreed that the specimen is part of `iv`, and okage added what it lacked:
`FontDatabase.LoadFontData` reads WOFF, WOFF2 and EOT, and
`FontDatabase.FaceName` reads the strings of the name table (okage D45).
`iv` requires okage v0.8.0, the first release with them.

The decoder loads the font into a font database of its own, so the text
draws in that font alone, and a character that it lacks draws its `.notdef`
glyph. It keeps the specimen of `fontimg`:

- The first line names the typographic family, the typographic subfamily
  (name ID 17) or else the subfamily (2), and the version (5), up to a
  semicolon, after which a font often puts the data of its build. `fontimg`
  named the family of name ID 1, so `Figtree-Italic[wght].ttf` was "Figtree
  Light, Regular Italic", and is now "Figtree, Light Italic".
- The alphabet, the digits and four pangrams at growing sizes follow. In
  `fontimg`, the sample text of a font (19) took the place of all of them,
  so IBM Plex Sans showed only its sample text. A maintainer asked that each
  font show its characters at each size, so the sample text is now one more
  line, after the first.
- A face that draws a line of emoji in color gets that line after the
  characters. A font of emoji, which draws no lowercase letter, gets its
  first line in the font of the cards (`internal/font`), and the line of
  emoji at the sizes of the pangrams. A line that no font draws is left
  out. Noto Color Emoji of zuzu tests it, so its 10.7 MB are not test data.
- A size is in points and the margin in millimetres, at `--font-dpi`, as
  in `fontimg`. Each test font comes out within one pixel of its old size.

`--font-style` and `--font-variant` took the bare integers of canvas, and
`--font-style` changed nothing. `--font-weight`, from 1 to 1000, and
`--font-italic` replace them. They select the weight and the style that okage
shapes with, and set the axes `wght`, `ital` and `slnt` of a variable font.
The default is the weight and the style of the face, so in a collection,
`--page` selects the face. When `--font-weight` is given, the first line
names it.

`internal/font` loads the fonts of the system once, with
`FontDatabase.LoadOSFonts`, as the renderer of okage loads them, and measures
a string with `svg.ShapeText` in the family and the weight that the card
names. `iv` first copied the function of the renderer that loads them. The
maintainers of okage exported it in v0.9.0 (okage D46), and `iv` uses it. Each contact and audio card in `testdata` draws
the same pixels as before. tdewolff/canvas, tdewolff/font and
kenshaw/fontimg left `go.mod`, and the release build went from 107.67 MB to
99.81 MB. `testdata/fontimg` has a WOFF and a WOFF2 of Noto Mono, which draw
the same pixels as its TrueType, and IBM Plex Sans from zuzu, six faces in
one collection in WOFF2. Each of its faces draws the same pixels as the same
face of the collection that `woff2_decompress` gives. `pfb` left the extensions, because nothing
read it.

### D49. xo/ivy draws documents, ebooks and page descriptions in Go. Decided. Amends D27. Amends D28. Amends D34. Amended by D50.

D28 gave the formats of mupdf to four outside commands and one package:
`soffice` of LibreOffice, `ebook-convert` of calibre, `xpstopng` of libgxps,
`pdftoppm` of poppler, and oov/psd. On 2026-11-18, a maintainer asked to
move to xo/ivy, which decodes and draws such formats in pure Go with okage.
ivy had no tag yet, so `iv` requires its pushed commit `e0b710b`. That
commit requires okage v0.10.0 and magic v0.7.0, and `iv` moved to them. Later
on the same day, `iv` moved to the ivy commit `6e838f4`, okage v0.12.0 and
magic v0.8.0. On 2026-11-19, `iv` moved to the ivy commit `72caf3e`, which
requires okage v0.14.0 and magic v0.9.1. Later on that day, `iv` moved to the
ivy commit `8bb9f50`, which requires okage v0.17.0, and then to the ivy commit
`871c3f6`, which requires okage v0.18.0. On 2026-11-20, `iv` moved to the ivy
commit `c64d600`, which has the package `icc` (D57).

`decoder/ivy` registers one decoder for each format of ivy that `iv` takes,
named `ivy-` and the name of the format, from the `Format` value that the
package of the format exports. The decoders `libreoffice`, `calibre`,
`gxps`, `poppler` and `psd` are gone, and CI installs only `ffmpeg`. `iv`
gains CHM, DjVu, EMF and WMF, the previews of camera RAW files, EPS,
iWork, Visio and Publisher, OpenRaster and Krita, and CSV as a table. It
loses WordPerfect, which only LibreOffice read.

- A document with pages draws the page of `--page` over white. It draws at
  `--dpi`, or at the lower resolution that draws the page at the size that
  it is shown at. A page of a raster image (Photoshop, OpenRaster, camera
  RAW) draws at its own size, with its transparent pixels, so that `--bg`
  applies to it as to any other image.
- A document with no pages, such as a Visio drawing, draws its preview that
  is closest to the display size.
- A document lays out its text in the fonts that it embeds, then in the
  fonts of the system. `iv` does not bundle the fonts module of ivy, which
  holds fonts with the metrics of the fonts of Office, because it adds about
  10.9 MB.
- A decoder does not register a generic mime type, such as `text/csv` or
  `application/zip`, which would claim every file of that type. The
  pipeline routes such a file by its extension. A compound file of Office
  97 to 2003, which libmagic types as `application/x-ole-storage`, is
  routed by its extension too. A table with no extension, as from a pipe,
  is a CSV.
- Each decoder of ivy comes before libvips, which still draws a pdf that
  ivy cannot. blitz keeps HTML, Markdown and text, and the archives decoder
  keeps the comic archives, because ivy does not read CBR.

`testdata/calibre`, `gxps`, `libreoffice` and `psd` became `testdata/ivy`.
Each office document, ebook, XPS and Photoshop file of the test data draws
close to the old output, and the Photoshop files keep the same alpha. ivy
cannot draw a WMF image in `file-sample_100kB.doc`, so the page has no chart
and `iv` warns that it is drawn in part. The cover of `sample.epub` has a
white strip at its right side, where the page is wider than the cover.

One page, at a display of 1200x1200, as one run of `iv`:

| File | Before | ivy |
| --- | --- | --- |
| `file-sample_100kB.docx` | 505 ms | 142 ms |
| `sample.epub` | 1619 ms | 155 ms |
| `example.xps` | 845 ms | 99 ms |
| `file-sample_150kB.pdf` | 53 ms | 73 ms |

The release build went from 99.81 MB to 117.51 MB. ivy adds 8.25 MB of code,
okage, golang.org/x/image and golang.org/x/text add 3.6 MB, and the tables
of the runtime grow with the code.

### D50. ivy draws every pdf, and asks for a missing password. Decided. Amends D49.

D49 kept the `vips-pdf` decoder after `ivy-pdf`. It drew a pdf that ivy
could not, and it asked for the password of an encrypted pdf at the
terminal. On 2026-11-18, a maintainer asked to remove it and to give its
work to ivy.

- `vips-pdf` is gone. The `vips` decoder takes only images, so libvips draws
  no pdf. If ivy cannot draw a pdf, `iv` shows the error of ivy.
- The ivy decoders give ivy the password of `--password`, and a function
  that asks for the password at the terminal while it is missing or wrong.
  The function asks at most three times, as `vips-pdf` did. If stdin is not
  a terminal, it does not ask, and the error says to give `--password`.
- Each format of ivy that takes a password asks for it in the same way, such
  as docx, pptx, odf and the Office 97 to 2003 formats, and not only a pdf.
- The plain and the encrypted pdf moved from `testdata/vips` to
  `testdata/ivy`. `TestPassword` in `decoder/ivy` gives the password with
  `--password`, with answers at the terminal, with too many wrong answers,
  and with no terminal.

### D51. A spreadsheet is drawn in landscape by default. Decided.

ivy draws a sheet on the paper that its file gives. A file that gives no
paper takes the paper of `ivy.Config`, and the zero paper is US Letter in
portrait. A table is usually wider than it is tall, so on 2026-11-19 a
maintainer asked to draw a spreadsheet in landscape by default.

- The ivy decoders give `ivy.Letter.Landscape()` as the paper of csv, xlsx
  and xls files, and of an OpenDocument spreadsheet. Any other document
  keeps the zero paper.
- The odf format of ivy holds text documents, spreadsheets and
  presentations, and a decode function does not get the mime type. The
  decoder reads the mime type from the first entry of the zip, which the
  OpenDocument package stores first and uncompressed.
- ivy uses the paper only where the file gives none. A sheet that gives its
  orientation keeps it. An xlsx sheet that gives its own paper size and no
  orientation is in portrait, because ivy reads it so. The `file_example_*` sheets in
  `testdata/ivy` give A4 in portrait, and stay in portrait.
- `TestPaper` in `decoder/ivy` checks each case, with the new
  `testdata/ivy/sales.csv`. `TestIsODS` checks the test of the mime type.

### D52. A scheduled workflow keeps the AUR packages current. Decided. Amended by D66.

The Announce Release workflow bumped the AUR packages `iv-cli` and
`iv-cli-bin` when a release was published. Nothing tried a bump again when
it failed, or when the event did not arrive. On 2026-11-15, the bump of
`iv-cli-bin` for v0.23.0 failed, because GitHub gave a 404 for an asset of
the release that it had just published. On 2026-11-19, a maintainer asked to
move the bump to a workflow that runs on a schedule.

- `.github/workflows/aur.yml` runs every 12 hours, at 00:23 and 12:23 UTC,
  and can also be started by hand. It does not run on the release event.
- Its first job reads the tag of the latest published release with `gh`,
  and the version of each package from the RPC interface of the AUR. A
  package is behind when its `pkgver` is lower than the release. A package
  that is at the release, or past it, is left alone.
- A second job bumps only the packages that are behind, in an Arch Linux
  container, as Announce Release did. It does not start when no package is
  behind. If a bump fails, the next run tries again.
- Announce Release keeps the Homebrew formula and the Discord announcement.
  D66 moves the Homebrew formula to a scheduled workflow.
- GitHub turns off a scheduled workflow in a public repository after 60 days
  with no activity. A maintainer then turns it on again on the Actions tab.

### D53. The Release workflow runs only for a version tag. Decided.

The Release workflow ran on each push. It built the binaries for Linux,
macOS and Windows on seven runners, and drafted a release only when the push
was a tag. On 2026-11-19, a maintainer asked that a push only be tested, to
stay inside the limits of GitHub Actions.

- The Release workflow runs only for a pushed tag that starts with `v`. Each
  of its runs drafts a release.
- The Test workflow runs on each push and each pull request, as D14 says. It
  builds no binaries for a release. Its `go build ./...` step makes sure
  that the code compiles.
- A change to `build.sh` or to the Release workflow is tested only by the
  next tag. The job that drafts the release needs every build, so a draft
  is made only when every build passed.

### D54. A script makes the screenshots of the README in Ghostty. Decided.

On 2026-11-19, a maintainer asked for screenshots in `README.md` of what the
output of `iv` looks like in Ghostty, made by a script.

- `docs/screenshots/screenshots.sh` builds `iv`, and makes eight screenshots
  of files in `testdata/`: a png, an svg, a Mermaid flowchart, a Vega-Lite
  chart, a Markdown document, a Word document, a font and a WIFI code.
- It runs on Sway. For each screenshot, it opens a Ghostty window of
  1400x880 pixels with the default configuration of Ghostty, on a workspace
  of its own, so that no other window covers it. It runs `iv` in the window,
  captures the window with `grim`, and compresses the PNG with `optipng`.
- Each window ends its command, and so closes with no confirmation, when
  its screenshot is taken, or when the script ends early.
- The window shows the command line of `iv`. The script gives `iv` a width
  of one column less and a height of four rows less than the terminal, so
  that the command line stays on screen, and the command line does not show
  those options.
- A screen lock makes each screenshot black, so the script must run while
  the screen is not locked.

### D55. A document is written to a pdf with ivy, as vectors. Decided.

On 2026-11-19, a maintainer asked to write a pdf with the PDF writer of
ivy. The ivy commit `8bb9f50` has it, in `pdf/pdfenc`. It writes the pages
of a document through okage, with their paths and their images as vectors.
In that commit, the glyphs of the text are paths, so a reader cannot search
or copy the text. Later on that day, `iv` moved to the ivy commit `871c3f6`,
which writes the text as text, in subsets of the fonts that it embeds. The
glyphs of a pdf or a PostScript file are still paths.

- `ivctx.Config.Encoder` names the encoder that writes the output. When it
  is `pdf`, `decoder/ivy` returns the page that it drew, wrapped in a type
  that keeps the document open and writes it with `pdfenc`. For any other
  encoder, the decoder returns the image as before, so that the terminal
  encoder gets the concrete image type that it encodes fastest.
- An encoder can take a decoded image before `ivimg.Prepare` scales and
  composites it, through `encoder.Source`. The `pdf` encoder takes an image
  that has a `WritePDF` method. A document is paper, so `-W`, `-H` and
  `--bg` do not change it.
- The pdf holds every page of the document, or only the page that `--page`
  selects, when `--page` is given.
- Any other image, and a format of ivy with no pages, such as a Visio
  preview, goes on a page with blitz, as before. `--pdf-page` and
  `--pdf-margin` apply only to such an image.

### D56. Pure Go codecs replace libvips. Decided. Amends D5. Amends D8. Amends D9. Amends D22. Amends D27. Amended by D57.

On 2026-11-20, a maintainer asked to replace libvips with pure Go packages,
many of them written with coding agents in 2026, and to measure them against
libvips first. The maintainer chose to drop FITS, NIfTI and JPEG XS, which
have no maintained pure Go package, and to keep OpenEXR, Radiance HDR, PFM,
MATLAB and the file format of libvips.

| Decoder    | Encoder    | Package                                     |
| ---------- | ---------- | ------------------------------------------- |
| `heic`     | `heic`     | `github.com/gen2brain/h265`                 |
| `avif`     | `avif`     | `github.com/gen2brain/gav1d`                |
| `jxl`      | `jxl`      | `github.com/gen2brain/jxl`                  |
|            | `vpx`      | `github.com/gen2brain/vpx`, a lossy webp    |
| `jpeg2000` | `jpeg2000` | `github.com/mrjoshuak/go-jpeg2000`          |
| `exr`      |            | `github.com/mrjoshuak/go-openexr`           |
| `hdr`      |            | `github.com/mdouchement/hdr`, HDR and PFM   |
| `matlab`   |            | `github.com/scigolib/matlab`                |
| `vipsfile` |            | written in `iv`                             |
|            | `tiff`     | `golang.org/x/image/tiff`                   |
|            | `gif`      | `image/gif`                                 |

- The test files come from the test data of these packages, or were made
  from `png/rose.png`. `testdata/README.md` names each one.
- `vips/`, `decoder/vips`, `encoder/vips` and `--vips-concurrency` are gone.
  So are `ivimg.Resizer`, `ivimg.Flattener` and `ivimg.JPEGEncoder`, which
  only the image of libvips implemented, the `ErrUnsupportedFormat` errors,
  the `unsupported` map of `ivcmd/decode_test.go`, and
  `TestEncodeAfterDecode`. The `bmp` decoder no longer hands a bitmap that it
  cannot read to libvips.
- `iv` is still cgo code, because blitz is. CI installs fontconfig and
  freetype on Linux, nothing on macOS, and gcc on Windows. The Go cache of CI
  no longer has the flags of libvips in its key.
- `internal/scrgb` encodes the linear light of OpenEXR, Radiance HDR, PFM
  and a float `.v` file as sRGB, and clips it, as libvips did.
- The `exr` decoder takes the color out of the alpha, as OpenEXR stores it.
  libvips read OpenEXR through ImageMagick, which did not, and drew a file
  with only a G channel in gray. `exr/rose.exr` matches its source to 90 dB
  with the new decoder, and to 17 dB with libvips.
- The PFM reader of mdouchement/hdr returns the rows from the bottom up, as
  the file stores them, and stops at a comment line, which libvips writes.
  The `hdr` decoder turns the rows over and skips the comment.
- `iv` routed only an `image/` mime type to libvips, so it never drew a
  MATLAB file. scigolib/matlab v0.3.19 reads version 5, and returns each array
  of a version 7.3 file with one dimension, so a version 7.3 file has no
  image (see `docs/BACKLOG.md`).
- The decoders do not convert colors with an ICC profile or the CICP values
  of a file, as libvips did. ivy has the code, but not in a package that `iv`
  can import (see `docs/BACKLOG.md`).
- The tag v1.5.12 of go-jpeg2000 on GitHub differs from the code that the
  checksum database recorded, so `go get` with `GOPROXY=direct` fails. The
  module proxy serves the recorded code.

On 10 files that both read, a render to Kitty took 0.88 times as long as
with libvips, a render to iTerm 1.29 times, and a render to Sixel the same.
Encoding a photo in 9 formats took 1.33 times as long. The Measures section
of `README.md` has the numbers.

### D57. The decoders convert colors to sRGB with the icc package of ivy. Decided. Amends D56.

D56 left the colors of a file as they were decoded. On 2026-11-20, a
maintainer asked the ivy session to make its port of moxcms public, and ivy
`c64d600` has it as the package `icc`. `internal/cms` calls it for the
decoders:

- The `heic`, `avif` and `jxl` decoders give the ICC profile of the file, or
  its CICP code points when it has no profile. The `jpeg2000` decoder gives
  the profile of a jp2 file. go-jpeg2000 converts the color spaces of the
  format to sRGB itself.
- The `jpeg` decoder converts a CMYK or YCCK JPEG with the profile in its
  APP2 segments, or with the default CMYK profile of ivy when it has none.
  Before, the Go decoder converted CMYK with a formula and no profile.
- An image in sRGB, or with no color space, is returned as it is. An image
  with the PQ or the HLG transfer of HDR is returned as it is too. The `icc`
  package does no tone mapping, and such an image would look dark.
- The rendering intent is relative colorimetric, the default of libvips. A
  profile that `icc` cannot use leaves the image as it is, with a warning.
- A png and an RGB jpeg keep the bytes of their files (D22), and their
  profiles are not applied. `x/image/tiff` does not decode a CMYK tiff.

`jpeg/rose-cmyk-icc.jpg` is the rose in CMYK with the SWOP profile of colord.
Its decode matches the conversion of lcms to 53.6 dB, where the decode of
the Go formula reached 23.0 dB. `jxl/xyb_p3.jxl`, in DCI-P3, moved from 22.5
dB to 34.7 dB of the conversion of libvips.

The conversion costs time for a file that embeds a profile. A render to
Kitty of `heic/dome.heic`, of 36 megapixels, took 930 ms in place of 590 ms,
and of `avif/precision.avif` 18 ms in place of 9 ms. Both profiles describe
sRGB, so the conversion does not change them (see `docs/BACKLOG.md`).

### D58. --page selects a top-level image of a heic or avif file. Decided.

A maintainer noted on 2026-11-20 that `heic/cyberpunk.heic` holds more than
one image. libheif lists 12 top-level images in it, and 2 in
`heic/dome.heic`. libvips loaded the image that `--page` selected, and D56
lost that, because gen2brain/h265 and gen2brain/gav1d decode only the primary
image, which the `pitm` box of the file names.

- `internal/heif` reads the boxes `pitm`, `iinf` and `iref` of the file. An
  image item is top level when it is not hidden, and is not a thumbnail, an
  auxiliary image or a tile of a grid, as libheif reads it. The order is the
  order of `iinf`. `TestParse` checks the list of each test file against the
  list of `heif-info` of libheif 1.21.
- For `--page N`, the decoder gives the codec a copy of the file whose `pitm`
  box names the Nth top-level image. Without `--page`, or with a page past
  the last image, the codec decodes the primary image, as libvips did.
- The frames of an image sequence, such as an animated avif, are not pages
  yet (see `docs/BACKLOG.md`).

### D59. The verbose output says which image of a dynamic wallpaper is shown. Decided.

A maintainer found on 2026-11-20 that `heic/dome.heic` and
`heic/cyberpunk.heic` are dynamic wallpapers of macOS, and asked that `-v`
show this metadata for the page that is shown.

- The XMP of the primary image holds a property of the namespace
  `apple_desktop`: `h24` names the time of the day of each image, `solar`
  the position of the sun, and `apr` only the images of the light and the
  dark appearance. Its value is a binary property list in base64, which
  `howett.net/plist` reads.
- The `heic` and `avif` decoders read the XMP with `RawXMP` of the codec,
  from the file as it was, because the XMP belongs to the primary image.
  `heif.Select` gives the index of the image that is shown.
- `dome.heic` names its first image for the light appearance, from 00:00,
  and its second for the dark appearance, from 12:00. `cyberpunk.heic` has 12
  images, from 00:00 to 23:00, and names images 4 and 10 for the light and
  the dark appearance.

### D60. The verbose output holds all of the metadata of an image file. Decided.

A maintainer asked on 2026-11-20 that `-v` show the metadata that each image
format holds, with no field left out, and asked whether a known Exif package
could do it.

- `internal/meta` reads the file with `github.com/bep/imagemeta`, which Hugo
  uses. It reads every Exif, IPTC and XMP tag of a JPEG, TIFF, PNG, WebP,
  HEIC, AVIF or JPEG XL file and of a DNG, CR2, NEF, ARW or PEF file, and the
  size and the frame count, and is tested against `exiftool -n -json`. Its
  format detection is not implemented, so `iv` gives the format from the mime
  type, or from the extension of a camera RAW file. A JPEG XL file can hold
  its metadata compressed with brotli, which `andybalholm/brotli` reads.
- The package adds what imagemeta does not read: the `tEXt`, `zTXt` and
  `iTXt` chunks of a PNG, the COM segments of a JPEG, the comment extensions
  of a GIF, and the description of an ICC profile in a JPEG, a PNG and a
  WebP. A text chunk is read as UTF-8 when it is valid UTF-8, because many
  programs write UTF-8 where the format says Latin-1.
- Each tag is written as it is, with its source and its IFD or namespace,
  and bytes in hexadecimal. The GPS tags are written too, as the maintainer
  asked.
- `internal/cms` names the profile that it converts with, as in "converted
  to sRGB with the profile "SWOP TR003 Coated"".
- An Exif reader of its own would take about 200 lines for the common
  fields, and the tag tables of exiftool for all of them, so `iv` did not
  write one.
- The test files `sunrise.*` of bep/imagemeta hold full Exif, IPTC and XMP,
  and the test checks a tag of each source.

### D61. Every animated format plays, and --page selects a frame. Decided. Amended by D63.

A maintainer asked on 2026-11-20 for the decoders of gen2brain that were
worth adding. Before, `--animate` played a gif, a lottie and a video, and an
animated png, webp, jpeg xl, or a heic or avif sequence showed its first
frame.

- `decoder.Frames` takes the frames of an animation, each a whole image, with
  their delays. For `--animate`, it returns an `Anim` that plays them.
  Otherwise it returns the frames as a set of images, from which the pipeline
  takes the frame that `--page` selects. A delay that is not positive is a
  tenth of a second, as a browser shows it.
- The `png` decoder decodes an APNG with `pngn.DecodeAll` of gen2brain/pngn,
  when the frames are asked for. A still png keeps the bytes of its file.
- The `webp` decoder decodes with gen2brain/vpx, which is a port of libwebp,
  and replaces the decoder `nativewebp`. Its test files moved to
  `testdata/webp` (D11). The `nativewebp` encoder stays (see D63).
- The `jxl` decoder, and the `heic` and `avif` decoders for a file with one
  top-level image, return the frames of `DecodeAll`, with their colors
  converted (D57). In a heic or avif file with several top-level images,
  `--page` selects an image (D58).
- The `gif` decoder composites its frames for `--page` as for `--animate`.
- `gif/animated.gif` was made again as an APNG, an animated webp and an
  animated jpeg xl, and frames 1, 3 and 6 of each are the same pixels as the
  frames of the gif.

### D62. jpegn decodes a JPEG, and a large one at a fraction of its size. Decided. Amends D22.

A maintainer asked on 2026-11-20 to adopt `gen2brain/jpegn` after the
animations, if a benchmark showed that it pays off. jpegn is a pure Go JPEG
codec with SIMD. It reads what `image/jpeg` reads, and can decode at 1/2,
1/4 or 1/8 of the size inside the inverse transform.

- `ivimg` decodes a JPEG with jpegn. A JPEG that is not scaled still goes to
  the terminal as the bytes of its file (D22).
- `ivimg.Reducer` is an image that can decode itself at a smaller size.
  `ivimg.Resize` scales a `Reducer` that is made smaller from its reduced
  decode. The JPEG of `ivimg` decodes at the smallest fraction that is still
  at least the size that it is shown at.
- To fit 1920x1200, a JPEG of 6016x6016 took 165 ms in place of 837 ms with
  `image/jpeg`, and one of 4000x3000 took 112 ms in place of 315 ms. jpegn at
  full size took 623 ms and 237 ms. The result matches the decode at full
  size to 47 dB and 48 dB.

### D63. vpx encodes webp, and nativewebp is gone. Decided. Amends D61.

D61 moved the decoding of webp to gen2brain/vpx and kept the `nativewebp`
encoder, which then was the only use of `github.com/HugoSmits86/nativewebp`.
A maintainer asked on 2026-11-20 to drop it.

- `encoder/webp` registers `webp`, which writes a webp with no loss, as
  `nativewebp` did, and `webp-lossy`, which replaces the encoder `vpx`.
  `--out` selects `webp` for the webp extension, because it comes first by
  name.
- For the photo of `BenchmarkEncode`, `nativewebp` took 249 ms for 791 KB,
  and vpx with no loss took 276 ms for 731 KB. The file of vpx decodes to the
  same pixels as the photo.
- `--encoder nativewebp` is now `--encoder webp`, and `--encoder vpx` is now
  `--encoder webp-lossy`.

### D64. ivy draws LaTeX and plain TeX, and zuzu draws the math of KaTeX. Decided.

The ivy commit f96aae4 added the package `latex`, a copy of the TeX engine
of go-tex/engine, and zuzu v0.7.5 draws the math of KaTeX in a label of
Mermaid. A maintainer asked on 2026-11-21 for test files of both.

- `decoder/ivy` registers `ivy-latex` for the extensions `tex`, `latex` and
  `ltx`. The engine sets a document in the Go fonts, and its math in STIX
  Two Math, when it has no TeX fonts.
- `iv` gives the engine no TeX files: no fonts, no TeX distribution, and not
  the folder of the document. A document reads no class, package, figure or
  `\input` file, and the engine skips a command that it does not know. The
  three ways to give it those files are an item in `BACKLOG.md`.
- libmagic gives `text/x-tex` to any text with commands of TeX in it, such
  as a Mermaid diagram or a Markdown file with math. A TeX mime type routes a
  file to `ivy-latex` only with a TeX extension, or with no extension, as
  `text/csv` does for `ivy-csv` (D51). libmagic gives `text/plain` to a plain
  TeX file, and its extension routes it.
- The `zuzu` decoder gives each parse the math font of zuzu, STIX Two Math,
  with `zuzu.WithMath`. Without it, zuzu draws a formula as its source, and
  reports an issue. ivy already holds the same font, from okage, so `iv`
  does not grow.
- `testdata/ivy` holds `article.tex` and `plain.tex`, and `testdata/zuzu`
  holds `flowchart-math.mmd` and `sequence-math.mmd`. `TestDecodeMath` in
  `decoder/zuzu` fails when zuzu reports an issue for either diagram.

### D65. zuzu takes the emoji font and the time when it parses. Decided.

zuzu v0.10.0 accepts `zuzu.WithEmoji` in `zuzu.Parse`, because a sequence
diagram, a pie and a mindmap lay out their labels when they compile, and
adds `zuzu.WithNow` (zuzu D38). A maintainer asked on 2026-11-26 to take
zuzu v0.10.0.

- The `zuzu` decoder gives the emoji font to the parse, and no longer to the
  render. The render uses the font of the chart. Before, a sequence diagram
  measured an emoji with no font. `testdata/zuzu/emoji-sequence.mmd` drew
  360 pixels wide then, and draws 420 pixels wide now, with more room
  between its participants.
- The decoder gives the parse the time of the run with `zuzu.WithNow`, so
  that a gantt draws the marker of today, as Mermaid does. A gantt whose
  dates do not hold today draws no marker. The same file can therefore draw
  differently on another day, and no test compares the pixels of a gantt.

### D66. A scheduled workflow keeps the Homebrew formula current. Decided. Amends D52.

Announce Release bumped the formula in the `kenshaw/homebrew-iv` tap when a
release was published, as it did for the AUR packages before D52. Nothing
tried the bump again when it failed, or when the event did not arrive. On
2026-11-25, the AUR workflow pushed the v0.24.0 bump of `iv-cli-bin` with no
release notes, because the GitHub API gave a 403 to a request with no token.
Announce Release made the same request with no token, and did not stop on a
failure. On 2026-11-27, a maintainer asked to move the Homebrew bump to a
workflow that runs on a schedule, as the AUR bump does.

- `.github/workflows/homebrew.yml` runs every 12 hours, at 00:33 and 12:33
  UTC, and can also be started by hand. It does not run on the release
  event.
- It reads the tag of the latest published release with `gh`, and the
  version in the `url` of `Formula/iv.rb` in the tap. The formula is behind
  when that version is lower than the release. A formula that is at the
  release, or past it, is left alone.
- It bumps the formula only when it is behind. If a bump fails, the next run
  tries again.
- The Homebrew and AUR workflows send the token of the workflow when they
  read the release notes from the GitHub API. A bump stops, and pushes
  nothing, when the request fails or the release has no notes.
- Announce Release keeps only the Discord announcement.

## Open questions for the maintainers

1. Windows builds under UCRT64 because mupdf needed it (D14). mupdf is gone
   (D28). Does Windows still need UCRT64? Nothing has tested MINGW64.
2. CI installs the decoder commands on every runner (D27), and the first run
   can show a libvips format or a package that a runner does not have. Each
   one is then either installed or added to `untested` with its reason.
