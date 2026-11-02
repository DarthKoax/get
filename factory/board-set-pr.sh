#!/usr/bin/env bash
set -euo pipefail

JOB_ID="$1"
PR_URL="$2"

BOARD="factory/board.json"
LOCK="factory/.board.lock"

mkdir -p factory
touch "$LOCK"

(
  flock -x 200
  jq --arg id "$JOB_ID" --arg pr "$PR_URL" \
    '(.jobs[] | select(.id == $id)) |= (.stage = "done" | .worktree = null | .pr = $pr)' \
    "$BOARD" > "$BOARD.tmp" && mv "$BOARD.tmp" "$BOARD"
) 200>"$LOCK"
