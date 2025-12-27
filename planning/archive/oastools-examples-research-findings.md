# oastools Examples Expansion: Research Findings

**Document Version:** 1.0  
**Date:** December 2025
**Purpose:** Research synthesis documenting how comparable OpenAPI tooling software provides examples, informing the oastools examples expansion strategy.

---

## Research Methodology

This analysis examined eight major OpenAPI tooling projects across the ecosystem to identify patterns and best practices for structuring examples. Projects were selected based on popularity, language diversity, and functional overlap with oastools capabilities.

**Projects Examined:**

| Project | Language | Primary Function | Repository |
|---------|----------|------------------|------------|
| OpenAPI Generator | Java (multi-target) | Code generation | github.com/OpenAPITools/openapi-generator |
| Swagger Codegen | Java (multi-target) | Code generation | github.com/swagger-api/swagger-codegen |
| oapi-codegen | Go | Code generation | github.com/oapi-codegen/oapi-codegen |
| go-swagger | Go | Full toolkit | github.com/go-swagger/go-swagger |
| libopenapi | Go | Parsing/validation | github.com/pb33f/libopenapi |
| kin-openapi | Go | Parsing/validation | github.com/getkin/kin-openapi |
| Connexion | Python | Server framework | github.com/spec-first/connexion |
| NSwag | .NET | Full toolkit | github.com/RicoSuter/NSwag |

---

## Key Finding 1: Petstore as Universal Standard

The Swagger Petstore specification serves as the de facto standard example across virtually all OpenAPI tooling. Every examined project uses some variant of Petstore for primary demonstrations.

### Petstore Variants in Use

Projects maintain multiple Petstore variants for different purposes:

**Standard Petstore** (`petstore.yaml` / `petstore.json`): The canonical 20-operation API definition used for basic feature demonstrations and quickstart guides.

**Minimal Petstore** (`petstore-minimal.json`): A reduced specification with 3-5 operations, used specifically for rapid onboarding examples where the full Petstore would be overwhelming.

**Testing Petstore** (`petstore-with-fake-endpoints-models-for-testing.yaml`): An extended specification including edge cases such as special characters in names, unicode handling, multiple authentication schemes, deeply nested schemas, circular references, and various parameter types. Swagger Codegen and OpenAPI Generator maintain this variant specifically for comprehensive test coverage.

### Implications for oastools

The current oastools PetStore example aligns with industry practice. However, the absence of a minimal variant creates an onboarding gap. New users must understand a 20+ operation API before experiencing any success with the toolkit. Adding a minimal "quickstart" specification would address this gap while maintaining Petstore compatibility for users familiar with the ecosystem standard.

---

## Key Finding 2: Three Dominant Directory Organization Patterns

Analysis revealed three distinct organizational philosophies for example directories, each suited to different tool types.

### Pattern A: Client/Server with Language Nesting

**Used by:** OpenAPI Generator, Swagger Codegen

This pattern organizes examples first by generation target (client vs. server), then by API, then by language and framework:

```
samples/
├── client/
│   └── petstore/
│       ├── java/
│       │   ├── feign/
│       │   ├── resttemplate/
│       │   └── okhttp-gson/
│       ├── go/
│       │   └── go-petstore/
│       └── python/
│           └── petstore-api/
├── server/
│   └── petstore/
│       ├── spring-boot/
│       ├── jaxrs/
│       └── flask/
└── openapi3/
    ├── client/petstore/
    └── server/petstore/
```

**Rationale:** Multi-language generators need to demonstrate each target language. Users typically know their language first and want to find relevant examples quickly.

**Relevance to oastools:** Limited. oastools targets Go exclusively, eliminating the need for language-based organization.

### Pattern B: Framework-First Organization

**Used by:** oapi-codegen

This pattern organizes examples by framework implementation, enabling direct comparison across web frameworks:

```
examples/
├── petstore-expanded/
│   ├── chi/
│   ├── echo/
│   ├── fiber/
│   ├── gin/
│   ├── gorilla/
│   └── stdhttp/
├── authenticated-api/
├── extensions/
│   ├── xgotype/
│   └── xgoname/
└── import-mapping/
```

**Rationale:** Go developers often have strong framework preferences. Showing identical functionality across frameworks allows direct comparison and validation that generated code integrates correctly with their chosen framework.

**Relevance to oastools:** High. The oastools generator supports multiple routers (stdlib, chi). Framework variants would demonstrate this capability effectively.

### Pattern C: Feature/Version Separation

**Used by:** Connexion, NSwag

This pattern organizes examples by OpenAPI version first, then by feature demonstrated:

```
examples/
├── openapi3/
│   ├── helloworld/
│   ├── jwt/
│   ├── sqlalchemy/
│   └── restyresolver/
└── swagger2/
    ├── oauth2/
    └── sqlalchemy/
```

**Rationale:** Version-specific features require version-specific examples. Users migrating from OAS 2.0 to 3.x need to see what changes.

**Relevance to oastools:** Moderate. While oastools supports OAS 2.0 through 3.2.0, version-specific features are less pronounced in Go code generation than in runtime validation frameworks.

---

## Key Finding 3: Test Fixtures Reveal Different Philosophies

The relationship between examples and test fixtures varies significantly across projects.

### Library Pattern: Separate testdata Directories

Go-native libraries (kin-openapi, libopenapi) follow standard Go conventions with `testdata/` subdirectories within each package:

```
openapi3/
├── testdata/
│   ├── valid/
│   │   ├── petstore.yaml
│   │   └── complex-refs.yaml
│   └── invalid/
│       └── missing-paths.yaml
openapi3filter/
└── testdata/
    └── filter-specs/
```

### Generator Pattern: Centralized Test Resources

Code generators (OpenAPI Generator, Swagger Codegen) centralize test specifications in `src/test/resources/` with version-based organization:

```
modules/swagger-codegen/src/test/resources/
├── 2_0/
│   ├── petstore.yaml
│   └── petstore-with-fake-endpoints.yaml
├── 3_0/
│   └── petstore.yaml
└── 3_1/
    └── petstore-3.1.yaml
```

### Issue-Linked Pattern: Living Documentation

go-swagger employs a unique approach that ties test fixtures to GitHub issues:

```
fixtures/
├── bugs/
│   ├── 1746/swagger.yml
│   ├── 2043/swagger.yml
│   └── 2156/swagger.yml
├── enhancements/
│   └── 1557/swagger.yml
└── codegen/
    └── todolist.yml
```

**Rationale:** This creates living documentation where each fixture references its originating issue. Developers can understand why a fixture exists and what edge case it addresses.

**Relevance to oastools:** The issue-linked pattern could enhance the existing `testdata/` structure. Currently, oastools uses `testdata/corpus/` for integration testing but lacks issue-linked fixtures for edge cases discovered during development.

---

## Key Finding 4: README Documentation Approaches

README quality and comprehensiveness varies dramatically across projects.

### Comprehensive In-Repo Documentation

oapi-codegen provides the most extensive in-repo documentation, with a README exceeding 10,000 words covering:

- Installation and CLI usage
- YAML configuration with JSON Schema validation
- Implementation examples for each supported framework
- OpenAPI extension documentation (x-go-type, x-go-name)
- Import mapping guides
- Troubleshooting sections

Configuration files include schema references for IDE validation:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/oapi-codegen/oapi-codegen/HEAD/configuration-schema.json
package: api
generate:
  chi-server: true
  strict-server: true
```

### External Documentation Sites

Complex tools favor external documentation sites over extensive READMEs:

| Project | Documentation Site | README Role |
|---------|-------------------|-------------|
| go-swagger | goswagger.io | Installation and quick links |
| libopenapi | pb33f.io | Architecture overview and getting started |
| NSwag | GitHub Wiki (43 pages) | Minimal with wiki links |
| OpenAPI Generator | openapi-generator.tech | CLI reference only |

### Generated README Templates

Code generators produce standardized READMEs with their output:

```markdown
# petstore-api

Petstore API client generated by OpenAPI Generator

## Installation

pip install petstore-api

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /pets | List all pets |
| POST | /pets | Create a pet |

## Models

- Pet
- Error

## Authentication

Configure API key: `client.configuration.api_key['api_key'] = 'YOUR_API_KEY'`
```

**Relevance to oastools:** The existing PetStore example includes a generated README. The pattern of including both a hand-written overview README and generated usage READMEs within examples is industry-standard and should be maintained.

---

## Key Finding 5: Standalone Runnability is Universal

Every examined project ensures examples are immediately runnable without modification.

### Build Configuration Completeness

Examples include complete build configurations appropriate to their language:

| Language | Included Files |
|----------|---------------|
| Go | go.mod, go.sum |
| Java | pom.xml or build.gradle |
| Python | setup.py, requirements.txt |
| .NET | .csproj |
| JavaScript | package.json |

### Entry Point Conventions

Go examples consistently include minimal `main.go` files demonstrating immediate usage:

```go
func main() {
    swagger, _ := api.GetSwagger()
    e := echo.New()
    api.RegisterHandlers(e, &PetstoreServer{})
    e.Logger.Fatal(e.Start(":8080"))
}
```

### Preserved Implementation Files

go-swagger distinguishes generated code from user implementation code through a `configure_*.go` pattern. This file is generated once but not overwritten on regeneration, allowing users to add implementation code without losing it:

```
restapi/
├── configure_petstore.go   # User-modifiable, preserved
├── doc.go                  # Generated, overwritten
├── embedded_spec.go        # Generated, overwritten
└── server.go               # Generated, overwritten
```

**Relevance to oastools:** The current PetStore example is runnable and includes go.mod. The preservation pattern for user code is worth considering for future generator enhancements but is not directly applicable to examples.

---

## Key Finding 6: Progressive Complexity Patterns

Tools consistently progress examples from minimal to complex:

### Typical Progression

| Stage | Example Type | Features Demonstrated |
|-------|-------------|----------------------|
| 1. Quickstart | helloworld, ping | Single endpoint, minimal spec, fastest success |
| 2. Core CRUD | petstore, todo-list | Multiple endpoints, models, basic validation |
| 3. Authentication | jwt, oauth2, apikey | Security schemes, token handling |
| 4. Database | sqlalchemy examples | Model relationships, persistence |
| 5. Advanced | strict server, custom extensions | Type safety, import mapping, custom templates |

### NSwag's Alternative: Generation Mode Progression

NSwag progresses by generation approach rather than API complexity:

1. Project-based generation (.nswag referencing .csproj)
2. Assembly-based generation (from compiled DLL)
3. Reflection-based generation (runtime inspection)
4. Middleware-based generation (integrated in application startup)

**Relevance to oastools:** The complexity progression pattern aligns with oastools' capability tiers. A quickstart example is notably absent from the current structure and represents a clear gap.

---

## Key Finding 7: Real-World Specification References

Projects reference production APIs to demonstrate scalability, though actual generated code is rarely committed.

### Commonly Referenced Specifications

| Specification | Usage Pattern | Projects Referencing |
|--------------|---------------|---------------------|
| Stripe | Performance benchmarks | libopenapi, go-swagger |
| Kubernetes | Complex schema handling | OpenAPI Generator |
| GitHub | Large operation counts | Multiple |
| DigitalOcean | Multi-file resolution | libopenapi |

### libopenapi's Approach

libopenapi explicitly references complex specifications in documentation without including generated output:

> "libopenapi has been tested against thousands of OpenAPI specifications, including the Stripe API (13MB), Kubernetes API, and DigitalOcean API (1,300+ files)."

The repository includes a synthetic "BurgerShop" API for examples, keeping committed examples small while referencing real-world specs for credibility.

**Relevance to oastools:** The existing corpus (Stripe, GitHub, Discord, etc.) provides excellent real-world validation. Selected corpus specifications could be elevated to examples with generated code committed, demonstrating oastools handling production-scale APIs.

---

## Key Finding 8: Configuration File Convergence

Modern tools have converged on declarative YAML/JSON configuration over CLI flags.

### oapi-codegen Configuration

```yaml
# cfg.yaml
package: api
output: server.gen.go
generate:
  chi-server: true
  models: true
  embedded-spec: true
output-options:
  skip-prune: true
```

### NSwag Configuration

```json
{
  "runtime": "Net70",
  "documentGenerator": {
    "aspNetCoreToOpenApi": {
      "project": "MyProject.csproj"
    }
  },
  "codeGenerators": {
    "openApiToCSharpClient": {
      "className": "MyClient",
      "namespace": "MyNamespace"
    }
  }
}
```

### OpenAPI Generator Configuration

```yaml
generatorName: spring
inputSpec: petstore.yaml
outputDir: generated/spring-server
additionalProperties:
  artifactId: springboot-petstore
  basePackage: com.example.petstore
```

**Relevance to oastools:** Examples could include configuration files demonstrating declarative generation, complementing CLI-based examples. This would show users both interaction patterns.

---

## Synthesis: Patterns Applicable to oastools

Based on this research, the following patterns are directly applicable to the oastools examples expansion:

### Immediate Adoption

1. **Minimal Quickstart**: Add a 20-line specification example demonstrating fastest path to success
2. **Framework Variants**: Organize generator examples by router (stdlib, chi) following oapi-codegen's pattern
3. **README Template**: Standardize example READMEs with purpose, prerequisites, quick start, and regeneration commands
4. **go.mod Completeness**: Ensure all examples are standalone modules (already implemented)

### Considered Adoption

1. **Real-World References**: Include generated code from 2-4 corpus specifications (weather.gov, Discord, GitHub)
2. **Workflow Organization**: Organize non-generator examples by workflow (validate, convert, merge) rather than package name
3. **Issue-Linked Fixtures**: Extend testdata with fixtures linked to resolved issues

### Deferred or Rejected

1. **Language Nesting**: Not applicable (Go-only toolkit)
2. **External Documentation Site**: Already implemented via MkDocs at erraggy.github.io/oastools
3. **Preserved Implementation Files**: Generator-level feature, not examples-specific

---

## Corpus Analysis for Example Selection

The oastools corpus contains 10 specifications spanning OAS versions 2.0 through 3.1.0. The following analysis evaluates each for suitability as a committed example.

### Recommended for Examples

| Specification | Version | Size | Recommendation Rationale |
|--------------|---------|------|--------------------------|
| US NWS (weather.gov) | 3.0.3 | 200KB | Public domain, no auth complexity, validates cleanly, represents government/public data domain |
| Discord | 3.1.0 | 2MB | OAS 3.1 representation, popular domain, manageable size, demonstrates modern spec features |
| GitHub | 3.0.3 | 8MB | Industry standard reference, extensive operation count (800+), demonstrates scale handling |

### Recommended for Reference Only

| Specification | Version | Size | Reference Rationale |
|--------------|---------|------|---------------------|
| Stripe | 3.0.0 | 13MB | Too large for committed example, reference in benchmarks and file-splitting documentation |
| Microsoft Graph | 3.0.4 | 34MB | Excessive size, useful for stress testing documentation only |

### Not Recommended

| Specification | Version | Exclusion Rationale |
|--------------|---------|---------------------|
| Petstore | 2.0 | Already represented |
| DigitalOcean | 3.0.0 | 496 validation errors, would confuse example users |
| Plaid | 3.0.0 | Financial domain less recognizable than alternatives |
| Asana | 3.0.0 | Less industry recognition than GitHub/Discord |
| Google Maps | 3.0.3 | Requires API key for meaningful client usage |

---

## References

### Repository URLs Examined

- https://github.com/OpenAPITools/openapi-generator
- https://github.com/swagger-api/swagger-codegen
- https://github.com/oapi-codegen/oapi-codegen
- https://github.com/go-swagger/go-swagger
- https://github.com/pb33f/libopenapi
- https://github.com/getkin/kin-openapi
- https://github.com/spec-first/connexion
- https://github.com/RicoSuter/NSwag

### Documentation Sites Examined

- https://openapi-generator.tech
- https://goswagger.io
- https://pb33f.io/libopenapi/
- https://connexion.readthedocs.io

### oastools Internal References

- Makefile corpus-download target (lines 399-426)
- internal/corpusutil/corpus.go
- planning/archive/Top10-Public-OAS-Docs-FromNotebookLM-DeepResearch.md

---

## Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | December 2025 | Initial research synthesis |
