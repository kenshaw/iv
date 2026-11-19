#!/bin/bash

# screenshots.sh makes the screenshots in README.md. It builds iv, and for
# each screenshot opens a Ghostty window on Sway, runs iv in it, and captures
# the window with grim.
#
# It needs Sway, Ghostty 1.4.0 or later, grim, jq and python3, and uses
# optipng when it is installed. The windows open on a workspace of their own,
# on the focused output, and the script goes back to the workspace that was
# focused when it ends. Do not use the keyboard or the mouse until then.
#
# Usage: docs/screenshots/screenshots.sh [NAME...]
#
# With no NAME, it makes every screenshot.

set -euo pipefail

SRC=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
OUT=$SRC/docs/screenshots

# the size of each window, in pixels
WIDTH=1400
HEIGHT=880

# SHOTS holds the name of each screenshot, then the arguments of iv, which are
# relative to the root of the repository
SHOTS=(
  "photo|testdata/png/dice.png"
  "svg|testdata/okage/choropleth.svg"
  "mermaid|testdata/zuzu/flowchart.mmd"
  "markdown|testdata/blitz/sample.md"
  "docx|testdata/ivy/file-sample_100kB.docx"
  "chart|testdata/zuzu/sales.vl.json"
  "font|testdata/fontimg/Ubuntu-R.ttf"
  "wifi|WIFI:S:mynetwork;T:WPA;P:secret;;"
)

for cmd in swaymsg ghostty grim jq python3 go; do
  if ! command -v $cmd &> /dev/null; then
    echo "error: $cmd is not in \$PATH" >&2
    exit 1
  fi
done
if [ -z "${SWAYSOCK:-}" ]; then
  echo "error: Sway is not running" >&2
  exit 1
fi

TMP=$(mktemp -d)
# each window ends its command, and closes, when $TMP/close exists, so that no
# window stays open when the script ends early
export SHOT_CLOSE_ALL=$TMP/close
WORKSPACE=$(swaymsg -t get_workspaces | jq -r '.[] | select(.focused) | .name')
trap 'touch "$SHOT_CLOSE_ALL"; sleep 1; rm -rf "$TMP"; swaymsg -q workspace "$WORKSPACE"' EXIT

(set -x; go build -C "$SRC" -o "$TMP/bin/iv" .)

# wait waits until the file exists, for at most 60 seconds
wait_for() {
  for _ in $(seq 600); do
    if [ -e "$1" ]; then
      return
    fi
    sleep 0.1
  done
  echo "error: timed out waiting for $1" >&2
  return 1
}

# window prints the con_id and the rectangle of the window with the title
window() {
  swaymsg -t get_tree | jq -r --arg title "$1" \
    '.. | objects | select(.name? == $title) | "\(.id) \(.rect.x),\(.rect.y) \(.rect.width)x\(.rect.height)"'
}

# shot makes the screenshot NAME of iv with the arguments ARG...
shot() {
  local name=$1 title="iv screenshot $1" id rect
  shift
  export SHOT_READY=$TMP/$name.ready SHOT_DONE=$TMP/$name.done SHOT_CLOSE=$TMP/$name.close
  # the window waits until it has its size, then shows the command line and
  # runs iv, which fits the image to the window. It then waits for the script
  # to take the screenshot, and its command ends, which closes the window
  # with no confirmation
  PATH=$TMP/bin:$PATH ghostty \
    --gtk-single-instance=false \
    --config-default-files=false \
    --title="$title" \
    --window-decoration=none \
    --font-size=14 \
    --window-padding-x=16 \
    --window-padding-y=12 \
    --cursor-style-blink=false \
    --confirm-close-surface=false \
    --working-directory="$SRC" \
    -e bash -c '
      until [ -e "$SHOT_READY" ]; do
        [ -e "$SHOT_CLOSE_ALL" ] && exit
        sleep 0.1
      done
      clear
      # an argument with a character that the shell reads is quoted
      printf "\e[1;32m$\e[0m iv"
      for arg in "$@"; do
        if [[ "$arg" =~ ^[A-Za-z0-9_./:=-]+$ ]]; then
          printf " %s" "$arg"
        else
          printf " %s" "'\''$arg'\''"
        fi
      done
      printf "\n"
      # iv keeps back two rows of the terminal, for the file name and the
      # prompt. The command line takes a third row. After an image whose
      # height is a whole number of rows, Ghostty puts the cursor one row
      # lower, which takes a fourth. So iv gets a height of four rows less
      # than the terminal, and the command line stays on screen. iv also
      # gets a width of one column less. The command line does not show -W
      # and -H
      read -r rows cols xpixel ypixel < <(python3 -c "
import fcntl, struct, sys, termios
print(*struct.unpack(\"HHHH\", fcntl.ioctl(sys.stdin, termios.TIOCGWINSZ, bytes(8))))")
      iv -W $((xpixel * (cols - 1) / cols)) -H $((ypixel * (rows - 4) / rows)) "$@"
      status=$?
      printf "\e[1;32m$\e[0m "
      echo $status > "$SHOT_DONE"
      until [ -e "$SHOT_CLOSE" ] || [ -e "$SHOT_CLOSE_ALL" ]; do sleep 0.1; done
    ' iv "$@" &> "$TMP/$name.log" &
  for _ in $(seq 100); do
    id=$(window "$title" | cut -d' ' -f1)
    if [ -n "$id" ]; then
      break
    fi
    sleep 0.1
  done
  if [ -z "$id" ]; then
    echo "error: no window for $name" >&2
    cat "$TMP/$name.log" >&2
    return 1
  fi
  swaymsg -q "[con_id=$id] floating enable, border none, resize set $WIDTH $HEIGHT, move position center"
  # give the terminal time to take its new size
  sleep 1
  touch "$SHOT_READY"
  wait_for "$SHOT_DONE"
  # a screen that shows an error is not a screenshot
  if [ "$(cat "$SHOT_DONE")" != 0 ]; then
    echo "error: iv failed for $name" >&2
    return 1
  fi
  rect=$(window "$title" | cut -d' ' -f2-)
  # the pointer goes to the corner of the output, out of the window, which is
  # the only window of the workspace
  swaymsg -q "seat - cursor set 0 0"
  # give the terminal time to draw the image
  sleep 2
  grim -g "$rect" "$OUT/$name.png"
  touch "$SHOT_CLOSE"
  if command -v optipng &> /dev/null; then
    optipng -quiet -o2 "$OUT/$name.png"
  fi
  echo "$OUT/$name.png"
}

# no other window can cover a window of this workspace
swaymsg -q workspace "iv screenshots"

for s in "${SHOTS[@]}"; do
  name=${s%%|*}
  if [ $# -ne 0 ] && [[ ! " $* " =~ " $name " ]]; then
    continue
  fi
  shot "$name" "${s#*|}"
done
