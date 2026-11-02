#!/usr/bin/env bash
set -euo pipefail

JOB_ID="${1:-}"
BOARD="factory/board.json"

if [ ! -f "$BOARD" ]; then
  echo "error: factory/board.json not found" >&2
  exit 1
fi

checkout_branch() {
  local branch="$1"
  git fetch origin "$branch" && git checkout "$branch"
}

if [ -z "$JOB_ID" ]; then
  if ! command -v fzf >/dev/null 2>&1; then
    echo "no job specified, checking out main"
    checkout_branch "main"
    exit 0
  fi

  selection=$(
    {
      echo "main | main | default"
      jq -r '.jobs[] | "\(.id) | \(.branch) | \(.stage)"' "$BOARD"
    } | fzf --header="Select a branch or job"
  ) || true

  if [ -z "$selection" ]; then
    echo "no selection, checking out main"
    checkout_branch "main"
    exit 0
  fi

  id=$(echo "$selection" | cut -d'|' -f1 | tr -d ' ')
  if [ "$id" = "main" ]; then
    checkout_branch "main"
    exit 0
  fi
  JOB_ID="$id"
fi

branch=$(jq -r --arg id "$JOB_ID" '.jobs[] | select(.id == $id) | .branch' "$BOARD")
if [ -z "$branch" ] || [ "$branch" = "null" ]; then
  echo "error: job '$JOB_ID' not found in factory/board.json" >&2
  exit 1
fi

echo "checking out $branch (job: $JOB_ID)"
checkout_branch "$branch"
