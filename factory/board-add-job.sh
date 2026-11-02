#!/usr/bin/env bash
set -euo pipefail

JOB_ID="$1"
BRANCH="$2"
WORKTREE="$3"

BOARD="factory/board.json"
LOCK="factory/.board.lock"

mkdir -p factory
touch "$LOCK"

(
  flock -x 200
  jq --arg id "$JOB_ID" --arg br "$BRANCH" --arg wt "$WORKTREE" \
    '.jobs += [{"id":$id,"branch":$br,"worktree":$wt,"stage":"spec","round":0,"reviews":{}}]' \
    "$BOARD" > "$BOARD.tmp" && mv "$BOARD.tmp" "$BOARD"
) 200>"$LOCK"
