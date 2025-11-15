# Data Model: PowerXPlugin Testing Strategy

## Entities

### TestLayer
- **Description**: Represents a logical grouping of automated checks (unit, integration, E2E, CLI).
- **Attributes**:
  - `name` (string; enum: `unit`, `integration`, `e2e`, `cli`)
  - `scope` (string; textual description of coverage)
  - `entry_command` (string; command or script entry point)
  - `dependencies` (string array; required tools/services)
  - `artifacts` (string array; expected outputs)
  - `expected_duration` (integer; minutes)
- **Relationships**:
  - `TestLayer` produces many `TestArtifact` records.

### TestArtifact
- **Description**: Evidence generated while running a test layer.
- **Attributes**:
  - `name` (string; e.g., `coverage.out`, `playwright-report`)
  - `location` (path; e.g., `tmp/coverage.html`)
  - `type` (enum: `log`, `report`, `screenshot`, `bundle`)
  - `retention_policy` (string; e.g., `retain until next run`, `archive on release`)
  - `producer_layer` (foreign key → TestLayer.name)
  - `consumers` (string array; roles or processes that use the artifact)

### TestWorkflow
- **Description**: Automation entry point binding multiple test layers.
- **Attributes**:
  - `name` (enum: `smoke`, `regression`)
  - `layers` (ordered array of TestLayer references)
  - `script_entry` (string; path to shell script or Makefile target)
  - `documentation` (path; e.g., `docs/test/testing_usage.md#smoke`)
  - `success_criteria` (string array; pointers to spec success criteria)
- **Relationships**:
  - `TestWorkflow` orchestrates many `TestLayer` instances.

## Validation Rules
- `TestLayer.entry_command` MUST be runnable from repository root.
- `TestArtifact.location` MUST reside in `tmp/` or a directory explicitly allowed by docs/test/testing_usage.md.
- `TestWorkflow.layers` MUST include at least one layer; smoke workflow MUST include `unit` + `cli`; regression workflow MUST include all four layers.

## State/Process Notes
- When scripts run, they should emit `TestArtifact` metadata for auditing (e.g., echo saved file locations).
- Failure in any layer should halt the owning workflow unless explicitly marked as non-blocking (e.g., flaky E2E with retry).
