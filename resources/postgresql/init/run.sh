#!/usr/bin/env bash
set -euo pipefail

export PGPASSWORD="$POSTGRES_PASSWORD"

psql --host "$POSTGRES_HOST" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -p 5432 \
  -v DB_NAME="$db_name" \
  -v APP_USER="$username" \
  -v APP_PASSWORD="$password" \
  -v RO_USER="$ro_username" \
  -v RO_PASSWORD="$ro_password" \
  -f /sql/create-db.sql