# Phase 0 Backlog Template

Use this template to capture the baseline audit outputs for Phase 0.

## Metadata

- Date:
- Author/LLM:
- Commit/Branch:
- Scope reviewed:

## 1) Service Inventory

List each domain service and summarize key findings.

| Service | Handler Duplication | Error Handling Drift | Logging Drift | Priority |
|---|---|---|---|---|
| cart |  |  |  | high/med/low |
| customer |  |  |  | high/med/low |
| order |  |  |  | high/med/low |
| product |  |  |  | high/med/low |
| eventreader |  |  |  | high/med/low |

## 2) Duplication Hotspots

Capture repeated patterns suitable for promotion to platform packages.

| Pattern | Current Locations | Candidate Platform Package | Reuse Count | Notes |
|---|---|---|---|---|
| JSON decode helper |  | `internal/platform/httpx` |  |  |
| JSON write helper |  | `internal/platform/httpx` |  |  |
| path param checks |  | `internal/platform/httpx` |  |  |
| HTTP error response mapping mechanics |  | `internal/platform/httperr` |  |  |

## 3) Architecture Boundary Audit

Document imports that violate or risk violating clean architecture.

### Forbidden Patterns Checked

- [ ] `internal/platform/*` importing `internal/service/*`
- [ ] cross-domain service imports (`internal/service/x` importing `internal/service/y`)
- [ ] domain logic embedded in platform packages

### Findings

| File | Violation Type | Severity | Proposed Fix |
|---|---|---|---|
|  |  | high/med/low |  |

## 4) Error Handling Baseline

Measure current consistency and gaps.

| Check | Count | Notes |
|---|---:|---|
| uses of `errors.Is` |  |  |
| direct `== Err...` checks |  |  |
| string matching on `err.Error()` |  |  |
| missing `%w` wrapping opportunities |  |  |

### Sentinel Error Coverage

| Domain Package | Sentinel Errors Present | Missing Domain Sentinels |
|---|---|---|
| cart |  |  |
| customer |  |  |
| order |  |  |
| product |  |  |

## 5) Logging Baseline

Assess consistency against `logging-standard.md`.

| Service | Component Naming Consistent | Structured Fields Consistent | Sensitive Data Risk | Top Fixes |
|---|---|---|---|---|
| cart | yes/no | yes/no | high/med/low |  |
| customer | yes/no | yes/no | high/med/low |  |
| order | yes/no | yes/no | high/med/low |  |
| product | yes/no | yes/no | high/med/low |  |

### Field Key Drift

| Concept | Variants Found | Preferred Key |
|---|---|---|
| cart id |  | `cart_id` |
| request id |  | `request_id` |
| event type |  | `event_type` |
| component |  | `component` |

## 6) Candidate Standardization Backlog

Create implementation-ready work items.

| ID | Title | Phase | Scope | Risk | Dependencies | Acceptance Criteria |
|---|---|---|---|---|---|---|
| STD-001 | Introduce `httpx` decode/write helpers | 1/2 | platform + pilot service | med | none | helper package added + pilot migrated |
| STD-002 | Remove `err.Error()` string matching | 3 | domain handlers | low | STD-001 | all handlers use sentinel + `errors.Is` |
| STD-003 | Standardize component logger names | 5 | all services | low | none | all services use stable `snake_case` components |

## 7) Prioritized Migration Order

Record service migration order and rationale.

1. Service:
   - Rationale:
   - Risk notes:
2. Service:
   - Rationale:
   - Risk notes:
3. Service:
   - Rationale:
   - Risk notes:

## 8) Verification Baseline

- `go test ./...` result:
- Known flaky tests:
- Known environment dependencies:

## 9) Phase 0 Exit Checklist

- [ ] Inventory complete across all target services.
- [ ] Boundary matrix documented and reviewed.
- [ ] Logging baseline recorded with concrete fixes.
- [ ] Backlog items are phase-tagged and dependency-tagged.
- [ ] Migration order is explicit.

## 10) Handoff Notes For Next LLM

- Highest-risk areas:
- Recommended first implementation task:
- Constraints to preserve (API behavior, error messages, etc.):
