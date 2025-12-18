#!/usr/bin/env bash
# scripts/Makefile/export_env.sh
set -euo pipefail

(
  set -a
  [ -f config/.env ] && source config/.env
  [ -f config/.env.local ] && source config/.env.local
  env
) | sed 's/^/export /'

