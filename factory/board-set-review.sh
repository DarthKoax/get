#!/usr/bin/env bash
set -euo pipefail

JOB_ID="$1"
REVIEWER="$2"
VERDICT="$3"

BOARD="factory/board.json"
LOCK="factory/.board.lock"

mkdir -p factory
touch "$LOCK"

(
  flock -x 200
  jq --arg id "$JOB_ID" --arg r "$REVIEWER" --arg v "$VERDICT" \
    '(.jobs[] | select(.id == $id)).reviews[$r] = $v' \
    "$BOARD" > "$BOARD.tmp" && mv "$BOARD.tmp" "$BOARD"
) 200>"$LOCK"
