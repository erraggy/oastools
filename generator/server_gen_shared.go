package generator

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/parser"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Server Generation Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// wrapperSuffixes is the ordered list of suffixes to try when naming server wrapper types.
var wrapperSuffixes = []string{"Request", "Input", "Req"}

// resolveWrapperName picks a wrapper type name that doesn't collide with schema types.
// It tries {methodName}Request, then Input, then Req, then numeric fallback.
func resolveWrapperName(methodName string, schemaTypes map[string]bool) string {
	for _, suffix := range wrapperSuffixes {
		candidate := methodName + suffix
		if !schemaTypes[candidate] {
			return candidate
		}
	}
	const maxAttempts = 1000
	for i := 2; i <= maxAttempts; i++ {
		candidate := fmt.Sprintf("%sRequest%d", methodName, i)
		if !schemaTypes[candidate] {
			return candidate
		}
	}
	// All candidates exhausted — practically unreachable.
	// Return a novel numeric suffix so generated code fails with a clear compile error
	// rather than silently reusing a known-colliding name.
	return fmt.Sprintf("%sRequest%d", methodName, maxAttempts+1)
}

// buildServerMethodSignature builds an interface method signature for an operation.
// This is 100% identical between OAS 2.0 and OAS 3.x.
func buildServerMethodSignature(path, method string, op *parser.Operation, responseType string, schemaTypes map[string]bool) string {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)
	wrapperName := resolveWrapperName(methodName, schemaTypes)

	// Write comment - handle multiline descriptions properly
	if op.Summary != "" {
		buf.WriteString(formatMultilineComment(op.Summary, methodName, "\t"))
	} else if op.Description != "" {
		buf.WriteString(formatMultilineComment(op.Description, methodName, "\t"))
	}
	if op.Deprecated {
		buf.WriteString("\t// Deprecated: This operation is deprecated.\n")
	}

	buf.WriteString(fmt.Sprintf("\t%s(ctx context.Context, req *%s) (%s, error)\n", methodName, wrapperName, responseType))

	return buf.String()
}

// clientMethodGenerator is a callback for generating a client method.
type clientMethodGenerator func(path, method string, op *parser.Operation) (string, error)

// generateGroupClientMethods generates client methods for all operations in a group.
// This is 100% identical between OAS 2.0 and OAS 3.x.
func generateGroupClientMethods(
	buf *bytes.Buffer,
	group FileGroup,
	opToPathMethod map[string]OperationMapping,
	result *GenerateResult,
	addIssue issueAdder,
	generateMethod clientMethodGenerator,
) {
	for _, opID := range group.Operations {
		info, ok := opToPathMethod[opID]
		if !ok {
			continue
		}

		code, err := generateMethod(info.Path, info.Method, info.Op)
		if err != nil {
			addIssue(fmt.Sprintf("paths.%s.%s", info.Path, info.Method),
				fmt.Sprintf("failed to generate client method: %v", err), SeverityWarning)
			continue
		}
		buf.WriteString(code)
		result.GeneratedOperations++
	}
}

// writeNotImplementedError writes the ErrNotImplemented variable and NotImplementedError type.
// This is identical across OAS 2.0, OAS 3.x, and split-file generation.
func writeNotImplementedError(buf *bytes.Buffer) {
	buf.WriteString("// ErrNotImplemented is returned by UnimplementedServer methods.\n")
	buf.WriteString("var ErrNotImplemented = &NotImplementedError{}\n\n")
	buf.WriteString("// NotImplementedError indicates an operation is not implemented.\n")
	buf.WriteString("type NotImplementedError struct{}\n\n")
	buf.WriteString("func (e *NotImplementedError) Error() string { return \"not implemented\" }\n\n")
}

// generateServerMiddlewareShared generates the server middleware file.
// This is 100% identical between OAS 2.0 and OAS 3.x.
func generateServerMiddlewareShared(result *GenerateResult, addIssue issueAdder) error {
	data := ServerMiddlewareFileData{
		Header: HeaderData{
			PackageName: result.PackageName,
		},
	}

	formatted, err := executeTemplate("middleware.go.tmpl", data)
	if err != nil {
		addIssue("server_middleware.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	result.Files = append(result.Files, GeneratedFile{
		Name:    "server_middleware.go",
		Content: formatted,
	})
	return nil
}

// serverRouterContext holds the context for generating server router code.
type serverRouterContext struct {
	paths        parser.Paths
	oasVersion   parser.OASVersion
	httpMethods  []string
	packageName  string
	serverRouter string
	schemaTypes  map[string]bool
	result       *GenerateResult
	addIssue     issueAdder
	// paramToBindData converts a parameter to ParamBindData (version-specific)
	paramToBindData func(param *parser.Parameter) ParamBindData
}

// generateServerRouterShared generates HTTP router code.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateServerRouterShared(ctx *serverRouterContext) error {
	if len(ctx.paths) == 0 {
		return nil
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := maputil.SortedKeys(ctx.paths)

	// Build router data
	data := ServerRouterFileData{
		Header: HeaderData{
			PackageName: ctx.packageName,
		},
		Operations: make([]RouterOperationData, 0),
	}

	for _, path := range pathKeys {
		pathItem := ctx.paths[path]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, ctx.oasVersion)
		for _, method := range ctx.httpMethods {
			op := operations[method]
			if op == nil {
				continue
			}

			methodName := operationToMethodName(op, path, method)
			if generatedMethods[methodName] {
				continue
			}
			generatedMethods[methodName] = true

			opData := RouterOperationData{
				Path:        path,
				Method:      strings.ToUpper(method),
				MethodName:  methodName,
				RequestType: resolveWrapperName(methodName, ctx.schemaTypes),
			}

			// Collect path parameters with type info for proper conversion in templates
			for _, param := range op.Parameters {
				if param != nil && param.In == parser.ParamInPath {
					opData.PathParams = append(opData.PathParams, ctx.paramToBindData(param))
				}
			}

			data.Operations = append(data.Operations, opData)
		}
	}

	// Select template based on router type
	templateName := "router.go.tmpl"
	if ctx.serverRouter == "chi" {
		templateName = "router_chi.go.tmpl"
	}

	formatted, err := executeTemplate(templateName, data)
	if err != nil {
		ctx.addIssue("server_router.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "server_router.go",
		Content: formatted,
	})
	return nil
}

// serverStubsContext holds the context for generating server stubs.
type serverStubsContext struct {
	paths       parser.Paths
	oasVersion  parser.OASVersion
	httpMethods []string
	packageName string
	schemaTypes map[string]bool
	result      *GenerateResult
	addIssue    issueAdder
	// getResponseType returns the Go type for the operation's response given the method name
	getResponseType func(methodName string) string
}

// generateServerStubsShared generates testable stub implementations.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateServerStubsShared(ctx *serverStubsContext) error {
	if len(ctx.paths) == 0 {
		return nil
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := maputil.SortedKeys(ctx.paths)

	// Build stubs data
	data := ServerStubsFileData{
		Header: HeaderData{
			PackageName: ctx.packageName,
		},
		Operations: make([]StubOperationData, 0),
	}

	for _, path := range pathKeys {
		pathItem := ctx.paths[path]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, ctx.oasVersion)
		for _, method := range ctx.httpMethods {
			op := operations[method]
			if op == nil {
				continue
			}

			methodName := operationToMethodName(op, path, method)
			if generatedMethods[methodName] {
				continue
			}
			generatedMethods[methodName] = true

			opData := StubOperationData{
				MethodName:   methodName,
				RequestType:  resolveWrapperName(methodName, ctx.schemaTypes),
				ResponseType: ctx.getResponseType(methodName),
			}

			data.Operations = append(data.Operations, opData)
		}
	}

	formatted, err := executeTemplate("stubs.go.tmpl", data)
	if err != nil {
		ctx.addIssue("server_stubs.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "server_stubs.go",
		Content: formatted,
	})
	return nil
}

// baseServerContext holds the context for generating base server code.
type baseServerContext struct {
	paths       parser.Paths
	oasVersion  parser.OASVersion
	httpMethods []string
	packageName string
	needsTime   bool // Whether the time import is needed
	schemaTypes map[string]bool
	result      *GenerateResult
	addIssue    issueAdder
	// generateMethodSignature generates a server method signature for the interface
	generateMethodSignature func(path, method string, op *parser.Operation) string
	// getResponseType returns the Go type for the operation's response
	getResponseType func(op *parser.Operation) string
	// generateRequestTypes is an optional callback to generate request types inline.
	// When set, it's called after the interface and before UnimplementedServer.
	// This is used for single-file mode. For split mode, leave nil.
	generateRequestTypes func(buf *bytes.Buffer, generatedMethods map[string]bool)
}

// generateBaseServerShared generates the base server.go with interface and unimplemented server.
// Returns a map of generated method names (for use by group file generation).
func generateBaseServerShared(ctx *baseServerContext) (map[string]bool, error) {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", ctx.packageName))

	// Write imports - include net/http upfront to avoid expensive goimports scanning
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"net/http\"\n")
	if ctx.needsTime {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString(")\n\n")

	// Track generated methods to avoid duplicates (can happen with duplicate operationIds).
	// NOTE: This map must be local per file generation to avoid stale data in split mode.
	generatedMethods := make(map[string]bool)

	// Generate server interface (must be complete)
	buf.WriteString("// ServerInterface represents the server API.\n")
	buf.WriteString("type ServerInterface interface {\n")

	if ctx.paths != nil {
		for _, path := range maputil.SortedKeys(ctx.paths) {
			pathItem := ctx.paths[path]
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, ctx.oasVersion)
			for _, method := range ctx.httpMethods {
				op := operations[method]
				if op == nil {
					continue
				}

				methodName := operationToMethodName(op, path, method)
				if generatedMethods[methodName] {
					ctx.addIssue(fmt.Sprintf("paths.%s.%s", path, method),
						fmt.Sprintf("duplicate method name %s - skipping", methodName), SeverityWarning)
					continue
				}
				generatedMethods[methodName] = true

				sig := ctx.generateMethodSignature(path, method, op)
				buf.WriteString(sig)
			}
		}
	}

	buf.WriteString("}\n\n")

	// Generate request types if callback is provided (single-file mode)
	if ctx.generateRequestTypes != nil {
		ctx.generateRequestTypes(&buf, generatedMethods)
	}

	// Write unimplemented server (must be complete)
	buf.WriteString("// UnimplementedServer provides default implementations that return errors.\n")
	buf.WriteString("type UnimplementedServer struct{}\n\n")

	// Track generated UnimplementedServer methods separately to avoid duplicates.
	// We can't reuse generatedMethods because it's used to check if a method was
	// added to the interface (i.e., wasn't filtered as duplicate).
	generatedUnimplemented := make(map[string]bool)

	if ctx.paths != nil {
		for _, path := range maputil.SortedKeys(ctx.paths) {
			pathItem := ctx.paths[path]
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, ctx.oasVersion)
			for _, method := range ctx.httpMethods {
				op := operations[method]
				if op == nil {
					continue
				}

				methodName := operationToMethodName(op, path, method)
				// Only generate if the method was added to the interface (not a duplicate)
				if !generatedMethods[methodName] {
					continue
				}
				// Skip if already generated for UnimplementedServer
				if generatedUnimplemented[methodName] {
					continue
				}
				generatedUnimplemented[methodName] = true

				responseType := ctx.getResponseType(op)
				wrapperName := resolveWrapperName(methodName, ctx.schemaTypes)

				buf.WriteString(fmt.Sprintf("func (s *UnimplementedServer) %s(ctx context.Context, req *%s) (%s, error) {\n",
					methodName, wrapperName, responseType))
				buf.WriteString(fmt.Sprintf("\treturn %s, ErrNotImplemented\n", zeroValue(responseType)))
				buf.WriteString("}\n\n")
			}
		}
	}

	// Write error type
	writeNotImplementedError(&buf)

	// Format and append the file
	appendFormattedFile(ctx.result, "server.go", &buf, ctx.addIssue)

	return generatedMethods, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Split Server Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// splitServerContext holds context for generating split server files.
type splitServerContext struct {
	paths               parser.Paths
	oasVersion          parser.OASVersion
	splitPlan           *SplitPlan
	result              *GenerateResult
	addIssue            issueAdder
	generateBaseServer  func() (map[string]bool, error)
	generateRequestType func(path, method string, op *parser.Operation) string
}

// generateSplitServerShared generates server code split across multiple files.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateSplitServerShared(ctx *splitServerContext) error {
	// Generate the base server.go (interface, unimplemented, error types)
	// Returns the set of methods that were actually generated (excludes duplicates)
	generatedMethods, err := ctx.generateBaseServer()
	if err != nil {
		return err
	}

	// Build a map of operation ID to path/method for quick lookup
	opToPathMethod := buildOperationMap(ctx.paths, ctx.oasVersion)

	// Generate a server file for each group (request types only)
	for _, group := range ctx.splitPlan.Groups {
		if group.IsShared {
			continue // Skip shared types group
		}

		if err := generateServerGroupFileShared(&serverGroupContext{
			group:               group,
			opToPathMethod:      opToPathMethod,
			generatedMethods:    generatedMethods,
			packageName:         ctx.result.PackageName,
			result:              ctx.result,
			addIssue:            ctx.addIssue,
			generateRequestType: ctx.generateRequestType,
		}); err != nil {
			ctx.addIssue(fmt.Sprintf("server_%s.go", group.Name), fmt.Sprintf("failed to generate: %v", err), SeverityWarning)
		}
	}

	return nil
}

// serverGroupContext holds context for generating a single server group file.
type serverGroupContext struct {
	group               FileGroup
	opToPathMethod      map[string]OperationMapping
	generatedMethods    map[string]bool
	packageName         string
	result              *GenerateResult
	addIssue            issueAdder
	generateRequestType func(path, method string, op *parser.Operation) string
}

// generateServerGroupFileShared generates a server_{group}.go file with request types.
// The generatedMethods map indicates which methods were added to the interface.
// This is shared between OAS 2.0 and OAS 3.x generators.
//
//nolint:unparam // error return kept for API consistency and future extensibility
func generateServerGroupFileShared(ctx *serverGroupContext) error {
	var buf bytes.Buffer

	// Write header with comment about the group
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n")
	buf.WriteString(fmt.Sprintf("// This file contains %s server request types.\n\n", ctx.group.DisplayName))
	buf.WriteString(fmt.Sprintf("package %s\n\n", ctx.packageName))

	// Write minimal imports - formatAndFixImports will add/remove as needed
	buf.WriteString("import (\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString(")\n\n")

	// Generate request types for each operation in this group
	for _, opID := range ctx.group.Operations {
		// Skip if method was not added to interface (was filtered as duplicate)
		if !ctx.generatedMethods[opID] {
			continue
		}

		info, ok := ctx.opToPathMethod[opID]
		if !ok {
			continue
		}

		reqType := ctx.generateRequestType(info.Path, info.Method, info.Op)
		if reqType != "" {
			buf.WriteString(reqType)
		}
	}

	// Format and append the file
	fileName := fmt.Sprintf("server_%s.go", ctx.group.Name)
	appendFormattedFile(ctx.result, fileName, &buf, ctx.addIssue)

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Client Generation Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// generateBaseClientShared generates the base client.go with struct, constructor, and options.
// This is 100% identical between OAS 2.0 and OAS 3.x.
func generateBaseClientShared(packageName string, info *parser.Info, result *GenerateResult, addIssue issueAdder) error {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Write imports (base client needs these imports for clientHelpers)
	buf.WriteString("import (\n")
	buf.WriteString("\t\"bytes\"\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"encoding/json\"\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"io\"\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString("\t\"net/url\"\n")
	buf.WriteString("\t\"strings\"\n")
	buf.WriteString(")\n\n")

	// Write client struct, types, constructor, and options using shared boilerplate
	writeClientBoilerplate(&buf, info)

	// Write helper functions
	buf.WriteString(clientHelpers)

	// Format and append the file
	appendFormattedFile(result, "client.go", &buf, addIssue)

	return nil
}
