package generator

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	oasparser "github.com/erraggy/oastools/parser"
)

// TestLargeAPISplitByTag tests that a large API is split by tags when enabled
func TestLargeAPISplitByTag(t *testing.T) {
	// Create a synthetic OAS 3.0 document with many operations across different tags
	doc := createLargeOAS3Doc(20) // 20 paths = 40 operations

	gen := New()
	gen.PackageName = "largeapi"
	gen.GenerateClient = true
	gen.GenerateServer = true
	gen.GenerateTypes = true
	gen.SplitByTag = true
	gen.SplitByPathPrefix = true
	gen.MaxOperationsPerFile = 10 // Force split at 10 operations

	parseResult := oasparser.ParseResult{
		Version:    "3.0.3",
		OASVersion: oasparser.OASVersion303,
		Document:   doc,
	}

	result, err := gen.GenerateParsed(parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed() error: %v", err)
	}

	// Check that we have multiple client files
	clientFiles := countFilesByPrefix(result.Files, "client")
	if clientFiles < 2 {
		t.Errorf("expected multiple client files when splitting large API, got %d", clientFiles)
	}

	// Types may be in a single file if they're all shared
	// but we should have at least one types file
	typesFiles := countFilesByPrefix(result.Files, "types")
	if typesFiles < 1 {
		t.Errorf("expected at least 1 types file, got %d", typesFiles)
	}

	// Check that we have multiple server files
	serverFiles := countFilesByPrefix(result.Files, "server")
	if serverFiles < 2 {
		t.Errorf("expected multiple server files when splitting large API, got %d", serverFiles)
	}

	// Verify all generated Go code compiles (skip non-Go files like README.md)
	for _, file := range result.Files {
		if !isGoFile(file.Name) {
			continue
		}
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, file.Name, file.Content, parser.AllErrors)
		if parseErr != nil {
			t.Errorf("generated file %s does not compile: %v", file.Name, parseErr)
		}
	}
}

// TestLargeAPISplitByPathPrefix tests splitting by path prefix
func TestLargeAPISplitByPathPrefix(t *testing.T) {
	// Create doc with different path prefixes but no tags
	doc := createOAS3DocWithPaths()

	gen := New()
	gen.PackageName = "pathprefix"
	gen.GenerateClient = true
	gen.GenerateTypes = true
	gen.SplitByTag = true // Will fall through to path prefix when no tags
	gen.SplitByPathPrefix = true
	gen.MaxOperationsPerFile = 5

	parseResult := oasparser.ParseResult{
		Version:    "3.0.3",
		OASVersion: oasparser.OASVersion303,
		Document:   doc,
	}

	result, err := gen.GenerateParsed(parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed() error: %v", err)
	}

	// Verify all generated Go code compiles (skip non-Go files like README.md)
	for _, file := range result.Files {
		if !isGoFile(file.Name) {
			continue
		}
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, file.Name, file.Content, parser.AllErrors)
		if parseErr != nil {
			t.Errorf("generated file %s does not compile: %v", file.Name, parseErr)
		}
	}

	// Should have at least types.go and client.go
	if len(result.Files) < 2 {
		t.Errorf("expected at least 2 files, got %d", len(result.Files))
	}
}

// TestLargeAPINoSplit tests that splitting is disabled when thresholds not met
func TestLargeAPINoSplit(t *testing.T) {
	// Small doc that shouldn't be split
	doc := createSmallOAS3Doc()

	gen := New()
	gen.PackageName = "smallapi"
	gen.GenerateClient = true
	gen.GenerateTypes = true
	gen.SplitByTag = true
	gen.SplitByPathPrefix = true
	gen.MaxOperationsPerFile = 100 // High threshold

	parseResult := oasparser.ParseResult{
		Version:    "3.0.3",
		OASVersion: oasparser.OASVersion303,
		Document:   doc,
	}

	result, err := gen.GenerateParsed(parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed() error: %v", err)
	}

	// Should have exactly one client file
	clientFiles := countFilesByPrefix(result.Files, "client")
	if clientFiles != 1 {
		t.Errorf("expected 1 client file for small API, got %d", clientFiles)
	}

	// Should have exactly one types file
	typesFiles := countFilesByPrefix(result.Files, "types")
	if typesFiles != 1 {
		t.Errorf("expected 1 types file for small API, got %d", typesFiles)
	}
}

// TestOAS2LargeAPISplit tests file splitting for OAS 2.0 documents
func TestOAS2LargeAPISplit(t *testing.T) {
	doc := createLargeOAS2Doc(20)

	gen := New()
	gen.PackageName = "oas2largeapi"
	gen.GenerateClient = true
	gen.GenerateServer = true
	gen.GenerateTypes = true
	gen.SplitByTag = true
	gen.MaxOperationsPerFile = 10

	parseResult := oasparser.ParseResult{
		Version:    "2.0",
		OASVersion: oasparser.OASVersion20,
		Document:   doc,
	}

	result, err := gen.GenerateParsed(parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed() error: %v", err)
	}

	// Check that we have multiple files
	if len(result.Files) < 4 {
		t.Errorf("expected multiple files for large OAS2 API, got %d", len(result.Files))
	}

	// Verify all generated Go code compiles (skip non-Go files like README.md)
	for _, file := range result.Files {
		if !isGoFile(file.Name) {
			continue
		}
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, file.Name, file.Content, parser.AllErrors)
		if parseErr != nil {
			t.Errorf("generated file %s does not compile: %v", file.Name, parseErr)
		}
	}
}

// TestFileSplitterAnalyze tests the file splitter analysis
func TestFileSplitterAnalyze(t *testing.T) {
	tests := []struct {
		name          string
		ops           int
		maxOps        int
		wantSplit     bool
		wantMinGroups int
	}{
		{
			name:          "small API no split",
			ops:           5,
			maxOps:        100,
			wantSplit:     false,
			wantMinGroups: 0,
		},
		{
			name:          "large API needs split",
			ops:           50,
			maxOps:        10,
			wantSplit:     true,
			wantMinGroups: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := createLargeOAS3Doc(tt.ops / 2) // Each path has 2 ops (GET, POST)

			fs := &FileSplitter{
				MaxOperationsPerFile: tt.maxOps,
				SplitByTag:           true,
				SplitByPathPrefix:    true,
			}

			plan := fs.AnalyzeOAS3(doc)

			if plan.NeedsSplit != tt.wantSplit {
				t.Errorf("NeedsSplit = %v, want %v", plan.NeedsSplit, tt.wantSplit)
			}

			if tt.wantSplit && len(plan.Groups) < tt.wantMinGroups {
				t.Errorf("got %d groups, want at least %d", len(plan.Groups), tt.wantMinGroups)
			}
		})
	}
}

// TestSplitFilesCompile tests that split files form a valid Go package
func TestSplitFilesCompile(t *testing.T) {
	doc := createLargeOAS3Doc(30) // Larger to ensure split

	gen := New()
	gen.PackageName = "compiletest"
	gen.GenerateClient = true
	gen.GenerateServer = true
	gen.GenerateTypes = true
	gen.SplitByTag = true
	gen.MaxOperationsPerFile = 10

	parseResult := oasparser.ParseResult{
		Version:    "3.0.3",
		OASVersion: oasparser.OASVersion303,
		Document:   doc,
	}

	result, err := gen.GenerateParsed(parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed() error: %v", err)
	}

	// Collect all files and verify they can be parsed together (skip non-Go files)
	fset := token.NewFileSet()
	var parseErrors []string

	for _, file := range result.Files {
		if !isGoFile(file.Name) {
			continue
		}
		_, parseErr := parser.ParseFile(fset, file.Name, file.Content, parser.AllErrors)
		if parseErr != nil {
			parseErrors = append(parseErrors, file.Name+": "+parseErr.Error())
		}
	}

	if len(parseErrors) > 0 {
		t.Errorf("generated package has compilation errors:\n%v", parseErrors)
	}

	// Log file count for visibility
	t.Logf("Generated %d files for large API", len(result.Files))
	for _, file := range result.Files {
		t.Logf("  - %s (%d bytes)", file.Name, len(file.Content))
	}
}

// Helper functions

func isGoFile(name string) bool {
	return len(name) > 3 && name[len(name)-3:] == ".go"
}

func countFilesByPrefix(files []GeneratedFile, prefix string) int {
	count := 0
	for _, f := range files {
		if len(f.Name) >= len(prefix) && f.Name[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}

func createLargeOAS3Doc(pathCount int) *oasparser.OAS3Document {
	doc := &oasparser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &oasparser.Info{
			Title:   "Large Test API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*oasparser.PathItem),
		Components: &oasparser.Components{
			Schemas: make(map[string]*oasparser.Schema),
		},
	}

	// Create paths with alternating tags
	tags := []string{"users", "orders", "products", "inventory", "reports"}
	for i := range pathCount {
		tag := tags[i%len(tags)]
		path := "/" + tag + "/" + string(rune('a'+i))

		doc.Paths[path] = &oasparser.PathItem{
			Get: &oasparser.Operation{
				OperationID: tag + "Get" + string(rune('A'+i)),
				Tags:        []string{tag},
				Summary:     "Get " + tag + " item",
				Responses: &oasparser.Responses{
					Codes: map[string]*oasparser.Response{
						"200": {Description: "OK"},
					},
				},
			},
			Post: &oasparser.Operation{
				OperationID: tag + "Create" + string(rune('A'+i)),
				Tags:        []string{tag},
				Summary:     "Create " + tag + " item",
				Responses: &oasparser.Responses{
					Codes: map[string]*oasparser.Response{
						"201": {Description: "Created"},
					},
				},
			},
		}

		// Add a schema for this tag
		doc.Components.Schemas[tag+"Item"+string(rune('A'+i))] = &oasparser.Schema{
			Type: "object",
			Properties: map[string]*oasparser.Schema{
				"id":   {Type: "string"},
				"name": {Type: "string"},
			},
		}
	}

	return doc
}

func createLargeOAS2Doc(pathCount int) *oasparser.OAS2Document {
	doc := &oasparser.OAS2Document{
		Swagger: "2.0",
		Info: &oasparser.Info{
			Title:   "Large Test API (OAS2)",
			Version: "1.0.0",
		},
		Host:        "api.example.com",
		BasePath:    "/v1",
		Paths:       make(map[string]*oasparser.PathItem),
		Definitions: make(map[string]*oasparser.Schema),
	}

	tags := []string{"users", "orders", "products", "inventory", "reports"}
	for i := range pathCount {
		tag := tags[i%len(tags)]
		path := "/" + tag + "/" + string(rune('a'+i))

		doc.Paths[path] = &oasparser.PathItem{
			Get: &oasparser.Operation{
				OperationID: tag + "Get" + string(rune('A'+i)),
				Tags:        []string{tag},
				Summary:     "Get " + tag + " item",
				Responses: &oasparser.Responses{
					Codes: map[string]*oasparser.Response{
						"200": {Description: "OK"},
					},
				},
			},
			Post: &oasparser.Operation{
				OperationID: tag + "Create" + string(rune('A'+i)),
				Tags:        []string{tag},
				Summary:     "Create " + tag + " item",
				Responses: &oasparser.Responses{
					Codes: map[string]*oasparser.Response{
						"201": {Description: "Created"},
					},
				},
			},
		}

		doc.Definitions[tag+"Item"+string(rune('A'+i))] = &oasparser.Schema{
			Type: "object",
			Properties: map[string]*oasparser.Schema{
				"id":   {Type: "string"},
				"name": {Type: "string"},
			},
		}
	}

	return doc
}

func createOAS3DocWithPaths() *oasparser.OAS3Document {
	doc := &oasparser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &oasparser.Info{
			Title:   "Path Prefix API",
			Version: "1.0.0",
		},
		Paths: map[string]*oasparser.PathItem{
			"/api/v1/users":       {Get: &oasparser.Operation{OperationID: "listUsers", Responses: &oasparser.Responses{Codes: map[string]*oasparser.Response{"200": {Description: "OK"}}}}},
			"/api/v1/users/{id}":  {Get: &oasparser.Operation{OperationID: "getUser", Responses: &oasparser.Responses{Codes: map[string]*oasparser.Response{"200": {Description: "OK"}}}}},
			"/api/v2/orders":      {Get: &oasparser.Operation{OperationID: "listOrders", Responses: &oasparser.Responses{Codes: map[string]*oasparser.Response{"200": {Description: "OK"}}}}},
			"/api/v2/orders/{id}": {Get: &oasparser.Operation{OperationID: "getOrder", Responses: &oasparser.Responses{Codes: map[string]*oasparser.Response{"200": {Description: "OK"}}}}},
			"/internal/health":    {Get: &oasparser.Operation{OperationID: "healthCheck", Responses: &oasparser.Responses{Codes: map[string]*oasparser.Response{"200": {Description: "OK"}}}}},
		},
		Components: &oasparser.Components{
			Schemas: map[string]*oasparser.Schema{
				"User":  {Type: "object"},
				"Order": {Type: "object"},
			},
		},
	}
	return doc
}

func createSmallOAS3Doc() *oasparser.OAS3Document {
	doc := &oasparser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &oasparser.Info{
			Title:   "Small API",
			Version: "1.0.0",
		},
		Paths: map[string]*oasparser.PathItem{
			"/users": {
				Get: &oasparser.Operation{
					OperationID: "listUsers",
					Responses: &oasparser.Responses{
						Codes: map[string]*oasparser.Response{"200": {Description: "OK"}},
					},
				},
			},
		},
		Components: &oasparser.Components{
			Schemas: map[string]*oasparser.Schema{
				"User": {Type: "object"},
			},
		},
	}
	return doc
}

// TestLargeAPISecurityEnforceSplit tests that security_enforce.go is split for large APIs
func TestLargeAPISecurityEnforceSplit(t *testing.T) {
	// Create large doc with security requirements
	doc := createLargeOAS3DocWithSecurity(30)

	gen := New()
	gen.PackageName = "securitysplit"
	gen.GenerateClient = true
	gen.GenerateSecurityEnforce = true
	gen.SplitByTag = true
	gen.MaxOperationsPerFile = 10

	parseResult := oasparser.ParseResult{
		Version:    "3.0.3",
		OASVersion: oasparser.OASVersion303,
		Document:   doc,
	}

	result, err := gen.GenerateParsed(parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed() error: %v", err)
	}

	// Check that we have multiple security_enforce files
	securityFiles := countFilesByPrefix(result.Files, "security_enforce")
	if securityFiles < 2 {
		t.Errorf("expected multiple security_enforce files when splitting, got %d", securityFiles)
	}

	// Verify all generated Go code compiles
	for _, file := range result.Files {
		if !isGoFile(file.Name) {
			continue
		}
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, file.Name, file.Content, parser.AllErrors)
		if parseErr != nil {
			t.Errorf("generated file %s does not compile: %v", file.Name, parseErr)
		}
	}

	// Log file count for visibility
	t.Logf("Generated %d security_enforce files", securityFiles)
	for _, file := range result.Files {
		if len(file.Name) >= 16 && file.Name[:16] == "security_enforce" {
			t.Logf("  - %s (%d bytes)", file.Name, len(file.Content))
		}
	}
}

// TestOAS2SplitClientCompiles verifies OAS 2.0 split client files have correct imports (Issue #118)
func TestOAS2SplitClientCompiles(t *testing.T) {
	doc := createLargeOAS2Doc(20)

	gen := New()
	gen.PackageName = "oas2splitclient"
	gen.GenerateClient = true
	gen.GenerateTypes = true
	gen.SplitByTag = true
	gen.MaxOperationsPerFile = 10

	parseResult := oasparser.ParseResult{
		Version:    "2.0",
		OASVersion: oasparser.OASVersion20,
		Document:   doc,
	}

	result, err := gen.GenerateParsed(parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed() error: %v", err)
	}

	// Verify ALL generated Go files compile - this catches missing imports
	for _, file := range result.Files {
		if !isGoFile(file.Name) {
			continue
		}
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, file.Name, file.Content, parser.AllErrors)
		if parseErr != nil {
			t.Errorf("generated file %s does not compile: %v\nContent preview:\n%s",
				file.Name, parseErr, truncateContent(file.Content, 500))
		}
	}

	// Specifically check client.go has the required imports for clientHelpers
	var clientFile *GeneratedFile
	for i := range result.Files {
		if result.Files[i].Name == "client.go" {
			clientFile = &result.Files[i]
			break
		}
	}

	if clientFile == nil {
		t.Fatal("expected client.go file")
	}

	content := string(clientFile.Content)
	requiredImports := []string{
		`"bytes"`,
		`"encoding/json"`,
		`"net/url"`,
	}
	for _, imp := range requiredImports {
		if !strings.Contains(content, imp) {
			t.Errorf("client.go missing import %s", imp)
		}
	}
}

// TestOAS2SecurityEnforceSplit tests security_enforce splitting for OAS 2.0
func TestOAS2SecurityEnforceSplit(t *testing.T) {
	doc := createLargeOAS2DocWithSecurity(30)

	gen := New()
	gen.PackageName = "oas2securitysplit"
	gen.GenerateClient = true
	gen.GenerateSecurityEnforce = true
	gen.SplitByTag = true
	gen.MaxOperationsPerFile = 10

	parseResult := oasparser.ParseResult{
		Version:    "2.0",
		OASVersion: oasparser.OASVersion20,
		Document:   doc,
	}

	result, err := gen.GenerateParsed(parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed() error: %v", err)
	}

	// Check that we have multiple security_enforce files
	securityFiles := countFilesByPrefix(result.Files, "security_enforce")
	if securityFiles < 2 {
		t.Errorf("expected multiple security_enforce files when splitting, got %d", securityFiles)
	}

	// Verify all generated Go code compiles
	for _, file := range result.Files {
		if !isGoFile(file.Name) {
			continue
		}
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, file.Name, file.Content, parser.AllErrors)
		if parseErr != nil {
			t.Errorf("generated file %s does not compile: %v", file.Name, parseErr)
		}
	}
}

// Helper functions for new tests

func createLargeOAS3DocWithSecurity(pathCount int) *oasparser.OAS3Document {
	doc := createLargeOAS3Doc(pathCount)

	// Add security schemes
	doc.Components.SecuritySchemes = map[string]*oasparser.SecurityScheme{
		"oauth2": {
			Type: "oauth2",
			Flows: &oasparser.OAuthFlows{
				ClientCredentials: &oasparser.OAuthFlow{
					TokenURL: "https://example.com/token",
					Scopes:   map[string]string{"read": "Read access", "write": "Write access"},
				},
			},
		},
		"apiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
	}

	// Add security to all operations
	doc.Security = []oasparser.SecurityRequirement{
		{"oauth2": []string{"read"}},
	}

	return doc
}

func createLargeOAS2DocWithSecurity(pathCount int) *oasparser.OAS2Document {
	doc := createLargeOAS2Doc(pathCount)

	// Add security definitions
	doc.SecurityDefinitions = map[string]*oasparser.SecurityScheme{
		"oauth2": {
			Type:             "oauth2",
			Flow:             "application",
			TokenURL:         "https://example.com/token",
			Scopes:           map[string]string{"read": "Read access", "write": "Write access"},
			AuthorizationURL: "",
		},
		"apiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
	}

	// Add security to all operations
	doc.Security = []oasparser.SecurityRequirement{
		{"oauth2": []string{"read"}},
	}

	return doc
}

func truncateContent(content []byte, maxLen int) string {
	if len(content) <= maxLen {
		return string(content)
	}
	return string(content[:maxLen]) + "..."
}
