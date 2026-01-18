# Integration Tests

This directory contains integration tests for oastools. These tests exercise the full pipeline from parsing through generation using declarative YAML scenarios.

## Running Tests

```bash
# Run all integration tests
make integration-test

# Run with verbose output
go test -tags=integration ./integration/... -v -count=1 -timeout=10m

# Run specific scenario category
go test -tags=integration ./integration/... -run="basics/"

# Run single scenario
go test -tags=integration ./integration/... -run="parse-oas30"

# Debug mode (dumps intermediate outputs)
INTEGRATION_DEBUG=1 go test -tags=integration ./integration/... -v
```

## Directory Structure

```
integration/
├── README.md                 # This file
├── integration_test.go       # Main test entry point
├── harness/                  # Test framework
│   ├── harness.go            # Core types and scenario runner
│   ├── loader.go             # YAML scenario loader
│   ├── pipeline.go           # Pipeline step execution
│   ├── assertions.go         # Result validation helpers
│   ├── problems.go           # Problem injectors for test scenarios
│   └── report.go             # Human-readable output
├── bases/                    # Minimal OAS fixtures
│   ├── petstore-oas2.yaml    # OAS 2.0 baseline
│   ├── petstore-oas30.yaml   # OAS 3.0.3 baseline
│   ├── petstore-oas31.yaml   # OAS 3.1.0 baseline (with webhooks)
│   └── petstore-oas32.yaml   # OAS 3.2.0 baseline (with QUERY)
├── scenarios/                # Test scenarios
│   ├── basics/               # Basic parse/validate tests
│   ├── fixer/                # Fixer scenarios (Phase 2)
│   ├── joiner/               # Joiner scenarios (Phase 3)
│   ├── converter/            # Converter scenarios (Phase 4)
│   ├── pipeline/             # Full pipeline scenarios (Phase 5)
│   └── edge-cases/           # Edge case scenarios
└── golden/                   # Expected outputs for snapshot testing
```

## Scenario File Format

Scenarios are YAML files that describe a test case:

```yaml
# Required: Short, descriptive name
name: "Parse OAS 3.0 document"

# Optional: Additional context
description: "Verifies basic OAS 3.0 parsing works"

# Base document from bases/ directory (without .yaml extension)
base: petstore-oas30

# Problems to inject (Phase 2+)
problems:
  missing-path-params:
    - path: "/pets/{petId}/photos/{photoId}"
      method: GET
  generic-schemas:
    - "Response[Pet]"
  duplicate-operationids:
    - id: "getPet"
      count: 3
  csv-enums:
    - schema: "PetStatus"
      values: "available,pending,sold"
  unused-schemas:
    - "OrphanedType"
  empty-paths:
    - "/deprecated/endpoint"

# Pipeline steps to execute (in order)
pipeline:
  - step: parse
  - step: validate
    expect: valid
    assertions:
      - schemas-exist:
          - Pet
          - NewPet

# Optional: Skip this scenario
skip: "Reason for skipping"

# Optional: Mark as known failing
expected-failure: "Known issue #123"

# Optional: Debug settings
debug:
  dump-after: [parse, validate]
  verbose: true
```

### Pipeline Steps

| Step | Description | Phase |
|------|-------------|-------|
| `parse` | Parse the base document | 1 |
| `parse-all` | Parse multiple input documents | 1 |
| `validate` | Validate the parsed document | 1 |
| `fix` | Apply fixes to the document | 2 |
| `fix-all` | Apply fixes to all documents | 2 |
| `join` | Join multiple documents | 3 |
| `convert` | Convert between OAS versions | 4 |
| `convert-all` | Convert all documents to target version | 4 |
| `generate` | Generate client/server code | 5 |
| `build` | Compile generated code | 5 |
| `diff` | Compare two documents | 6 |
| `overlay` | Apply overlay transformations | 6 |

### Expect Values

| Value | Description |
|-------|-------------|
| `valid` | Document should be valid (no validation errors) |
| `invalid` | Document should be invalid (has validation errors) |
| `error` | Step should fail with an error |
| `success` | Step should succeed (default) |

### Assertions

```yaml
assertions:
  # Check schema count
  - schema-count: 5

  # Check specific schemas exist
  - schemas-exist:
      - Pet
      - Order

  # Check schemas do NOT exist
  - schemas-not-exist:
      - OrphanedType

  # Check error count
  - error-count: 1

  # Check error message contains substring
  - error-contains: "missing required field"

  # Check fixes applied (Phase 2+)
  - fixes-applied:
      missing-path-params: 2
      generic-schemas: 1
```

## Adding New Scenarios

1. Create a new YAML file in the appropriate `scenarios/` subdirectory
2. Define the scenario name, base document, and pipeline
3. Run `make integration-test` to verify

Example:
```yaml
name: "My new scenario"
description: "Tests a specific feature"
base: petstore-oas30
pipeline:
  - step: parse
  - step: validate
    expect: valid
```

## Base Fixtures

Base fixtures are minimal but valid OAS documents:

| File | Version | Features |
|------|---------|----------|
| `petstore-oas2.yaml` | 2.0 | definitions, body parameters |
| `petstore-oas30.yaml` | 3.0.3 | components, requestBody |
| `petstore-oas31.yaml` | 3.1.0 | webhooks, nullable types |
| `petstore-oas32.yaml` | 3.2.0 | QUERY method |

Each has:
- 2-3 paths
- 2-4 schemas
- Valid structure (passes validation)

## Implementation Phases

- **Phase 1** (Current): Foundation - parse, validate
- **Phase 2**: Fixer coverage - all fix types
- **Phase 3**: Joiner coverage - all strategies
- **Phase 4**: Converter + cross-version
- **Phase 5**: Full pipeline + generator
- **Phase 6**: Remaining packages (differ, overlay, etc.)
