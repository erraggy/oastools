package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/differ"
	"github.com/erraggy/oastools/generator"
	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"gopkg.in/yaml.v3"
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
	case "help", "-h", "--help":
		printUsage()
	case "validate":
		if err := handleValidate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "parse":
		if err := handleParse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "join":
		if err := handleJoin(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "convert":
		if err := handleConvert(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "diff":
		if err := handleDiff(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "generate":
		if err := handleGenerate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

// validateCollisionStrategy validates a collision strategy name and returns an error if invalid.
// The strategyName parameter is used in the error message (e.g., "path-strategy").
func validateCollisionStrategy(strategyName, value string) error {
	if value != "" && !joiner.IsValidStrategy(value) {
		return fmt.Errorf("invalid %s '%s'. Valid strategies: %v", strategyName, value, joiner.ValidStrategies())
	}
	return nil
}

// parseFlags contains flags for the parse command
type parseFlags struct {
	resolveRefs       bool
	validateStructure bool
	quiet             bool
}

// setupParseFlags creates and configures a FlagSet for the parse command.
// Returns the FlagSet and a parseFlags struct with bound flag variables.
func setupParseFlags() (*flag.FlagSet, *parseFlags) {
	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	flags := &parseFlags{}

	fs.BoolVar(&flags.resolveRefs, "resolve-refs", false, "resolve external $ref references")
	fs.BoolVar(&flags.validateStructure, "validate-structure", false, "validate document structure during parsing")
	fs.BoolVar(&flags.quiet, "q", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.quiet, "quiet", false, "quiet mode: only output the document, no diagnostic messages")

	fs.Usage = func() {
		output := fs.Output()
		_, _ = fmt.Fprintf(output, "Usage: oastools parse [flags] <file|url|->\n\n")
		_, _ = fmt.Fprintf(output, "Parse and output OpenAPI document structure.\n\n")
		_, _ = fmt.Fprintf(output, "Flags:\n")
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(output, "\nExamples:\n")
		_, _ = fmt.Fprintf(output, "  oastools parse openapi.yaml\n")
		_, _ = fmt.Fprintf(output, "  oastools parse --resolve-refs openapi.yaml\n")
		_, _ = fmt.Fprintf(output, "  oastools parse --validate-structure https://example.com/api/openapi.yaml\n")
		_, _ = fmt.Fprintf(output, "  cat openapi.yaml | oastools parse -q -\n")
		_, _ = fmt.Fprintf(output, "\nPipelining:\n")
		_, _ = fmt.Fprintf(output, "  - Use '-' as the file path to read from stdin\n")
		_, _ = fmt.Fprintf(output, "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
	}

	return fs, flags
}

func handleParse(args []string) error {
	fs, flags := setupParseFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("parse command requires exactly one file path, URL, or '-' for stdin")
	}

	specPath := fs.Arg(0)

	// Create parser with options
	p := parser.New()
	p.ResolveRefs = flags.resolveRefs
	p.ValidateStructure = flags.validateStructure

	// Parse the file, URL, or stdin
	var result *parser.ParseResult
	var err error

	if specPath == "-" {
		result, err = p.ParseReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("parsing stdin: %w", err)
		}
	} else {
		result, err = p.Parse(specPath)
		if err != nil {
			return fmt.Errorf("parsing file: %w", err)
		}
	}

	// Print results (always to stderr to keep stdout clean for JSON output)
	if !flags.quiet {
		fmt.Fprintf(os.Stderr, "OpenAPI Specification Parser\n")
		fmt.Fprintf(os.Stderr, "============================\n\n")
		fmt.Fprintf(os.Stderr, "oastools version: %s\n", oastools.Version())
		if specPath == "-" {
			fmt.Fprintf(os.Stderr, "Specification: <stdin>\n")
		} else {
			fmt.Fprintf(os.Stderr, "Specification: %s\n", specPath)
		}
		fmt.Fprintf(os.Stderr, "OAS Version: %s\n", result.Version)
		fmt.Fprintf(os.Stderr, "Source Size: %s\n", parser.FormatBytes(result.SourceSize))
		fmt.Fprintf(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		fmt.Fprintf(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		fmt.Fprintf(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		fmt.Fprintf(os.Stderr, "Load Time: %v\n\n", result.LoadTime)

		// Print warnings
		if len(result.Warnings) > 0 {
			fmt.Fprintf(os.Stderr, "Warnings:\n")
			for _, warning := range result.Warnings {
				fmt.Fprintf(os.Stderr, "  - %s\n", warning)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}

		// Print errors
		if len(result.Errors) > 0 {
			fmt.Fprintf(os.Stderr, "Validation Errors:\n")
			for _, err := range result.Errors {
				fmt.Fprintf(os.Stderr, "  - %s\n", err)
			}
			fmt.Fprintf(os.Stderr, "\n")
			os.Exit(1)
		}

		// Print document info
		if result.Document != nil {
			switch doc := result.Document.(type) {
			case *parser.OAS2Document:
				fmt.Fprintf(os.Stderr, "Document Type: OpenAPI 2.0 (Swagger)\n")
				if doc.Info != nil {
					fmt.Fprintf(os.Stderr, "Title: %s\n", doc.Info.Title)
					fmt.Fprintf(os.Stderr, "Description: %s\n", doc.Info.Description)
					fmt.Fprintf(os.Stderr, "Version: %s\n", doc.Info.Version)
				}
				fmt.Fprintf(os.Stderr, "Paths: %d\n", len(doc.Paths))

			case *parser.OAS3Document:
				fmt.Fprintf(os.Stderr, "Document Type: OpenAPI 3.x\n")
				if doc.Info != nil {
					fmt.Fprintf(os.Stderr, "Title: %s\n", doc.Info.Title)
					if doc.Info.Summary != "" {
						fmt.Fprintf(os.Stderr, "Summary: %s\n", doc.Info.Summary)
					}
					fmt.Fprintf(os.Stderr, "Description: %s\n", doc.Info.Description)
					fmt.Fprintf(os.Stderr, "Version: %s\n", doc.Info.Version)
				}
				fmt.Fprintf(os.Stderr, "Servers: %d\n", len(doc.Servers))
				fmt.Fprintf(os.Stderr, "Paths: %d\n", len(doc.Paths))
				if len(doc.Webhooks) > 0 {
					fmt.Fprintf(os.Stderr, "Webhooks: %d\n", len(doc.Webhooks))
				}
			}
		}

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Raw Data (JSON):\n")
	}
	jsonData, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}
	fmt.Println(string(jsonData))

	if !flags.quiet {
		fmt.Fprintf(os.Stderr, "\nParsing completed successfully!\n")
	}
	return nil
}

// validateFlags contains flags for the validate command
type validateFlags struct {
	strict     bool
	noWarnings bool
	quiet      bool
	format     string
}

// setupValidateFlags creates and configures a FlagSet for the validate command.
// Returns the FlagSet and a validateFlags struct with bound flag variables.
func setupValidateFlags() (*flag.FlagSet, *validateFlags) {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags := &validateFlags{}

	fs.BoolVar(&flags.strict, "strict", false, "enable stricter validation beyond spec requirements")
	fs.BoolVar(&flags.noWarnings, "no-warnings", false, "suppress warning messages (only show errors)")
	fs.BoolVar(&flags.quiet, "q", false, "quiet mode: only output validation result, no diagnostic messages")
	fs.BoolVar(&flags.quiet, "quiet", false, "quiet mode: only output validation result, no diagnostic messages")
	fs.StringVar(&flags.format, "format", "text", "output format: text, json, or yaml")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: oastools validate [flags] <file|url|->\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Validate an OpenAPI specification file, URL, or stdin against the specification version it declares.\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(fs.Output(), "\nOutput Formats:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  text (default)  Human-readable text output\n")
		_, _ = fmt.Fprintf(fs.Output(), "  json            JSON format for programmatic processing\n")
		_, _ = fmt.Fprintf(fs.Output(), "  yaml            YAML format for programmatic processing\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nExamples:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools validate openapi.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools validate https://example.com/api/openapi.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools validate --strict api-spec.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools validate --no-warnings openapi.json\n")
		_, _ = fmt.Fprintf(fs.Output(), "  cat openapi.yaml | oastools validate -q -\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools validate --format json openapi.yaml | jq '.valid'\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nPipelining:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Use '-' as the file path to read from stdin\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Use --format json/yaml for structured output that can be parsed\n")
	}

	return fs, flags
}

func handleValidate(args []string) error {
	fs, flags := setupValidateFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("validate command requires exactly one file path, URL, or '-' for stdin")
	}

	specPath := fs.Arg(0)

	// Create validator with options
	v := validator.New()
	v.StrictMode = flags.strict
	v.IncludeWarnings = !flags.noWarnings

	// Validate the file, URL, or stdin with timing
	startTime := time.Now()
	var result *validator.ValidationResult
	var err error

	if specPath == "-" {
		// Read from stdin
		p := parser.New()
		parseResult, err := p.ParseReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("parsing stdin: %w", err)
		}
		result, err = v.ValidateParsed(*parseResult)
		if err != nil {
			return fmt.Errorf("validating from stdin: %w", err)
		}
	} else {
		result, err = v.Validate(specPath)
		if err != nil {
			return fmt.Errorf("validating file: %w", err)
		}
	}
	totalTime := time.Since(startTime)

	// Validate format flag
	if flags.format != "text" && flags.format != "json" && flags.format != "yaml" {
		return fmt.Errorf("invalid format '%s'. Valid formats: text, json, yaml", flags.format)
	}

	// Handle structured output formats
	if flags.format == "json" || flags.format == "yaml" {
		// Output structured format to stdout
		var data []byte
		var err error
		
		if flags.format == "json" {
			data, err = json.MarshalIndent(result, "", "  ")
		} else {
			data, err = yaml.Marshal(result)
		}
		
		if err != nil {
			return fmt.Errorf("marshaling validation result: %w", err)
		}
		
		fmt.Println(string(data))
		
		// Exit with error if validation failed
		if !result.Valid {
			os.Exit(1)
		}
		
		return nil
	}

	// Text format output (original behavior)
	// Print results (always to stderr to be consistent with parse and convert)
	if !flags.quiet {
		fmt.Fprintf(os.Stderr, "OpenAPI Specification Validator\n")
		fmt.Fprintf(os.Stderr, "================================\n\n")
		fmt.Fprintf(os.Stderr, "oastools version: %s\n", oastools.Version())
		if specPath == "-" {
			fmt.Fprintf(os.Stderr, "Specification: <stdin>\n")
		} else {
			fmt.Fprintf(os.Stderr, "Specification: %s\n", specPath)
		}
		fmt.Fprintf(os.Stderr, "OAS Version: %s\n", result.Version)
		fmt.Fprintf(os.Stderr, "Source Size: %s\n", parser.FormatBytes(result.SourceSize))
		fmt.Fprintf(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		fmt.Fprintf(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		fmt.Fprintf(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		fmt.Fprintf(os.Stderr, "Load Time: %v\n", result.LoadTime)
		fmt.Fprintf(os.Stderr, "Total Time: %v\n\n", totalTime)

		// Print errors
		if len(result.Errors) > 0 {
			fmt.Fprintf(os.Stderr, "Errors (%d):\n", result.ErrorCount)
			for _, err := range result.Errors {
				fmt.Fprintf(os.Stderr, "  %s\n", err.String())
			}
			fmt.Fprintf(os.Stderr, "\n")
		}

		// Print warnings
		if len(result.Warnings) > 0 {
			fmt.Fprintf(os.Stderr, "Warnings (%d):\n", result.WarningCount)
			for _, warning := range result.Warnings {
				fmt.Fprintf(os.Stderr, "  %s\n", warning.String())
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	// Print summary (always to stderr for consistency)
	if result.Valid {
		fmt.Fprintf(os.Stderr, "✓ Validation passed")
		if result.WarningCount > 0 {
			fmt.Fprintf(os.Stderr, " with %d warning(s)", result.WarningCount)
		}
		fmt.Fprintf(os.Stderr, "\n")
	} else {
		fmt.Fprintf(os.Stderr, "✗ Validation failed: %d error(s)", result.ErrorCount)
		if result.WarningCount > 0 {
			fmt.Fprintf(os.Stderr, ", %d warning(s)", result.WarningCount)
		}
		fmt.Fprintf(os.Stderr, "\n")
		os.Exit(1)
	}

	return nil
}

// joinFlags contains flags for the join command
type joinFlags struct {
	output            string
	pathStrategy      string
	schemaStrategy    string
	componentStrategy string
	noMergeArrays     bool
	noDedupTags       bool
}

// setupJoinFlags creates and configures a FlagSet for the join command.
// Returns the FlagSet and a joinFlags struct with bound flag variables.
func setupJoinFlags() (*flag.FlagSet, *joinFlags) {
	fs := flag.NewFlagSet("join", flag.ContinueOnError)
	flags := &joinFlags{}

	fs.StringVar(&flags.output, "o", "", "output file path (required)")
	fs.StringVar(&flags.output, "output", "", "output file path (required)")
	fs.StringVar(&flags.pathStrategy, "path-strategy", "", "collision strategy for paths (accept-left, accept-right, fail, fail-on-paths)")
	fs.StringVar(&flags.schemaStrategy, "schema-strategy", "", "collision strategy for schemas/definitions")
	fs.StringVar(&flags.componentStrategy, "component-strategy", "", "collision strategy for other components")
	fs.BoolVar(&flags.noMergeArrays, "no-merge-arrays", false, "don't merge arrays (servers, security, etc.)")
	fs.BoolVar(&flags.noDedupTags, "no-dedup-tags", false, "don't deduplicate tags by name")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: oastools join [flags] <file1> <file2> [file3...]\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Join multiple OpenAPI specification files into a single document.\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(fs.Output(), "\nCollision Strategies:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  accept-left      Keep the first value when collisions occur\n")
		_, _ = fmt.Fprintf(fs.Output(), "  accept-right     Keep the last value when collisions occur (overwrite)\n")
		_, _ = fmt.Fprintf(fs.Output(), "  fail             Fail with an error on any collision\n")
		_, _ = fmt.Fprintf(fs.Output(), "  fail-on-paths    Fail only on path collisions, allow schema collisions\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nExamples:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools join -o merged.yaml base.yaml extensions.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools join --path-strategy accept-left -o api.yaml spec1.yaml spec2.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools join --schema-strategy accept-right -o output.yaml api1.yaml api2.yaml api3.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nNotes:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - All input files must be the same major OAS version (2.0 or 3.x)\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - The output will use the version of the first input file\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Info section is taken from the first document by default\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Output file is written with restrictive permissions (0600) for security\n")
	}

	return fs, flags
}

func handleJoin(args []string) error {
	fs, flags := setupJoinFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("join command requires at least 2 input files")
	}

	if flags.output == "" {
		fs.Usage()
		return fmt.Errorf("output file is required (use -o or --output)")
	}

	filePaths := fs.Args()

	// Build configuration
	config := joiner.DefaultConfig()
	config.MergeArrays = !flags.noMergeArrays
	config.DeduplicateTags = !flags.noDedupTags

	// Validate and parse strategy flags
	if err := validateCollisionStrategy("path-strategy", flags.pathStrategy); err != nil {
		return err
	}
	if err := validateCollisionStrategy("schema-strategy", flags.schemaStrategy); err != nil {
		return err
	}
	if err := validateCollisionStrategy("component-strategy", flags.componentStrategy); err != nil {
		return err
	}

	// Apply validated strategies to config
	if flags.pathStrategy != "" {
		config.PathStrategy = joiner.CollisionStrategy(flags.pathStrategy)
	}
	if flags.schemaStrategy != "" {
		config.SchemaStrategy = joiner.CollisionStrategy(flags.schemaStrategy)
	}
	if flags.componentStrategy != "" {
		config.ComponentStrategy = joiner.CollisionStrategy(flags.componentStrategy)
	}

	// Validate output path before joining
	if err := validateOutputPath(flags.output, filePaths); err != nil {
		return err
	}

	// Create joiner and execute with timing
	startTime := time.Now()
	j := joiner.New(config)
	result, err := j.Join(filePaths)
	if err != nil {
		return fmt.Errorf("joining specifications: %w", err)
	}

	// Write result to file
	err = j.WriteResult(result, flags.output)
	totalTime := time.Since(startTime)
	if err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	// Print success message
	fmt.Printf("OpenAPI Specification Joiner\n")
	fmt.Printf("============================\n\n")
	fmt.Printf("oastools version: %s\n", oastools.Version())
	fmt.Printf("Successfully joined %d specification files\n", len(filePaths))
	fmt.Printf("Output: %s\n", flags.output)
	fmt.Printf("OAS Version: %s\n", result.Version)
	fmt.Printf("Paths: %d\n", result.Stats.PathCount)
	fmt.Printf("Operations: %d\n", result.Stats.OperationCount)
	fmt.Printf("Schemas: %d\n", result.Stats.SchemaCount)
	fmt.Printf("Total Time: %v\n\n", totalTime)

	if result.CollisionCount > 0 {
		fmt.Printf("Collisions resolved: %d\n\n", result.CollisionCount)
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings (%d):\n", len(result.Warnings))
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
		fmt.Println()
	}

	fmt.Printf("✓ Join completed successfully!\n")
	return nil
}

// validateOutputPath checks if the output path is safe to write to
func validateOutputPath(outputPath string, inputPaths []string) error {
	// Get absolute path of output file
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Check if output file would overwrite any input files
	for _, inputPath := range inputPaths {
		absInputPath, err := filepath.Abs(inputPath)
		if err != nil {
			return fmt.Errorf("invalid input path %s: %w", inputPath, err)
		}

		if absOutputPath == absInputPath {
			return fmt.Errorf("output file %s would overwrite input file %s", outputPath, inputPath)
		}
	}

	// Check if output file already exists and warn (but don't error)
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Fprintf(os.Stderr, "Warning: output file %s already exists and will be overwritten\n", outputPath)
	}

	return nil
}

// convertFlags contains flags for the convert command
type convertFlags struct {
	target     string
	output     string
	strict     bool
	noWarnings bool
	quiet      bool
}

// setupConvertFlags creates and configures a FlagSet for the convert command.
// Returns the FlagSet and a convertFlags struct with bound flag variables.
func setupConvertFlags() (*flag.FlagSet, *convertFlags) {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	flags := &convertFlags{}

	fs.StringVar(&flags.target, "t", "", "target OAS version (e.g., \"3.0.3\", \"2.0\", \"3.1.0\") (required)")
	fs.StringVar(&flags.target, "target", "", "target OAS version (e.g., \"3.0.3\", \"2.0\", \"3.1.0\") (required)")
	fs.StringVar(&flags.output, "o", "", "output file path (default: stdout)")
	fs.StringVar(&flags.output, "output", "", "output file path (default: stdout)")
	fs.BoolVar(&flags.strict, "strict", false, "fail on any conversion issues (even warnings)")
	fs.BoolVar(&flags.noWarnings, "no-warnings", false, "suppress warning and info messages")
	fs.BoolVar(&flags.quiet, "q", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.quiet, "quiet", false, "quiet mode: only output the document, no diagnostic messages")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: oastools convert [flags] <file|url|->\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Convert an OpenAPI specification from one version to another.\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(fs.Output(), "\nSupported Conversions:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - OAS 2.0 → OAS 3.x (3.0.0 through 3.2.0)\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - OAS 3.x → OAS 2.0\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - OAS 3.x → OAS 3.y (version updates)\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nExamples:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools convert -t 3.0.3 https://example.com/swagger.yaml -o openapi.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools convert -t 2.0 openapi-v3.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools convert --strict -t 3.1.0 swagger.yaml -o openapi-v3.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  cat swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nPipelining:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Use '-' as the file path to read from stdin\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nNotes:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Critical issues indicate features that cannot be converted (data loss)\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Warnings indicate lossy conversions or best-effort transformations\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Info messages provide context about conversion choices\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Always validate converted documents before deployment\n")
	}

	return fs, flags
}

func handleConvert(args []string) error {
	fs, flags := setupConvertFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("convert command requires exactly one file path, URL, or '-' for stdin")
	}

	specPath := fs.Arg(0)

	if flags.target == "" {
		fs.Usage()
		return fmt.Errorf("target version is required (use -t or --target)")
	}

	// Create converter with options
	c := converter.New()
	c.StrictMode = flags.strict
	c.IncludeInfo = !flags.noWarnings

	// Convert the file, URL, or stdin with timing
	startTime := time.Now()
	var result *converter.ConversionResult
	var err error

	if specPath == "-" {
		// Read from stdin
		p := parser.New()
		parseResult, err := p.ParseReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("parsing stdin: %w", err)
		}
		result, err = c.ConvertParsed(*parseResult, flags.target)
		if err != nil {
			return fmt.Errorf("converting from stdin: %w", err)
		}
	} else {
		result, err = c.Convert(specPath, flags.target)
		if err != nil {
			return fmt.Errorf("converting file: %w", err)
		}
	}
	totalTime := time.Since(startTime)

	// Print results (to stderr in quiet mode)
	if !flags.quiet {
		fmt.Fprintf(os.Stderr, "OpenAPI Specification Converter\n")
		fmt.Fprintf(os.Stderr, "===============================\n\n")
		fmt.Fprintf(os.Stderr, "oastools version: %s\n", oastools.Version())
		if specPath == "-" {
			fmt.Fprintf(os.Stderr, "Specification: <stdin>\n")
		} else {
			fmt.Fprintf(os.Stderr, "Specification: %s\n", specPath)
		}
		fmt.Fprintf(os.Stderr, "Source Version: %s\n", result.SourceVersion)
		fmt.Fprintf(os.Stderr, "Target Version: %s\n", result.TargetVersion)
		fmt.Fprintf(os.Stderr, "Source Size: %s\n", parser.FormatBytes(result.SourceSize))
		fmt.Fprintf(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		fmt.Fprintf(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		fmt.Fprintf(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		fmt.Fprintf(os.Stderr, "Load Time: %v\n", result.LoadTime)
		fmt.Fprintf(os.Stderr, "Total Time: %v\n\n", totalTime)

		// Print issues
		if len(result.Issues) > 0 {
			fmt.Fprintf(os.Stderr, "Conversion Issues (%d):\n", len(result.Issues))
			for _, issue := range result.Issues {
				fmt.Fprintf(os.Stderr, "  %s\n", issue.String())
			}
			fmt.Fprintf(os.Stderr, "\n")
		}

		// Print summary
		if result.Success {
			fmt.Fprintf(os.Stderr, "✓ Conversion successful")
			if result.InfoCount > 0 || result.WarningCount > 0 {
				fmt.Fprintf(os.Stderr, " (%d info, %d warnings)", result.InfoCount, result.WarningCount)
			}
			fmt.Fprintf(os.Stderr, "\n")
		} else {
			fmt.Fprintf(os.Stderr, "✗ Conversion completed with %d critical issue(s)", result.CriticalCount)
			if result.WarningCount > 0 {
				fmt.Fprintf(os.Stderr, ", %d warning(s)", result.WarningCount)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	// Write output
	data, err := marshalDocument(result.Document, result.SourceFormat)
	if err != nil {
		return fmt.Errorf("marshaling converted document: %w", err)
	}

	if flags.output != "" {
		if err := os.WriteFile(flags.output, data, 0600); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		if !flags.quiet {
			fmt.Fprintf(os.Stderr, "\nOutput written to: %s\n", flags.output)
		}
	} else {
		// Write to stdout
		if _, err = os.Stdout.Write(data); err != nil {
			return fmt.Errorf("writing converted document to stdout: %w", err)
		}
	}

	// Exit with error if conversion failed
	if !result.Success {
		os.Exit(1)
	}

	return nil
}

// marshalDocument marshals a document to bytes in the specified format
func marshalDocument(doc any, format parser.SourceFormat) ([]byte, error) {
	if format == parser.SourceFormatJSON {
		return json.MarshalIndent(doc, "", "  ")
	}
	return yaml.Marshal(doc)
}

// diffFlags contains flags for the diff command
type diffFlags struct {
	breaking bool
	noInfo   bool
	format   string
}

// setupDiffFlags creates and configures a FlagSet for the diff command.
// Returns the FlagSet and a diffFlags struct with bound flag variables.
func setupDiffFlags() (*flag.FlagSet, *diffFlags) {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	flags := &diffFlags{}

	fs.BoolVar(&flags.breaking, "breaking", false, "enable breaking change detection and categorization")
	fs.BoolVar(&flags.noInfo, "no-info", false, "exclude informational changes from output")
	fs.StringVar(&flags.format, "format", "text", "output format: text, json, or yaml")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: oastools diff [flags] <source> <target>\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Compare two OpenAPI specification files or URLs and report differences.\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(fs.Output(), "\nOutput Formats:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  text (default)  Human-readable text output\n")
		_, _ = fmt.Fprintf(fs.Output(), "  json            JSON format for programmatic processing\n")
		_, _ = fmt.Fprintf(fs.Output(), "  yaml            YAML format for programmatic processing\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nModes:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Default (Simple):\n")
		_, _ = fmt.Fprintf(fs.Output(), "    Reports all semantic differences between specifications without\n")
		_, _ = fmt.Fprintf(fs.Output(), "    categorizing them by severity or breaking change impact.\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "  --breaking (Breaking Change Detection):\n")
		_, _ = fmt.Fprintf(fs.Output(), "    Categorizes changes by severity and identifies breaking API changes:\n")
		_, _ = fmt.Fprintf(fs.Output(), "    - Critical: Removed endpoints or operations\n")
		_, _ = fmt.Fprintf(fs.Output(), "    - Error:    Removed required parameters, incompatible type changes\n")
		_, _ = fmt.Fprintf(fs.Output(), "    - Warning:  Deprecated operations, added required fields\n")
		_, _ = fmt.Fprintf(fs.Output(), "    - Info:     Additions, relaxed constraints, documentation updates\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nExamples:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools diff api-v1.yaml api-v2.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools diff --breaking api-v1.yaml api-v2.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools diff --breaking --no-info old.yaml new.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools diff --format json --breaking api-v1.yaml api-v2.yaml | jq '.HasBreakingChanges'\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools diff https://example.com/api/v1.yaml https://example.com/api/v2.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nExit Status:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  0    No differences found (or no breaking changes in --breaking mode)\n")
		_, _ = fmt.Fprintf(fs.Output(), "  1    Differences found (or breaking changes found in --breaking mode)\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nNotes:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Both specifications must be valid OpenAPI documents\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Cross-version comparison (2.0 vs 3.x) is supported with limitations\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Breaking change detection helps identify backward compatibility issues\n")
	}

	return fs, flags
}

func handleDiff(args []string) error {
	fs, flags := setupDiffFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("diff command requires exactly two file paths or URLs")
	}

	sourcePath := fs.Arg(0)
	targetPath := fs.Arg(1)

	// Validate format flag
	if flags.format != "text" && flags.format != "json" && flags.format != "yaml" {
		return fmt.Errorf("invalid format '%s'. Valid formats: text, json, yaml", flags.format)
	}

	// Create differ with options
	d := differ.New()
	if flags.breaking {
		d.Mode = differ.ModeBreaking
	} else {
		d.Mode = differ.ModeSimple
	}
	d.IncludeInfo = !flags.noInfo

	// Diff the files with timing
	startTime := time.Now()
	result, err := d.Diff(sourcePath, targetPath)
	totalTime := time.Since(startTime)
	if err != nil {
		return fmt.Errorf("comparing specifications: %w", err)
	}

	// Handle structured output formats
	if flags.format == "json" || flags.format == "yaml" {
		var data []byte
		var err error
		
		if flags.format == "json" {
			data, err = json.MarshalIndent(result, "", "  ")
		} else {
			data, err = yaml.Marshal(result)
		}
		
		if err != nil {
			return fmt.Errorf("marshaling diff result: %w", err)
		}
		
		fmt.Println(string(data))
		
		// Exit with error if breaking changes found (in breaking mode)
		if flags.breaking && result.HasBreakingChanges {
			os.Exit(1)
		}
		
		return nil
	}

	// Text format output (original behavior)
	// Print results
	fmt.Printf("OpenAPI Specification Diff\n")
	fmt.Printf("==========================\n\n")
	fmt.Printf("oastools version: %s\n", oastools.Version())
	fmt.Printf("Source: %s (%s)\n", sourcePath, result.SourceVersion)
	fmt.Printf("Target: %s (%s)\n", targetPath, result.TargetVersion)
	fmt.Printf("Total Time: %v\n\n", totalTime)

	if len(result.Changes) == 0 {
		fmt.Println("✓ No differences found - specifications are identical")
		return nil
	}

	// Print changes grouped by category if in breaking mode
	if flags.breaking {
		// Group changes by category
		categories := make(map[differ.ChangeCategory][]differ.Change)
		for _, change := range result.Changes {
			categories[change.Category] = append(categories[change.Category], change)
		}

		// Print each category
		categoryOrder := []differ.ChangeCategory{
			differ.CategoryEndpoint,
			differ.CategoryOperation,
			differ.CategoryParameter,
			differ.CategoryRequestBody,
			differ.CategoryResponse,
			differ.CategorySchema,
			differ.CategorySecurity,
			differ.CategoryServer,
			differ.CategoryInfo,
		}

		for _, category := range categoryOrder {
			changes := categories[category]
			if len(changes) == 0 {
				continue
			}

			fmt.Printf("%s Changes (%d):\n", category, len(changes))
			for _, change := range changes {
				fmt.Printf("  %s\n", change.String())
			}
			fmt.Println()
		}

		// Print summary
		fmt.Printf("Summary:\n")
		fmt.Printf("  Total changes: %d\n", len(result.Changes))
		if result.HasBreakingChanges {
			fmt.Printf("  ⚠️  Breaking changes: %d\n", result.BreakingCount)
		} else {
			fmt.Printf("  ✓ Breaking changes: 0\n")
		}
		fmt.Printf("  Warnings: %d\n", result.WarningCount)
		if d.IncludeInfo {
			fmt.Printf("  Info: %d\n", result.InfoCount)
		}

		// Exit with error if breaking changes found
		if result.HasBreakingChanges {
			os.Exit(1)
		}
	} else {
		// Simple mode - just print all changes
		fmt.Printf("Changes (%d):\n", len(result.Changes))
		for _, change := range result.Changes {
			fmt.Printf("  %s\n", change.String())
		}
	}

	return nil
}

// generateFlags contains flags for the generate command
type generateFlags struct {
	output       string
	packageName  string
	client       bool
	server       bool
	types        bool
	noPointers   bool
	noValidation bool
	strict       bool
	noWarnings   bool
}

// setupGenerateFlags creates and configures a FlagSet for the generate command.
// Returns the FlagSet and a generateFlags struct with bound flag variables.
func setupGenerateFlags() (*flag.FlagSet, *generateFlags) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	flags := &generateFlags{}

	fs.StringVar(&flags.output, "o", "", "output directory for generated files (required)")
	fs.StringVar(&flags.output, "output", "", "output directory for generated files (required)")
	fs.StringVar(&flags.packageName, "p", "api", "Go package name for generated code")
	fs.StringVar(&flags.packageName, "package", "api", "Go package name for generated code")
	fs.BoolVar(&flags.client, "client", false, "generate HTTP client code")
	fs.BoolVar(&flags.server, "server", false, "generate server interface code")
	fs.BoolVar(&flags.types, "types", true, "generate type definitions from schemas")
	fs.BoolVar(&flags.noPointers, "no-pointers", false, "don't use pointer types for optional fields")
	fs.BoolVar(&flags.noValidation, "no-validation", false, "don't include validation tags")
	fs.BoolVar(&flags.strict, "strict", false, "fail on any generation issues (even warnings)")
	fs.BoolVar(&flags.noWarnings, "no-warnings", false, "suppress warning and info messages")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: oastools generate [flags] <file|url>\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Generate Go code from an OpenAPI specification.\n\n")
		_, _ = fmt.Fprintf(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(fs.Output(), "\nExamples:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools generate --client -o ./client openapi.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools generate --server -o ./server -p myapi openapi.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools generate --client --server -o ./api petstore.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "  oastools generate --types -o ./models https://example.com/api/openapi.yaml\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nNotes:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - At least one of --client, --server, or --types must be enabled\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Types are always generated when --client or --server is enabled\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Generated code uses Go idioms and best practices\n")
		_, _ = fmt.Fprintf(fs.Output(), "  - Server interface is framework-agnostic\n")
	}

	return fs, flags
}

func handleGenerate(args []string) error {
	fs, flags := setupGenerateFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("generate command requires exactly one file path or URL")
	}

	specPath := fs.Arg(0)

	if flags.output == "" {
		fs.Usage()
		return fmt.Errorf("output directory is required (use -o or --output)")
	}

	// Ensure at least one generation mode is enabled
	if !flags.client && !flags.server && !flags.types {
		fs.Usage()
		return fmt.Errorf("at least one of --client, --server, or --types must be enabled")
	}

	// Create generator with options
	g := generator.New()
	g.PackageName = flags.packageName
	g.GenerateClient = flags.client
	g.GenerateServer = flags.server
	g.GenerateTypes = flags.types || flags.client || flags.server
	g.UsePointers = !flags.noPointers
	g.IncludeValidation = !flags.noValidation
	g.StrictMode = flags.strict
	g.IncludeInfo = !flags.noWarnings

	// Generate the code with timing
	startTime := time.Now()
	result, err := g.Generate(specPath)
	totalTime := time.Since(startTime)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Print results
	fmt.Printf("OpenAPI Code Generator\n")
	fmt.Printf("=====================\n\n")
	fmt.Printf("oastools version: %s\n", oastools.Version())
	fmt.Printf("Specification: %s\n", specPath)
	fmt.Printf("OAS Version: %s\n", result.SourceVersion)
	fmt.Printf("Source Size: %s\n", parser.FormatBytes(result.SourceSize))
	fmt.Printf("Package: %s\n", result.PackageName)
	fmt.Printf("Types: %d\n", result.GeneratedTypes)
	fmt.Printf("Operations: %d\n", result.GeneratedOperations)
	fmt.Printf("Total Time: %v\n\n", totalTime)

	// Print issues
	if len(result.Issues) > 0 {
		fmt.Printf("Generation Issues (%d):\n", len(result.Issues))
		for _, issue := range result.Issues {
			fmt.Printf("  %s\n", issue.String())
		}
		fmt.Println()
	}

	// Write files
	if err := result.WriteFiles(flags.output); err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	// Print generated files
	fmt.Printf("Generated Files (%d):\n", len(result.Files))
	for _, file := range result.Files {
		fmt.Printf("  - %s/%s (%d bytes)\n", flags.output, file.Name, len(file.Content))
	}
	fmt.Println()

	// Print summary
	if result.Success {
		fmt.Printf("✓ Generation successful")
		if result.InfoCount > 0 || result.WarningCount > 0 {
			fmt.Printf(" (%d info, %d warnings)", result.InfoCount, result.WarningCount)
		}
		fmt.Println()
	} else {
		fmt.Printf("✗ Generation completed with %d critical issue(s)", result.CriticalCount)
		if result.WarningCount > 0 {
			fmt.Printf(", %d warning(s)", result.WarningCount)
		}
		fmt.Println()
		os.Exit(1)
	}

	return nil
}

func printUsage() {
	fmt.Println(`oastools - OpenAPI Specification Tools

Usage:
  oastools <command> [options]

Commands:
  validate    Validate an OpenAPI specification file or URL
  convert     Convert between OpenAPI specification versions
  diff        Compare two OpenAPI specifications and detect changes
  generate    Generate Go client/server code from an OpenAPI specification
  join        Join multiple OpenAPI specification files
  parse       Parse and display an OpenAPI specification file or URL
  version     Show version information
  help        Show this help message

Examples:
  oastools validate openapi.yaml
  oastools validate https://example.com/api/openapi.yaml
  oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml
  oastools diff --breaking api-v1.yaml api-v2.yaml
  oastools generate --client -o ./client openapi.yaml
  oastools join -o merged.yaml base.yaml extensions.yaml
  oastools parse https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/examples/v3.0/petstore.yaml

Run 'oastools <command> --help' for more information on a command.`)
}
