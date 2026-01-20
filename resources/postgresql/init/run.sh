#!/usr/bin/env bash
set -euo pipefail

export PGPASSWORD="$POSTGRES_PASSWORD"

psql --host "$POSTGRES_HOST" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -p 5432 \
  -v DB_NAME="$DB_NAME" \
  -v APP_USER="$USERNAME" \
  -v APP_PASSWORD="$PASSWORD" \
  -v RO_USER="$RO_USERNAME" \
  -v RO_PASSWORD="$RO_PASSWORD" \
  -f /sql/create-db.sql