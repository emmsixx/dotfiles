#!/usr/bin/env bash
set -euo pipefail

source "$HOME/.secrets"

API_URL="https://phobos.m6.rs/api/upload"
AUTH_HEADER="authorization: $PHOBOS_TOKEN"

# Output directory and filename
OUT_DIR="$HOME/Pictures/Screenshots"
mkdir -p "$OUT_DIR"
OUT_FILE="$OUT_DIR/satty-$(date '+%Y%m%d-%H%M%S').png"

# Capture raw screenshot (still needs a temp file for satty input)
RAW_IMG="$(mktemp --suffix=.png)"

grim -g "$(slurp)" "$RAW_IMG" || {
  exit 1
}

if ! satty --filename "$RAW_IMG" --fullscreen --output-filename "$OUT_FILE"; then
  rm -f "$RAW_IMG"
  exit 1
fi

rm -f "$RAW_IMG" # cleanup raw temp

# If no output file, user used copy button — image already in clipboard
if [[ ! -s "$OUT_FILE" ]]; then
  exit 0
fi

MIME_TYPE="$(file --mime-type -b "$OUT_FILE")"

URL="$(
  curl \
    -sS \
    -H "$AUTH_HEADER" \
    -H 'content-type: multipart/form-data' \
    -H 'x-zipline-domain: m6.pics' \
    -F "file=@${OUT_FILE}" \
    "$API_URL" \
  | jq -r '.files[0].url'
)" || {
  exit 1
}

if [[ -z "$URL" || "$URL" == "null" ]]; then
  echo "Upload failed or response missing URL" >&2
  exit 1
fi

printf "%s" "$URL" | wl-copy
notify-send --app-name="phobos.m6.rs" --urgency=normal \
  "Upload Complete" "URL copied to clipboard."
echo "Copied to clipboard: $URL"
