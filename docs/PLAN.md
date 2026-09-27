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

The state below is at `d2151b1` (2026-09-18).

- One package per format registers a decoder or an encoder from `init`, and a
  pipeline in `decoder` runs them (D10). `iv --list` names every registered
  decoder and encoder.
- `iv` selects a decoder by the mime type that libmagic sniffs from the
  content (D6, D17).
- libvips decodes and encodes the formats that no Go package covers (D5, D8).
  mupdf, through go-fitz, renders the document formats. blitz renders
  Markdown, HTML and web pages (D16). The mermaid renderer runs in process on
  wazero (D18).
- Four decoders run an external command that must be in `$PATH`. `ffmpeg`
  runs `ffmpeg` and `ffprobe`, `tag` runs the same two for the waveform of
  an audio card, `libreoffice` runs `soffice`, and `binwalk` runs `binwalk`
  (D7).
- `iv` plays an animation from a gif, a lottie file or a video, with limits
  (D19).
- CI builds and tests on Linux amd64 and arm64, macOS and Windows (D14). A
  release builds for the same platforms (D15).
- `go test ./...` passes, and `./test.sh` renders every file in `testdata/`
  (D11).

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
| [D12](#d12-a-test-skips-when-its-tool-its-library-support-or-its-file-is-missing-proposed) | A test skips when its tool, its library support or its file is missing | Proposed |
| [D13](#d13-test-data-is-checked-out-with-no-line-ending-conversion-decided) | Test data is checked out with no line ending conversion | Decided |
| [D14](#d14-ci-tests-on-linux-macos-and-windows-and-linux-runs-ubuntu-2604-decided) | CI tests on Linux, macOS and Windows, and Linux runs ubuntu-26.04 | Decided |
| [D15](#d15-a-release-builds-for-linux-macos-and-windows-decided) | A release builds for Linux, macOS and Windows | Decided |
| [D16](#d16-markdown-html-and-web-pages-render-with-xoblitz-decided) | Markdown, HTML and web pages render with xo/blitz | Decided |
| [D17](#d17-mime-detection-uses-libmagic-through-xomagic-decided-amends-d6) | Mime detection uses libmagic through xo/magic | Decided. Amends D6 |
| [D18](#d18-mermaid-renders-in-process-with-xomermaid-on-wazero-decided-amends-d7) | Mermaid renders in process with xo/mermaid on wazero | Decided. Amends D7 |
| [D19](#d19-iv-plays-one-animation-to-the-terminal-for-at-most-one-minute-decided) | iv plays one animation to the terminal, for at most one minute | Decided |
| [D20](#d20-the-go-directive-follows-the-newest-release-of-go-proposed) | The go directive follows the newest release of Go | Proposed |
| [D21](#d21-iv-is-set-up-for-coding-agents-as-every-xo-repository-is-decided) | iv is set up for coding agents as every xo repository is | Decided |

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
(2026-06-23) added `--force-mime`, and `ce5809d` treats `application/octet-stream` as a font when
the extension is a font extension. Both came from issue 5, where `mimetype`
did not detect a TTF file.

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

### D12. A test skips when its tool, its library support or its file is missing. Proposed.

The tests call `t.Skip` when an external command is not in `$PATH`, when
libmagic is not available, when a test file is missing, when the network is
not available (`blitz.ErrFetch`), and when this build of libvips cannot read
or write a format (`ErrUnsupportedFormat`). `19c8f46` gives one case: the
libvips of MSYS2 has no pdf loader. The rule holds in every test that needs
one of these. Nothing shows that Ken chose it as a rule, so ask him.

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

### D20. The go directive follows the newest release of Go. Proposed.

The `go` directive in `go.mod` went from 1.21.6 in `3b94bb5` to 1.27.1 in
`2a02ec6`, and it moved with each release of Go in the commits that update
the dependencies. The README asks for Go 1.27 or later. The pattern holds
across the whole history, and nothing states it as a rule, so ask Ken.

### D21. iv is set up for coding agents as every xo repository is. Decided.

Ken decided on 2026-09-27 that every `xo` repository is set up for coding
agents the same way (dbmeta D110), and asked for the same setup here. `iv` is
a small library in this sense, so its decisions stay in this document and are
not split into one file each (dbmeta D111).

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

## Open questions for Ken

1. D12 and D20 are Proposed. Is each one a rule?
2. dbmeta D110 names the `xo` repositories, and `iv` is
   `github.com/kenshaw/iv`. D21 records that you asked for the same setup
   here. Is that right?
3. `iv` has no `.golangci.yml`, and CI runs no linter. Do you want the
   configuration that the `xo` repositories use?
4. `go-pedantry` and the code disagree in two places. The skill wants a
   receiver of one or two letters, and `ivcmd.Args` uses `args` and
   `ivctx.Mode` uses `mode`. The skill wants no "unable to" at the start of an
   error message, and `decoder/winres` writes "unable to load". Does the
   code or the skill win? If the code wins, `AGENTS.md` needs the rule.
5. `13ec599` removed the 32-bit ARM release, and the README still has a
   section on 32-bit targets. Are 32-bit targets still supported?
6. The static Linux build in `release.yml` is commented out, and `build.sh`
   still takes `-s`. Is the static build gone for good?
7. `notes.txt` is not tracked, because `.gitignore` ignores `*.txt`. Its items
   for this repository are now in [`BACKLOG.md`](BACKLOG.md). Do you want to
   keep the file?
