# Phase Playbooks

Each phase in this document is independently executable by another LLM.

Use this structure per phase:

- Entry criteria
- Scope
- Non-goals
- Ordered tasks
- Verification commands
- Exit criteria
- Rollback approach

---

## Phase 0 - Baseline and Guardrails

## Entry criteria
- None.

## Scope
- Assess current codebase consistency and duplication.
- Define standardization rules that all later phases follow.

## Non-goals
- No production code refactors.
- No new platform helper package implementation.

## Ordered tasks
1. Create a service-by-service inventory of:
   - Handler helper duplication (`decodeJSON`, `writeJSON`, param extraction).
   - Error handling style differences (`errors.Is` vs string compare).
   - Logging style differences (message shape, keys, component names).
2. Create boundary matrix documenting allowed and disallowed imports.
3. Define promote-to-platform criteria:
   - Reused by at least 2 services.
   - Domain-agnostic.
   - Stable API likelihood.
4. Produce prioritized migration backlog grouped by risk and impact.
5. Publish standards checklist for use in all PRs.

## Verification commands
- `go test ./...`
- `go list ./...`

## Exit criteria
- Baseline inventory committed under `docs/standardization`.
- Boundary rules explicitly documented.
- Migration backlog ordered by phases.

## Rollback approach
- Revert only planning documents introduced in this phase.

---

## Phase 1 - Platform Contract Hardening

## Entry criteria
- Phase 0 complete.

## Scope
- Create or refine domain-agnostic platform contracts used by multiple services.
- Define strict package responsibilities.

## Non-goals
- Full migration of domain services.
- Introducing platform code with domain knowledge.

## Ordered tasks
1. Confirm package responsibilities:
   - `internal/platform/httpx` (HTTP mechanics only).
   - `internal/platform/httperr` (shared transport-level response helpers only).
   - Existing `internal/platform/errors` remains generic response envelope utilities.
2. Draft initial APIs with minimal surface area:
   - JSON decode helper with strict parsing.
   - JSON response writer helper.
   - Optional path/query extraction helpers.
3. Define API stability rules:
   - Version compatibility expectations.
   - Deprecation policy for helper functions.
4. Add package docs describing no-domain-logic rule.
5. Add compile-time and unit tests for platform helpers.

## Verification commands
- `go test ./internal/platform/...`
- `go test ./...`

## Exit criteria
- Platform helper packages compile and are tested.
- APIs are documented and intentionally minimal.
- No platform package imports domain packages.

## Rollback approach
- Revert new platform helper packages and docs if APIs prove too broad.

---

## Phase 2 - Shared HTTP Handler Toolkit

## Entry criteria
- Phase 1 complete.

## Scope
- Apply shared handler helpers in one pilot service and validate ergonomics.

## Non-goals
- Full-codebase migration in this phase.

## Ordered tasks
1. Select pilot service (recommended: cart).
2. Replace local helper duplication with platform helpers:
   - Decode request body.
   - Write JSON/no-content responses.
   - Path param requirement checks.
3. Keep domain-specific error mapping in domain handlers.
4. Ensure response messages and status codes remain behavior-compatible.
5. Capture migration notes and API gaps found during pilot.

## Verification commands
- `go test ./internal/service/cart/...`
- `go test ./...`

## Exit criteria
- Pilot handler code uses platform helper package(s).
- No behavior regressions in API response contracts.
- Migration notes recorded for broader rollout.

## Rollback approach
- Revert pilot service changes while retaining platform package groundwork.

---

## Phase 3 - Domain Service Standardization

## Entry criteria
- Phase 2 complete with successful pilot.

## Scope
- Roll standard patterns across all domain services in controlled order.

## Non-goals
- Rewriting business logic.
- Introducing cross-domain coupling.

## Ordered tasks
1. Migrate service handlers in this order:
   - `customer`
   - `order`
   - `product`
   - `eventreader` (if applicable)
2. Apply consistent patterns:
   - Platform JSON helpers for transport mechanics.
   - `Err*` sentinel definitions for domain conditions.
   - `errors.Is` for domain error checks.
   - Consistent constructor and component naming style.
3. Remove dead/commented code while touching files.
4. Add/adjust tests for error mapping and response behavior where needed.
5. Update examples/docs to reflect final patterns.

## Verification commands
- `go test ./internal/service/customer/...`
- `go test ./internal/service/order/...`
- `go test ./internal/service/product/...`
- `go test ./internal/service/eventreader/...`
- `go test ./...`

## Exit criteria
- All domain handlers follow the same transport pattern.
- No string-matching on `err.Error()` for business decisions.
- Error messages/statuses remain stable unless explicitly approved.

## Rollback approach
- Revert migration service-by-service, not all at once.

---

## Phase 4 - Repository and Transaction Consistency

## Entry criteria
- Phase 3 complete.

## Scope
- Standardize data-access and transaction orchestration patterns across services.

## Non-goals
- Database schema redesign.
- New features.

## Ordered tasks
1. Define repository conventions:
   - Interface naming and file layout.
   - Error wrapping patterns.
   - Sentinel ownership by package.
2. Standardize transaction pattern:
   - Begin transaction.
   - Defer rollback guard.
   - Commit and post-commit actions.
3. Standardize outbox usage:
   - In-transaction write semantics.
   - Post-commit trigger behavior.
   - Concurrency safety expectations.
4. Audit for read-modify-write race risks and define mitigation pattern.
5. Create examples in one canonical service package.

## Verification commands
- `go test ./internal/service/...`
- `go test ./...`

## Exit criteria
- Repository and transaction patterns are consistent across services.
- Outbox integration follows a single documented pattern.
- High-risk race-prone flows are documented and mitigated.

## Rollback approach
- Revert by package slice (repository file groups), keeping unaffected services intact.

---

## Phase 5 - Logging Standardization and Debuggability

## Entry criteria
- Phase 2 complete minimum; recommended after Phase 4.

## Scope
- Standardize logging message shape and field conventions across all services.

## Non-goals
- Full observability platform rollout.
- Replacing logging backend.

## Ordered tasks
1. Apply `logging-standard.md` across services.
2. Normalize logger component names (`snake_case`, stable, scoped).
3. Ensure each log line has concise message plus structured fields.
4. Ensure sensitive fields are redacted or omitted.
5. Add service-level log coverage checklist:
   - Startup and shutdown.
   - External I/O boundaries.
   - Domain decision failures.
   - Retry loops and backoff.
6. Reduce noisy/duplicative logs while preserving debugging value.

## Verification commands
- `go test ./...`
- Run one local service and review sample logs for field consistency.

## Exit criteria
- Logging format and fields are consistent across domain and platform layers.
- No known sensitive values in logs.
- Debugging a failed request/event can be done via structured keys.

## Rollback approach
- Revert logger callsite edits by package if verbosity reduction removed needed signals.

---

## Phase 6 - Enforcement and Template Finalization

## Entry criteria
- Phases 0 through 5 complete.

## Scope
- Prevent regression by automating checks and publishing templates.

## Non-goals
- Major additional refactors.

## Ordered tasks
1. Add CI checks for:
   - `gofmt`/`goimports` compliance.
   - test suite pass.
   - static analysis/lint baselines.
2. Add architecture check policy:
   - detect forbidden import direction (`platform -> service`).
3. Publish starter templates:
   - new domain service package skeleton.
   - new handler skeleton using platform helper toolkit.
   - new repository skeleton with standard tx pattern.
4. Add PR checklist using standards from this program.

## Verification commands
- CI pipeline dry-run equivalent commands.
- `go test ./...`

## Exit criteria
- New code paths follow standards by default.
- CI blocks architectural and style drift.
- Templates are published and referenced in docs.

## Rollback approach
- Disable new CI checks one by one only if they produce false positives; keep templates/docs.
