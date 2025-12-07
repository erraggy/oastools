# Top 10 publicly accessible OpenAPI specifications for integration testing

These ten specifications represent the most popular, diverse, and well-documented OpenAPI documents available for public fetching. **All URLs have been verified as directly accessible** without authentication, making them ideal for automated integration tests of OAS tooling libraries.

## Selection highlights

The curated list spans **five OAS versions** (2.0, 3.0.0, 3.0.1, 3.0.3, 3.1.0), covers **eight distinct industries**, and ranges from a minimal 20-operation reference spec to massive enterprise APIs with 900+ schemas. OpenAPI 3.2.0 was released September 2025, but no public specifications have adopted it yet—this list will need updating once early adopters emerge.

---

## 1. Stripe API

| Attribute       | Value                                                                        |
| --------------- | ---------------------------------------------------------------------------- |
| **Source URL**  | `https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.json` |
| **OAS Version** | 3.0.0                                                                        |
| **Paths**       | ~500                                                                         |
| **Operations**  | ~700                                                                         |
| **Schemas**     | **900+**                                                                     |
| **File Size**   | ~14–16 MB                                                                    |
| **Format**      | JSON (YAML also available: `spec3.yaml`)                                     |

The payments industry gold standard. Stripe's specification is the **largest well-maintained public OAS document**, featuring comprehensive coverage of payments, subscriptions, invoicing, and financial operations. Uses vendor extensions like `x-expandableFields` and `x-stripeBypassValidation`. Updated frequently with versioned releases.

---

## 2. GitHub REST API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json` |
| **OAS Version** | 3.0.3 |
| **Paths** | ~400 |
| **Operations** | 600+ |
| **Schemas** | ~700 |
| **File Size** | ~424 KB (JSON), ~210 KB (YAML) |
| **Format** | JSON (YAML also available) |

GitHub's official REST API specification covers repositories, issues, pull requests, actions, and organizational management. Features `x-github` extensions for cloud-only flags and GitHub Apps compatibility. A **3.1.0 version** is available in the `descriptions-next/` folder for testing newer OAS features.

---

## 3. DigitalOcean API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/digitalocean/openapi/main/specification/DigitalOcean-public.v2.yaml` |
| **OAS Version** | 3.0.0 |
| **Paths** | 294+ |
| **Operations** | ~400 |
| **Schemas** | ~200+ (uses external $ref files) |
| **File Size** | ~150–200 KB (base file) |
| **Format** | YAML |

Comprehensive cloud infrastructure API covering Droplets, Kubernetes, databases, load balancers, and the new GradientAI platform. Uses extensive external `$ref` references across 1,300+ files—excellent for testing spec resolution and bundling capabilities.

---

## 4. Discord API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/discord/discord-api-spec/main/specs/openapi.json` |
| **OAS Version** | **3.1.0** |
| **Paths** | ~150 |
| **Operations** | ~300 |
| **Schemas** | ~400 |
| **File Size** | ~2–5 MB |
| **Format** | JSON |

The only **OpenAPI 3.1.0** spec from a major API provider in this list. Covers Discord's HTTP API v10 for bots, guilds, channels, and interactions. Uses custom `x-discord-union` extension for oneOf/anyOf patterns. A preview spec (`openapi_preview.json`) is also available for bleeding-edge features.

---

## 5. Twilio API (Core v2010)

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/twilio/twilio-oai/main/spec/json/twilio_api_v2010.json` |
| **OAS Version** | 3.0.1 |
| **Paths** | ~130 |
| **Operations** | ~250 |
| **Schemas** | ~150 |
| **File Size** | ~1–2 MB |
| **Format** | JSON (YAML also available) |

Twilio's core voice and SMS API specification. The `twilio-oai` repository contains **40+ separate spec files** for different services (Conversations, Messaging, Verify, Video, Studio)—useful for testing multi-file OAS workflows. GA status with active maintenance.

---

## 6. Google Maps Platform API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/googlemaps/openapi-specification/main/dist/google-maps-platform-openapi3.json` |
| **OAS Version** | 3.0.3 |
| **Paths** | ~50 |
| **Operations** | ~60 |
| **Schemas** | ~150 |
| **File Size** | ~500 KB |
| **Format** | JSON (YAML also available) |

Official specification for Directions, Geocoding, Distance Matrix, Elevation, Places, Roads, and Time Zone APIs. One of the few **Google APIs with a public OpenAPI spec**. Well-documented with complete request/response examples.

---

## 7. Asana API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/Asana/openapi/master/defs/asana_oas.yaml` |
| **OAS Version** | 3.0.0 |
| **Paths** | ~120 |
| **Operations** | ~250 |
| **Schemas** | ~150 |
| **File Size** | ~405 KB |
| **Format** | YAML |

The project management platform's official specification covers tasks, projects, portfolios, goals, teams, webhooks, and custom fields. Clean YAML structure makes it excellent for testing parser implementations.

---

## 8. GitLab REST API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://docs.gitlab.com/api/openapi/openapi.yaml` |
| **OAS Version** | 3.0.1 |
| **Paths** | 76 |
| **Operations** | 154 |
| **Schemas** | 53 |
| **File Size** | ~97 KB |
| **Format** | YAML |

GitLab's API v4 specification covers badges, branches, access requests, CI variables, jobs, and migrations. Intentionally smaller (partial API coverage)—useful for testing medium-sized specs. Licensed under CC BY-SA 4.0.

---

## 9. Petstore (Swagger 2.0 Classic)

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://petstore.swagger.io/v2/swagger.json` |
| **OAS Version** | **Swagger 2.0** |
| **Paths** | 14 |
| **Operations** | 20 |
| **Definitions** | 6 |
| **File Size** | ~20 KB |
| **Format** | JSON (YAML also available at `/swagger.yaml`) |

The canonical **Swagger 2.0 reference specification**. Despite its simplicity, it remains invaluable for testing backward compatibility with the older spec format. Includes OAuth 2.0 and API key authentication patterns.

---

## 10. Slack Web API

| Attribute | Value |
|-----------|-------|
| **Source URL** | `https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2.json` |
| **OAS Version** | **Swagger 2.0** |
| **Paths** | ~150 |
| **Operations** | ~150 |
| **Definitions** | ~150 |
| **File Size** | ~2–3 MB |
| **Format** | JSON |

A large, real-world **Swagger 2.0 specification** covering messaging, channels, users, and files. Repository was archived March 2024 but remains accessible. Provides a meaningful second Swagger 2.0 test case beyond Petstore, with substantial complexity.

---

## Quick reference table

| # | API | OAS Version | Paths | Operations | Schemas | Size | Format |
|---|-----|-------------|-------|------------|---------|------|--------|
| 1 | Stripe | 3.0.0 | ~500 | ~700 | 900+ | 14–16 MB | JSON |
| 2 | GitHub | 3.0.3 | ~400 | 600+ | ~700 | ~424 KB | JSON |
| 3 | DigitalOcean | 3.0.0 | 294 | ~400 | ~200 | ~200 KB | YAML |
| 4 | Discord | **3.1.0** | ~150 | ~300 | ~400 | 2–5 MB | JSON |
| 5 | Twilio | 3.0.1 | ~130 | ~250 | ~150 | 1–2 MB | JSON |
| 6 | Google Maps | 3.0.3 | ~50 | ~60 | ~150 | ~500 KB | JSON |
| 7 | Asana | 3.0.0 | ~120 | ~250 | ~150 | ~405 KB | YAML |
| 8 | GitLab | 3.0.1 | 76 | 154 | 53 | ~97 KB | YAML |
| 9 | Petstore | **2.0** | 14 | 20 | 6 | ~20 KB | JSON |
| 10 | Slack | **2.0** | ~150 | ~150 | ~150 | 2–3 MB | JSON |

---

## Version coverage summary

The selection achieves excellent OAS version diversity:

- **Swagger 2.0**: Petstore (minimal), Slack (large real-world)
- **OpenAPI 3.0.0**: Stripe, DigitalOcean, Asana
- **OpenAPI 3.0.1**: Twilio, GitLab
- **OpenAPI 3.0.3**: GitHub, Google Maps
- **OpenAPI 3.1.0**: Discord

**OpenAPI 3.2.0 note**: Released September 23, 2025, but no public API specifications have adopted it yet. Key 3.2.0 features include QUERY HTTP method support, streaming media types, tag hierarchy, and `components/mediaTypes`. Monitor the OpenAPI Initiative's examples repository for future 3.2.0 samples.

---

## Additional resources for edge-case testing

For comprehensive OAS tooling validation, consider supplementing with:

- **GitHub 3.1.0 spec**: `https://raw.githubusercontent.com/github/rest-api-description/main/descriptions-next/api.github.com/api.github.com.json`
- **Petstore 3.0 extended**: `https://raw.githubusercontent.com/openapitools/openapi-generator/master/modules/openapi-generator/src/test/resources/3_0/petstore.yaml`
- **Microsoft Graph** (very large): `https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml`
- **Box API**: `https://raw.githubusercontent.com/box/box-openapi/main/openapi.json`

All primary URLs verified accessible December 2025.