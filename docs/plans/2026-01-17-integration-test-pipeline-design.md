# Integration Test Pipeline Design

> **Status:** Draft - Pending Implementation
> **Created:** 2026-01-17
> **Author:** Claude (with Robbie)

## Overview

A comprehensive integration test suite that exercises the full oastools pipeline to validate all "battle-tested" claims. Uses a scenario-driven approach with declarative YAML test cases.

### Motivation

Rapid feature expansion has increased complexity across all packages. We need systematic proof that:
- Every fix type actually fixes what it claims
- Join strategies and semantic deduplication work correctly
- Version conversions can be joined with native-version documents
- Each package's documented features work as advertised

### Design Principles

- **Minimal viable fixtures** - Petstore-style base documents, just enough to trigger test cases
- **Pragmatic tooling** - Use what fits (walker for traversal, direct manipulation for additions)
- **YAGNI** - Start simple, add machinery only when needed
- **Pre-release focus** - Not CI on every commit; run before releases or ad-hoc

---

## Pre-Implementation Setup

### Linear Project Tracking

Before starting implementation, configure Linear for tracking phases:

```bash
# Add the Linear MCP server (already done)
claude mcp add --transport http linear-server https://mcp.linear.app/mcp

# Restart Claude Code to authenticate
# Then create issues for each phase
```

**Linear Structure:**
- Create a project: "Integration Test Pipeline"
- Create issues for each phase (see Implementation Phases below)
- Use labels: `phase-1`, `phase-2`, etc.

---

## Directory Structure

```
integration/
├── README.md                    # How to run, add scenarios, understand results
├── integration_test.go          # Main test entry point with go:build tag
├── harness/
│   ├── harness.go               # Core scenario runner
│   ├── loader.go                # YAML scenario loader & validation
│   ├── problems.go              # Problem injectors (adds issues to base docs)
│   ├── pipeline.go              # Pipeline step execution
│   ├── assertions.go            # Result validation helpers
│   └── report.go                # Human-readable test output
├── bases/
│   ├── petstore-oas2.yaml       # Clean OAS 2.0 baseline (~50 lines)
│   ├── petstore-oas30.yaml      # Clean OAS 3.0 baseline
│   ├── petstore-oas31.yaml      # Clean OAS 3.1 baseline (with webhooks)
│   └── petstore-oas32.yaml      # Clean OAS 3.2 baseline (with QUERY method)
├── scenarios/
│   ├── fixer/                   # One file per fix type or combination
│   ├── joiner/                  # Join strategy scenarios
│   ├── converter/               # Version conversion scenarios
│   ├── pipeline/                # Full end-to-end scenarios
│   └── edge-cases/              # Boundary conditions, error cases
└── golden/                      # Expected outputs for snapshot testing (optional)
```

### Key Decisions

- **Build tag `//go:build integration`** - Won't run with normal `go test`
- **Bases are minimal** - Just enough to be valid, easy to extend
- **Scenarios organized by focus area** - Easy to run subsets
- **Harness is internal** - Not exported, just supports the tests

---

## Scenario File Format

### Single Document Scenarios

```yaml
# integration/scenarios/fixer/all-fixes-combined.yaml
name: "All fix types applied together"
description: "Proves all 6 fix types work correctly when applied simultaneously"

# Base document to start with (from bases/ directory)
base: petstore-oas30

# Problems to inject into the base document
problems:
  missing-path-params:
    - path: "/pets/{petId}/photos/{photoId}"
      method: GET
  generic-schemas:
    - "Response[Pet]"
    - "PagedList[Pet,Cursor]"
  duplicate-operationids:
    - id: "getPet"
      count: 3
  csv-enums:
    - schema: "PetStatus"
      values: "available,pending,sold"
  unused-schemas:
    - "OrphanedLegacyType"
  empty-paths:
    - "/deprecated/endpoint"

# Pipeline steps to execute (in order)
pipeline:
  - step: parse
  - step: fix
    config:
      enabled: all
  - step: validate
    expect: valid
  - step: generate
    config:
      client: true
      server: true
      package: "petstore"
  - step: build
    expect: success

# Optional: capture intermediate results for debugging
debug:
  dump-after: [parse, fix, validate]
```

### Multi-Document (Join) Scenarios

```yaml
# integration/scenarios/joiner/semantic-dedup.yaml
name: "Semantic deduplication across services"

inputs:
  - base: petstore-oas30
    problems:
      # Pet schema defined here
  - base: petstore-oas30
    as: "inventory-service"
    problems:
      # Identical Pet schema with different name "InventoryPet"
      semantic-duplicate:
        - original: Pet
          duplicate-name: InventoryPet

pipeline:
  - step: parse-all
  - step: fix-all
  - step: join
    config:
      strategy: deduplicate-equivalent
      semantic-deduplication: true
  - step: validate
    expect: valid
    assertions:
      - schema-count: 1
```

### Expected Failure Scenarios

```yaml
# integration/scenarios/edge-cases/join-collision-fails.yaml
name: "Join fails on path collision with FailOnCollision strategy"

inputs:
  - base: petstore-oas30
  - base: petstore-oas30
    as: "other-service"

pipeline:
  - step: join
    config:
      strategy: fail-on-collision
    expect: error
    error-contains: "path collision"
```

### Scenario Schema Documentation

A JSON Schema or comprehensive docs for the scenario format should be included in `integration/README.md`.

---

## Harness Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          integration_test.go                            │
│  • Discovers scenarios via glob                                         │
│  • Runs each as t.Run() subtest                                         │
│  • Supports -run filtering                                              │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            harness/loader.go                            │
│  • LoadScenario(path) → *Scenario                                       │
│  • Validates scenario against schema                                    │
│  • Resolves base references to actual file paths                        │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           harness/problems.go                           │
│  • InjectProblems(doc, problems) → modified doc                         │
│  • Registry of problem injectors by type                                │
│  • Uses walker for traversal, direct manipulation for additions         │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           harness/pipeline.go                           │
│  • RunPipeline(scenario) → *PipelineResult                              │
│  • Executes steps in order, passing output to next step                 │
│  • Step executors for each package                                      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          harness/assertions.go                          │
│  • Declarative assertion types                                          │
│  • Schema count, fix counts, error matching, etc.                       │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            harness/report.go                            │
│  • GenerateReport(results) → human-readable summary                     │
│  • Optional: JUnit XML output for CI                                    │
│  • Debug dumps to temp directory                                        │
└─────────────────────────────────────────────────────────────────────────┘
```

### Key Behaviors

1. **Fail-fast by default** - Pipeline stops on first unexpected failure
2. **Debug mode** - `INTEGRATION_DEBUG=1` dumps all intermediate states
3. **Parallel-safe** - Each scenario runs in isolation with its own temp dir
4. **Clear error messages** - Which scenario, which step, expected vs actual, path to debug dump

---

## Problem Injectors

Registry of all problem types that can be injected into base documents:

### Fixer Problems

| Problem | Description |
|---------|-------------|
| `missing-path-params` | Add path with template variables but no parameter declarations |
| `generic-schemas` | Add schemas with bracket syntax (e.g., `Response[Pet]`) |
| `duplicate-operationids` | Create multiple operations with the same operationId |
| `csv-enums` | Store enum values as CSV string instead of array |
| `unused-schemas` | Add schemas not referenced anywhere |
| `empty-paths` | Add paths with no HTTP operations |

### Joiner Problems

| Problem | Description |
|---------|-------------|
| `duplicate-schema-identical` | Same name, same structure (should merge) |
| `duplicate-schema-different` | Same name, different structure (collision) |
| `duplicate-path` | Same path in multiple documents |
| `semantic-duplicate` | Different name, same structure (for dedup testing) |

### Converter Edge Cases

| Problem | Description |
|---------|-------------|
| `oas2-body-param` | OAS 2.0 body parameter (converts to requestBody) |
| `multiple-servers` | Multiple servers (lossy in 3→2 conversion) |
| `webhook` | OAS 3.1+ webhook (not in 2.0) |

### Validator Problems (for expect: invalid)

| Problem | Description |
|---------|-------------|
| `missing-required-field` | Omit a required field |
| `invalid-ref` | Reference to non-existent schema |
| `circular-ref` | Create circular schema reference |

### Implementation Approach

Use the right tool for each injector:
- **Walker** for find-and-modify patterns (e.g., duplicate operationIds)
- **Direct manipulation** for adding new structures (e.g., unused schemas)

---

## Assertions

### In-Scenario Assertion Types

```yaml
# Simple expectations
expect: valid      # No validation errors
expect: invalid    # Has validation errors
expect: error      # Step should fail
expect: success    # Step should succeed (default)

# Detailed assertions
assertions:
  - fixes-applied:
      missing-path-params: 2
      generic-schemas: 3
  - no-fixes-applied:
      - csv-enums
  - schema-count: 5
  - schemas-exist:
      - Pet
      - Order
  - schemas-not-exist:
      - InventoryPet
  - error-count: 1
  - error-contains: "NonExistent"
  - error-path: "$.paths./pets"
  - files-generated:
      - client.go
      - types.go
  - file-contains:
      file: client.go
      patterns:
        - "func.*ListPets"
```

### Failure Output

```
=== FAIL: scenarios/joiner/semantic-dedup.yaml
    Step: join
    Assertion: schema-count
    Expected: 5
    Actual: 7

    Hint: Schemas found: [Pet, InventoryPet, Order, OrderItem, ...]
    Debug: intermediate output written to /tmp/integration-debug-xxx/after-join.yaml
```

---

## Error Handling

### Expected Failures

Some scenarios intentionally test error cases:

```yaml
pipeline:
  - step: join
    config:
      strategy: fail-on-collision
    expect: error
    error-contains: "path collision"
```

### Skipped Scenarios

```yaml
# Skip with reason
skip: "generator doesn't support QUERY method yet"

# Or mark as expected failure (known issue)
expected-failure: "Known issue #142 - circular refs in composition"
```

---

## Test Execution

### Commands

```bash
# Run all integration tests
make integration-test

# Run specific category
go test -tags=integration ./integration/... -run=fixer/

# Run single scenario
go test -tags=integration ./integration/... -run="all-fixes-combined"

# Debug mode
INTEGRATION_DEBUG=1 go test -tags=integration ./integration/... -v
```

### Makefile Targets

```makefile
.PHONY: integration-test
integration-test:
	@echo "Running integration tests..."
	go test -tags=integration ./integration/... -v -count=1 -timeout=10m

.PHONY: integration-test-debug
integration-test-debug:
	INTEGRATION_DEBUG=1 go test -tags=integration ./integration/... -v -count=1 -timeout=10m
```

### Output Format

```
=== RUN   TestScenarios
=== RUN   TestScenarios/fixer/all-fixes-combined
    ✓ parse (12ms)
    ✓ fix (45ms) - 6 fixes applied
    ✓ validate (8ms)
    ✓ generate (234ms) - 4 files
    ✓ build (1.2s)
--- PASS: TestScenarios/fixer/all-fixes-combined (1.5s)
```

### Summary Report

```
================================================================================
INTEGRATION TEST SUMMARY
================================================================================
Scenarios:  23 passed, 0 failed, 2 skipped
Duration:   47.3s

Coverage:
  Fix types exercised:      6/6  ✓
  Join strategies tested:   7/7  ✓
  OAS versions tested:      4/4  ✓
================================================================================
```

---

## Base Fixtures

Minimal, valid OAS documents serving as starting points.

### Requirements

- **~50-80 lines each** - Small enough to read in one screen
- **2-3 paths, 2-3 schemas** - Enough to test joins/refs
- **Valid out of the box** - `TestBasesAreValid` should pass
- **Version-specific features** - Each includes at least one unique feature

### Base Documents

| Base | OAS Version | Unique Features |
|------|-------------|-----------------|
| `petstore-oas2.yaml` | 2.0 | definitions, body parameters |
| `petstore-oas30.yaml` | 3.0.3 | components, requestBody |
| `petstore-oas31.yaml` | 3.1.0 | webhooks, JSON Schema 2020-12 |
| `petstore-oas32.yaml` | 3.2.0 | QUERY method, additionalOperations |

---

## Implementation Phases

### Phase 0: Linear Setup

**Duration:** 15 minutes

1. Restart Claude Code to authenticate with Linear
2. Create project: "Integration Test Pipeline"
3. Create issues for phases 1-6
4. Link this design document

---

### Phase 1: Foundation

**Estimated Duration:** 2-3 hours

**Deliverables:**
- Directory structure
- Harness skeleton: loader, basic pipeline runner
- 4 base fixtures (OAS 2.0, 3.0, 3.1, 3.2)
- 2-3 simple scenarios: parse → validate
- `make integration-test` target
- README with scenario schema documentation

**Acceptance Criteria:**
- `make integration-test` runs and passes
- Bases validated as valid OAS documents
- Can add new scenarios by creating YAML files

---

### Phase 2: Fixer Coverage

**Estimated Duration:** 2-3 hours

**Deliverables:**
- All 6 problem injectors for fixer
- 6+ scenarios (one per fix type, plus combinations)
- Assertions for fix counts and types

**Acceptance Criteria:**
- Every fix type has at least one dedicated scenario
- Combined fix scenario passes
- Assertions verify correct fixes applied

---

### Phase 3: Joiner Coverage

**Estimated Duration:** 3-4 hours

**Deliverables:**
- Problem injectors for join conflicts
- Scenarios for all 7 collision strategies
- Semantic deduplication scenarios
- Assertions for schema counts, deduplication

**Acceptance Criteria:**
- All collision strategies tested
- Semantic deduplication proven to work
- Collision failures handled correctly

---

### Phase 4: Converter + Cross-Version

**Estimated Duration:** 2-3 hours

**Deliverables:**
- Version conversion scenarios (2.0↔3.x, 3.x↔3.y)
- Multi-version join scenarios (convert then join)
- Edge case scenarios (lossy conversions)

**Acceptance Criteria:**
- All version conversions tested
- Converted documents can be joined
- Lossy conversions emit warnings

---

### Phase 5: Full Pipeline + Generator

**Estimated Duration:** 3-4 hours

**Deliverables:**
- End-to-end scenarios: parse → fix → join → generate → build
- Generated code compilation verification
- "Microservices simulation" scenarios

**Acceptance Criteria:**
- Full pipeline runs successfully
- Generated code compiles
- Complex multi-service scenario passes

---

### Phase 6: Remaining Packages

**Estimated Duration:** 4+ hours (can be split)

**Deliverables:**
- Differ scenarios (breaking change detection)
- Overlay scenarios
- HTTPValidator scenarios (optional)
- Builder scenarios (optional)

**Acceptance Criteria:**
- All claimed features have test coverage
- Summary report shows complete coverage

---

## Features to Prove

Summary of all claims that need test coverage:

### Fixer (6 fix types)

- [ ] `missing-path-params` - Adds missing path parameter declarations
- [ ] `generic-schemas` - Renames schemas with special characters
- [ ] `duplicate-operationids` - Renames duplicate operation IDs
- [ ] `csv-enums` - Expands CSV enum strings to arrays
- [ ] `unused-schemas` - Removes unreferenced schemas
- [ ] `empty-paths` - Removes paths with no operations

### Joiner (7 strategies + features)

- [ ] `fail-on-collision` - Errors on any collision
- [ ] `accept-left` - Keeps left document's value
- [ ] `accept-right` - Keeps right document's value
- [ ] `fail-on-paths` - Fails only on path collisions
- [ ] `rename-left` - Renames left schema
- [ ] `rename-right` - Renames right schema
- [ ] `deduplicate-equivalent` - Merges identical schemas
- [ ] Semantic deduplication across documents
- [ ] Operation-aware renaming
- [ ] Namespace prefixes

### Converter

- [ ] OAS 2.0 → 3.0.x
- [ ] OAS 3.0.x → 2.0
- [ ] OAS 3.0.x → 3.1.x
- [ ] OAS 3.1.x → 3.0.x
- [ ] OAS 3.x → 3.2.0
- [ ] Lossy conversion warnings
- [ ] Converted docs can be joined

### Other Packages

- [ ] Parser: multi-version, $ref resolution, circular refs, resource limits
- [ ] Validator: structure, semantic, JSON Schema validation
- [ ] Differ: breaking change detection with severity
- [ ] Generator: client/server generation, security helpers, OAuth2, file splitting
- [ ] Builder: Go type reflection, generic support (optional)
- [ ] Overlay: JSONPath targeting, update/remove actions (optional)
- [ ] Walker: typed handlers, flow control (implicitly tested via others)
- [ ] HTTPValidator: parameter deserialization, schema validation (optional)

---

## Open Questions

None currently - design approved.

---

## Appendix: Scenario Schema Reference

(To be completed in Phase 1 - will include full JSON Schema or detailed field documentation)
