#!/usr/bin/env bash

# Smoke test for iv: renders every string form and test data file through the
# decoder/encoder pipeline. Pass -t to render to the terminal instead of
# discarding the output.

set -eu

# a binary built in the source directory wins, and $PATH is the fallback.
# `command -v` cannot fail the script here: under `set -e` it would take the
# whole run down before the ./iv check below ever ran
IVBIN=$(command -v iv || true)
if [ -e ./iv ]; then
  IVBIN=./iv
fi
if [ -z "$IVBIN" ]; then
  echo "error: no ./iv, and no iv in \$PATH (try: go build -o ./iv .)"
  exit 1
fi
IVBIN=$(realpath $IVBIN)

SRC=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

# extension of the files in testdata/strings, kept in step with StringExt in
# ivcmd/decode_test.go
STRING_EXT=.iv_test_string

# opens testdata/vips/file-sample_150kB.enc.pdf, kept in step with encPassword
# in ivcmd/decode_test.go
ENC_PASSWORD=password

# by default decode everything but throw the encoded output away; -t renders
# to the terminal instead
OUT=(--encoder png --out /dev/null)
if [ "${1:-}" = "-t" ]; then
  OUT=()
fi

LOG=$(mktemp)
trap 'rm -f "$LOG"' EXIT

fail=0

run() {
  printf '%-72s ' "$(echo "$1" | cut -c1-70)"
  # an encrypted pdf prompts for its password, and there is nobody here to
  # type one
  local pass=()
  case "$1" in
    *.enc.pdf) pass=(--password "$ENC_PASSWORD") ;;
  esac
  if $IVBIN -q "${OUT[@]}" "${pass[@]}" "$1" >"$LOG" 2>&1; then
    echo ok
  else
    echo "FAIL: $(head -1 "$LOG")"
    fail=1
  fi
}

# testdata/strings holds one command line argument per file -- a data: URL, a
# WIFI: code -- rather than something to open
echo '== strings =='
while read -r f; do
  run "$(cat "$f")"
done < <(find "$SRC/testdata/strings" -type f -name "*$STRING_EXT" | sort)

echo
echo '== files =='
while read -r f; do
  run "$f"
done < <(find "$SRC/testdata" -type f ! -name "*$STRING_EXT" | sort)

echo
if [ $fail -ne 0 ]; then
  echo 'one or more targets failed'
  exit 1
fi
echo 'all targets rendered'
