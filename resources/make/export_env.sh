#!/usr/bin/env bash
set -euo pipefail
(
  set -a
  [ -f .env ] && source .env
  [ -f .env.local ] && source .env.local
  env
) | grep -v '^=' | awk '{pos = index($0, "="); print substr($0, 1, pos-1) " := " substr($0, pos+1)}'