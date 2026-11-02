#!/bin/bash
# Convert server SQLite database to shortcuts.json format

set -e

DB_PATH=${1:-./data/shortcuts.db}
OUTPUT=${2:-shortcuts.json}

if [ ! -f "$DB_PATH" ]; then
    echo "Error: Database file not found: $DB_PATH"
    exit 1
fi

echo "Converting $DB_PATH to $OUTPUT..."

if command -v sqlite3 >/dev/null 2>&1; then
    sqlite3 -json "$DB_PATH" "SELECT name, url, COALESCE(search_url, '') as search_url, COALESCE(description, '') as description FROM shortcuts ORDER BY name;" | \
      jq 'if . == null or . == [] then {shortcuts: []} else {shortcuts: [.[] | {name: .name, url: .url, search_url: .search_url, description: .description}]} end' > "$OUTPUT"
else
    python3 - "$DB_PATH" "$OUTPUT" <<'PY'
import json, sqlite3, sys

db_path, output = sys.argv[1], sys.argv[2]
conn = sqlite3.connect(db_path)
conn.row_factory = sqlite3.Row
rows = conn.execute(
    "SELECT name, url, COALESCE(search_url, '') AS search_url, COALESCE(description, '') AS description FROM shortcuts ORDER BY name"
).fetchall()
data = {
    "shortcuts": [
        {
            "name": r["name"],
            "url": r["url"],
            "search_url": r["search_url"],
            "description": r["description"],
        }
        for r in rows
    ]
}
with open(output, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
conn.close()
PY
fi

echo "Exported to $OUTPUT"
echo "You can now use this file for static mode deployments."
