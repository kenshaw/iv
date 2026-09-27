# Contributing to iv

Read three documents before you change anything.

[`AGENTS.md`](AGENTS.md) holds the rules: the standing rules, the layout, the
Go conventions and the commands to run. It is written for an AI coding agent,
and everything in it applies to a person too. `CLAUDE.md` holds one line that
imports it, for Claude Code.

[`docs/PLAN.md`](docs/PLAN.md) holds the plan and every decision this project
made. The index at the top lists each decision with its status. Read the
status, because some decisions amend an earlier one. The open questions for
Ken are at the end.

[`docs/BACKLOG.md`](docs/BACKLOG.md) holds the work that is known and not
done.

Do not decide an open question on your own. Ask Ken.

## Build dependencies

`iv` is cgo code, and a Go toolchain on its own cannot build it. The
[Building](README.md#building) section of `README.md` names the libraries for
each platform.

## Before you send a change

Run these commands in the repository root. All of them must pass, and
`gofmt -l .` must print nothing:

```sh
gofmt -l .
go build ./...
go vet ./...
go test ./...
```

If you changed a decoder, an encoder or the pipeline, also run the smoke test.
It renders every file in `testdata/` through the command:

```sh
go build -o ./iv . && ./test.sh
```

## Agent skills

The repository carries two agent skills. A skill is a set of instructions
that a coding agent loads for a task. `simple-english` sets how prose is
written, and `go-pedantry` sets how Go is written.

`skills-lock.json` names the source of each skill. The `skills` command from
npm writes that file, and version 1.7.0 is the one measured. It writes each
skill into two folders. Codex and the other agents read
`.agents/skills/<name>`, and Claude Code reads `.claude/skills/<name>`.

To install or update a skill, run its command in the repository root:

```sh
npx skills@1.7.0 add AminBlg/SimpleEnglish --skill simple-english --agent codex claude-code --copy -y
npx skills@1.7.0 add oborchers/fractional-cto --skill go-pedantry --agent codex claude-code --copy -y
```

Keep `--copy`. Without it, the command writes `.claude/skills/<name>` as a
symbolic link. A Windows checkout writes a symbolic link as a text file, and
Claude Code then loads no skill and reports nothing. `TestSkillsAreCopies`
fails on a link, and it fails when the two folders differ. See D21.

`.claude/settings.local.json` holds the Claude Code permissions of one
person. The root `.gitignore` ignores it, and it ignores everything else in
`.claude/` except the skills.
