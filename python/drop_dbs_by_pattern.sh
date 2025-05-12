#!/bin/bash

# Usage:
#   ./drop_dbs_by_pattern.sh api_runner
#   ./drop_dbs_by_pattern.sh api_runner --force

set -e

PATTERN="$1"
FORCE="$2"

if [[ -z "$PATTERN" ]]; then
  echo "❌ Usage: $0 <pattern> [--force]"
  exit 1
fi

echo "🔍 Looking for databases matching pattern: '$PATTERN'"

DBS=$(psql -U postgres -d postgres -t -A -c "SELECT datname FROM pg_database WHERE datname LIKE '${PATTERN}%';")

if [[ -z "$DBS" ]]; then
  echo "✅ No matching databases found."
  exit 0
fi

echo "⚠️ Found databases:"
echo "$DBS"
echo

if [[ "$FORCE" != "--force" ]]; then
  read -p "❓ Drop these databases? [y/N] " confirm
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "❌ Cancelled."
    exit 1
  fi
fi

for db in $DBS; do
  echo "🗑️ Dropping $db..."
  psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS \"$db\";"
done

echo "✅ All matching databases dropped."
