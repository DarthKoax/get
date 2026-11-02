#!/usr/bin/env bash
set -euo pipefail

# Factory setup script - copies helper scripts and initializes factory structure
# Usage: bash setup-factory.sh [skill-base-dir]

SKILL_DIR="${1:-$(dirname "$(realpath "$0")")/..}"

echo "Setting up factory from: $SKILL_DIR"

# Create factory directory
mkdir -p factory/jobs

# Initialize board.json if it doesn't exist
if [ ! -f factory/board.json ]; then
  echo '{"jobs": []}' > factory/board.json
  echo "Created factory/board.json"
fi

# Copy helper scripts from skill directory
if [ -d "$SKILL_DIR/scripts" ]; then
  cp "$SKILL_DIR/scripts/"*.sh factory/
  chmod +x factory/*.sh
  echo "Copied helper scripts to factory/"
else
  echo "Warning: No scripts directory found in $SKILL_DIR"
fi

# Ensure .gitignore entries
for pat in "factory/jobs/" "factory/board.json" "factory/.board.lock"; do
  if [ -f .gitignore ]; then
    if ! grep -qF "$pat" .gitignore; then
      echo "$pat" >> .gitignore
    fi
  else
    echo "$pat" > .gitignore
  fi
done

# Inject checkout target into Makefile if not present
if [ ! -f Makefile ]; then
  cat > Makefile <<'EOF'
SHELL := /usr/bin/env bash -euo pipefail

.PHONY: checkout

## checkout: pull & checkout a factory feature branch (usage: make checkout [job=001-parent-direct-approve]); defaults to main
checkout:
	@bash factory/checkout.sh "$(job)"
EOF
  echo "Created Makefile with checkout target"
else
  if ! grep -qE '^checkout:' Makefile; then
    cat >> Makefile <<'EOF'

## checkout: pull & checkout a factory feature branch (usage: make checkout [job=001-parent-direct-approve]); defaults to main
checkout:
	@bash factory/checkout.sh "$(job)"
EOF
    echo "Added checkout target to existing Makefile"
  else
    echo "Makefile already has checkout target, skipping"
  fi
fi

echo "Factory setup complete!"
