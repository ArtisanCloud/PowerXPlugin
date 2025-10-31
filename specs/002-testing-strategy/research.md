# Research Notes: PowerXPlugin Testing Strategy

## Decision 1: Dual Delivery (Documentation + Automation)
- **Decision**: Provide both human-readable guides under `docs/test/` and executable scripts/Makefile targets for smoke and regression workflows.
- **Rationale**: Ensures contributors can follow instructions manually while CI/local environments benefit from one-touch automation, satisfying FR-001/FR-002 and the constitution's transparency rule.
- **Alternatives Considered**:
  - Documentation only — rejected because it relies on manual recall and increases drift risk.
  - Automation only — rejected because new contributors need narrative context and troubleshooting tips.

## Decision 2: Script Placement & Tooling
- **Decision**: House shell scripts under `scripts/testing/` and expose mirrored Makefile targets at repo root.
- **Rationale**: Keeps automation discoverable, avoids scattering shell files, and aligns with repository conventions for future CI integration.
- **Alternatives Considered**:
  - Embedding commands inside documentation without scripts — rejected due to maintainability.
  - Using Node-based runners — rejected; shell provides lowest dependency surface for Go/Node workflows.

## Decision 3: Artifact Locations & Naming
- **Decision**: Standardise generated artifacts to `tmp/` (coverage, reports) and reuse `skeleton/web-admin/test-results/` for Playwright outputs.
- **Rationale**: Matches current docs, keeps git tree clean, and simplifies clean-up operations inside scripts.
- **Alternatives Considered**:
  - Storing outputs alongside source directories — rejected to avoid accidental commits and clutter.
  - Creating new hidden directories — rejected to avoid onboarding confusion.

## Decision 4: Toolchain Baseline
- **Decision**: Require Go 1.21+, Node 18+, npm 9+, Playwright 1.48+ across workflows.
- **Rationale**: Aligns with project baseline (Phase 7 foundation) and ensures compatibility with existing skeleton/framework code.
- **Alternatives Considered**: Supporting earlier versions — rejected due to increased matrix complexity and lack of guarantees.

## Decision 5: Failure Diagnostics
- **Decision**: Capture logs in `/tmp` and point to troubleshooting sections in docs/test/testing_usage.md; scripts should emit helpful messages on missing tooling or service startup failures.
- **Rationale**: Speeds up incident triage and prevents silent failures during CI runs.
- **Alternatives Considered**: Let commands fail without context — rejected for poor developer experience.
