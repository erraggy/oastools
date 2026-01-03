// API Documentation Generator example demonstrating Markdown documentation generation.
//
// This example shows how to:
//   - Extract documentation from multiple handler types in a single pass
//   - Maintain state across nested handlers (path → operation → parameters/responses)
//   - Generate structured Markdown output from OpenAPI specifications
//   - Use walker for comprehensive documentation generation
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

// Documentation holds the complete API documentation structure.
type Documentation struct {
	Title       string
	Description string
	Version     string
	Servers     []ServerDoc
	Tags        []TagDoc
	Endpoints   []EndpointDoc
}

// ServerDoc represents server documentation.
type ServerDoc struct {
	URL         string
	Description string
}

// TagDoc represents tag documentation.
type TagDoc struct {
	Name        string
	Description string
}

// EndpointDoc represents endpoint documentation.
type EndpointDoc struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Parameters  []ParamDoc
	Responses   []ResponseDoc
}

// ParamDoc represents parameter documentation.
type ParamDoc struct {
	Name        string
	In          string
	Required    bool
	Description string
}

// ResponseDoc represents response documentation.
type ResponseDoc struct {
	StatusCode  string
	Description string
}

func main() {
	specPath := findSpecPath()

	// Parse the specification
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(specPath),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Initialize documentation
	doc := &Documentation{}

	// Track current context for nested handlers
	var currentPath string
	var currentEndpoint *EndpointDoc

	// Walk the document with comprehensive handlers
	err = walker.Walk(parseResult,
		// Extract API info (title, description, version)
		walker.WithInfoHandler(func(info *parser.Info, path string) walker.Action {
			doc.Title = info.Title
			doc.Description = info.Description
			doc.Version = info.Version
			return walker.Continue
		}),

		// Collect server URLs and descriptions
		walker.WithServerHandler(func(server *parser.Server, path string) walker.Action {
			doc.Servers = append(doc.Servers, ServerDoc{
				URL:         server.URL,
				Description: server.Description,
			})
			return walker.Continue
		}),

		// Collect tags with descriptions
		walker.WithTagHandler(func(tag *parser.Tag, path string) walker.Action {
			doc.Tags = append(doc.Tags, TagDoc{
				Name:        tag.Name,
				Description: tag.Description,
			})
			return walker.Continue
		}),

		// Track current path for operation context
		walker.WithPathHandler(func(pathTemplate string, pathItem *parser.PathItem, path string) walker.Action {
			currentPath = pathTemplate
			return walker.Continue
		}),

		// Collect operation details, create new EndpointDoc
		walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
			endpoint := EndpointDoc{
				Method:      strings.ToUpper(method),
				Path:        currentPath,
				OperationID: op.OperationID,
				Summary:     op.Summary,
				Description: op.Description,
				Tags:        op.Tags,
				Parameters:  []ParamDoc{},
				Responses:   []ResponseDoc{},
			}
			doc.Endpoints = append(doc.Endpoints, endpoint)
			currentEndpoint = &doc.Endpoints[len(doc.Endpoints)-1]
			return walker.Continue
		}),

		// Add parameter to current endpoint
		walker.WithParameterHandler(func(param *parser.Parameter, path string) walker.Action {
			if currentEndpoint != nil && strings.Contains(path, ".parameters[") {
				// Only add parameters that are part of operations (not path-level duplicates)
				currentEndpoint.Parameters = append(currentEndpoint.Parameters, ParamDoc{
					Name:        param.Name,
					In:          param.In,
					Required:    param.Required,
					Description: param.Description,
				})
			}
			return walker.Continue
		}),

		// Add response to current endpoint
		walker.WithResponseHandler(func(statusCode string, resp *parser.Response, path string) walker.Action {
			if currentEndpoint != nil {
				currentEndpoint.Responses = append(currentEndpoint.Responses, ResponseDoc{
					StatusCode:  statusCode,
					Description: resp.Description,
				})
			}
			return walker.Continue
		}),
	)
	if err != nil {
		log.Fatalf("Walk error: %v", err)
	}

	// Sort endpoints by path then method for consistent output
	sort.Slice(doc.Endpoints, func(i, j int) bool {
		if doc.Endpoints[i].Path != doc.Endpoints[j].Path {
			return doc.Endpoints[i].Path < doc.Endpoints[j].Path
		}
		return doc.Endpoints[i].Method < doc.Endpoints[j].Method
	})

	// Generate and print Markdown documentation
	generateMarkdown(doc)
}

func generateMarkdown(doc *Documentation) {
	// Title and version
	fmt.Printf("# %s\n\n", doc.Title)
	fmt.Printf("**Version:** %s\n\n", doc.Version)

	// Description
	if doc.Description != "" {
		fmt.Printf("%s\n\n", doc.Description)
	}

	// Servers section
	if len(doc.Servers) > 0 {
		fmt.Println("## Servers")
		fmt.Println()
		fmt.Println("| Environment | URL |")
		fmt.Println("|-------------|-----|")
		for _, server := range doc.Servers {
			env := server.Description
			if env == "" {
				env = "Server"
			}
			fmt.Printf("| %s | %s |\n", env, server.URL)
		}
		fmt.Println()
	}

	// Tags section
	if len(doc.Tags) > 0 {
		fmt.Println("## Tags")
		fmt.Println()
		for _, tag := range doc.Tags {
			if tag.Description != "" {
				fmt.Printf("- **%s** - %s\n", tag.Name, tag.Description)
			} else {
				fmt.Printf("- **%s**\n", tag.Name)
			}
		}
		fmt.Println()
	}

	// Endpoints section
	fmt.Println("## Endpoints")
	fmt.Println()

	for i, endpoint := range doc.Endpoints {
		// Endpoint header
		fmt.Printf("### %s %s\n\n", endpoint.Method, endpoint.Path)

		// Operation ID and summary
		if endpoint.OperationID != "" {
			fmt.Printf("**%s**", endpoint.OperationID)
			if endpoint.Summary != "" {
				fmt.Printf(": %s", endpoint.Summary)
			}
			fmt.Println()
			fmt.Println()
		} else if endpoint.Summary != "" {
			fmt.Printf("%s\n\n", endpoint.Summary)
		}

		// Description
		if endpoint.Description != "" {
			fmt.Printf("%s\n\n", endpoint.Description)
		}

		// Parameters table
		if len(endpoint.Parameters) > 0 {
			fmt.Println("**Parameters:**")
			fmt.Println()
			fmt.Println("| Name | In | Required | Description |")
			fmt.Println("|------|-----|----------|-------------|")
			for _, param := range endpoint.Parameters {
				required := "No"
				if param.Required {
					required = "Yes"
				}
				description := param.Description
				if description == "" {
					description = "-"
				}
				fmt.Printf("| %s | %s | %s | %s |\n", param.Name, param.In, required, description)
			}
			fmt.Println()
		}

		// Responses table
		if len(endpoint.Responses) > 0 {
			fmt.Println("**Responses:**")
			fmt.Println()
			fmt.Println("| Status | Description |")
			fmt.Println("|--------|-------------|")
			for _, resp := range endpoint.Responses {
				description := resp.Description
				if description == "" {
					description = "-"
				}
				fmt.Printf("| %s | %s |\n", resp.StatusCode, description)
			}
			fmt.Println()
		}

		// Separator between endpoints (except for last)
		if i < len(doc.Endpoints)-1 {
			fmt.Println("---")
			fmt.Println()
		}
	}
}

// findSpecPath locates the petstore-3.0.yaml file relative to the source file.
func findSpecPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot determine source file location")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "testdata", "petstore-3.0.yaml")
}
