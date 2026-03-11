# Phase 4 Progress - Repository and Transaction Consistency

## Metadata

- Date: 2026-03-11
- Author/LLM: OpenCode (GPT-5.3 Codex)
- Scope: Phase 4 only (`internal/service/*` repository and transaction consistency)
- Sequencing basis: `docs/standardization/roadmap.md`, `docs/standardization/phase-playbooks.md`, `docs/standardization/phase-3-progress.md`

## Checklist

| Item | Status | Evidence Paths |
|---|---|---|
| Confirm next executable phase and entry criteria (`Phase 3 -> Phase 4`) | done | `docs/standardization/roadmap.md`, `docs/standardization/phase-playbooks.md`, `docs/standardization/phase-3-progress.md` |
| Define repository conventions (interface/sentinel/error-wrapping expectations) | done | `docs/standardization/repository-transaction-standard.md` |
| Normalize repository error wrapping semantics (`%w` for chained errors) | done | Updated repository files under `internal/service/{cart,customer,order,product}/repository*.go`; verification command shows zero `%v` wraps in `fmt.Errorf` for repository files |
| Standardize transaction orchestration pattern | done | Existing `BeginTx -> rollback guard -> commit` pattern documented and validated via references in `docs/standardization/repository-transaction-standard.md` |
| Standardize outbox in-transaction write pattern | done | `docs/standardization/repository-transaction-standard.md`; callsites verified by `rg -n 'outboxWriter\.WriteEvent\(ctx, tx' internal/service/*/repository*.go` |
| Audit read-modify-write race risks and define mitigation | done | Risk findings + mitigation pattern documented in `docs/standardization/repository-transaction-standard.md` |
| Create canonical examples in one service package | done | `internal/service/cart/repository_tx_patterns.md` |

## Files Updated in Scope

- `internal/service/cart/repository_checkout.go`
- `internal/service/cart/repository_contact.go`
- `internal/service/cart/repository_crud.go`
- `internal/service/cart/repository_items.go`
- `internal/service/cart/repository_payment.go`
- `internal/service/customer/repository_crud.go`
- `internal/service/customer/repository_query.go`
- `internal/service/customer/repository_util.go`
- `internal/service/order/repository_crud.go`
- `internal/service/product/repository_bulk.go`
- `internal/service/product/repository_crud.go`
- `internal/service/product/repository_image.go`
- `internal/service/product/repository_query.go`
- `internal/service/product/repository_util.go`
- `docs/standardization/repository-transaction-standard.md`
- `internal/service/cart/repository_tx_patterns.md`

## Machine-Verifiable Done Report

| Check ID | Command | Expected | Actual | Result |
|---|---|---|---|---|
| P4-ENTRY-001 | `go test ./internal/platform/...` | exit code 0 | pass | PASS |
| P4-ENTRY-002 | `go test ./...` | exit code 0 | pass | PASS |
| P4-ENTRY-003 | `go list ./...` | package list emitted | package list emitted | PASS |
| P4-VERIFY-001 | `go test ./internal/service/...` | exit code 0 | pass | PASS |
| P4-VERIFY-002 | `go test ./...` | exit code 0 | pass | PASS |
| P4-VERIFY-003 | `go list ./...` | package list emitted | package list emitted | PASS |
| P4-VERIFY-004 | `go test ./internal/platform/...` | exit code 0 | pass | PASS |
| P4-VERIFY-005 | `rg -n '"go-shopping-poc/internal/service/' internal/platform --glob '*.go'` | 0 matches | 0 matches | PASS |
| P4-CHECK-001 | `rg -n 'fmt\.Errorf\("[^"]*%v' internal/service/*/repository*.go` | 0 matches | 0 matches | PASS |
| P4-CHECK-002 | `rg -n 'BeginTx\(ctx, nil\)' internal/service/*/repository*.go` | transaction entrypoints listed | entrypoints listed across cart/customer/order/product | PASS |
| P4-CHECK-003 | `rg -n 'outboxWriter\.WriteEvent\(ctx, tx' internal/service/*/repository*.go` | in-transaction outbox writes listed | listed across cart/customer/order/product | PASS |

## Failed Command Notes

- Initial Phase 4 verification attempt failed:
  - Command: `go test ./internal/service/...`
  - Root cause: automated `%v -> %w` replacement changed `fmt.Sprintf("%v", err)` to invalid `fmt.Sprintf("%w", err)` in `internal/service/product/repository_util.go:31`.
  - Fix: restored `%v` at `internal/service/product/repository_util.go:31`, then re-ran verification successfully.

## Deferred / Out-of-Scope Findings

- Concurrency mitigation implementation (for example `SELECT ... FOR UPDATE` or CAS updates) is documented but not behavior-changed in this phase to preserve runtime semantics.
- Logging-key normalization remains Phase 5 scope.
