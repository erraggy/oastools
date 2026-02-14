package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/cmd/oastools/commands"
	"github.com/erraggy/oastools/internal/mcpserver"
)

// validCommands lists all valid command names for typo suggestions
var validCommands = []string{
	"validate", "fix", "convert", "diff", "generate", "join", "mcp", "overlay", "parse", "walk", "version", "help",
}

// levenshteinDistance calculates the minimum edit distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range len(b) + 1 {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// suggestCommand returns the closest matching command if the edit distance is <= 2
func suggestCommand(input string) string {
	var bestMatch string
	bestDistance := 3 // Only suggest if distance <= 2

	for _, cmd := range validCommands {
		dist := levenshteinDistance(input, cmd)
		if dist < bestDistance {
			bestDistance = dist
			bestMatch = cmd
		}
	}

	return bestMatch
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "-v", "--version":
		fmt.Printf("oastools v%s\n", oastools.Version())
		fmt.Printf("commit: %s\n", oastools.Commit())
		fmt.Printf("built: %s\n", oastools.BuildTime())
		fmt.Printf("go: %s\n", oastools.GoVersion())
	case "help", "-h", "--help":
		printUsage()
	case "validate":
		if err := commands.HandleValidate(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "parse":
		if err := commands.HandleParse(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "join":
		if err := commands.HandleJoin(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "convert":
		if err := commands.HandleConvert(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "diff":
		if err := commands.HandleDiff(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "generate":
		if err := commands.HandleGenerate(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "fix":
		if err := commands.HandleFix(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "overlay":
		if err := commands.HandleOverlay(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "walk":
		if err := commands.HandleWalk(os.Args[2:]); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "mcp":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := mcpserver.Run(ctx); err != nil {
			commands.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		commands.Writef(os.Stderr, "Unknown command: %s\n", command)
		if suggestion := suggestCommand(command); suggestion != "" {
			commands.Writef(os.Stderr, "Did you mean: %s?\n", suggestion)
		}
		commands.Writef(os.Stderr, "\n")
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`oastools - OpenAPI Specification Tools

Usage:
  oastools <command> [options]

Commands:
  validate    Validate an OpenAPI specification file or URL
  fix         Apply automatic fixes to common OpenAPI specification issues
  convert     Convert between OpenAPI specification versions
  diff        Compare two OpenAPI specifications and detect changes
  generate    Generate Go client/server code from an OpenAPI specification
  join        Join multiple OpenAPI specification files
  overlay     Apply or validate OpenAPI Overlay documents
  parse       Parse and display an OpenAPI specification file or URL
  walk        Query and explore OpenAPI specification documents
  mcp         Start an MCP server over stdio
  version     Show version information
  help        Show this help message

Examples:
  oastools validate openapi.yaml
  oastools fix api.yaml | oastools validate -q -
  oastools validate https://example.com/api/openapi.yaml
  oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml
  oastools diff --breaking api-v1.yaml api-v2.yaml
  oastools generate --client -o ./client openapi.yaml
  oastools join -o merged.yaml base.yaml extensions.yaml
  oastools overlay apply -s openapi.yaml changes.yaml -o production.yaml
  oastools parse https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/examples/v3.0/petstore.yaml

Run 'oastools <command> --help' for more information on a command.`)
}
