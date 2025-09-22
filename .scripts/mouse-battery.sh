#!/bin/bash
set -euo pipefail

ICON="󰍽"
LOW=20
MED=50

fmt() { printf '{"text":"%s","class":"%s"}\n' "$1" "$2"; }

out="$(timeout 3s solaar show 2>/dev/null || true)"
[ -n "$out" ] || { fmt "" "mouse-notfound"; exit 0; }

block="$(printf '%s\n' "$out" \
  | awk '
    /^[^[:space:]]/ { if (blk) print blk; blk=""; }  # new top-level block
    { blk = blk $0 "\n" }
    END { if (blk) print blk }
  ' \
  | awk 'BEGIN{RS="";FS="\n"} /Kind[[:space:]]*:[[:space:]]*mouse/{print; exit}')"

[ -n "$block" ] || block="$out"

line="$(printf '%s\n' "$block" | awk '/^\s*Battery:/ {print; exit}')"
[ -n "$line" ] || { fmt "" "mouse-notfound"; exit 0; }

# Parse percentage (first number only) and state
perc="$(printf '%s' "$line" | sed -n 's/.*Battery:[[:space:]]*\([0-9]\+\).*/\1/p')"
state="$(printf '%s' "$line" | grep -oEi '(discharging|charging|full|good|low)' | head -n1 | tr '[:upper:]' '[:lower:]' || true)"

# Build output
if [ -n "$perc" ]; then
  if ! printf '%s' "$perc" | grep -qE '^[0-9]+$'; then
    fmt "" "mouse-unknown"; exit 0
  fi
  text="${perc}% ${ICON}"
  case "$state" in
    discharging|"")
      class="mouse-discharging"
      if [ "$perc" -le "$LOW" ]; then
        class="$class mouse-low"
      elif [ "$perc" -le "$MED" ]; then
        class="$class mouse-medium"
      else
        class="$class mouse-high"
      fi
      ;;
    charging) class="mouse-charging" ;;
    full|good) class="mouse-full" ;;
    low) class="mouse-discharging mouse-low" ;;
    *) class="mouse-unknown" ;;
  esac
  fmt "$text" "$class"; exit 0
else
  case "$state" in
    full|good)
      fmt "100% ${ICON}" "mouse-full"; exit 0
      ;;
    *)
      fmt " ${ICON}" "mouse-notfound"; exit 0
      ;;
  esac
fi
