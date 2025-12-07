# OpenAPI Specification Corpus

This directory contains cached copies of public OpenAPI specifications used for integration testing.

## Downloading the Corpus

Run the following command to download all specifications:

```bash
make corpus-download
```

## Specifications

| # | API | OAS Version | Format | Size | Valid | Source |
|---|-----|-------------|--------|------|-------|--------|
| 1 | Petstore | 2.0 | JSON | ~20 KB | Yes | petstore.swagger.io |
| 2 | DigitalOcean | 3.0.0 | YAML | ~200 KB | Yes | github.com/digitalocean |
| 3 | Asana | 3.0.0 | YAML | ~405 KB | No | github.com/Asana |
| 4 | Google Maps | 3.0.3 | JSON | ~500 KB | No | github.com/googlemaps |
| 5 | US NWS | 3.0.3 | JSON | ~800 KB | No | api.weather.gov |
| 6 | Plaid | 3.0.0 | YAML | ~1.2 MB | No | github.com/plaid |
| 7 | Discord | 3.1.0 | JSON | ~3 MB | Yes | github.com/discord |
| 8 | GitHub | 3.0.3 | JSON | ~5 MB | No | github.com/github |
| 9 | Stripe | 3.0.0 | JSON | ~14 MB | Yes | github.com/stripe |
| 10 | Microsoft Graph | 3.0.4 | YAML | ~15 MB | No | github.com/microsoftgraph |

## Files

- `petstore-swagger.json` - Swagger 2.0 reference implementation
- `digitalocean-public.v2.yaml` - Cloud infrastructure API
- `asana-oas.yaml` - Project management API
- `google-maps-platform.json` - Google Maps Platform APIs
- `nws-openapi.json` - US National Weather Service API
- `plaid-2020-09-14.yml` - FinTech banking API
- `discord-openapi.json` - Discord HTTP API v10 (OAS 3.1.0)
- `github-api.json` - GitHub REST API
- `stripe-spec3.json` - Stripe payments API
- `msgraph-openapi.yaml` - Microsoft Graph API

## Notes

- **Valid** column indicates whether the spec passes `oastools validate --strict`
- Files are not committed to Git (see `.gitignore`)
- Specifications may change over time; re-run `make corpus-download` to update
- Large files (Stripe, Microsoft Graph) may take longer to parse

## Selection Criteria

These specifications were selected based on:
1. **Popularity** - Widely-used APIs with active developer communities
2. **Size** - Range from minimal (20KB) to massive (15MB)
3. **Diversity** - Multiple OAS versions, formats, and domains
4. **Accessibility** - All publicly available without authentication

See `planning/Top10-Public-OAS-Docs-CombinedSummary.md` for detailed research.
