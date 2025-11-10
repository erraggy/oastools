package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/erraggy/oastools/internal/parser"
	"github.com/erraggy/oastools/internal/validator"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "-v", "--version":
		fmt.Printf("oastools v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	case "validate":
		handleValidate(os.Args[2:])
	case "parse":
		handleParse(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleParse(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: parse command requires a file path\n\n")
		fmt.Println("Usage: oastools parse <file>")
		os.Exit(1)
	}

	filePath := args[0]

	// Create parser
	p := parser.New()

	// Parse the file
	result, err := p.Parse(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("OpenAPI Specification Parser\n")
	fmt.Printf("============================\n\n")
	fmt.Printf("File: %s\n", filePath)
	fmt.Printf("Version: %s\n\n", result.Version)

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings:\n")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
		fmt.Println()
	}

	// Print errors
	if len(result.Errors) > 0 {
		fmt.Printf("Validation Errors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
		fmt.Println()
		os.Exit(1)
	}

	// Print document info
	if result.Document != nil {
		switch doc := result.Document.(type) {
		case *parser.OAS2Document:
			fmt.Printf("Document Type: OpenAPI 2.0 (Swagger)\n")
			if doc.Info != nil {
				fmt.Printf("Title: %s\n", doc.Info.Title)
				fmt.Printf("Description: %s\n", doc.Info.Description)
				fmt.Printf("Version: %s\n", doc.Info.Version)
			}
			fmt.Printf("Paths: %d\n", len(doc.Paths))

		case *parser.OAS3Document:
			fmt.Printf("Document Type: OpenAPI 3.x\n")
			if doc.Info != nil {
				fmt.Printf("Title: %s\n", doc.Info.Title)
				if doc.Info.Summary != "" {
					fmt.Printf("Summary: %s\n", doc.Info.Summary)
				}
				fmt.Printf("Description: %s\n", doc.Info.Description)
				fmt.Printf("Version: %s\n", doc.Info.Version)
			}
			fmt.Printf("Servers: %d\n", len(doc.Servers))
			fmt.Printf("Paths: %d\n", len(doc.Paths))
			if len(doc.Webhooks) > 0 {
				fmt.Printf("Webhooks: %d\n", len(doc.Webhooks))
			}
		}
	}

	// Output raw data as JSON for inspection
	fmt.Printf("\nRaw Data (JSON):\n")
	jsonData, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))

	fmt.Printf("\nParsing completed successfully!\n")
}

func handleValidate(args []string) {
	// Parse flags
	var strict bool
	var noWarnings bool
	var filePath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--strict":
			strict = true
		case "--no-warnings":
			noWarnings = true
		case "-h", "--help":
			printValidateUsage()
			return
		default:
			if filePath == "" {
				filePath = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument '%s'\n", arg)
				os.Exit(1)
			}
		}
	}

	if filePath == "" {
		fmt.Fprintf(os.Stderr, "Error: validate command requires a file path\n\n")
		printValidateUsage()
		os.Exit(1)
	}

	// Create validator with options
	v := validator.New()
	v.StrictMode = strict
	v.IncludeWarnings = !noWarnings

	// Validate the file
	result, err := v.Validate(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error validating file: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("OpenAPI Specification Validator\n")
	fmt.Printf("================================\n\n")
	fmt.Printf("File: %s\n", filePath)
	fmt.Printf("Version: %s\n\n", result.Version)

	// Print errors
	if len(result.Errors) > 0 {
		fmt.Printf("Errors (%d):\n", result.ErrorCount)
		for _, err := range result.Errors {
			fmt.Printf("  %s\n", err.String())
		}
		fmt.Println()
	}

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings (%d):\n", result.WarningCount)
		for _, warning := range result.Warnings {
			fmt.Printf("  %s\n", warning.String())
		}
		fmt.Println()
	}

	// Print summary
	if result.Valid {
		fmt.Printf("✓ Validation passed")
		if result.WarningCount > 0 {
			fmt.Printf(" with %d warning(s)", result.WarningCount)
		}
		fmt.Println()
	} else {
		fmt.Printf("✗ Validation failed: %d error(s)", result.ErrorCount)
		if result.WarningCount > 0 {
			fmt.Printf(", %d warning(s)", result.WarningCount)
		}
		fmt.Println()
		os.Exit(1)
	}
}

func printValidateUsage() {
	fmt.Println(`Usage: oastools validate [options] <file>

Validate an OpenAPI specification file against the specification version it declares.

Options:
  --strict        Enable stricter validation beyond spec requirements
  --no-warnings   Suppress warning messages (only show errors)
  -h, --help      Show this help message

Examples:
  oastools validate openapi.yaml
  oastools validate --strict api-spec.yaml
  oastools validate --no-warnings openapi.json`)
}

func printUsage() {
	fmt.Println(`oastools - OpenAPI Specification Tools

Usage:
  oastools <command> [options]

Commands:
  validate    Validate an OpenAPI specification file
  join        Join multiple OpenAPI specification files
  parse       Parse and display an OpenAPI specification
  version     Show version information
  help        Show this help message

Examples:
  oastools validate openapi.yaml
  oastools join base.yaml extensions.yaml

Run 'oastools <command> --help' for more information on a command.`)
}
