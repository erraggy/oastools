package main

import (
	"fmt"
	"os"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/cmd/oastools/commands"
	"github.com/erraggy/oastools/internal/cliutil"
)

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
			cliutil.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "parse":
		if err := commands.HandleParse(os.Args[2:]); err != nil {
			cliutil.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "join":
		if err := commands.HandleJoin(os.Args[2:]); err != nil {
			cliutil.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "convert":
		if err := commands.HandleConvert(os.Args[2:]); err != nil {
			cliutil.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "diff":
		if err := commands.HandleDiff(os.Args[2:]); err != nil {
			cliutil.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "generate":
		if err := commands.HandleGenerate(os.Args[2:]); err != nil {
			cliutil.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "fix":
		if err := commands.HandleFix(os.Args[2:]); err != nil {
			cliutil.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "overlay":
		if err := commands.HandleOverlay(os.Args[2:]); err != nil {
			cliutil.Writef(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		cliutil.Writef(os.Stderr, "Unknown command: %s\n\n", command)
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
