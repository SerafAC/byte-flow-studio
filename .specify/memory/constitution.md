<!--
SYNC IMPACT REPORT
==================
Version change: (unversioned) → 1.1.0
Bump type: MINOR — added Governance section; populated Technical Standards (was TODO)

Modified principles:
  - None (principles I–V unchanged)

Added sections:
  - Governance (new)

Removed sections:
  - None

Filled placeholders:
  - TODO(TECH_STACK) resolved: Go 1.23+, TypeScript 5.x, SQLite (modernc.org/sqlite)

Templates checked:
  - .specify/templates/plan-template.md   ✅ Constitution Check gate references constitution dynamically — no update needed
  - .specify/templates/spec-template.md   ✅ Aligned with Principle I (spec-driven) and acceptance scenario format
  - .specify/templates/tasks-template.md  ✅ Test-First ordering (tests before impl) matches Principle II; observability tasks present

Deferred TODOs:
  - None remaining

Follow-up:
  - If a new language/runtime is adopted, update Technical Standards and bump PATCH or MINOR accordingly.
-->

# Byte Flow Studio Constitution

## Core Principles

### I. Spec-Driven Development

All features MUST begin with a feature specification (`spec.md`) before any implementation work
starts. Specifications MUST include user stories with priorities, functional requirements, and
measurable success criteria.

- No code MUST be written without an approved spec.
- Requirements marked `[NEEDS CLARIFICATION]` MUST be resolved before the planning phase begins.
- Acceptance scenarios MUST be written in Given/When/Then format and be independently testable.

**Rationale**: Prevents scope creep, ensures alignment before costly implementation, and produces
living documentation that reflects intent rather than accident.

### II. Test-First (NON-NEGOTIABLE)

TDD is mandatory for all new features and bug fixes.

- Tests MUST be written before implementation code.
- Tests MUST fail (red) before implementation begins.
- The Red → Green → Refactor cycle MUST be observed and verifiable in commit history.
- Tests that are written *after* implementation do not satisfy this principle.

**Rationale**: Post-hoc tests validate only what was built, not what was intended. Test-first
drives better interfaces and surfaces design flaws early.

### III. Component Modularity

Features MUST be designed as self-contained, composable units.

- Each feature module MUST be independently testable without requiring other features to be running.
- Modules MUST expose clear contracts (interfaces/APIs); internal implementation details MUST NOT
  leak across boundaries.
- Shared infrastructure (auth, logging, DB) MUST live in foundational layers, never duplicated
  across feature modules.
- UI views MUST be separated from business logic implementation.

**Rationale**: Enables incremental delivery, independent team workflows, and reduces blast radius
of changes.

### IV. Observability

All features MUST be debuggable without attaching a debugger.

- Structured logging MUST be used for all significant operations (start, completion, failure).
- Errors MUST include enough context to reproduce and diagnose the issue (no bare "something
  failed" messages).
- Log levels MUST be used consistently: DEBUG for trace-level detail, INFO for lifecycle events,
  WARN for recoverable anomalies, ERROR for failures requiring attention.

**Rationale**: Production systems cannot be paused. Observable systems fail gracefully and recover
faster.

### V. Simplicity (YAGNI)

The minimum complexity needed to satisfy current requirements is always the right complexity.

- Features, abstractions, and configurations for hypothetical future requirements MUST NOT be added.
- Each added abstraction MUST justify itself against a concrete, present need.
- When two solutions are equivalent in correctness, the simpler MUST be chosen.
- Violations MUST be documented in the plan's Complexity Tracking table with explicit justification.

**Rationale**: Premature abstractions become technical debt. Simplicity preserves optionality and
reduces onboarding cost.

## Technical Standards

ByteFlow Studio is a multiplatform desktop application with a Go backend and a TypeScript frontend.

- **Backend**: Go 1.23+ — all core processing, pipeline execution, and data persistence logic.
- **Frontend**: TypeScript 5.x — UI layer only; MUST NOT contain business logic.
- **Storage**: SQLite via `modernc.org/sqlite` (CGO-free, cross-compilable). All workflow
  definitions and session data are stored in a single `.byteflow` file per project.
- **Dependencies**: All dependencies MUST be pinned to explicit versions; floating version ranges
  (e.g., `^`, `~`, `*`) MUST NOT be used in production manifests.
- **Configuration**: All environment-specific configuration MUST be externalized (environment
  variables or config files); secrets MUST NOT be committed to version control.
- **Language/runtime choices** for any new dependency MUST be documented in the relevant
  feature's `plan.md` Technical Context section.

## Development Workflow

### Feature Lifecycle

1. **Specify** — Create `spec.md` using `/speckit.specify`; resolve all `[NEEDS CLARIFICATION]` items.
2. **Plan** — Generate `plan.md` using `/speckit.plan`; pass Constitution Check gates.
3. **Task breakdown** — Generate `tasks.md` using `/speckit.tasks`.
4. **Implement** — Execute tasks using `/speckit.implement`; follow Test-First principle strictly.
5. **Analyze** — Run `/speckit.analyze` to verify cross-artifact consistency before marking complete.

### Quality Gates

- Constitution Check in `plan.md` MUST be explicitly passed before Phase 0 research begins.
- All user stories MUST have at least one passing acceptance test before a feature is considered done.
- No PR MUST be merged with failing tests or unresolved `[NEEDS CLARIFICATION]` items in the spec.

### Branching

- Feature branches MUST follow the naming convention: `###-feature-name` (e.g., `001-user-auth`).
- The `main` branch MUST always be in a deployable state.

## Governance

This constitution supersedes all other project practices and conventions. Any practice that
conflicts with a stated principle MUST be resolved in favour of the constitution.

### Amendment Procedure

1. Propose the amendment in writing, stating which principle or section is affected and why.
2. Record the change in this file under a new version using semantic versioning:
   - **MAJOR**: Backward-incompatible changes — principle removals or redefinitions that
     invalidate existing specs, plans, or tests.
   - **MINOR**: New principles, sections, or materially expanded guidance.
   - **PATCH**: Clarifications, wording improvements, or non-semantic refinements.
3. Update `LAST_AMENDED_DATE` to the date of the change.
4. Run `/speckit.analyze` on any active feature specs to verify no cross-artifact conflicts
   were introduced by the amendment.
5. Propagate necessary updates to `.specify/templates/` files if the amendment changes
   mandatory workflow gates or task structures.

### Compliance

- Every plan's Constitution Check MUST enumerate each principle and confirm compliance or
  record a justified exception in the Complexity Tracking table.
- All PRs MUST reference the feature spec; reviewers MUST verify Test-First evidence in
  commit history before approving.

**Version**: 1.1.0 | **Ratified**: 2026-03-19 | **Last Amended**: 2026-03-21
