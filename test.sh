#!/bin/bash

# Smoke test for iv: renders every string form and test data file through the
# decoder/encoder pipeline. Pass -t to render to the terminal instead of
# discarding the output.

set -eu

IVBIN=$(which iv)
if [ -e ./iv ]; then
  IVBIN=./iv
fi
IVBIN=$(realpath $IVBIN)

SRC=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

# by default decode everything but throw the encoded output away; -t renders
# to the terminal instead
OUT=(--encoder png --out /dev/null)
if [ "${1:-}" = "-t" ]; then
  OUT=()
fi

strings=(
  "WIFI:S:testssid;T:WPA;P:secret;;"
  "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxMDAiIGhlaWdodD0iMTAwIj48Y2lyY2xlIGN4PSI1MCIgY3k9IjUwIiByPSI0MCIgc3Ryb2tlPSJncmVlbiIgc3Ryb2tlLXdpZHRoPSI0IiBmaWxsPSJ5ZWxsb3ciIC8+PC9zdmc+"
  "data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A//www.w3.org/2000/svg%22%20width%3D%2264px%22%20height%3D%2264px%22%20viewBox%3D%220%200%2064%2064%22%20version%3D%221.1%22%3E%3Crect%20fill%3D%22%2350c848%22%20cx%3D%2232%22%20cy%3D%2232%22%20width%3D%2264%22%20height%3D%2264%22%20r%3D%2232%22/%3E%3Ctext%20x%3D%2250%25%22%20y%3D%2250%25%22%20fill%3D%22%23fff%22%20text-anchor%3D%22middle%22%20font-size%3D%2228%22%20dy%3D%22.1em%22%3EKS%3C/text%3E%3C/svg%3E"
  "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
)

LOG=$(mktemp)
trap 'rm -f "$LOG"' EXIT

fail=0

run() {
  printf '%-72s ' "$(echo "$1" | cut -c1-70)"
  if $IVBIN -q "${OUT[@]}" "$1" >"$LOG" 2>&1; then
    echo ok
  else
    echo "FAIL: $(head -1 "$LOG")"
    fail=1
  fi
}

echo '== strings =='
for s in "${strings[@]}"; do
  run "$s"
done

echo
echo '== files =='
while read -r f; do
  run "$f"
done < <(find "$SRC/testdata" -type f ! -name '*.enc.pdf' | sort)

echo
if [ $fail -ne 0 ]; then
  echo 'one or more targets failed'
  exit 1
fi
echo 'all targets rendered'
