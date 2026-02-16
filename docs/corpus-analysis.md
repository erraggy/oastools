# Corpus Analysis: 10 Real-World OpenAPI Specs

> Generated 2026-02-15 using only the oastools MCP server tools (`parse`, `walk_operations`, `walk_schemas`, `walk_parameters`, `walk_responses`, `walk_security`, `walk_headers`, `walk_refs`) with the `group_by` aggregation feature from #321.

The corpus lives in `testdata/corpus/` and contains specs from major API providers spanning Swagger 2.0 through OAS 3.1.

---

## Parse Summary

| Spec | OAS | Format | Paths | Operations | Schemas | Tags |
|------|-----|--------|------:|-----------:|--------:|-----:|
| Petstore | 2.0 | JSON | 14 | 20 | 6 | 3 |
| Google Maps | 3.0.3 | JSON | 17 | 17 | 75 | 9 |
| NWS Weather | 3.0.3 | JSON | 60 | 60 | 103 | 0 |
| Asana | 3.0.0 | YAML | 158 | 217 | 244 | 42 |
| Discord | 3.1.0 | JSON | 137 | 227 | 492 | 0 |
| Plaid | 3.0.0 | YAML | 334 | 324 | 2,179 | 1 |
| DigitalOcean | 3.0.0 | YAML | 356 | 544 | 658 | 48 |
| Stripe | 3.0.0 | JSON | 415 | 588 | 1,321 | 0 |
| GitHub | 3.0.3 | JSON | 720 | 1,078 | 904 | 46 |
| MS Graph | 3.0.4 | YAML | 10,405 | 16,098 | 4,294 | 457 |
| **TOTALS** | | | **12,616** | **19,173** | **10,276** | |

The corpus spans 3 orders of magnitude in every dimension (20 to 16,098 operations), covers all major OAS versions (2.0, 3.0.0, 3.0.3, 3.0.4, 3.1.0), and includes both JSON (6 specs) and YAML (4 specs).

---

## Operations by Method

Using `walk_operations` with `group_by=method`:

| Spec | GET | POST | PUT | PATCH | DELETE |
|------|----:|-----:|----:|------:|-------:|
| Petstore | 8 | 7 | 2 | - | 3 |
| Google Maps | 16 | 1 | - | - | - |
| NWS Weather | 60 | - | - | - | - |
| Asana | 101 | 75 | 21 | - | 20 |
| Discord | 97 | 42 | 20 | 31 | 37 |
| Plaid | 2 | 322 | - | - | - |
| DigitalOcean | 283 | 110 | 55 | 12 | 84 |
| Stripe | 263 | 293 | - | - | 32 |
| GitHub | 568 | 171 | 112 | 61 | 166 |
| MS Graph | 8,473 | 3,361 | 179 | 1,976 | 2,109 |

### Patterns

- **NWS is read-only**: all 60 operations are GET. A pure data-retrieval API.
- **Plaid is POST-only** (99.4%): the RPC-over-HTTP pattern where every endpoint is a command, not a resource operation.
- **Stripe leans POST > GET** (293 vs 263): payment domain is write-heavy. Zero PUT/PATCH -- all mutations use POST.
- **MS Graph favors PATCH over PUT** (1,976 vs 179): OData convention for partial updates.
- **GitHub uses all 5 methods** with the widest distribution, reflecting classic REST design.

---

## Component Schema Types

Using `walk_schemas` with `group_by=type` and `component=true`:

| Spec | object | string | array | integer | boolean | number | (none) | nullable unions |
|------|-------:|-------:|------:|--------:|--------:|-------:|-------:|------:|
| Petstore | 6 | 14 | 2 | 9 | 1 | - | 1 | - |
| Google Maps | 64 | 161 | 63 | 15 | 25 | 44 | 74 | - |
| NWS | 78 | 228 | 60 | 12 | 4 | 6 | 225 | - |
| Asana | 262 | 590 | 85 | 30 | 57 | 16 | 416 | - |
| Discord | 425 | 451 | 165 | 330 | 145 | 10 | 1,634 | 771 |
| Plaid | 1,674 | 2,978 | 567 | 317 | 211 | 384 | 3,624 | - |
| DigitalOcean | 786 | 1,511 | 391 | 423 | 161 | 46 | 1,169 | - |
| Stripe | 1,439 | 3,277 | 296 | 685 | 394 | 23 | 2,746 | - |
| GitHub | 2,508 | 20,123 | 661 | 2,810 | 3,157 | 64 | 2,636 | - |
| MS Graph | 6,660 | 7,747 | 2,835 | 4 | 1,394 | 1,018 | 10,007 | - |

### Patterns

- **Discord is the only OAS 3.1 spec** and uniquely shows nullable union types (`string, null`: 312, `integer, null`: 158, etc. -- 771 total). This is the 3.1 way of expressing nullability via JSON Schema's `type: [string, null]` instead of 3.0's `nullable: true`.
- **GitHub has a massive string bias** (20,123 string schemas, 63% of all component schemas). Many enums and scalar properties are expanded as individual schemas.
- **Typeless schemas `""` are pervasive**: every OAS 3.0 spec has large counts of schemas without an explicit `type`. These are typically `allOf`/`anyOf`/`oneOf` compositions or `$ref` wrappers.
- **MS Graph uses almost zero integers** (only 4!) but 1,018 `number` types. Their OData convention prefers `number` even for ID/count fields.

---

## Response Status Codes

Using `walk_responses` with `group_by=status_code`:

| Spec | 2xx | 3xx | 4xx | 5xx | default | other |
|------|----:|----:|----:|----:|--------:|------:|
| Petstore | 9 | - | 20 | - | 4 | 3 |
| Google Maps | 17 | - | 3 | - | - | - |
| NWS | 59 | 3 | - | - | 60 | 7 |
| Asana | 217 | - | 886 | 218 | - | 34 |
| Discord | 237 | - | 454 | - | - | 2 |
| Plaid | 325 | - | 1 | - | 279 | - |
| DigitalOcean | 544 | - | 993 | 1,088 | 541 | 277 |
| Stripe | 588 | - | - | - | 588 | - |
| GitHub | 1,081 | 33 | 1,269 | 141 | - | 410 |
| MS Graph | 16,098 | - | 16,098 | 16,098 | - | 1,157 |

### Error-handling styles

- **Stripe: `default` catch-all** -- every operation has exactly `200` + `default`. Simplest pattern.
- **MS Graph: wildcard ranges** -- uses `2XX`, `4XX`, `5XX` on every operation. Most systematic but least specific.
- **Discord: mixed wildcards** -- `4XX` wildcard alongside exact `429`. Combines range and exact codes.
- **GitHub: most granular** -- 25 distinct status codes including rare ones like `405`, `406`, `413`. Best client error-handling guidance.
- **Asana: exhaustive error codes** -- every operation specifies `400`, `401`, `403`, `404`, `500` individually.

---

## Parameters by Location

Using `walk_parameters` with `group_by=location`:

| Spec | path | query | header | body/formData | (unresolved $ref) |
|------|-----:|------:|-------:|--------------:|------:|
| Petstore | 9 | 4 | 1 | 11 | - |
| Google Maps | - | 77 | - | - | 114 |
| NWS | 44 | 73 | 2 | - | 92 |
| Asana | 39 | 299 | - | - | 414 |
| Discord | 170 | 91 | - | - | - |
| Plaid | 1 | - | 2 | - | 5 |
| DigitalOcean | 142 | 103 | 4 | - | 719 |
| Stripe | 436 | 951 | - | - | - |
| GitHub | 169 | 302 | - | - | 2,832 |
| MS Graph | 21,825 | 13,471 | 2,611 | - | 15,304 |

### Patterns

- **Empty location `""` = unresolved `$ref`**: parameters that reference `#/components/parameters/...` show empty location until `resolve_refs=true` is used. GitHub has 2,832 of these.
- **Petstore is the only spec with `body` and `formData`**: these are Swagger 2.0 parameter locations, replaced by `requestBody` in OAS 3.0.
- **MS Graph has 2,611 header parameters**: OData-standard headers like `ConsistencyLevel`, `$top`, `$filter`.
- **Stripe is query-heavy** (951 query params): filtering and pagination options on list endpoints.

---

## Security Schemes

Using `walk_security`:

| Spec | Scheme(s) | Type(s) |
|------|-----------|---------|
| Petstore | api_key, petstore_auth | apiKey (header), OAuth2 |
| Google Maps | ApiKeyAuth | apiKey (query) |
| NWS | apiKeyAuth, userAgent | apiKey (header) x2 |
| Asana | oauth2, personalAccessToken | OAuth2, HTTP Bearer |
| Discord | BotToken, OAuth2 | apiKey (header), OAuth2 |
| Plaid | clientId, plaidVersion, secret | apiKey (header) x3 |
| DigitalOcean | bearer_auth | HTTP Bearer |
| Stripe | basicAuth, bearerAuth | HTTP Basic, HTTP Bearer |
| GitHub | *(none defined)* | - |
| MS Graph | *(none defined)* | - |

### Patterns

- **GitHub and MS Graph define zero security schemes** despite being authenticated APIs. Auth is handled outside the spec.
- **Google Maps puts the API key in the query string** -- the only spec to do this.
- **NWS uses User-Agent as a security scheme** -- creative abuse tracking via a required header.
- **Plaid requires 3 simultaneous header keys** (clientId + secret + plaidVersion) -- multi-key auth.

---

## Response Headers

Using `walk_headers` with `group_by=name`:

| Spec | Total Headers | Top Header | Occurrences |
|------|--------:|------------|------:|
| Discord | 1,200 | X-RateLimit-Bucket/Limit/Remaining/Reset/Reset-After | 240 each |
| DigitalOcean | 1,069 | ratelimit-limit/remaining/reset | 354 each |
| GitHub | 244 | Link | 193 |
| NWS | 137 | X-Correlation-Id / X-Request-Id / X-Server-Id | 44 each |
| Stripe | 0 | - | - |
| Asana | 0 | - | - |
| Plaid | 0 | - | - |

### Patterns

- **Discord and DigitalOcean document rate-limiting headers on every response**: Discord has 5 rate-limit headers per response (1,200 total across 240 operations).
- **GitHub's `Link` header appears 193 times**: the HATEOAS pagination mechanism (`rel="next"`, `rel="last"`).
- **Stripe documents zero response headers** despite having rate limits in practice.
- **GitHub has a casing inconsistency**: both `Link` and `link`, `Location` and `location` appear as separate header names. HTTP headers are case-insensitive, so these should be merged.

---

## Top $ref Hotspots

Using `walk_refs` (top 10 per spec, ranked by reference count):

| Spec | #1 Most-Referenced | Count | Type |
|------|-------------------|------:|------|
| Stripe | `schemas/error` | 588 | schema |
| Discord | `schemas/SnowflakeType` | 554 | schema |
| DigitalOcean | `responses/server_error` | 544 | response |
| GitHub | `responses/not_found` | 487 | response |
| Plaid | `schemas/APIClientID` | 324 | schema |
| Asana | `responses/BadRequest` | 216 | response |
| NWS | `responses/Error` | 60 | response |
| Google Maps | `schemas/LatLngLiteral` | 13 | schema |

### Patterns

- **Error responses dominate**: the most-referenced component in 4/8 specs is an error response. This validates extracting errors into reusable `$ref` components.
- **Discord's SnowflakeType is referenced 554 times**: their Snowflake ID system permeates every schema.
- **GitHub's `owner` and `repo` path parameters** are referenced 480 and 479 times -- nearly every endpoint is scoped to a repository.
- **Plaid's `APIClientID` (324 refs)** reflects their triple-key auth pattern embedded in every request schema.

### GitHub -- Full Top 10

| Ref | Count | Type |
|-----|------:|------|
| responses/not_found | 487 | response |
| parameters/owner | 480 | parameter |
| parameters/repo | 479 | parameter |
| schemas/simple-user | 399 | schema |
| parameters/org | 330 | parameter |
| responses/forbidden | 318 | response |
| schemas/organization-simple-webhooks | 263 | schema |
| schemas/simple-installation | 252 | schema |
| parameters/per-page | 241 | parameter |
| schemas/enterprise-webhooks | 234 | schema |

### Discord -- Full Top 10

| Ref | Count | Type |
|-----|------:|------|
| schemas/SnowflakeType | 554 | schema |
| headers/X-RateLimit-Bucket | 239 | header |
| headers/X-RateLimit-Limit | 239 | header |
| headers/X-RateLimit-Remaining | 239 | header |
| headers/X-RateLimit-Reset | 239 | header |
| headers/X-RateLimit-Reset-After | 239 | header |
| responses/ClientErrorResponse | 227 | response |
| responses/ClientRatelimitedResponse | 227 | response |
| schemas/UserResponse | 49 | schema |
| schemas/MessageComponentTypes | 39 | schema |

---

## Tag Distribution

Using `walk_operations` with `group_by=tag` (top 15):

### GitHub (46 tags)

| Tag | Operations |
|-----|----------:|
| repos | 204 |
| actions | 184 |
| orgs | 104 |
| issues | 49 |
| codespaces | 48 |
| users | 47 |
| apps | 37 |
| activity | 32 |
| teams | 32 |
| packages | 27 |
| pulls | 27 |
| projects | 26 |
| dependabot | 22 |
| migrations | 22 |
| code-scanning | 21 |

### DigitalOcean (48 tags)

| Tag | Operations |
|-----|----------:|
| GradientAI Platform | 84 |
| Databases | 69 |
| Monitoring | 61 |
| Apps | 34 |
| Kubernetes | 28 |
| Container Registries | 19 |
| Droplets | 19 |
| Container Registry | 18 |
| Firewalls | 11 |
| Uptime | 11 |
| Load Balancers | 10 |
| VPCs | 10 |
| Block Storage | 9 |
| Functions | 9 |
| Partner Network Connect | 9 |

### Asana (42 tags)

| Tag | Operations |
|-----|----------:|
| Tasks | 27 |
| Projects | 19 |
| Goals | 12 |
| Portfolios | 12 |
| Custom fields | 8 |
| Tags | 8 |
| Users | 8 |
| Sections | 7 |
| Teams | 7 |
| Time tracking entries | 6 |
| Workspaces | 6 |
| Allocations | 5 |
| Budgets | 5 |
| Goal relationships | 5 |
| Memberships | 5 |

---

## Corpus Fingerprint

| Dimension | Smallest | Largest | Ratio |
|-----------|----------|---------|------:|
| Operations | Petstore (20) | MS Graph (16,098) | 805x |
| Schemas | Petstore (6) | MS Graph (4,294) | 716x |
| Paths | Petstore (14) | MS Graph (10,405) | 743x |
| $ref targets | Petstore (6) | Plaid (2,049) | 341x |
| Response headers | Stripe (0) | Discord (1,200) | -- |

### Coverage matrix

| Dimension | Values in Corpus |
|-----------|-----------------|
| OAS versions | 2.0, 3.0.0, 3.0.3, 3.0.4, 3.1.0 |
| Formats | JSON (6), YAML (4) |
| API styles | REST (GitHub), RPC-over-HTTP (Plaid), OData (MS Graph), read-only (NWS, Google Maps) |
| Auth types | OAuth2, Bearer, Basic, API Key (header & query), multi-key, User-Agent, undeclared |
| Error patterns | default catch-all, wildcard ranges, exhaustive codes, mixed |
