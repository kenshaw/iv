# Plan

This document holds the plan for `iv`, then every decision that shapes it,
then the open questions for Ken.

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

The state below is the working tree of 2026-10-15, after D22 to D33.

- One package per format registers a decoder or an encoder from `init`, and a
  pipeline in `decoder` runs them (D10). `iv --list` names every registered
  decoder and encoder.
- `iv` selects a decoder by the mime type that libmagic sniffs from the
  content (D6, D17).
- An image reaches the encoder in the form that it already has: the bytes of
  its PNG or JPEG file, which decodes its pixels only when they are needed,
  or the image that libvips holds, until a step changes its pixels (D22,
  D30). `ivimg.Prepare` scales and composites stills and animation frames
  alike.
- libvips decodes, scales and encodes the formats that no Go package covers
  (D5, D8, D22). blitz renders Markdown, HTML and web pages (D16). The
  mermaid renderer runs in process on wazero (D18). oov/psd decodes Photoshop
  files (D28).
- Seven decoders run an external command that must be in `$PATH` (D7,
  D28):
  `ffmpeg` and `tag` run `ffmpeg` and `ffprobe`, `libreoffice` runs
  `soffice`, `calibre` runs `ebook-convert`, `gxps` runs `xpstopng`, `poppler`
  runs `pdftoppm` where libvips has no pdf loader, and `binwalk` runs
  `binwalk`.
- `iv` plays an animation from a gif, a lottie file or a video, with limits
  (D19).
- CI builds, tests and lints on Linux amd64 and arm64, macOS and Windows
  (D14, D29). A release builds for the same 64-bit platforms (D15, D32).
- `go test ./...` passes, and fails when a command or a file that a test
  needs is missing (D27). `./test.sh` renders every file in `testdata/`
  (D11). The benchmarks in `ivcmd` compare the releases (D26).

## Direction

No document states a direction beyond the work in
[`BACKLOG.md`](BACKLOG.md). Ask Ken before you start a new format or a new
feature that is not in that list.

## Decisions

Each decision that shapes `iv`, in the order that it was made. The entries are
append only. A decision is never edited to change its conclusion. When a later
decision changes one, the older entry keeps its text, and its heading names
the decision that changed it.

Each heading carries a status after the title. `Decided` means that the
history shows that Ken chose it. `Proposed` means that the code follows it
and nothing shows that Ken chose it, so ask him. After the status, `Amends Dn`
or `Amended by Dn` names a decision that this one changes or that changes it.

D1 to D20 were written on 2026-09-27 from the history of the repository, from
`3b94bb5` (2024-02-02) to `d2151b1` (2026-09-18), and from the closed issues
and pull requests on GitHub. Most commit messages are short, so a reason
usually comes from a code comment, a workflow comment, the README or a comment
by Ken on GitHub. If none of these gives a reason, the entry says "Reason not
recorded".

A reference to a decision of another repository names that repository, as in
dbmeta D110. A bare number, as in D3, names a decision in this document.

| Decision | Title | Status |
| --- | --- | --- |
| [D1](#d1-iv-draws-images-with-terminal-graphics-and-never-as-text-decided) | iv draws images with terminal graphics and never as text | Decided |
| [D2](#d2-speed-is-not-a-goal-decided) | Speed is not a goal | Decided |
| [D3](#d3-svg-renders-with-xoresvg-decided) | SVG renders with xo/resvg | Decided |
| [D4](#d4-the-command-line-is-parsed-with-xoox-decided) | The command line is parsed with xo/ox | Decided |
| [D5](#d5-libvips-decodes-the-formats-that-no-go-package-covers-decided-amended-by-d8) | libvips decodes the formats that no Go package covers | Decided. Amended by D8 |
| [D6](#d6-iv-selects-a-decoder-by-the-mime-type-of-the-content-decided-amended-by-d17) | iv selects a decoder by the mime type of the content | Decided. Amended by D17 |
| [D7](#d7-an-external-command-is-found-in-path-when-a-decoder-first-uses-it-decided-amended-by-d18) | An external command is found in $PATH when a decoder first uses it | Decided. Amended by D18 |
| [D8](#d8-the-libvips-binding-is-cshumvipsgen-decided-amends-d5) | The libvips binding is cshum/vipsgen | Decided. Amends D5 |
| [D9](#d9-iv-is-cgo-code-decided) | iv is cgo code | Decided |
| [D10](#d10-decoders-and-encoders-register-themselves-and-one-pipeline-runs-them-decided) | Decoders and encoders register themselves, and one pipeline runs them | Decided |
| [D11](#d11-test-data-has-one-folder-per-decoder-and-testsh-renders-all-of-it-decided) | Test data has one folder per decoder, and test.sh renders all of it | Decided |
| [D12](#d12-a-test-skips-when-its-tool-its-library-support-or-its-file-is-missing-proposed-amended-by-d27) | A test skips when its tool, its library support or its file is missing | Proposed. Amended by D27 |
| [D13](#d13-test-data-is-checked-out-with-no-line-ending-conversion-decided) | Test data is checked out with no line ending conversion | Decided |
| [D14](#d14-ci-tests-on-linux-macos-and-windows-and-linux-runs-ubuntu-2604-decided) | CI tests on Linux, macOS and Windows, and Linux runs ubuntu-26.04 | Decided |
| [D15](#d15-a-release-builds-for-linux-macos-and-windows-decided) | A release builds for Linux, macOS and Windows | Decided |
| [D16](#d16-markdown-html-and-web-pages-render-with-xoblitz-decided) | Markdown, HTML and web pages render with xo/blitz | Decided |
| [D17](#d17-mime-detection-uses-libmagic-through-xomagic-decided-amends-d6) | Mime detection uses libmagic through xo/magic | Decided. Amends D6 |
| [D18](#d18-mermaid-renders-in-process-with-xomermaid-on-wazero-decided-amends-d7) | Mermaid renders in process with xo/mermaid on wazero | Decided. Amends D7 |
| [D19](#d19-iv-plays-one-animation-to-the-terminal-for-at-most-one-minute-decided) | iv plays one animation to the terminal, for at most one minute | Decided |
| [D20](#d20-the-go-directive-follows-the-newest-release-of-go-decided) | The go directive follows the newest release of Go | Decided |
| [D21](#d21-iv-is-set-up-for-coding-agents-as-every-xo-repository-is-decided) | iv is set up for coding agents as every xo repository is | Decided |
| [D22](#d22-an-image-reaches-the-encoder-in-the-form-that-it-already-has-decided-amended-by-d28-amended-by-d33) | An image reaches the encoder in the form that it already has | Decided. Amended by D28. Amended by D33 |
| [D23](#d23-a-jpeg-with-an-exif-orientation-a-cmyk-jpeg-and-an-animated-png-keep-no-bytes-decided) | A JPEG with an EXIF orientation, a CMYK JPEG and an animated PNG keep no bytes | Decided |
| [D24](#d24---fg-and---border-set-the-colors-and-the-border-of-a-qr-code-decided) | --fg and --border set the colors and the border of a QR code | Decided |
| [D25](#d25---mermaid-bg-sets-the-background-in-the-mermaid-renderer-decided) | --mermaid-bg sets the background in the mermaid renderer | Decided |
| [D26](#d26-benchmarks-compare-the-releases-and-the-readme-holds-the-numbers-decided) | Benchmarks compare the releases, and the README holds the numbers | Decided |
| [D27](#d27-a-test-fails-when-what-it-needs-is-missing-decided-amends-d12) | A test fails when what it needs is missing | Decided. Amends D12 |
| [D28](#d28-mupdf-is-removed-and-calibre-libgxps-oovpsd-and-poppler-take-its-formats-decided-amends-d22) | mupdf is removed, and calibre, libgxps, oov/psd and poppler take its formats | Decided. Amends D22 |
| [D29](#d29-golangci-lint-checks-the-code-in-ci-with-the-rules-of-the-xo-repositories-decided) | golangci-lint checks the code in CI, with the rules of the xo repositories | Decided |
| [D30](#d30-a-png-or-a-jpeg-decodes-its-pixels-only-when-they-are-needed-decided) | A PNG or a JPEG decodes its pixels only when they are needed | Decided |
| [D31](#d31-the-code-follows-go-pedantry-where-the-two-disagreed-decided) | The code follows go-pedantry where the two disagreed | Decided |
| [D32](#d32-iv-is-built-for-64-bit-targets-only-and-has-no-static-build-decided) | iv is built for 64-bit targets only, and has no static build | Decided |
| [D33](#d33-kitty-gets-raw-pixels-compressed-with-zlib-when-they-compress-well-decided-amends-d22) | Kitty gets raw pixels compressed with zlib when they compress well | Decided. Amends D22 |

### D1. iv draws images with terminal graphics and never as text. Decided.

The first commit, `3b94bb5` (2024-02-02), draws with `github.com/kenshaw/rasterm`,
which writes Sixel, iTerm and Kitty graphics. `encoder/rasterm` does the same
today. If the terminal has none of the three, `iv` stops with the error that
`rasterm` returns.

Ken closed pull request 4, "Added ascii support and auto width", on
2025-08-23. He wrote that `iv` is an image viewer for terminals that support
terminal graphics, and that he will not add support for terminals that do
not. In issue 3 he said that Kitty detection belongs in `rasterm`, and that
the `TERM_GRAPHICS` environment variable selects a protocol.

### D2. Speed is not a goal. Decided.

Issue 1, "iv is very slow", was closed on 2024-02-18. Ken wrote that he has no
intention to make `iv` run faster. He named two causes: Go is slow to decode
and encode images, and Sixel is slower than the other protocols.

### D3. SVG renders with xo/resvg. Decided.

`542deaf` (2024-02-02), the first release, added `github.com/xo/resvg`.
`decoder/resvg` registers it for plain and gzipped SVG. Reason not recorded.

### D4. The command line is parsed with xo/ox. Decided.

`3451594` (2024-12-04) replaced `spf13/cobra` and `spf13/pflag` with
`github.com/xo/ox`. `edb8b3d` added back the short flags that the change lost,
and `be6b7fe` (2025-05-27) moved the version to `ox.VersionString`. The flags
are the `ox` struct tags on `ivcmd.Args`, and `ivctx` registers its own types
with `ox`. Reason not recorded.

### D5. libvips decodes the formats that no Go package covers. Decided. Amended by D8.

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

### D7. An external command is found in $PATH when a decoder first uses it. Decided. Amended by D18.

`4298105` (2025-01-30) runs `soffice` for office documents. `930d808`
(2025-01-30), "Switching to ffmpeg command for videos", replaced the
`go-astiav` library with the `ffmpeg` command. `dbb6c69` (2025-08-28) ran
`mmdc` for mermaid. Each looked up its command once, on first use.

`3890a24` (2026-09-11) moved the lookup to `internal/cmd` and added `binwalk`
as the decoder of last resort. The README gives the rule for a missing
command: the file fails to render with `<command> not in path`, and is not
skipped without a message. The audio card is the exception. Without
`ffmpeg`, it is drawn with no waveform.

### D8. The libvips binding is cshum/vipsgen. Decided. Amends D5.

`ad71d8f` (2026-02-01), "Change vips package", replaced `davidbyttow/govips`
with `github.com/cshum/vipsgen`. The commit gives no reason. The README gives
one result: `vipsgen` generates its bindings against libvips 8.18, so `iv`
needs libvips 8.18 or later. `1b24400` (2026-06-23) later set the `Unlimited`
and `Memory` load options of `vipsgen`. Ken wrote on issue 5 that the new
`vipsgen` can override the memory security limits of libvips.

### D9. iv is cgo code. Decided.

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

`5abc5a3` added `RegisterBuiltin` on the same day. It marks a format that is
in the `image.Decode` registry of Go, so that a container such as a comic
archive can render it. `bc0c9bf` (2026-09-12) reworked the pipeline for
scaling, which closed issue 2, "Add resize feature". Ken wrote on issue 2 that
a Claude agent did this work.

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
libmagic is not available, when a test file is missing, when the network is
not available (`blitz.ErrFetch`), and when this build of libvips cannot read
or write a format (`ErrUnsupportedFormat`). `19c8f46` gives one case: the
libvips of MSYS2 has no pdf loader. The rule holds in every test that needs
one of these. Nothing shows that Ken chose it as a rule, so ask him.

Ken did not make it a rule. D27 replaces it: a test fails.

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
The reason is in the workflow and the README: the mupdf of go-fitz calls
`__intrinsic_setjmpex`, and only the UCRT runtime has it.

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

### D16. Markdown, HTML and web pages render with xo/blitz. Decided.

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
comment in `decoder/mime.go` gives one limit: the text analysis of libmagic
read out of bounds and stopped the process on the mermaid and dot test files.

### D18. Mermaid renders in process with xo/mermaid on wazero. Decided. Amends D7.

`dbb6c69` (2025-08-28) rendered mermaid with the `mmdc` command, and `cab5f7e`
added its default icon pack and a white background. `2a02ec6` (2026-09-13)
replaced `mmdc` with `github.com/xo/mermaid`, which runs on
`github.com/tetratelabs/wazero`. It replaced `--mermaid-icons` with
`--mermaid-theme`, and it keeps the compiled code in the user cache folder,
under `iv/wazero`. The README gives the reason: the renderer needs no browser
and no Node. It also gives the cost: the renderer is a reimplementation, so
the output is like mermaid and not the same, and icon packs do not work.

### D19. iv plays one animation to the terminal, for at most one minute. Decided.

`e3d2ea5` (2026-09-18) added `--animate` for a gif, a lottie file and a video,
and `d2151b1` changed how the frame rate is chosen. The README gives the
rules:

- `--animate` takes exactly one file, and it does not work with `--out`.
- One pass plays for at most one minute.
- A video is sampled at 12 frames per second across its length.
- Playback follows the clock and drops a frame that is late.
- `--loop` and `--fps` imply `--animate`.

An animated WebP or APNG does not play. No text gives a reason, and
[`BACKLOG.md`](BACKLOG.md) holds the item.

### D20. The go directive follows the newest release of Go. Decided.

The `go` directive in `go.mod` went from 1.21.6 in `3b94bb5` to 1.27.1 in
`2a02ec6`, and it moved with each release of Go in the commits that update
the dependencies. The README asks for Go 1.27 or later. The pattern holds
across the whole history. Ken made it a rule on 2026-10-15.

### D21. iv is set up for coding agents as every xo repository is. Decided.

Ken decided on 2026-09-27 that every `xo` repository is set up for coding
agents the same way (dbmeta D110), and asked for the same setup here. `iv` is
a small library in this sense, so its decisions stay in this document and are
not split into one file each (dbmeta D111).

On 2026-10-15 Ken said that every `kenshaw` repository follows dbmeta D110
too, and not only `iv`.

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

### D22. An image reaches the encoder in the form that it already has. Decided. Amended by D28. Amended by D33.

Ken asked on 2026-10-15 for this pipeline, after `rasterm` v0.1.18 began to
send the bytes that an image holds without an encode. The `ivimg` package
holds it, and every still and every animation frame goes through it the
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

Ken decided on 2026-10-15, and asked for the orientation to be applied. A JPEG
whose orientation is not 1 is turned the right way up when it is decoded
(`ivimg.orient`), so it shows as the camera meant, and the turned image is new
pixels with no bytes.

ffmpeg wrote its snapshots and frames with the `apng` muxer, which writes an
animated PNG of one frame. They are now plain PNGs, written with the `png`
codec and the `image2pipe` muxer, so that they keep their bytes.

### D24. --fg and --border set the colors and the border of a QR code. Decided.

`--fg` and `--border` set the colors and the quiet zone of a QR code until
`3a1e7b4` (2026-09-13), which changed the `qr` decoder to draw black on white
with the quiet zone of `go-qrcode`, and gave no reason. From then on nothing
read the two settings. The first form of this decision removed the flags, and
Ken asked on 2026-10-15 for them to be restored for QR codes.

`--fg` is the color of the modules, and black by default. `--bg` is the
background, and a transparent background, the default, draws white, because a
code with no background is hard to read on a dark terminal. `--border` is the
quiet zone in pixels, 40 by default, which is the 4 modules that the standard
asks for at 10 pixels a module. The defaults draw the same code as v0.18.1,
byte for byte. `TestDecodeColors` covers them.

### D25. --mermaid-bg sets the background in the mermaid renderer. Decided.

`--mermaid-bg` did nothing. The background was composited when the mime type
was `text/plain`, and the pipeline only saw the mime type of the svg that the
mermaid decoder hands back. The renderer draws its own background, so the
color now goes to it as the `background` theme variable. The flag has no
default, so each theme keeps its own background, and the default theme is
white as before. By hand, `--mermaid-bg red` gave a red background and
`--mermaid-theme dark` the dark background of the theme. Ken decided on
2026-10-15.

### D26. Benchmarks compare the releases, and the README holds the numbers. Decided.

Ken asked on 2026-10-15 for benchmarks of the decoding, compared with the
earlier releases, in the README. `ivcmd/bench_test.go` holds
`BenchmarkDecode` and `BenchmarkRender`. The second runs the `render` method
of the command, from the file to the terminal graphics in `io.Discard`, with
the protocol that `TERM_GRAPHICS` selects. The method has the same signature
from v0.17.0, so the same file runs against each release. The Performance
section of `README.md` holds the results and the commands that made them.

### D27. A test fails when what it needs is missing. Decided. Amends D12.

Ken decided on 2026-10-15 that a test fails, and does not skip, when an
external command, libmagic, a test file or a libvips format is missing. A
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

### D28. mupdf is removed, and calibre, libgxps, oov/psd and poppler take its formats. Decided. Amends D22.

Ken decided on 2026-10-15 to remove `github.com/gen2brain/go-fitz`, which
linked a static mupdf, from a module of 258 MB. It decoded EPUB, MOBI, XPS,
FictionBook and Photoshop files. Its libjpeg was the second libjpeg in the
binary that broke the JPEG encoder of libvips (D22): with fitz linked, a save
failed with "Wrong JPEG library version", and without it the save works. It
was also why Windows builds under UCRT64 (D14), and why 32-bit targets needed
a stub.

Each format went to the replacement that Ken chose:

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

Ken decided on 2026-10-15 to add the configuration of the xo repositories and
run it in CI. `.golangci.yml` sets `default: all`, and disables each linter
that makes idiomatic Go worse here, with its reason beside it. A real defect
got a code change: a context stored in `decoder.Sink`, a `context.Background`
in the command, an error left unchecked, and a hand-written filter that
`slices.DeleteFunc` replaces. The lint job pins golangci-lint v2.14.0, which
found 0 issues.

### D30. A PNG or a JPEG decodes its pixels only when they are needed. Decided.

Ken decided on 2026-10-15. `ivimg.DecodePNG` and `ivimg.DecodeJPEG` read the
size and the color model from the header, and decode the pixels when a step
or an encoder asks for them. A terminal that gets the bytes of the file never
decodes them: the render of `png/precision.png` for Kitty went from 20.8 ms
to 2.2 ms. A file with a good header and bad data now fails when the pixels
are asked for. The image keeps the error, and the scaler, the compositor and
the encoder entries return it.

### D31. The code follows go-pedantry where the two disagreed. Decided.

Ken decided on 2026-10-15 that the skill wins. Each receiver has one or two
letters, and an error message does not start with "unable to". A context is
passed, and never stored in a struct.

### D32. iv is built for 64-bit targets only, and has no static build. Decided.

Ken decided on 2026-10-15. `13ec599` had removed the 32-bit ARM release, and
the README still described 32-bit targets. The README section and the 32-bit
ARM path of `build.sh` are removed, and the stub of go-fitz went with D28. The
static Linux build was commented out of the release, and `-s` of `build.sh`
is removed.

### D33. Kitty gets raw pixels compressed with zlib when they compress well. Decided. Amends D22.

Ken chose the design on 2026-10-15, from measurements of each file that Kitty
gets as raw pixels in iv, and the rasterm session built it as rasterm D18 and
D19. iv requires rasterm v0.1.20. For raw pixels of over 64 KiB, the Kitty
encoder writes one zlib stream at level 1, and checks after each 64 KiB that
the output so far is under 25% of the input so far. At the first check that
fails, it sends the pixels raw. A PNG is still sent as its bytes.

The first form checked only the first 64 KiB, which is the top rows of an
image. A photo with a flat top then compressed whole, at 11 ms where raw took
1.35 ms. Ken chose on 2026-10-15 to check after each 64 KiB. In iv, a page of
markdown now goes to Kitty as 186 KB in place of 8 MB of raw RGBA, and a
photo stays raw.

rasterm v0.1.19 flushed the zlib stream at each check, and Ghostty 1.3.1 then
closed itself on `gxps/example.xps` at 1390x1966, over ssh. Each flush wrote
a stored block of 1 byte and an empty one after each block of 65535 bytes.
Python's zlib and Zig 0.16 inflate that stream, so the fault is in the
inflater of that Ghostty, likely an older Zig. The same pixels in one stream
with no flush, raw, and as a PNG all worked. rasterm v0.1.20 checks without a
flush, so the stream is one plain stream of all the pixels, and Ken confirmed
that it works in Ghostty.

## Open questions for Ken

1. Windows builds under UCRT64 because mupdf needed it (D14). mupdf is gone
   (D28). Does Windows still need UCRT64? Nothing has tested MINGW64.
2. CI installs the decoder commands on every runner (D27), and the first run
   can show a libvips format or a package that a runner does not have. Each
   one is then either installed or added to `untested` with its reason.
