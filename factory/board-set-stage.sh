#!/usr/bin/env bash
set -euo pipefail

JOB_ID="$1"
STAGE="$2"
ROUND="${3:-}"

BOARD="factory/board.json"
LOCK="factory/.board.lock"

mkdir -p factory
touch "$LOCK"

EXPR=".stage = \"$STAGE\""
[ -n "$ROUND" ] && EXPR="$EXPR | .round = $ROUND"

(
  flock -x 200
  jq --arg id "$JOB_ID" '(.jobs[] | select(.id == $id)) |= ('"$EXPR"')' \
    "$BOARD" > "$BOARD.tmp" && mv "$BOARD.tmp" "$BOARD"
) 200>"$LOCK"
