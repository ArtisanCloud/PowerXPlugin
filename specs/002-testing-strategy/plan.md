# Implementation Plan: PowerXPlugin Testing Strategy

**Branch**: `002-testing-strategy` | **Date**: 2025-10-31 | **Spec**: `specs/002-testing-strategy/spec.md`
**Input**: Feature specification from `/specs/002-testing-strategy/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement a repository-wide testing strategy that delivers (1) smoke and full regression workflows callable via documented commands and executable scripts, (2) guidance for layering and extending automated tests, and (3) troubleshooting plus artifact management standards aligned with the existing docs (`docs/test/testing_strategy.md`, `docs/test/testing_usage.md`). Work focuses on codifying workflows, shipping automation under `scripts/`/Makefile targets, and updating documentation to keep frameworks and CLI outputs consistent.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.21+, Node.js 18+, npm 9+, Bash (POSIX)  
**Primary Dependencies**: Go toolchain (`go test`, `go tool cover`), Playwright 1.48+, Nuxt CLI (`nuxi`), jq/standard UNIX utilities  
**Storage**: N/A (documentation and scripts only)  
**Testing**: `go test`, Playwright (`npx playwright test`), Shell-based validation scripts  
**Target Platform**: Developer workstations & CI runners (Linux/macOS), GitHub Actions  
**Project Type**: Multi-repo tooling/documentation within PowerXPlugin mono-repo  
**Performance Goals**: Smoke workflow ≤5 minutes (backend unit tests <10s, contract validation <30s, CLI scaffold check <30s); full regression ≤60 minutes (Playwright suite <60s per run, retry rate ≤1%); Playwright suite stable with ≤1% flake rate  
**Constraints**: Must honour constitution principles (Go+Nuxt baseline, CLI/script parity); workflows usable without network (except npm install/Playwright install)  
**Scale/Scope**: Team of ~10 contributors, multiple concurrent feature branches, release cadence weekly

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **双重使命仓库**: Plan delivers both documentation and executable scripts ensuring skeleton/framework parity via automated verification. ✅  
- **契约优先兼容性**: Contract validation scripts will enforce schema alignment (`docs/contracts/**`), maintaining single source of truth. ✅  
- **Go + Nuxt 基线**: Workflows target existing Go/Node stacks; no deviation introduced. ✅  
- **脚手架与 CLI 纪律**: CLI automation checks (`px-plugin init`) baked into smoke/regression flows, reinforcing CLI obligations. ✅  
- **透明交付与一致性**: Artifacts (coverage, Playwright) stored under `tmp/`/`skeleton/.../test-results/` with documentation updates, meeting transparency standards. ✅

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
docs/
└── test/
    ├── testing_strategy.md          # Existing strategy reference
    ├── testing_usage.md             # Execution manual
    ├── research.md                  # Phase 0 output (to add)
    └── quickstart.md                # Phase 1 output (to add)

scripts/
├── testing/
│   ├── smoke.sh                     # Planned smoke workflow (new)
│   ├── regression.sh                # Planned full regression automation (new)
│   ├── validate-contracts.sh        # Planned shared validator (new)
│   └── README.md                    # Script catalogue (new)
└── Makefile targets (root)          # `make test-smoke`, `make test-regression`, etc.

tmp/
└── coverage.html                    # Generated coverage artifacts (ignored by VCS)

docs/contracts/                      # Existing schemas, consumed by scripts
skeleton/web-admin/tests/e2e/        # Playwright suite extended per guidance
```

**Structure Decision**: Formalise testing assets under `docs/test/` for documentation and `scripts/testing/` plus Makefile targets for automation. Existing framework/skeleton directories remain unchanged; new scripts will orchestrate commands across `framework/`, `skeleton/`, `tools/cli/`, and `docs/contracts/`.

## Cross-References
- Data Model: [data-model.md](./data-model.md)
- Workflow Contracts: [contracts/testing-workflows.md](./contracts/testing-workflows.md)
- Quickstart: [quickstart.md](./quickstart.md)
- Research Decisions: [research.md](./research.md)

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |

## Phase 0: Outline & Research
- Validate dual-delivery approach (docs + scripts) and record rationale in `research.md`.
- Confirm toolchain baselines, artifact locations, and diagnostics policy.
- Identify any remaining open questions (none at this stage).

## Phase 1: Design & Contracts
- Author `data-model.md` capturing `TestLayer`, `TestArtifact`, `TestWorkflow`.
- Produce `contracts/testing-workflows.md` describing script behaviours and exit criteria.
- Draft `quickstart.md` highlighting prerequisites, smoke/regression execution, and artifact review.
- Plan script/Makefile structure in this document and cross-reference docs/test usage.

## Phase 2: Implementation Preparation
- Implement `scripts/testing/*.sh` and associated Makefile targets.
- Update `docs/test/testing_strategy.md` & `docs/test/testing_usage.md` with final command names.
- Ensure generated artifacts align with documented paths (`tmp/coverage.html`, Playwright reports).
- Integrate smoke/regression scripts into CI (update `.github/workflows/test.yml`, enable parallel backend/contracts/cli jobs, configure Playwright browser caching).
- Emit runtime metrics within smoke/regression scripts and document how to capture them for SC-001/SC-002 verification.
- Deliver a `scripts/testing/audit-test-adoption.sh` workflow (and documentation) to track SC-003 test-adoption criteria across recent PRs.
