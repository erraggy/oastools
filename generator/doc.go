// Package generator provides Go code generation from OpenAPI Specification documents.
//
// The generator creates idiomatic Go code for API clients and server stubs from
// OAS 2.0 and OAS 3.x specifications. Generated code emphasizes type safety,
// proper error handling, and clean interfaces.
//
// # Quick Start
//
// Generate a client using functional options:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("openapi.yaml"),
//		generator.WithPackageName("petstore"),
//		generator.WithClient(true),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := result.WriteFiles("./generated"); err != nil {
//		log.Fatal(err)
//	}
//
// Or use a reusable Generator instance:
//
//	g := generator.New()
//	g.PackageName = "petstore"
//	g.GenerateClient = true
//	g.GenerateServer = true
//	result, _ := g.Generate("openapi.yaml")
//	result.WriteFiles("./generated")
//
// # Generation Modes
//
// The generator supports three modes:
//   - Client: HTTP client with methods for each operation
//   - Server: Interface definitions and request/response types
//   - Types: Schema-only generation (models)
//
// # Security Generation
//
// When generating clients, the generator automatically creates security helper
// functions based on the security schemes defined in the OpenAPI specification.
// These helpers are generated as ClientOption functions that configure authentication.
//
// Security scheme types and generated helpers:
//   - apiKey (header): With{Name}APIKey(key string) ClientOption
//   - apiKey (query): With{Name}APIKeyQuery(key string) ClientOption
//   - apiKey (cookie): With{Name}APIKeyCookie(key string) ClientOption
//   - http/basic: With{Name}BasicAuth(username, password string) ClientOption
//   - http/bearer: With{Name}BearerToken(token string) ClientOption
//   - oauth2: With{Name}OAuth2Token(token string) ClientOption
//   - openIdConnect: With{Name}Token(token string) ClientOption
//
// Enable security generation with WithSecurity(true) or WithGenerateSecurity(true).
//
// # OAuth2 Flow Generation
//
// For APIs using OAuth2, the generator can create full OAuth2 client implementations
// with support for all standard flows:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("openapi.yaml"),
//		generator.WithPackageName("api"),
//		generator.WithClient(true),
//		generator.WithOAuth2Flows(true),
//	)
//
// Generated OAuth2 code includes:
//   - OAuth2Config struct for client configuration
//   - OAuth2Token struct with access/refresh tokens
//   - OAuth2Client with flow-specific methods
//   - GetAuthorizationURL() for authorization code flow
//   - ExchangeCode() to exchange auth codes for tokens
//   - GeneratePKCEChallenge() for PKCE challenge generation (RFC 7636)
//   - GetAuthorizationURLWithPKCE() for secure authorization with PKCE
//   - ExchangeCodeWithPKCE() to exchange auth codes with code verifier
//   - GetClientCredentialsToken() for client credentials flow
//   - GetPasswordToken() for password flow (with warnings)
//   - GetImplicitAuthorizationURL() for implicit flow (deprecated)
//   - RefreshToken() for token refresh
//   - WithOAuth2AutoRefresh() ClientOption for automatic token refresh
//
// # Credential Management
//
// The generator can create credential provider interfaces for flexible authentication:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("openapi.yaml"),
//		generator.WithPackageName("api"),
//		generator.WithClient(true),
//		generator.WithCredentialMgmt(true),
//	)
//
// Generated credential code includes:
//   - CredentialProvider interface
//   - MemoryCredentialProvider for testing
//   - EnvCredentialProvider for environment variables
//   - CredentialChain for fallback providers
//   - WithCredentialProvider() ClientOption
//
// # Security Enforcement
//
// Generate security validation middleware for server implementations:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("openapi.yaml"),
//		generator.WithPackageName("api"),
//		generator.WithServer(true),
//		generator.WithSecurityEnforce(true),
//	)
//
// Generated enforcement code includes:
//   - SecurityRequirement struct
//   - OperationSecurityRequirements map
//   - SecurityValidator for request validation
//   - RequireSecurityMiddleware for enforcement
//
// # OpenID Connect Discovery
//
// For APIs using OpenID Connect, generate automatic discovery and configuration:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("openapi.yaml"),
//		generator.WithPackageName("api"),
//		generator.WithClient(true),
//		generator.WithOIDCDiscovery(true),
//	)
//
// Generated OIDC code includes:
//   - OIDCConfiguration struct
//   - OIDCDiscoveryClient for .well-known discovery
//   - NewOAuth2ClientFromOIDC() helper
//
// # File Splitting for Large APIs
//
// For large APIs (like Microsoft Graph), the generator can split output across
// multiple files based on operation tags or path prefixes:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("large-api.yaml"),
//		generator.WithPackageName("api"),
//		generator.WithClient(true),
//		generator.WithMaxLinesPerFile(2000),
//		generator.WithSplitByTag(true),
//	)
//
// File splitting options:
//   - WithMaxLinesPerFile(n): Maximum lines per generated file (default: 2000)
//   - WithMaxTypesPerFile(n): Maximum types per file (default: 200)
//   - WithMaxOperationsPerFile(n): Maximum operations per file (default: 100)
//   - WithSplitByTag(bool): Split by operation tags (default: true)
//   - WithSplitByPathPrefix(bool): Split by path prefix (default: true)
//
// # README Generation
//
// Generate a README.md file documenting the generated code:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("openapi.yaml"),
//		generator.WithPackageName("api"),
//		generator.WithClient(true),
//		generator.WithReadme(true),
//	)
//
// The generated README includes:
//   - API overview and version info
//   - Generated file descriptions
//   - Security configuration examples
//   - Regeneration command
//
// # Server Extensions
//
// When generating server code, additional extensions provide a complete server
// framework with runtime validation, request binding, routing, and testing support:
//
//	result, err := generator.GenerateWithOptions(
//		generator.WithFilePath("openapi.yaml"),
//		generator.WithPackageName("api"),
//		generator.WithServer(true),
//		generator.WithServerAll(), // Enable all server extensions
//	)
//
// Server extension options:
//   - WithServerResponses(bool): Typed response writers with Status*() methods
//   - WithServerBinder(bool): Request parameter binding using httpvalidator
//   - WithServerMiddleware(bool): Validation middleware for request/response validation
//   - WithServerRouter(string): HTTP router generation ("stdlib", "chi")
//   - WithServerStubs(bool): Configurable stub implementations for testing
//   - WithServerAll(): Enable all server extensions at once
//
// Generated server extension files:
//   - server_responses.go: Per-operation response types with WriteTo() methods
//   - server_binder.go: RequestBinder with Bind{Operation}Request() methods
//   - server_middleware.go: ValidationMiddleware with configurable error handling
//   - server_router.go: ServerRouter implementing http.Handler
//   - server_stubs.go: StubServer with configurable function fields for testing
//
// Example router setup with error logging:
//
//	router, _ := NewServerRouter(server, parsed,
//		WithMiddleware(ValidationMiddleware(parsed)),
//		WithErrorHandler(func(r *http.Request, err error) {
//			log.Printf("Error: %s %s: %v", r.Method, r.URL.Path, err)
//		}),
//	)
//	http.ListenAndServe(":8080", router)
//
// # Type Mapping
//
// OpenAPI types are mapped to Go types as follows:
//   - string → string (with format handling: date-time→time.Time, uuid→string, etc.)
//   - integer → int64 (int32 for format: int32)
//   - number → float64 (float32 for format: float)
//   - boolean → bool
//   - array → []T
//   - object → struct or map[string]T
//
// Optional fields use pointers, and nullable fields in OAS 3.1+ are handled
// with pointer types or generic Option[T] types (configurable).
//
// # Generated Files
//
// The generator produces the following files:
//   - types.go: Model structs from components/schemas
//   - client.go: HTTP client (when GenerateClient is true)
//   - server.go: Server interface (when GenerateServer is true)
//   - security_helpers.go: Security ClientOption functions (when GenerateSecurity is true)
//   - {name}_oauth2.go: OAuth2 client for each OAuth2 scheme (when GenerateOAuth2Flows is true)
//   - credentials.go: Credential provider interfaces (when GenerateCredentialMgmt is true)
//   - security_enforce.go: Security validation (when GenerateSecurityEnforce is true)
//   - oidc_discovery.go: OIDC discovery client (when GenerateOIDCDiscovery is true)
//   - README.md: Documentation (when GenerateReadme is true)
//   - server_responses.go: Response types (when ServerResponses or ServerAll is set)
//   - server_binder.go: Request binding (when ServerBinder or ServerAll is set)
//   - server_middleware.go: Validation middleware (when ServerMiddleware or ServerAll is set)
//   - server_router.go: HTTP router (when ServerRouter or ServerAll is set)
//   - server_stubs.go: Test stubs (when ServerStubs or ServerAll is set)
//
// See the exported GenerateResult and GenerateIssue types for complete details.
//
// # Related Packages
//
// The generator integrates with other oastools packages:
//   - [github.com/erraggy/oastools/parser] - Parse specifications before code generation
//   - [github.com/erraggy/oastools/validator] - Validate specifications before generation
//   - [github.com/erraggy/oastools/fixer] - Fix common validation errors before generation
//   - [github.com/erraggy/oastools/converter] - Convert OAS versions before generation
//   - [github.com/erraggy/oastools/joiner] - Join specifications before generation
//   - [github.com/erraggy/oastools/differ] - Compare specifications to understand changes
//   - [github.com/erraggy/oastools/builder] - Programmatically build specifications
package generator
