# Standardization Program

This directory contains a staged plan to make the codebase a consistent, idiomatic foundation for future development while preserving strict separation between domain services and shared platform services.

## Objectives

- Create a predictable and idiomatic Go codebase across all services.
- Preserve and enforce clean architecture boundaries:
  - `internal/platform/*` = reusable technical capabilities.
  - `internal/service/*` = domain business logic and policies.
- Reduce duplication safely through reusable platform packages where appropriate.
- Standardize logging so messages are concise, consistent, and useful for debugging.
- Enable independent phase-by-phase execution by different LLMs.

## How To Use These Docs

1. Read `roadmap.md` for phases, dependencies, and sequencing.
2. Execute one phase from `phase-playbooks.md` at a time.
3. Apply logging rules from `logging-standard.md` in every phase.
4. Do not start a phase unless all prerequisites and entry checks are satisfied.

## Program Rules

- Behavior-preserving refactors by default unless phase explicitly permits behavior changes.
- Small batches and frequent verification (`go test ./...` at phase milestones).
- No domain logic in platform packages.
- No platform package imports from domain packages.
- Prefer introducing standards and helper APIs before mass migration.

## Deliverables

- `roadmap.md`: phase sequence, goals, and quality gates.
- `phase-playbooks.md`: execution-ready steps, acceptance criteria, and rollback per phase.
- `logging-standard.md`: logging conventions, message format, and anti-patterns.
- `phase-0-baseline.md`: completed Phase 0 baseline audit, boundary findings, and phase-tagged migration backlog.
- `templates/*`: canonical starter templates for domain services, handlers, and repository transaction flows.
