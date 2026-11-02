#!/usr/bin/env bash
set -euo pipefail

JOB_ID="$1"
JQ_EXPR="$2"
BOARD="factory/board.json"
LOCK="factory/.board.lock"

mkdir -p factory
touch "$LOCK"

(
  flock -x 200
  # Ensure the job exists
  jq --arg id "$JOB_ID" 'if (.jobs | map(.id) | index($id)) == null then .jobs += [{"id": $id}] else . end' "$BOARD" > "$BOARD.tmp" && mv "$BOARD.tmp" "$BOARD"
  # Apply caller expression scoped to the job
  jq --arg id "$JOB_ID" '(.jobs[] | select(.id == $id)) |= ('"$JQ_EXPR"')' "$BOARD" > "$BOARD.tmp" && mv "$BOARD.tmp" "$BOARD"
) 200>"$LOCK"
