# oastools Examples Expansion: Implementation Plan

**Document Version:** 1.1
**Date:** December 2025
**Purpose:** Comprehensive plan for expanding the examples directory in erraggy/oastools to provide greater value for developers learning and evaluating the toolkit.

---

## ✅ Tier 1 Complete (PR #196)

**Completed:** December 26, 2025
**PR:** #196 (merged with squash)

Tier 1 has been fully implemented. See [Current State](#current-state-post-tier-1) for the resulting structure.

### Resume Instructions for Tier 2

To continue with Tier 2 (Package Coverage Expansion):

1. Read this document for context
2. Create a feature branch: `git checkout -b feat/tier2-workflow-examples`
3. Start with Session A deliverables (validate-and-fix, version-conversion, multi-api-merge)
4. Follow the [Example README Template](#example-readme-template) for consistency
5. Run `make check` before committing

---

## Executive Summary

The oastools repository originally contained a single example (PetStore) demonstrating full-featured code generation. This plan proposes expanding the examples directory to showcase the breadth of oastools capabilities across its 10 modular packages, leveraging the existing corpus of real-world specifications and following patterns established by leading OpenAPI tooling projects.

Three implementation tiers are presented, ranging from a focused enhancement (Tier 1) to a comprehensive examples ecosystem (Tier 3). Each tier builds upon the previous, allowing incremental adoption based on available effort and maintenance capacity.

---

## Current State Analysis

### Existing Examples Structure

```
examples/
├── README.md           # Overview and navigation
└── petstore/
    ├── README.md       # Auto-generated usage documentation
    ├── go.mod          # Standalone Go module
    ├── go.sum
    ├── types.go        # Schema model structs
    ├── client.go       # HTTP client
    ├── server.go       # Server interface
    ├── server_router.go
    ├── server_middleware.go
    ├── server_binder.go
    ├── server_responses.go
    ├── server_stubs.go
    ├── security_helpers.go
    ├── oauth2_petstore_auth.go
    ├── credentials.go
    ├── security_enforce.go
    └── oidc_discovery.go
```

### Strengths of Current Approach

The PetStore example effectively demonstrates the generator package's full capabilities including client generation, server interfaces, OAuth2 flows, OIDC discovery, and credential management. It serves as a comprehensive reference for users interested specifically in code generation.

### Gaps Identified

1. **Package Coverage**: Only 1 of 10 packages (generator) has example coverage. Parser, validator, fixer, converter, joiner, differ, overlay, builder, and httpvalidator lack dedicated examples.

2. **Use Case Diversity**: No examples demonstrate common workflows such as multi-document merging, breaking change detection, or specification validation pipelines.

3. **OAS Version Representation**: Only OAS 2.0 is represented (PetStore source). No examples showcase OAS 3.0.x, 3.1.x, or 3.2.0 specifications.

4. **Progressive Complexity**: No "hello world" minimal example exists for rapid onboarding. Users must understand the full PetStore to begin.

5. **Real-World Specifications**: The corpus contains 10 carefully selected public APIs, but none are represented in examples despite being available for integration testing.

6. **Workflow Examples**: No examples demonstrate chaining operations (parse → validate → fix → convert) or the parse-once optimization pattern that delivers 9-154x performance improvements.

---

## Current State (Post Tier 1)

After PR #196, the examples directory now has this structure:

```
examples/
├── README.md                    # Feature matrix, quick start table, OAS coverage
├── quickstart/
│   ├── README.md               # 5-minute getting started guide
│   ├── spec.yaml               # 27-line OAS 3.0.3 spec
│   ├── main.go                 # ~100 lines: parse → validate → inspect
│   └── go.mod
├── validation-pipeline/
│   ├── README.md               # CI/CD-focused validation guide
│   ├── main.go                 # ~130 lines with source map support
│   └── go.mod
└── petstore/
    ├── README.md               # Overview linking to both variants
    ├── spec/
    │   └── petstore-v2.json    # Downloaded source specification
    ├── stdlib/                 # net/http router (moved from root)
    │   ├── go.mod
    │   └── [15 generated files]
    └── chi/                    # Chi router variant (newly generated)
        ├── go.mod
        └── [15 generated files]
```

### Tier 1 Deliverables Status

| Deliverable | Status | Notes |
|-------------|--------|-------|
| Quickstart example | ✅ Complete | 27-line spec, ~100 lines Go |
| Validation pipeline example | ✅ Complete | Source map support, severity reporting |
| Chi router variant | ✅ Complete | Full feature parity with stdlib |
| Petstore restructure | ✅ Complete | spec/, stdlib/, chi/ organization |
| Enhanced README | ✅ Complete | Feature matrix, time estimates, OAS coverage |

### Implementation Notes from Tier 1

**CLI Flag Corrections:**
The original plan had incorrect flag names. Correct flags are:
- `--server-router chi` (not `-router chi`)
- `--max-lines-per-file` (not `-split-threshold`)

**API Discovery:**
- `ParseResult.Errors` is `[]error`, not a structured type with `.Path`/`.Message` fields
- Use `fmt.Printf("- %v\n", e)` for error display, not field access

**Code Style:**
All examples follow idiomatic Go practices:
- All errors are checked (no blank identifier `_` for error returns)
- Clear error messages with context
- Proper exit codes (0 for success, 1 for failure)

---

## Implementation Options

### Tier 1: Focused Enhancement ✅ COMPLETE

**Effort Estimate:** 1-2 Claude Code sessions
**Actual Effort:** 1 Claude Code session
**Maintenance Burden:** Low
**Value Delivered:** High for generator users, moderate for other packages

> **Status:** Completed in PR #196 (merged December 26, 2025)

This tier added framework variants to the existing PetStore example and introduced a minimal quickstart example, following the oapi-codegen pattern of framework-first organization.

#### Proposed Structure

```
examples/
├── README.md                    # Enhanced overview with navigation
├── quickstart/
│   ├── README.md               # 5-minute getting started guide
│   ├── spec.yaml               # Minimal 20-line OpenAPI spec
│   └── main.go                 # Single-file demonstration
├── petstore/
│   ├── README.md               # Existing (enhanced)
│   ├── spec/
│   │   └── petstore-v2.json    # Source specification
│   ├── stdlib/                 # Current net/http implementation
│   │   ├── go.mod
│   │   └── [generated files]
│   └── chi/                    # Chi router variant
│       ├── go.mod
│       └── [generated files]
└── validation-pipeline/
    ├── README.md               # Demonstrates parse → validate → report
    ├── go.mod
    └── main.go
```

#### Deliverables

1. **Quickstart Example**: A minimal example with a 20-line specification and single main.go that parses, validates, and generates types. Demonstrates core value proposition in under 50 lines of code.

2. **Framework Variants**: Add Chi router variant alongside existing stdlib implementation, demonstrating the `-router chi` flag capability.

3. **Validation Pipeline Example**: Standalone example showing parser and validator packages working together, introducing the parse-once pattern.

4. **Enhanced README**: Update examples/README.md with a feature matrix table, quick command reference, and links to deep dive documentation.

#### Acceptance Criteria

- [x] All examples compile with `go build ./...`
- [x] Each example directory contains a README with purpose, commands, and expected output
- [x] Quickstart example runnable in under 2 minutes from clone
- [x] `make check` passes after changes
- [x] All 4706 tests passing
- [x] Code review passed (prereview skill with 4 agents)

---

### Tier 2: Package Coverage Expansion ← NEXT

**Effort Estimate:** 3-4 Claude Code sessions  
**Maintenance Burden:** Moderate  
**Value Delivered:** High across all packages

This tier extends Tier 1 by adding dedicated examples for each major package category, organized by workflow rather than package name to emphasize practical use cases.

#### Proposed Structure

```
examples/
├── README.md
├── quickstart/                  # From Tier 1
├── petstore/                    # From Tier 1 (enhanced)
│
├── workflows/
│   ├── README.md               # Workflow overview
│   │
│   ├── validate-and-fix/
│   │   ├── README.md           # Validation with auto-fix
│   │   ├── go.mod
│   │   ├── main.go
│   │   └── specs/
│   │       ├── invalid.yaml    # Intentionally problematic spec
│   │       └── fixed.yaml      # Expected output
│   │
│   ├── version-conversion/
│   │   ├── README.md           # OAS 2.0 ↔ 3.x conversion
│   │   ├── go.mod
│   │   ├── main.go
│   │   └── specs/
│   │       ├── swagger-v2.json
│   │       └── openapi-v3.yaml # Expected conversion output
│   │
│   ├── multi-api-merge/
│   │   ├── README.md           # Joining multiple specs
│   │   ├── go.mod
│   │   ├── main.go
│   │   └── specs/
│   │       ├── users-api.yaml
│   │       ├── orders-api.yaml
│   │       └── merged.yaml     # Expected output
│   │
│   ├── breaking-change-detection/
│   │   ├── README.md           # Diff for CI/CD pipelines
│   │   ├── go.mod
│   │   ├── main.go
│   │   └── specs/
│   │       ├── v1.yaml
│   │       └── v2.yaml         # Contains breaking changes
│   │
│   ├── overlay-transformations/
│   │   ├── README.md           # Applying overlays
│   │   ├── go.mod
│   │   ├── main.go
│   │   └── specs/
│   │       ├── base.yaml
│   │       └── overlay.yaml
│   │
│   └── http-validation/
│       ├── README.md           # Runtime request/response validation
│       ├── go.mod
│       └── main.go
│
└── programmatic-api/
    ├── README.md               # Builder package demonstration
    ├── go.mod
    └── main.go                 # Construct spec from code
```

#### Deliverables

1. **Workflow Examples**: Six workflow-focused examples demonstrating practical use cases that span multiple packages.

2. **Synthetic Test Specifications**: Purpose-built YAML/JSON files that clearly demonstrate each workflow's input and expected output.

3. **Builder Example**: Standalone demonstration of programmatic API construction, showing an alternative to hand-written YAML.

4. **Cross-Reference Documentation**: Each example README links to relevant deep dive documentation and pkg.go.dev references.

#### Acceptance Criteria

- All Tier 1 criteria plus:
- Each workflow example demonstrates at least one package not covered by other examples
- Examples include expected output files for comparison
- README files explain when and why to use each workflow
- Performance notes included where parse-once optimization applies

---

### Tier 3: Comprehensive Examples Ecosystem

**Effort Estimate:** 5-7 Claude Code sessions  
**Maintenance Burden:** High (corpus sync, regeneration on releases)  
**Value Delivered:** Maximum (production reference implementations)

This tier creates a full examples ecosystem including real-world API examples from the corpus, integration with the documentation site, and automated regeneration infrastructure.

#### Proposed Structure

```
examples/
├── README.md
├── quickstart/                  # From Tier 1
├── petstore/                    # From Tier 1
├── workflows/                   # From Tier 2
├── programmatic-api/            # From Tier 2
│
├── real-world/
│   ├── README.md               # Overview of corpus-based examples
│   │
│   ├── github/
│   │   ├── README.md           # GitHub API (OAS 3.0.3, large)
│   │   ├── go.mod
│   │   ├── client.go           # Generated client
│   │   └── types.go            # Generated types
│   │
│   ├── stripe/
│   │   ├── README.md           # Stripe API (OAS 3.0.0, very large)
│   │   ├── go.mod
│   │   └── [split files]       # Demonstrates file splitting
│   │
│   ├── discord/
│   │   ├── README.md           # Discord API (OAS 3.1.0)
│   │   ├── go.mod
│   │   └── [generated files]
│   │
│   └── weather-gov/
│       ├── README.md           # US NWS API (OAS 3.0.3, public data)
│       ├── go.mod
│       └── [generated files]
│
├── oas-versions/
│   ├── README.md               # Version-specific examples
│   ├── oas-2.0/                # Swagger 2.0 features
│   ├── oas-3.0/                # OAS 3.0.x features
│   ├── oas-3.1/                # OAS 3.1.x (JSON Schema alignment)
│   └── oas-3.2/                # OAS 3.2.0 (Moonwalk) features
│
├── advanced/
│   ├── README.md
│   │
│   ├── custom-templates/
│   │   ├── README.md           # Template customization
│   │   └── [files]
│   │
│   ├── schema-naming/
│   │   ├── README.md           # Custom naming strategies
│   │   └── [files]
│   │
│   └── collision-handling/
│       ├── README.md           # Joiner collision strategies
│       └── [files]
│
└── scripts/
    ├── regenerate-all.sh       # Regenerate all examples
    └── verify-examples.sh      # CI verification script
```

#### Deliverables

1. **Real-World Examples**: Generated code from 4 corpus specifications representing different sizes, OAS versions, and domains.

2. **OAS Version Showcase**: Dedicated examples highlighting version-specific features (webhooks in 3.1, etc.).

3. **Advanced Examples**: Demonstrations of customization points including template overrides, naming strategies, and collision handling.

4. **Automation Scripts**: Shell scripts for regenerating examples on new releases and verifying example integrity in CI.

5. **Documentation Integration**: MkDocs navigation updates to include examples section with rendered README files.

#### Acceptance Criteria

- All Tier 1 and Tier 2 criteria plus:
- Real-world examples regenerate successfully from corpus URLs
- Scripts execute without error on Linux and macOS
- Examples directory size remains reasonable (< 50MB committed, large specs gitignored with fetch instructions)
- Documentation site builds and renders examples section
- GitHub Actions workflow added for example verification

---

## Recommended Implementation Sequence

### Phase 1: Foundation (Tier 1) ✅ COMPLETE

Executed in a single Claude Code session (PR #196):

1. ✅ Created quickstart example with minimal specification (27 lines)
2. ✅ Restructured petstore to include spec/ subdirectory
3. ✅ Added Chi router variant to petstore
4. ✅ Created validation-pipeline example with source maps
5. ✅ Updated examples/README.md with feature matrix
6. ✅ Ran `make check` and verified (4706 tests passing)

### Phase 2: Workflow Coverage (Tier 2) ← RESUME HERE

Execute across 2-3 Claude Code sessions:

**Session A:**
1. Create workflows/validate-and-fix example
2. Create workflows/version-conversion example
3. Create workflows/multi-api-merge example

**Session B:**
1. Create workflows/breaking-change-detection example
2. Create workflows/overlay-transformations example
3. Create workflows/http-validation example

**Session C:**
1. Create programmatic-api/builder example
2. Cross-reference all README files
3. Update main examples/README.md
4. Run full verification

### Phase 3: Ecosystem (Tier 3)

Execute across 3-4 Claude Code sessions:

**Session A:**
1. Create real-world/weather-gov (smallest corpus spec with valid output)
2. Create real-world/discord (OAS 3.1 representation)
3. Create scripts/regenerate-all.sh

**Session B:**
1. Create real-world/github (demonstrates scale)
2. Create real-world/stripe (demonstrates file splitting)
3. Update gitignore for large generated files

**Session C:**
1. Create oas-versions examples
2. Create advanced examples

**Session D:**
1. Create scripts/verify-examples.sh
2. Add GitHub Actions workflow
3. Integrate with MkDocs documentation
4. Final verification pass

---

## Specification Selection for Real-World Examples

Based on corpus analysis and research findings, the following specifications are recommended for Tier 3:

| Specification | OAS Version | Size | Selection Rationale |
|--------------|-------------|------|---------------------|
| US NWS (weather.gov) | 3.0.3 | ~200KB | Public domain, no auth complexity, stable |
| Discord | 3.1.0 | ~2MB | OAS 3.1 representation, popular domain |
| GitHub | 3.0.3 | ~8MB | Industry standard reference, extensive operations |
| Stripe | 3.0.0 | ~13MB | Demonstrates file splitting, complex schemas |

Specifications **excluded** from examples:

| Specification | Exclusion Rationale |
|--------------|---------------------|
| Microsoft Graph | Too large (34MB), excessive generation time |
| Plaid | Financial domain may confuse general audience |
| DigitalOcean | Known validation errors (496+) |
| Asana | Less recognizable than alternatives |
| Google Maps | Requires API key for meaningful client use |

---

## Example README Template

Each example should follow this structure for consistency:

```markdown
# [Example Name]

[One-sentence description of what this example demonstrates.]

## What You'll Learn

- [Learning outcome 1]
- [Learning outcome 2]

## Prerequisites

- Go 1.24+
- oastools CLI installed (`go install github.com/erraggy/oastools/cmd/oastools@latest`)

## Quick Start

[Numbered steps to run the example]

## Generated Files

| File | Purpose |
|------|---------|
| file.go | Description |

## Key Concepts

[Brief explanation of the oastools features demonstrated]

## Next Steps

- [Link to relevant deep dive documentation]
- [Link to related example]

## Regeneration

To regenerate this example:

\`\`\`bash
oastools [command with flags]
\`\`\`

---

Generated by [oastools](https://github.com/erraggy/oastools) v[VERSION]
```

---

## Maintenance Considerations

### Regeneration Policy

Examples should be regenerated:
- On each minor version release (v1.x.0)
- When generator templates change
- When new features are added that examples should demonstrate

### Corpus Synchronization

Real-world examples depend on external URLs that may change. Mitigation strategies:

1. **Pinned Versions**: Use versioned/tagged URLs where available (GitHub raw with commit SHA)
2. **Local Fallback**: Include instructions for using cached corpus (`make corpus-download`)
3. **CI Verification**: Nightly workflow to verify external URLs remain accessible

### Size Management

To prevent repository bloat:
- Commit only essential generated files for small specs
- Use .gitignore for large generated outputs (Stripe, GitHub)
- Provide regeneration commands in README
- Consider git LFS for any files exceeding 1MB

---

## Success Metrics

| Metric | Tier 1 Target | Tier 2 Target | Tier 3 Target |
|--------|---------------|---------------|---------------|
| Package coverage | 2/10 (generator, validator) | 8/10 | 10/10 |
| OAS version coverage | 1 (2.0) | 3 (2.0, 3.0, 3.1) | 4 (all supported) |
| Time to first success | < 5 minutes | < 5 minutes | < 5 minutes |
| Total example count | 4 | 11 | 20+ |
| Documentation links | Basic | Cross-referenced | Full integration |

---

## Appendix A: CLI Commands for Example Generation

> **Note:** CLI flags corrected during Tier 1 implementation. Use `--flag` format, not `-flag`.

### Quickstart (Types Only)

```bash
oastools generate --types -p quickstart -o examples/quickstart examples/quickstart/spec.yaml
```

### PetStore with Chi Router (Used in Tier 1)

```bash
oastools generate --server --server-all --server-router chi --client \
  --security-enforce --oauth2-flows --oidc-discovery --credential-mgmt \
  -p petstore -o examples/petstore/chi \
  examples/petstore/spec/petstore-v2.json
```

### Real-World: Weather.gov

```bash
oastools generate --client --types -p weathergov \
  -o examples/real-world/weather-gov \
  https://api.weather.gov/openapi.json
```

### Real-World: Stripe (with file splitting)

```bash
oastools generate --client --types -p stripe \
  --max-lines-per-file 5000 \
  -o examples/real-world/stripe \
  https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.json
```

---

## Appendix B: Synthetic Specification Templates

### Minimal Quickstart Spec

```yaml
openapi: "3.0.3"
info:
  title: Quickstart API
  version: "1.0.0"
paths:
  /hello:
    get:
      operationId: sayHello
      summary: Returns a greeting
      responses:
        "200":
          description: A greeting message
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Greeting"
components:
  schemas:
    Greeting:
      type: object
      required: [message]
      properties:
        message:
          type: string
```

### Invalid Spec for Fixer Demo

```yaml
openapi: "3.0.3"
info:
  title: Fixable API
  version: "1.0.0"
paths:
  /items:
    get:
      # Missing operationId (fixer can generate)
      responses:
        "200":
          description: Items list
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/item"  # lowercase (fixer can normalize)
components:
  schemas:
    item:  # lowercase name
      type: object
      properties:
        id:
          type: integer
        Name:  # inconsistent casing
          type: string
```

---

## Appendix C: Decision Matrix

Use this matrix to select the appropriate implementation tier:

| Factor | Choose Tier 1 | Choose Tier 2 | Choose Tier 3 |
|--------|---------------|---------------|---------------|
| Available time | Limited | Moderate | Extensive |
| Maintenance capacity | Low | Medium | High |
| Primary audience | Generator users | All package users | Enterprise evaluators |
| Documentation maturity | Deep dives exist | Deep dives exist | Full docs integration needed |
| CI/CD infrastructure | Basic | Moderate | Advanced |

---

## Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | December 2025 | Initial plan |
| 1.1 | December 26, 2025 | Tier 1 complete (PR #196). Added implementation notes, CLI corrections, API discoveries. Updated status tracking. |
