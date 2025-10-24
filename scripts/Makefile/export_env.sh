#!/usr/bin/env bash
# scripts/Makefile/export_env.sh
set -euo pipefail

(
  set -a
  [ -f .env ] && source .env
  [ -f .env.local ] && source .env.local
  env
) | sed 's/^/export /'

