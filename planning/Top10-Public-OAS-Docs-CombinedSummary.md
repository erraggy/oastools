# Top 10 Public OpenAPI Specifications for Integration Testing

## Combined Summary from Multi-Platform Research

This document synthesizes research from three AI platforms to identify the most suitable public OpenAPI specifications for `oastools` integration testing. Each URL has been **verified accessible and confirmed to return well-formed OAS documents** as of December 2025.

### Methodology

Sources were weighted by research depth and accuracy:
1. **NotebookLM Deep Research** (highest weight) - Comprehensive analysis with structural metrics
2. **Claude Opus 4.5** (medium weight) - Curated list with version diversity focus
3. **GitHub Copilot + Gemini Pro 3** (lowest weight) - Broad survey with some broken links

Selection criteria prioritized:
- **URL accessibility** - All URLs verified to return HTTP 200
- **OAS validity** - Confirmed valid OpenAPI/Swagger structure with version declaration
- **Version diversity** - Coverage across OAS 2.0, 3.0.x, and 3.1.x
- **Size spectrum** - From minimal reference specs to enterprise-scale documents
- **Domain diversity** - Multiple industries and API patterns

### Verification Results

| Status | Count | Description |
|--------|-------|-------------|
| Verified Valid | 14 | Accessible and well-formed OAS |
| Invalid/Inaccessible | 6 | 404 errors or incomplete specifications |

**Notable failures:**
- Google Ads API (`v16` and `v17`) - 404 Not Found
- FHIR R4 (`fhir-swagger` repo) - 404 Not Found
- Twilio Core API - Incomplete (missing required root fields)
- Kubernetes - Schema fragment only (not complete OAS)
- AWS S3 / Uber - 404 (as noted in Copilot research)

### oastools Validation Results

Each specification was validated using `oastools validate --strict --no-warnings`.

For specifications with external HTTP `$ref` references, parsing used:
`oastools parse --resolve-refs --resolve-http-refs <url>`

| # | API | oastools Validation | Result |
|---|-----|---------------------|--------|
| 1 | Microsoft Graph | FAILED | 30,294 errors |
| 2 | Stripe | PASSED | Valid specification |
| 3 | GitHub | FAILED | 8,074 errors |
| 4 | Plaid | FAILED | 101 errors |
| 5 | Discord | PASSED | Valid specification |
| 6 | DigitalOcean | FAILED | 1,918 errors (path params not declared) |
| 7 | Google Maps | FAILED | 228 errors |
| 8 | US NWS | FAILED | 156 errors |
| 9 | Petstore 2.0 | PASSED | Valid specification |
| 10 | Asana | FAILED | 302 errors |

**Summary:** 3 passed, 7 failed validation

**Common validation errors:** Most failures involve path template parameters not declared in the operation's parameters list (e.g., `{owner}` in path but missing from `parameters` array). This is a strict validation check per the OAS specification.

**Note:** DigitalOcean provides a pre-bundled spec via their CI at `https://api-engineering.nyc3.digitaloceanspaces.com/spec-ci/DigitalOcean-public.v2.yaml`. The bundled version (2.4MB) parses successfully with 354 paths, 542 operations, and 651 schemas. Validation fails due to path template parameters not declared in operation parameters (same issue as GitHub and others).

---

## Combined Top 10 Specifications

### 1. Microsoft Graph v1.0

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml` |
| **OAS Version** | 3.0.4 |
| **Paths** | ~6,500 |
| **Operations** | ~18,000 |
| **Schemas** | ~3,000 |
| **File Size** | ~15 MB |
| **Format** | YAML |

The largest single-file OAS document available. Essential for **stress testing parser performance and memory management**. Features deep OData-style component referencing and hierarchical paths typical of enterprise APIs.

**Source:** NotebookLM (#1), Claude (additional resources)

---

### 2. Stripe API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.json` |
| **OAS Version** | 3.0.0 |
| **Paths** | ~300-500 |
| **Operations** | ~700-900 |
| **Schemas** | ~400-900 |
| **File Size** | ~2.5-14 MB |
| **Format** | JSON |

The payments industry gold standard. Features extensive use of `anyOf`/`oneOf` for polymorphism, callbacks, webhooks, and vendor extensions (`x-expandableFields`, `x-stripeBypassValidation`). Ideal for testing **schema composition and async webhook modeling**.

**Source:** All three platforms (unanimous)

---

### 3. GitHub REST API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json` |
| **OAS Version** | 3.0.3 |
| **Paths** | ~400-1,000 |
| **Operations** | ~600-3,000 |
| **Schemas** | ~600-800 |
| **File Size** | ~424 KB (JSON) |
| **Format** | JSON |

One of the most popular developer APIs. Features custom media types (`vnd.github+json`), extensive `$ref` linking, and `x-github` extensions. Tests **content negotiation and modular reference resolution**.

**Note:** A 3.1.0 version is available at `descriptions-next/api.github.com/api.github.com.json` for testing newer OAS features.

**Source:** All three platforms (unanimous)

---

### 4. Plaid API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/plaid/plaid-openapi/master/2020-09-14.yml` |
| **OAS Version** | 3.0.0 |
| **Paths** | ~50-150 |
| **Operations** | ~100-250 |
| **Schemas** | ~150-200 |
| **File Size** | ~400 KB - 1.2 MB |
| **Format** | YAML |

Modern FinTech API with multiple server definitions (Production/Sandbox) and strict security requirements. Tests **environment-aware client configuration and security scheme enforcement**.

**Source:** NotebookLM (#4), Copilot (#6)

---

### 5. Discord API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/discord/discord-api-spec/main/specs/openapi.json` |
| **OAS Version** | **3.1.0** |
| **Paths** | ~150 |
| **Operations** | ~300 |
| **Schemas** | ~400 |
| **File Size** | ~2-5 MB |
| **Format** | JSON |

The **only OpenAPI 3.1.0 specification** from a major API provider in this corpus. Covers Discord HTTP API v10 with custom `x-discord-union` extensions. Essential for testing **OAS 3.1 features** including JSON Schema Draft 2020-12 alignment and type arrays.

**Source:** Claude (#4)

---

### 6. DigitalOcean API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://api-engineering.nyc3.digitaloceanspaces.com/spec-ci/DigitalOcean-public.v2.yaml` |
| **OAS Version** | 3.0.0 |
| **Paths** | 354 |
| **Operations** | 542 |
| **Schemas** | 651 |
| **File Size** | ~2.4 MB (bundled) |
| **Format** | YAML |

Clean, resource-oriented cloud provider API. The bundled version consolidates 1,300+ external `$ref` files into a single file. Excellent for testing **large schema counts and RESTful design patterns**.

**Note:** The corpus uses DigitalOcean's pre-bundled CI artifact. The unbundled source at `github.com/digitalocean/openapi` uses non-standard `$ref` in `info.description`.

**Source:** Claude (#3), Copilot (#8)

---

### 7. Google Maps Platform API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/googlemaps/openapi-specification/main/dist/google-maps-platform-openapi3.json` |
| **OAS Version** | 3.0.3 |
| **Paths** | ~50 |
| **Operations** | ~60 |
| **Schemas** | ~150 |
| **File Size** | ~500 KB |
| **Format** | JSON |

One of the few Google APIs with a public OpenAPI spec. Covers Directions, Geocoding, Distance Matrix, Elevation, Places, Roads, and Time Zone APIs. Tests **geo-coordinate parameter handling and well-documented request/response examples**.

**Source:** Claude (#6)

---

### 8. US National Weather Service API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://api.weather.gov/openapi.json` |
| **OAS Version** | 3.0.3 |
| **Paths** | ~50 |
| **Operations** | ~120 |
| **Schemas** | ~100 |
| **File Size** | ~800 KB |
| **Format** | JSON |

Public utility API with JSON-LD structure and custom extensions for data discovery. Tests **schema extensibility, non-standard field handling, and geo-coordinate parameter encoding**.

**Source:** NotebookLM (#7)

---

### 9. Swagger Petstore (OAS 2.0)

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://petstore.swagger.io/v2/swagger.json` |
| **OAS Version** | **Swagger 2.0** |
| **Paths** | 14 |
| **Operations** | 20-21 |
| **Definitions** | 6 |
| **File Size** | ~20 KB |
| **Format** | JSON |

The canonical **Swagger 2.0 reference specification**. Essential for testing **backward compatibility and 2.0 → 3.x conversion**. Includes OAuth 2.0 and API key authentication patterns using legacy `definitions`, `consumes`, and `produces` fields.

**Source:** All three platforms (unanimous)

---

### 10. Asana API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/Asana/openapi/master/defs/asana_oas.yaml` |
| **OAS Version** | 3.0.0 |
| **Paths** | ~120 |
| **Operations** | ~250 |
| **Schemas** | ~150 |
| **File Size** | ~405 KB |
| **Format** | YAML |

Clean YAML structure covering tasks, projects, portfolios, goals, teams, webhooks, and custom fields. Tests **YAML parser robustness and well-structured project management domain modeling**.

**Source:** Claude (#7)

---

## Quick Reference Table

| # | API | OAS Version | Paths | Operations | Schemas | Size | Format |
|---|-----|-------------|-------|------------|---------|------|--------|
| 1 | Microsoft Graph | 3.0.4 | ~6,500 | ~18,000 | ~3,000 | ~15 MB | YAML |
| 2 | Stripe | 3.0.0 | ~300 | ~900 | ~400 | ~2.5 MB | JSON |
| 3 | GitHub | 3.0.3 | ~400 | ~600 | ~700 | ~424 KB | JSON |
| 4 | Plaid | 3.0.0 | ~150 | ~250 | ~200 | ~1.2 MB | YAML |
| 5 | Discord | **3.1.0** | ~150 | ~300 | ~400 | ~2-5 MB | JSON |
| 6 | DigitalOcean | 3.0.0 | 354 | 542 | 651 | ~2.4 MB | YAML |
| 7 | Google Maps | 3.0.3 | ~50 | ~60 | ~150 | ~500 KB | JSON |
| 8 | US NWS | 3.0.3 | ~50 | ~120 | ~100 | ~800 KB | JSON |
| 9 | Petstore | **2.0** | 14 | 21 | 6 | ~20 KB | JSON |
| 10 | Asana | 3.0.0 | ~120 | ~250 | ~150 | ~405 KB | YAML |

---

## Version Coverage Summary

| OAS Version | Count | Specifications |
|-------------|-------|----------------|
| Swagger 2.0 | 1 | Petstore |
| OpenAPI 3.0.0 | 4 | Stripe, Plaid, DigitalOcean, Asana |
| OpenAPI 3.0.3 | 3 | GitHub, Google Maps, US NWS |
| OpenAPI 3.0.4 | 1 | Microsoft Graph |
| OpenAPI 3.1.0 | 1 | Discord |

**OpenAPI 3.2.0 Note:** Released September 2025, but no public API specifications have adopted it yet. Monitor the OpenAPI Initiative's examples repository for future 3.2.0 samples.

---

## Format Distribution

| Format | Count | Specifications |
|--------|-------|----------------|
| JSON | 6 | Stripe, GitHub, Discord, Google Maps, US NWS, Petstore |
| YAML | 4 | Microsoft Graph, Plaid, DigitalOcean, Asana |

---

## Domain Coverage

| Domain | Specifications |
|--------|----------------|
| FinTech/Payments | Stripe, Plaid |
| Developer Tools | GitHub, DigitalOcean |
| Communications | Discord |
| Geo/Mapping | Google Maps |
| Weather/Public Utility | US NWS |
| Project Management | Asana |
| Enterprise Cloud | Microsoft Graph |
| Reference/Tutorial | Petstore |

---

## Honorable Mentions (Verified Valid)

These specifications were verified valid but not included in the top 10:

| API | URL | OAS Version | Notes |
|-----|-----|-------------|-------|
| GitLab | `https://docs.gitlab.com/api/openapi/openapi.yaml` | 3.0.1 | Partial API coverage, ~97 KB |
| Slack Web API | `https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2.json` | 2.0 | Archived repo, RPC-style API |
| Data.gov Admin | `https://api-umbrella.readthedocs.io/en/latest/_static/admin-api-swagger.yml` | 2.0 | Federal agency API management |
| Petstore 3.0 | `https://petstore3.swagger.io/api/v3/openapi.json` | 3.0.4 | Modern reference implementation |
| Box Platform | `https://raw.githubusercontent.com/box/box-openapi/main/openapi.json` | 3.0.2 | Enterprise file storage |
| GitHub 3.1.0 | `https://raw.githubusercontent.com/github/rest-api-description/main/descriptions-next/api.github.com/api.github.com.json` | 3.1.0 | Preview/next version |

---

## Integration Test Recommendations

Based on the combined research, the following test categories are recommended:

### Performance & Scalability
- **Microsoft Graph** - Parser efficiency, memory management, deep dereferencing
- **Stripe** - Large schema count, polymorphic types

### Legacy & Conversion
- **Petstore 2.0** - Baseline backward compatibility
- **Slack** (honorable mention) - Real-world Swagger 2.0 complexity

### Version-Specific Features
- **Discord** - OAS 3.1.0 features (type arrays, JSON Schema 2020-12)
- **GitHub** - OAS 3.0.3 with path to 3.1.0 migration testing

### Security & Configuration
- **Plaid** - Multi-server definitions, strict security requirements
- **Asana** - OAuth2 and personal access token patterns

### Reference Resolution
- **DigitalOcean** - External `$ref` files, bundling
- **GitHub** - Extensive internal `$ref` linking

### Domain-Specific Patterns
- **Google Maps** - Geo-coordinate parameters
- **US NWS** - JSON-LD extensions, public API patterns

---

*Last verified: December 6, 2025*
*oastools version: v1.18.0 (with HTTP $ref resolution support)*
*Sources: NotebookLM Deep Research, Claude Opus 4.5, GitHub Copilot + Gemini Pro 3*
