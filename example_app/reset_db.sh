#!/bin/bash
set -euo pipefail

DB_NAME=${DB_NAME:-example}
DB_USER=${DB_USER:-app_user}
DB_HOST=${DB_HOST:-0.0.0.0}
DB_PORT=${DB_PORT:-5432}
DB_ADMIN_USER=${DB_ADMIN_USER:-postgres}
DB_ADMIN_PASS=${DB_ADMIN_PASS:-postgres}

# Optional: Export PGPASSWORD for non-interactive psql
export PGPASSWORD="$DB_ADMIN_PASS"

echo "🔪 Terminating all connections to $DB_NAME..."
psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -c "
  SELECT pg_terminate_backend(pid)
  FROM pg_stat_activity
  WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();
"

echo "🔄 Dropping database: $DB_NAME"
psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;"

echo "🆕 Creating database: $DB_NAME"
psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

echo "✅ Database reset complete."
