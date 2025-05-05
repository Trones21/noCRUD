#!/bin/bash
set -euo pipefail

DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASS=$DB_PASS
DB_HOST=${DB_HOST:-0.0.0.0}
DB_PORT=${DB_PORT:-5432}
DB_ADMIN_USER=${DB_ADMIN_USER:-postgres}
DB_ADMIN_PASS=${DB_ADMIN_PASS:-postgres}

# Optional: Export PGPASSWORD for non-interactive psql
export PGPASSWORD="$DB_ADMIN_PASS"

echo "🔍 Checking Postgres availability..."
until psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -c '\q' 2>/dev/null; do
  echo "⏳ Waiting for Postgres to be ready..."
  sleep 1
done

echo "🔪 Terminating all connections to $DB_NAME..."
psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -c "
  SELECT pg_terminate_backend(pid)
  FROM pg_stat_activity
  WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();
"

echo "🔄 Dropping database: $DB_NAME"
psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;"


if psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -tc "SELECT 1 FROM pg_roles WHERE rolname = '$DB_USER'" | grep -q 1; then
    echo "👤 Role '$DB_USER' already exists. Skipping."
else
    echo "👤 Creating user: $DB_USER"
    psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASS';"
fi


echo "🆕 Creating database: $DB_NAME"
psql -h "$DB_HOST" -U "$DB_ADMIN_USER" -p "$DB_PORT" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

echo "✅ Database reset complete."
