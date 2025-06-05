#!/bin/bash

# Usage:
#   ./drop_dbs_by_pattern.sh api_runner
#   ./drop_dbs_by_pattern.sh api_runner --force

export PGPASSWORD=postgres

set -e

PATTERN="$1"
FORCE="$2"

if [[ -z "$PATTERN" ]]; then
  echo "❌ Usage: $0 <pattern> [--force]"
  exit 1
fi

echo "🔍 Looking for databases matching pattern: '$PATTERN'"

DBS=$(psql -U postgres -d postgres -h 0.0.0.0 -t -A -c "SELECT datname FROM pg_database WHERE datname LIKE '${PATTERN}%';")

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
  echo "🛑 Terminating sessions for $db..."
  psql -U postgres -d postgres -h 0.0.0.0 -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$db' AND pid <> pg_backend_pid();"
  
  echo "🗑️ Dropping $db..."
  psql -U postgres -d postgres -h 0.0.0.0 -c "DROP DATABASE IF EXISTS \"$db\";"
done

echo "✅ All matching databases dropped."
