#!/usr/bin/env bash

set -euo pipefail

echo "[phase-6] checking gofmt compliance"
UNFORMATTED=""
for f in $(git ls-files '*.go'); do
  result=$(gofmt -l "$f")
  if [[ -n "$result" ]]; then
    UNFORMATTED="${UNFORMATTED:+${UNFORMATTED}
}${result}"
  fi
done
if [[ -n "${UNFORMATTED}" ]]; then
  echo "gofmt check failed; run gofmt on:"
  printf '%s\n' "${UNFORMATTED}"
  exit 1
fi

if command -v goimports >/dev/null 2>&1; then
  echo "[phase-6] checking goimports compliance"
  UNIMPORTS=""
  for f in $(git ls-files '*.go'); do
    result=$(goimports -l "$f")
    if [[ -n "$result" ]]; then
      UNIMPORTS="${UNIMPORTS:+${UNIMPORTS}
}${result}"
    fi
  done
  if [[ -n "${UNIMPORTS}" ]]; then
    echo "goimports check failed; run goimports on:"
    printf '%s\n' "${UNIMPORTS}"
    exit 1
  fi
else
  echo "[phase-6] goimports not installed; skipping goimports check"
fi

echo "[phase-6] running go vet"
go vet ./...

echo "[phase-6] running tests"
go test ./...

echo "[phase-6] verifying package graph"
go list ./... >/dev/null

echo "[phase-6] checking architecture boundary: internal/platform must not import internal/service"
if rg -n '"go-shopping-poc/internal/service/' internal/platform --glob '*.go'; then
  echo "architecture check failed"
  exit 1
fi

echo "[phase-6] checking logging drift: With(\"handler\") is forbidden"
if rg -n 'With\("handler"' internal/service internal/platform --glob '*.go'; then
  echo "logging drift check failed: handler key found"
  exit 1
fi

echo "[phase-6] checking logging drift: operation key should be present"
if ! rg -n '"operation"\s*,' internal/service internal/platform --glob '*.go' >/dev/null; then
  echo "logging coverage check failed: operation key not found"
  exit 1
fi

echo "[phase-6] checking sensitive logger keys"
if rg -n 'logger\.(Debug|Info|Warn|Error)\(.*"(email|card_number|card_cvv|token|secret)"\s*,' internal/service internal/platform --glob '*.go'; then
  echo "sensitive logging check failed"
  exit 1
fi

echo "[phase-6] all checks passed"
