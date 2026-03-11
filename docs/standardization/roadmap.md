# Standardization Roadmap

This roadmap defines the order of operations for standardizing architecture and coding style across platform and domain services.

## Phase Sequence

1. Phase 0 - Baseline and Guardrails
2. Phase 1 - Platform Contract Hardening
3. Phase 2 - Shared HTTP Handler Toolkit
4. Phase 3 - Domain Service Standardization
5. Phase 4 - Repository and Transaction Consistency
6. Phase 5 - Logging Standardization and Debuggability
7. Phase 6 - Enforcement and Template Finalization

## Why This Sequence

- Build guardrails first so later refactors do not drift.
- Stabilize platform contracts before bulk domain migration.
- Migrate domain services only after reusable helpers exist.
- Standardize logging once structural patterns are in place.
- Add enforcement after standards are proven in real code.

## Dependency Graph

- Phase 0 is required by all phases.
- Phase 1 is required by Phase 2 and Phase 3.
- Phase 2 is required by Phase 3.
- Phase 3 is required by Phase 4.
- Phase 5 can begin after Phase 2 pilot, but must complete before Phase 6.
- Phase 6 depends on all previous phases.

## Per-Phase Outcomes

## Phase 0 - Baseline and Guardrails
- Define architectural constraints and coding standards as enforceable rules.
- Establish baseline metrics and inventory duplication hotspots.
- Produce prioritized migration backlog.

## Phase 1 - Platform Contract Hardening
- Introduce or refine platform-level shared packages that are domain-agnostic.
- Lock package responsibilities and import boundaries.
- Document stable APIs for shared utilities.

## Phase 2 - Shared HTTP Handler Toolkit
- Provide common handler primitives (JSON decode/encode, path params, common request failure patterns).
- Keep response semantics domain-owned while reducing boilerplate.
- Validate helper APIs in one pilot service before broad rollout.

## Phase 3 - Domain Service Standardization
- Apply common patterns across `internal/service/*` packages in a fixed migration order.
- Normalize error handling style (`Err*`, wrapping, `errors.Is`).
- Keep domain behavior unchanged while improving consistency.

## Phase 4 - Repository and Transaction Consistency
- Standardize transaction patterns, rollback safety, and outbox integration conventions.
- Normalize repository interface patterns and error semantics.
- Ensure read-modify-write flows are concurrency-safe and explicit.

## Phase 5 - Logging Standardization and Debuggability
- Enforce concise and structured logging message patterns.
- Normalize context fields and component names.
- Add redaction and sensitive-data handling consistency.

## Phase 6 - Enforcement and Template Finalization
- Add CI checks, style checks, and architecture checks.
- Publish canonical templates for new services and handlers.
- Prevent future drift through automation and docs.

## Definition of Done (Program)

- Platform/domain dependency boundaries are verifiably clean.
- Shared utilities are adopted consistently by all domain services.
- Error and logging patterns are consistent across services.
- A new service can be bootstrapped from templates without ad-hoc decisions.
- CI includes checks that detect style and architecture regressions.
