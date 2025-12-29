# PetStore Example

A comprehensive code generation example demonstrating the full capabilities of oastools generator, including client, server, security, and credential management.

## What You'll Learn

- How to generate complete Go client and server code from an OpenAPI specification
- Framework variants: stdlib (net/http) and Chi router
- Security features: OAuth2 flows, OIDC discovery, credential management

## Prerequisites

- Go 1.24+
- oastools CLI installed (`go install github.com/erraggy/oastools/cmd/oastools@latest`)

## Variants

| Directory | Router | Description |
|-----------|--------|-------------|
| [stdlib/](stdlib/) | net/http | Standard library HTTP server with path matching |
| [chi/](chi/) | go-chi/chi | Chi router with native path parameter extraction |

## Source Specification

- **File:** [spec/petstore-v2.json](spec/petstore-v2.json)
- **API:** Swagger Petstore API
- **OAS Version:** 2.0 (Swagger)
- **Source URL:** https://petstore.swagger.io/v2/swagger.json

## Generated Features

Both variants include:

| Feature | Files Generated |
|---------|----------------|
| Type definitions | `types.go` |
| HTTP client | `client.go` |
| Server interface | `server.go` |
| Router | `server_router.go` |
| Request binding | `server_binder.go` |
| Response writers | `server_responses.go` |
| Validation middleware | `server_middleware.go` |
| Stub implementations | `server_stubs.go` |
| OAuth2 flows | `oauth2_petstore_auth.go` |
| OIDC discovery | `oidc_discovery.go` |
| Credential management | `credentials.go` |
| Security enforcement | `security_enforce.go`, `security_helpers.go` |

## Regeneration

**stdlib variant:**
```bash
oastools generate --server --server-all --client --security-enforce \
  --oauth2-flows --oidc-discovery --credential-mgmt \
  -p petstore -o examples/petstore/stdlib \
  examples/petstore/spec/petstore-v2.json
```

**chi variant:**
```bash
oastools generate --server --server-all --server-router chi --client \
  --security-enforce --oauth2-flows --oidc-discovery --credential-mgmt \
  -p petstore -o examples/petstore/chi \
  examples/petstore/spec/petstore-v2.json
```

## Next Steps

- [Generator Documentation](../../packages/generator/)
- [Quickstart Example](../quickstart/) - Minimal introduction
- [Validation Pipeline Example](../validation-pipeline/) - Parse and validate workflow

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
