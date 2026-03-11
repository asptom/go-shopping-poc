# Phase 0 Baseline and Guardrails

## Metadata

- Date: 2026-03-11
- Author/LLM: OpenCode (GPT-5.3 Codex)
- Commit/Branch: `4d66678` on `gpt_refactor`
- Scope reviewed: `internal/service/{cart,customer,order,product,eventreader}`, `internal/platform/*`, `docs/standardization/*`

## 1) Service Inventory

| Service | Handler Duplication | Error Handling Drift | Logging Drift | Priority |
|---|---|---|---|---|
| cart | Local helper stack (`decodeJSON`, `writeJSON`, `requiredPathParam`) concentrated in one file (`internal/service/cart/handlers.go`), but pattern still duplicates transport mechanics present in other services; 7 decode and 6 encode callsites in this service. | Uses `errors.Is` consistently in handlers/services (21 matches), but repository layer has many `%v` wraps with `err` (63 potential `%w` opportunities). | Uses `component` in handlers/services (`internal/service/cart/handlers.go`, `internal/service/cart/service.go`) but no `operation`/`request_id`/`duration_ms` keys (0 matches each). | med |
| customer | Inline JSON decode/encode + path param checks repeated in one large handler file: 7 decode, 5 encode, 17 `chi.URLParam` callsites in `internal/service/customer/handler.go`. | No `errors.Is` usage (0), no sentinel comparison in handler path (0 direct `== Err...`), and 22 `%v` wrap opportunities in repository paths (for example `internal/service/customer/repository_crud.go`). | Service/repository logging exists, but HTTP handlers do not log boundary events; one sensitive-field log at `internal/service/customer/service.go:180` logs raw `email`. | high |
| order | Inline transport mechanics in `internal/service/order/handler.go` (1 decode, 2 encode, 4 URL param checks). | Mixed style: `errors.Is` used in repository/service (6), but handlers use direct sentinel equality 5 times (`internal/service/order/handler.go:29`, `internal/service/order/handler.go:73`, `internal/service/order/handler.go:77`, `internal/service/order/handler.go:108`, `internal/service/order/handler.go:112`); 26 `%v` wrap opportunities. | Uses `component` logger in service/repository (`internal/service/order/service.go`, `internal/service/order/repository.go`), but missing standard keys `operation`/`request_id`/`duration_ms` (0 each). | high |
| product | Inline response encoding and URL param validation duplicated across endpoints in `internal/service/product/handler.go` (8 encode, 7 URL param checks). | Contains 1 explicit string compare on `err.Error()` at `internal/service/product/repository_query.go:396`; 60 `%v` wrap opportunities in repository layer. | Rich log volume (70 callsites) and `component` usage exists, but no `operation`/`request_id`/`duration_ms` keys (0 each); some message text is handler-name centric (for example `GetDirectImage...`) rather than stable verb-object style in `internal/service/product/handler.go`. | high |
| eventreader | No HTTP handler duplication (0 decode/encode/param callsites in service path). | No sentinel matching or `errors.Is` usage (0/0); errors mostly flow through platform utilities and event handlers. | Event handlers use `With("handler", ...)` instead of standard `component` at `internal/service/eventreader/eventhandlers/on_product_created.go:24` and `internal/service/eventreader/eventhandlers/on_customer_created.go:24`; no `operation`/`request_id` keys (0 each). | med |

## 2) Duplication Hotspots

| Pattern | Current Locations | Candidate Platform Package | Reuse Count | Notes |
|---|---|---|---|---|
| JSON decode helper | `internal/service/cart/handlers.go` (`decodeJSON`), plus inline decoder calls in `internal/service/customer/handler.go` and `internal/service/order/handler.go` | `internal/platform/httpx` | 15 callsites across 3 services | Meets promote criteria: reused by >=2 services, domain-agnostic, stable API likely. |
| JSON write helper | `internal/service/cart/handlers.go` (`writeJSON`), plus inline `json.NewEncoder(w).Encode` in `internal/service/customer/handler.go`, `internal/service/order/handler.go`, `internal/service/product/handler.go` | `internal/platform/httpx` | 21 callsites across 4 services | Strongest low-risk extraction candidate for Phase 1/2. |
| Path param checks | Local helper in `internal/service/cart/handlers.go` and repeated `chi.URLParam(...); if == ""` checks in `internal/service/customer/handler.go`, `internal/service/order/handler.go`, `internal/service/product/handler.go` | `internal/platform/httpx` | 29 callsites across 4 services | Single `RequirePathParam` helper would remove repeated request-validation branches. |
| HTTP error response mapping mechanics | Repeated `errors.SendError`/`apierrors.SendError` use in `internal/service/cart/handlers.go`, `internal/service/customer/handler.go`, `internal/service/order/handler.go`, `internal/service/product/handler.go` | `internal/platform/httperr` | 133 callsites across 4 services | Mechanic is shared; status/message semantics stay domain-owned per phase plan. |

## 3) Architecture Boundary Audit

### Forbidden Patterns Checked

- [x] `internal/platform/*` importing `internal/service/*`
- [x] cross-domain service imports (`internal/service/x` importing `internal/service/y`)
- [x] domain logic embedded in platform packages

### Boundary Matrix (Allowed vs Disallowed)

| Source | Allowed | Disallowed |
|---|---|---|
| `internal/platform/*` | stdlib, `internal/contracts/*`, third-party libs, other `internal/platform/*` | `internal/service/*` imports, domain-specific naming/contracts |
| `internal/service/*` | stdlib, `internal/platform/*`, `internal/contracts/*` | imports of other domain services |

### Findings

| File | Violation Type | Severity | Proposed Fix |
|---|---|---|---|
| `internal/platform/sse/handler.go` | Platform package contains cart-specific transport semantics (`cartID` extraction and messages). | high | In Phase 1, generalize API to entity/topic-agnostic subscription key and move cart naming to service layer adapter. |
| `internal/platform/sse/hub.go` | Platform package models subscribers by `cartID` and logs cart-specific context. | high | Rename map key abstraction to generic stream/entity key and keep cart labels only in cart service wrappers. |
| `internal/platform/sse/client.go` | Platform type carries `cartID` field (domain term in platform core). | high | Replace field with neutral key (`streamID`/`entityID`) and keep behavior unchanged via adapter migration. |

Audit counts:
- `internal/platform/* -> internal/service/*` imports: 0 matches.
- Cross-domain `internal/service/x -> internal/service/y` imports: 0 matches.
- Cart-specific terms inside `internal/platform/sse/*.go`: 52 matches.

## 4) Error Handling Baseline

| Check | Count | Notes |
|---|---:|---|
| uses of `errors.Is` | 27 | Concentrated in cart/order and SQL no-row handling (examples: `internal/service/cart/handlers.go:89`, `internal/service/order/repository_crud.go:207`). |
| direct `== Err...` checks | 5 | All in order HTTP handlers (`internal/service/order/handler.go:29`, `internal/service/order/handler.go:73`, `internal/service/order/handler.go:77`, `internal/service/order/handler.go:108`, `internal/service/order/handler.go:112`). |
| string matching on `err.Error()` | 1 | `internal/service/product/repository_query.go:396` checks `"sql: no rows in result set"`. |
| missing `%w` wrapping opportunities | 171 | Pattern: `fmt.Errorf("...%v", err)` with sentinel prefix (examples: `internal/service/cart/repository_crud.go:49`, `internal/service/order/repository_crud.go:39`, `internal/service/product/repository_query.go:58`). |

Per-service error-style snapshot:
- cart: `errors.Is` 21, direct `== Err` 0, `err.Error()` matching 0, `%v` opportunities 63.
- customer: `errors.Is` 0, direct `== Err` 0, `err.Error()` matching 0, `%v` opportunities 22.
- order: `errors.Is` 6, direct `== Err` 5, `err.Error()` matching 0, `%v` opportunities 26.
- product: `errors.Is` 0, direct `== Err` 0, `err.Error()` matching 1, `%v` opportunities 60.
- eventreader: `errors.Is` 0, direct `== Err` 0, `err.Error()` matching 0, `%v` opportunities 0.

### Sentinel Error Coverage

| Domain Package | Sentinel Errors Present | Missing Domain Sentinels |
|---|---|---|
| cart | 15 sentinels across `internal/service/cart/repository.go` and `internal/service/cart/errors.go` (examples: `ErrCartNotFound`, `ErrCartMustHaveItemsForCheckout`) | None obvious for current scope; Phase 3 should enforce consistent usage paths (`errors.Is` everywhere). |
| customer | 6 sentinels in `internal/service/customer/repository.go` (examples: `ErrCustomerNotFound`, `ErrTransactionFailed`) | Domain-level sentinels for patch validation / business constraints are not explicit; many flows still return ad-hoc `fmt.Errorf` strings. |
| order | 7 sentinels in `internal/service/order/repository.go` (examples: `ErrOrderNotFound`, `ErrInvalidStatusTransition`) | Handler layer still bypasses `errors.Is` style despite sentinel availability. |
| product | 7 sentinels in `internal/service/product/repository.go` (examples: `ErrProductNotFound`, `ErrEventWriteFailed`) | `errors.Is` usage is sparse; one `err.Error()` string comparison bypasses sentinel semantics. |

## 5) Logging Baseline

| Service | Component Naming Consistent | Structured Fields Consistent | Sensitive Data Risk | Top Fixes |
|---|---|---|---|---|
| cart | no | no | med | Keep `component` but replace `handler` key usage in event handlers; introduce `operation` key at key callsites; normalize key names (`cartID` -> `cart_id` where applicable). |
| customer | partial | no | high | Add boundary logs in HTTP handlers; remove/mask raw email logging in `internal/service/customer/service.go:180`; introduce `operation` and request correlation fields. |
| order | partial | no | low | Convert `handler` logger key in event handler to `component`; add `operation` fields around create/cancel/status transitions. |
| product | yes | no | med | Normalize message text in direct-image path; add `operation` and duration fields for expensive I/O (`GetObject`, ingestion batch processing). |
| eventreader | no | no | low | Replace `With("handler", ...)` with `With("component", ...)` in both event handlers and add `operation` field (`process_customer_created`, `process_product_created`). |

Baseline counts against `logging-standard.md`:
- Structured logger usage (`slog`): present; no `log.Printf`/`fmt.Print*` usage found in target service paths.
- Logger constructor `With("component", ...)`: 17 matches across target services.
- Non-standard `With("handler", ...)`: 5 matches (`internal/service/eventreader/eventhandlers/*.go`, `internal/service/order/eventhandlers/on_cart_checked_out.go`, `internal/service/cart/eventhandlers/*.go`).
- `operation` key usage: 0 matches across target services.
- `request_id` key usage: 0 matches across target services.
- `duration_ms` key usage: 0 matches across target services.
- Potential sensitive log fields: 1 match (`internal/service/customer/service.go:180` logs `"email", customer.Email`).

### Field Key Drift

| Concept | Variants Found | Preferred Key |
|---|---|---|
| cart id | `cart_id`, `cartID`, `cartId` | `cart_id` |
| request id | none observed in target services | `request_id` |
| event type | `event_type` (present), plus message text embedding event names (for example `"Ignoring non-ProductCreated event"`) | `event_type` |
| component | `component`, `handler` | `component` |

## 6) Candidate Standardization Backlog

| ID | Title | Phase | Scope | Risk | Dependencies | Acceptance Criteria |
|---|---|---|---|---|---|---|
| STD-001 | Generalize `internal/platform/sse` domain terms | 1 | platform/sse only | high | none | `internal/platform/sse/*` contains no cart-specific identifiers/messages; service adapters preserve runtime behavior and routes. |
| STD-002 | Introduce `httpx` decode/encode/path helpers | 1 | `internal/platform/httpx` | med | none | Package provides strict decode, JSON write, and required path-param APIs with tests; no domain imports. |
| STD-003 | Introduce `httperr` mapping helpers | 1 | `internal/platform/httperr` | med | STD-002 | Shared response mechanics available without owning domain message/status decisions. |
| STD-004 | Pilot cart handler migration to `httpx`/`httperr` | 2 | `internal/service/cart/handlers.go` | med | STD-002, STD-003 | Cart handler removes local helper duplication; responses/statuses remain behavior-compatible. |
| STD-005 | Normalize handler transport patterns across customer/order/product | 3 | `internal/service/customer/handler.go`, `internal/service/order/handler.go`, `internal/service/product/handler.go` | med | STD-004 | Decode/encode/path/error mechanics use platform helpers consistently; route behavior unchanged. |
| STD-006 | Eliminate error-style drift (`== Err`, `err.Error()` matching) | 3 | order/product handler+repo paths | low | STD-005 | 0 direct `== Err...` in handlers and 0 `err.Error()` string matches for control flow; `errors.Is` used for sentinel checks. |
| STD-007 | Normalize `%w` wrapping in repository/service error chains | 3/4 | cart/customer/order/product repositories | med | STD-006 | `%v`-based wrapped error returns replaced where wrapping intent exists; `errors.Is` compatibility verified by tests. |
| STD-008 | Apply logging standard keys and sensitive-data policy | 5 | all target services | low | STD-005 | `component` key used consistently; `operation` added at boundary/failure points; raw `email` logging removed/masked; no `handler` logger key remains. |
| STD-009 | Add architecture/logging regression checks in CI | 6 | repo-wide tooling | low | STD-001, STD-008 | CI checks fail on `platform -> service` imports, cart terms in `internal/platform/sse`, and forbidden logging-key drift (`handler`, missing standard keys in templates). |

## 7) Prioritized Migration Order

1. Service: customer
   - Rationale: Highest HTTP transport duplication density in one file (`internal/service/customer/handler.go`) and immediate sensitive-log issue (`internal/service/customer/service.go:180`). Good first standardization target after cart pilot because blast radius is centralized.
   - Risk notes: Preserve message text and status codes; customer patch/PUT semantics are strict and should not be behavior-altered.
2. Service: order
   - Rationale: Small handler surface with clear error-style drift (5 direct `== Err...` checks) makes it a fast consistency win.
   - Risk notes: Order state transition errors are business-critical; maintain exact validation behavior and status mappings.
3. Service: product
   - Rationale: Largest handler/repository surface (multiple read endpoints + ingestion/repository complexity); includes the only `err.Error()` string compare hotspot.
   - Risk notes: Preserve direct image streaming behavior and ingestion side effects; avoid changing event publication semantics.
4. Service: eventreader
   - Rationale: Smallest service scope; mostly logging-key normalization and alignment after shared patterns stabilize.
   - Risk notes: Keep handler registration behavior and event-type filtering unchanged.

Migration order rationale (phase-aware):
- Keep Phase 2 cart pilot as already recommended by playbook.
- Then prioritize services with highest transport/error inconsistency and lowest architectural coupling first (customer -> order).
- Defer largest code surface (product) until helpers are proven.
- Finish with eventreader for logging and key consistency without HTTP migration overhead.

## 8) Verification Baseline

- `go test ./...` result: pass.
- `go list ./...` result: pass.
- Known flaky tests: none observed in this run.
- Known environment dependencies: none required for these commands in this run; many packages report `[no test files]`.

Command failure summary:
- No required verification commands failed.

## 9) Phase 0 Exit Checklist

- [x] Inventory complete across all target services.
- [x] Boundary matrix documented and reviewed.
- [x] Logging baseline recorded with concrete fixes.
- [x] Backlog items are phase-tagged and dependency-tagged.
- [x] Migration order is explicit.

## 10) Handoff Notes For Next LLM

- Highest-risk areas: `internal/platform/sse/*` domain leakage (cart-specific semantics in platform), and repository-wide error wrapping style drift where `%v` may hide unwrap behavior.
- Recommended first implementation task: execute STD-001 (platform SSE generalization) and STD-002 (`httpx`) before any domain migration.
- Constraints to preserve (API behavior, error messages, etc.):
  - Keep existing HTTP status codes and response shapes in handlers.
  - Keep domain-level error text exposed to clients unchanged unless explicitly approved.
  - Keep event topics/types and outbox behavior unchanged while refactoring mechanics.
  - Maintain clean architecture rule: `internal/platform/*` remains domain-agnostic; `internal/service/*` contains domain behavior.
