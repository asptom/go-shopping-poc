# Phase 6 Progress - Enforcement and Template Finalization

## Metadata

- Date: 2026-03-11
- Author/LLM: OpenCode (GPT-5.3 Codex)
- Scope: Phase 6 only (CI enforcement checks and canonical templates)
- Sequencing basis: `docs/standardization/roadmap.md`, `docs/standardization/phase-playbooks.md`, `docs/standardization/phase-5-progress.md`

## Checklist

| Item | Status | Evidence Paths |
|---|---|---|
| Confirm Phase 6 is the next allowed phase (`Phase 5 -> Phase 6`) | done | `docs/standardization/roadmap.md`, `docs/standardization/phase-playbooks.md`, `docs/standardization/phase-5-progress.md` |
| Add CI checks for `gofmt`/`goimports` compliance | done | `scripts/ci/standardization_checks.sh`, `.github/workflows/standardization-ci.yml` |
| Add CI checks for test suite pass | done | `scripts/ci/standardization_checks.sh` (`go test ./...`) |
| Add static analysis/lint baseline check | done | `scripts/ci/standardization_checks.sh` (`go vet ./...`) |
| Add architecture check for forbidden import direction (`platform -> service`) | done | `scripts/ci/standardization_checks.sh` (`rg -n '"go-shopping-poc/internal/service/' internal/platform --glob '*.go'`) |
| Add logging drift guard to prevent `With("handler")` regression | done | `scripts/ci/standardization_checks.sh` (`rg -n 'With\("handler"' ...`) |
| Publish domain service skeleton template | done | `docs/standardization/templates/domain-service-template.md` |
| Publish handler skeleton using `httpx`/`httperr` | done | `docs/standardization/templates/http-handler-template.md` |
| Publish repository skeleton with standard tx/outbox flow | done | `docs/standardization/templates/repository-tx-template.md` |
| Reference templates in standardization docs | done | `docs/standardization/README.md` |

## Files Updated in Scope

- `.github/workflows/standardization-ci.yml`
- `scripts/ci/standardization_checks.sh`
- `docs/standardization/templates/domain-service-template.md`
- `docs/standardization/templates/http-handler-template.md`
- `docs/standardization/templates/repository-tx-template.md`
- `docs/standardization/README.md`

## Machine-Verifiable Done Report

| Check ID | Command | Expected | Actual | Result |
|---|---|---|---|---|
| P6-VERIFY-001 | `scripts/ci/standardization_checks.sh` | exit code 0 | pass | PASS |
| P6-VERIFY-002 | `go test ./...` | exit code 0 | pass | PASS |
| P6-VERIFY-003 | `go list ./...` | package list emitted | package list emitted | PASS |
| P6-CHECK-ARCH-001 | `rg -n '"go-shopping-poc/internal/service/' internal/platform --glob '*.go'` | 0 matches | 0 matches | PASS |
| P6-CHECK-LOG-001 | `rg -n 'With\("handler"' internal/service internal/platform --glob '*.go'` | 0 matches | 0 matches | PASS |
| P6-CHECK-LOG-002 | `rg -n '"operation"\s*,' internal/service internal/platform --glob '*.go'` | operation field matches present | matches present across domain/platform files | PASS |

## Failed Command Notes

- No required verification commands failed.
- `goimports` was not installed in the local environment, so the script skipped local goimports verification and reported it explicitly. CI installs `goimports` before running the script to enforce the check in pipeline runs.

## Deferred / Out-of-Scope Findings

- Existing service-level `*-lint` make targets in `resources/make/services.mk` use `golangci-lint`; this Phase 6 implementation keeps a repo-wide baseline on `go vet` in CI script to avoid introducing a new global tool dependency in local runs.
