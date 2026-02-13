package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// HandleWalk routes the walk command to the appropriate subcommand handler.
func HandleWalk(args []string) error {
	if len(args) == 0 {
		printWalkUsage()
		return fmt.Errorf("walk command requires a subcommand")
	}

	subcommand := args[0]

	// Handle --help at walk level
	if subcommand == "--help" || subcommand == "-h" || subcommand == "help" {
		printWalkUsage()
		return nil
	}

	subArgs := args[1:]

	switch subcommand {
	case "operations":
		return handleWalkOperations(subArgs)
	case "schemas":
		return handleWalkSchemas(subArgs)
	case "parameters":
		return handleWalkParameters(subArgs)
	case "responses":
		return handleWalkResponses(subArgs)
	case "security":
		return handleWalkSecurity(subArgs)
	case "paths":
		return handleWalkPaths(subArgs)
	default:
		printWalkUsage()
		return fmt.Errorf("unknown walk subcommand: %s", subcommand)
	}
}

// WalkFlags contains common flags shared by all walk subcommands.
type WalkFlags struct {
	Format      string // Output format: text, json, yaml.
	Quiet       bool   // Suppress headers and decoration for piping.
	Detail      bool   // Show full node instead of summary table.
	Extension   string // Extension filter expression (e.g., "x-internal=true").
	ResolveRefs bool   // Resolve $ref pointers before output.
}

// parseSpec parses an OAS file from a file path, URL, or stdin ("-").
// When resolveRefs is true, all $ref pointers are resolved during parsing.
func parseSpec(specPath string, resolveRefs bool) (*parser.ParseResult, error) {
	p := parser.New()
	p.ResolveRefs = resolveRefs

	if specPath == StdinFilePath {
		return p.ParseReader(os.Stdin)
	}
	return p.Parse(specPath)
}

// renderNoResults prints an informative message when no results match the filters.
func renderNoResults(nodeType string, quiet bool) {
	if !quiet {
		Writef(os.Stderr, "No %s matched the given filters.\n", nodeType)
	}
}

// matchPath checks if a path template matches a pattern.
// Supports simple glob matching where * matches exactly one path segment
// (e.g., /pets/* matches /pets/123 but not /pets/123/details).
func matchPath(pathTemplate, pattern string) bool {
	if pattern == "" {
		return true
	}
	if strings.Contains(pattern, "*") {
		patternParts := strings.Split(pattern, "/")
		pathParts := strings.Split(pathTemplate, "/")
		if len(patternParts) != len(pathParts) {
			return false
		}
		for i, pp := range patternParts {
			if pp == "*" {
				continue
			}
			if pp != pathParts[i] {
				return false
			}
		}
		return true
	}
	return pathTemplate == pattern
}

// matchStatusCode checks if a status code matches a pattern.
// Supports wildcards like "2xx" or "4xx" which match any 3-digit code starting with
// the same digit. Non-wildcard patterns require exact match (e.g., "default").
func matchStatusCode(code, pattern string) bool {
	if pattern == "" {
		return true
	}
	pattern = strings.ToLower(pattern)
	code = strings.ToLower(code)
	if len(pattern) == 3 && strings.HasSuffix(pattern, "xx") {
		return len(code) == 3 && code[0] == pattern[0]
	}
	return code == pattern
}

func printWalkUsage() {
	Writef(os.Stderr, `Usage: oastools walk <subcommand> [flags] <file|url|->

Query and explore OpenAPI specification documents.

Subcommands:
  operations    List or inspect operations
  schemas       List or inspect schemas
  parameters    List or inspect parameters
  responses     List or inspect responses
  security      List or inspect security schemes
  paths         List or inspect path items

Common Flags:
  --format      Output format: text (default), json, yaml
  -q, --quiet   Suppress headers and decoration for piping
  --detail      Show full node instead of summary table
  --extension   Filter by extension (e.g., x-internal=true)
  --resolve-refs  Resolve $ref pointers in detail output

Examples:
  oastools walk operations api.yaml
  oastools walk operations --method get --path /pets --detail api.yaml
  oastools walk schemas --name Pet --detail api.yaml
  oastools walk operations --extension x-audited-by api.yaml
  oastools walk responses --status '4xx' -q --detail --format json api.yaml | jq

Run 'oastools walk <subcommand> --help' for subcommand-specific flags.
`)
}
