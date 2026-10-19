# iv

`iv` is a command-line image viewer. It draws an image in the terminal with
Sixel, iTerm or Kitty graphics, through `github.com/kenshaw/rasterm`. It opens
images, vector graphics, diagrams, documents, video, audio cover art, fonts,
the icons in a Windows executable, and arguments such as a `data:` URL. It can
also write the result to a file. It is one Go module, `github.com/kenshaw/iv`,
and one command.

`iv` is for viewing. Do not add a text-mode renderer, such as ASCII output
(D1). Speed is not a goal (D2).

## Standing rules

These hold in every `xo` repository, for every coding agent (dbmeta D110,
adopted here as D21).

1. Stage changes for review. Commit and push only when Ken says so.
2. Load the `simple-english` skill before you write any text that a person
   reads: project documentation, a code comment, an error message or a commit
   message. Follow it for that text.
3. In a Go project, load the `go-pedantry` skill before you write or review Go
   code. Follow it where it does not conflict with a rule in this file. A rule
   in this file wins where the two conflict.

`CLAUDE.md` holds one line that imports this file, so that Claude Code and
every other agent read the same rules. Edit this file, not that one.

## Which document to read

| If you are | Read |
| --- | --- |
| asking why something is the way it is | the decision index in `docs/PLAN.md` |
| about to decide something that is not written down | the open questions at the end of `docs/PLAN.md`. Ask Ken |
| looking for work that is known and not done | `docs/BACKLOG.md` |
| building `iv`, or installing its C libraries | the Building section of `README.md` |
| adding or changing a decoder | the Layout section below, then D10 and a decoder of the same kind |
| adding or changing test data | `testdata/README.md`, then D11 and D13 |
| adding a decoder that runs an external command | `internal/cmd`, D7 and the "Optional tools" section of `README.md` |
| changing mime detection | `decoder/mime.go`, D6 and D17 |
| changing how an image is scaled, composited or handed to an encoder | the package comment of `ivimg`, then D22 and D23 |
| changing the libvips code | `vips/`, `decoder/vips`, `encoder/vips`, D5, D8 and D22 |
| measuring a change for speed | the Performance section of `README.md`, and D26 |
| changing animation | `ivcmd/animate.go`, `decoder/anim.go` and D19 |
| changing a workflow | D14 and D15, and the comments in `.github/workflows/` |
| writing a document, a code comment, an error message or a commit message | the `simple-english` skill. Load it first |
| writing or reviewing Go code | the `go-pedantry` skill. Load it first, then the Go conventions below |
| adding or updating an agent skill | the Agent skills section of `CONTRIBUTING.md`, and D21 |

`CONTRIBUTING.md` is the same guide for a person, and shorter. `README.md` is
for a person who uses `iv`.

A reference to a decision of another repository names that repository, as in
dbmeta D110. A bare number, as in D3, names a decision of this repository, in
`docs/PLAN.md`.

## Layout

- `main.go` runs the command with `github.com/xo/ox` (D4). It holds nothing
  else.
- `ivcmd/` holds the command: the flags, as `ox` struct tags on `Args`, and
  the code that renders each target. `animate.go` plays an animation.
- `ivctx/` holds the configuration, which travels on the context, and
  computes the size that an image is displayed at (`Scale`).
- `ivimg/` holds the steps between the decoder and the encoder (D22):
  `Prepare` scales an image and composites it over the background,
  `DecodePNG` and `DecodeJPEG` keep the bytes of a file with its image, and
  `Raw` chooses the form that each protocol gets.
- `decoder/` holds the decoder registry and the pipeline (D10). A decoder
  returns an image, a set of images, or a new input for the pipeline. The
  pipeline stops after `maxDepth` hand-offs. `mime.go` sniffs the mime type
  with libmagic (D17).
- `decoder/<name>/` holds one decoder. It registers itself from `init` with
  `decoder.Register`, or with `decoder.RegisterBuiltin` for a format in the
  `image.Decode` registry of Go. `decoder/png` is the smallest example.
- `decoder/all/` imports every decoder. A new decoder must be added there.
- `encoder/` and `encoder/<name>/` do the same for the encoders, and
  `encoder/all/` imports every encoder. `encoder/rasterm` draws to the
  terminal.
- `vips/` holds the libvips code that `decoder/vips` and `encoder/vips` share
  (D8). Its `Image` keeps a decoded image in libvips, and scales,
  composites and encodes it there (D22).
- `internal/cmd/` finds an external command in `$PATH` on first use and runs
  it (D7). `internal/font/` finds the font that the generated cards use, and
  measures text in it.
- `testdata/` holds one folder per decoder, and `testdata/README.md` lists
  them (D11). Git does not convert the line endings of these files (D13).
- `test.sh` renders every file in `testdata/` through a built `iv`.
- `build.sh` makes a release build. The release workflow calls it.

## Go conventions

These are the conventions that the code follows today.

- `iv` is cgo code (D9). It needs libvips 8.18 or later, and fontconfig and
  freetype. `README.md` names the packages for each platform.
- A package comment starts with "Package <name> supplies a ... decoder for
  iv", or the same for an encoder.
- A decoder or an encoder registers itself from `init` and never from a
  function that a test calls. `decoder.Register` panics on a name that is
  registered already.
- `context.Context` is the first parameter of a decode or an encode function.
  `ivctx` reads the configuration from it.
- A decoder for a format that a terminal can show as it is keeps the bytes
  of the file with the image, through `decoder.BytesDecoder` and
  `ivimg.DecodePNG` or `ivimg.DecodeJPEG`. A step that changes the pixels
  returns a new image that holds no bytes (D22).
- libvips decodes when the pixels are asked for, which can be after the
  pipeline closed the file. Load a libvips image from bytes, never from a
  stream of the input (D22).
- Make a resvg renderer with all of its options in `resvg.New`. From resvg
  v0.12.1 a renderer cannot change after it is made, and a change to
  `resvg.Default` is ignored. The svg decoder builds its renderer in its init
  func. `TestSerifIsLatin` fails when the font families stop taking effect.
- Do not compare two `image.Image` values with `==`. A `rasterm.PNG` holds a
  slice, and the comparison panics. An encoder that encodes the pixels itself
  calls `ivimg.Unwrap` first, because the encoders of the standard library are
  fast only for their own concrete types.
- Wrap an error with `%w`. An error message is lower case, and it starts with
  the thing that failed, as in `mime type %q: %w` or `decoder %q: not
  registered`.
- A sentinel error is a package variable made with `errors.New`, with the
  `Err` prefix, such as `decoder.ErrNotSupported`. Compare errors with
  `errors.Is`.
- If a library cannot handle a format in this build, return
  `ErrUnsupportedFormat` from `decoder` or `encoder`, so that a test can skip.
- Code that works only on some platforms uses a build constraint in the file
  name or a `//go:build` line, as `decoder/blitz/link_windows.go` and
  `ivctx/term_unix.go` do.
- A test is in the package that it tests, not in an `_test` package.
- A test fails when an external command, libmagic, a test file or a libvips
  format is missing (D27). It does not skip. The one exception is a command
  that CI does not install on an operating system, which `untested` in
  `ivcmd/decode_test.go` names with its reason. No test reaches the network.
  A test that needs a web server runs one with `httptest`.
- A comment inside a function is usually lower case, with no period at the
  end.

## Testing

Run these commands in the repository root before you stage a change. All of
them must pass, and `gofmt -l .` must print nothing:

```sh
gofmt -l .
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

CI runs `go test -race -v ./...` on Linux amd64 and arm64, macOS and Windows
(D14). It runs `golangci-lint run ./...` with the release that the Lint job
in `.github/workflows/test.yml` pins (D29). The tests need every decoder
command: `ffmpeg`, `soffice`, `ebook-convert`, `xpstopng`, `pdftoppm` and
`binwalk` (D27).

If you changed a decoder, an encoder or the pipeline, also run the smoke test:

```sh
go build -o ./iv . && ./test.sh
```

`test.sh -t` draws each file to the terminal instead of discarding it.

If you changed how an image is decoded, scaled or encoded, run the
benchmarks, once for each protocol, and compare them with the last release
(D26). The Performance section of `README.md` holds the commands.

`.golangci.yml` sets `default: all`, and disables a linter with its reason
beside it. A linter that makes idiomatic Go worse is disabled, and only a real
defect gets a code change. Ask which one it is before you change the code
(D29).

`docs_test.go` in the root holds the tests for this setup (D21). They fail on
a skill that is a link or that differs between its two copies, on a
`CLAUDE.md` that holds more than `@AGENTS.md`, on another `.md` file in the
root, and on a decision index that does not match the headings.

## Writing documentation

A new document goes in `docs/`. Only `README.md`, `AGENTS.md`, `CLAUDE.md` and
`CONTRIBUTING.md` belong in the root, and `TestTheRootHoldsFourDocuments`
enforces that. Add a new document to the table above and to the Documents
section of `README.md`.

A decision goes in `docs/PLAN.md` and nowhere else (dbmeta D111). Give it the
next number and a heading such as `### D<n>. Title. Decided.` Use `Proposed`
if Ken did not choose it. If it changes an earlier decision, write
`Amends D5` in its heading and `Amended by D<n>` in the heading of the earlier
one. Add its row to the index. `TestTheDecisionIndexIsComplete` prints the
row when it is missing, and `TestAnAmendmentPointsBothWays` fails unless both
headings name each other.

Work that is known and not done goes in `docs/BACKLOG.md`, with where it came
from. When an item is done, delete it, and record anything decided in
`docs/PLAN.md`.
