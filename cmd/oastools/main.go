package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/differ"
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
		handleValidate(os.Args[2:])
	case "parse":
		handleParse(os.Args[2:])
	case "join":
		handleJoin(os.Args[2:])
	case "convert":
		handleConvert(os.Args[2:])
	case "diff":
		handleDiff(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleParse(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: parse command requires a file path or URL\n\n")
		fmt.Println("Usage: oastools parse <file|url>")
		os.Exit(1)
	}

	specPath := args[0]

	// Create parser with version in User-Agent
	p := parser.New()

	// Parse the file or URL
	result, err := p.Parse(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("OpenAPI Specification Parser\n")
	fmt.Printf("============================\n\n")
	fmt.Printf("Specification: %s\n", specPath)
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Source Size: %s\n", parser.FormatBytes(result.SourceSize))
	fmt.Printf("Load Time: %v\n\n", result.LoadTime)

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
	var specPath string

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
			if specPath == "" {
				specPath = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument '%s'\n", arg)
				os.Exit(1)
			}
		}
	}

	if specPath == "" {
		fmt.Fprintf(os.Stderr, "Error: validate command requires a file path or URL\n\n")
		printValidateUsage()
		os.Exit(1)
	}

	// Create validator with options
	v := validator.New()
	v.StrictMode = strict
	v.IncludeWarnings = !noWarnings

	// Validate the file or URL with timing
	startTime := time.Now()
	result, err := v.Validate(specPath)
	totalTime := time.Since(startTime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error validating file: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("OpenAPI Specification Validator\n")
	fmt.Printf("================================\n\n")
	fmt.Printf("Specification: %s\n", specPath)
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Source Size: %s\n", parser.FormatBytes(result.SourceSize))
	fmt.Printf("Load Time: %v\n", result.LoadTime)
	fmt.Printf("Total Time: %v\n\n", totalTime)

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
	fmt.Println(`Usage: oastools validate [options] <file|url>

Validate an OpenAPI specification file or URL against the specification version it declares.

Options:
  --strict        Enable stricter validation beyond spec requirements
  --no-warnings   Suppress warning messages (only show errors)
  -h, --help      Show this help message

Examples:
  oastools validate openapi.yaml
  oastools validate https://example.com/api/openapi.yaml
  oastools validate --strict api-spec.yaml
  oastools validate --no-warnings openapi.json`)
}

// parseJoinFlags parses command-line arguments for the join command
// Returns config, filePaths, outputPath, showHelp flag, and error
func parseJoinFlags(args []string) (joiner.JoinerConfig, []string, string, bool, error) {
	var outputPath string
	var pathStrategy string
	var schemaStrategy string
	var componentStrategy string
	var noMergeArrays bool
	var noDedupTags bool
	var filePaths []string

	// Parse flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-o", "--output":
			if i+1 >= len(args) {
				return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("%s requires an argument", arg)
			}
			outputPath = args[i+1]
			i++
		case "--path-strategy":
			if i+1 >= len(args) {
				return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("%s requires an argument", arg)
			}
			pathStrategy = args[i+1]
			i++
		case "--schema-strategy":
			if i+1 >= len(args) {
				return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("%s requires an argument", arg)
			}
			schemaStrategy = args[i+1]
			i++
		case "--component-strategy":
			if i+1 >= len(args) {
				return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("%s requires an argument", arg)
			}
			componentStrategy = args[i+1]
			i++
		case "--no-merge-arrays":
			noMergeArrays = true
		case "--no-dedup-tags":
			noDedupTags = true
		case "-h", "--help":
			return joiner.JoinerConfig{}, nil, "", true, nil
		default:
			filePaths = append(filePaths, arg)
		}
	}

	if len(filePaths) < 2 {
		return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("join command requires at least 2 input files")
	}

	if outputPath == "" {
		return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("output file is required (use -o or --output)")
	}

	// Build configuration
	config := joiner.DefaultConfig()
	config.MergeArrays = !noMergeArrays
	config.DeduplicateTags = !noDedupTags

	// Validate and parse strategy flags
	if pathStrategy != "" {
		if !joiner.IsValidStrategy(pathStrategy) {
			return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("invalid path-strategy '%s'. Valid strategies: %v", pathStrategy, joiner.ValidStrategies())
		}
		config.PathStrategy = joiner.CollisionStrategy(pathStrategy)
	}
	if schemaStrategy != "" {
		if !joiner.IsValidStrategy(schemaStrategy) {
			return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("invalid schema-strategy '%s'. Valid strategies: %v", schemaStrategy, joiner.ValidStrategies())
		}
		config.SchemaStrategy = joiner.CollisionStrategy(schemaStrategy)
	}
	if componentStrategy != "" {
		if !joiner.IsValidStrategy(componentStrategy) {
			return joiner.JoinerConfig{}, nil, "", false, fmt.Errorf("invalid component-strategy '%s'. Valid strategies: %v", componentStrategy, joiner.ValidStrategies())
		}
		config.ComponentStrategy = joiner.CollisionStrategy(componentStrategy)
	}

	return config, filePaths, outputPath, false, nil
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

func handleJoin(args []string) {
	config, filePaths, outputPath, showHelp, err := parseJoinFlags(args)
	if showHelp {
		printJoinUsage()
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		printJoinUsage()
		os.Exit(1)
	}

	// Validate output path before joining
	if err := validateOutputPath(outputPath, filePaths); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create joiner and execute with timing
	startTime := time.Now()
	j := joiner.New(config)
	result, err := j.Join(filePaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error joining specifications: %v\n", err)
		os.Exit(1)
	}

	// Write result to file
	err = j.WriteResult(result, outputPath)
	totalTime := time.Since(startTime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	// Print success message
	fmt.Printf("OpenAPI Specification Joiner\n")
	fmt.Printf("============================\n\n")
	fmt.Printf("Successfully joined %d specification files\n", len(filePaths))
	fmt.Printf("Output: %s\n", outputPath)
	fmt.Printf("Version: %s\n", result.Version)
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
}

func printJoinUsage() {
	fmt.Println(`Usage: oastools join [options] <file1> <file2> [file3...]

Join multiple OpenAPI specification files into a single document.

Required Options:
  -o, --output <file>              Output file path

Strategy Options:
  --path-strategy <strategy>       Collision strategy for paths
                                   (accept-left, accept-right, fail, fail-on-paths)
                                   Default: fail
  --schema-strategy <strategy>     Collision strategy for schemas/definitions
                                   Default: accept-left
  --component-strategy <strategy>  Collision strategy for other components
                                   Default: accept-left

Other Options:
  --no-merge-arrays               Don't merge arrays (servers, security, etc.)
  --no-dedup-tags                 Don't deduplicate tags by name
  -h, --help                      Show this help message

Collision Strategies:
  accept-left      Keep the first value when collisions occur
  accept-right     Keep the last value when collisions occur (overwrite)
  fail             Fail with an error on any collision
  fail-on-paths    Fail only on path collisions, allow schema collisions

Examples:
  oastools join -o merged.yaml base.yaml extensions.yaml
  oastools join --path-strategy accept-left -o api.yaml spec1.yaml spec2.yaml
  oastools join --schema-strategy accept-right -o output.yaml api1.yaml api2.yaml api3.yaml

Notes:
  - All input files must be the same major OAS version (2.0 or 3.x)
  - The output will use the version of the first input file
  - Info section is taken from the first document by default
  - Output file is written with restrictive permissions (0600) for security`)
}

// marshalDocument marshals a document to bytes in the specified format
func marshalDocument(doc any, format parser.SourceFormat) ([]byte, error) {
	if format == parser.SourceFormatJSON {
		return json.MarshalIndent(doc, "", "  ")
	}
	return yaml.Marshal(doc)
}

func handleConvert(args []string) {
	// Parse flags
	var targetVersion string
	var outputPath string
	var strict bool
	var noWarnings bool
	var specPath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-t", "--target":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires an argument\n", arg)
				os.Exit(1)
			}
			targetVersion = args[i+1]
			i++
		case "-o", "--output":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: %s requires an argument\n", arg)
				os.Exit(1)
			}
			outputPath = args[i+1]
			i++
		case "--strict":
			strict = true
		case "--no-warnings":
			noWarnings = true
		case "-h", "--help":
			printConvertUsage()
			return
		default:
			if specPath == "" {
				specPath = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument '%s'\n", arg)
				os.Exit(1)
			}
		}
	}

	if specPath == "" {
		fmt.Fprintf(os.Stderr, "Error: convert command requires a file path or URL\n\n")
		printConvertUsage()
		os.Exit(1)
	}

	if targetVersion == "" {
		fmt.Fprintf(os.Stderr, "Error: target version is required (use -t or --target)\n\n")
		printConvertUsage()
		os.Exit(1)
	}

	// Create converter with options
	c := converter.New()
	c.StrictMode = strict
	c.IncludeInfo = !noWarnings

	// Convert the file or URL with timing
	startTime := time.Now()
	result, err := c.Convert(specPath, targetVersion)
	totalTime := time.Since(startTime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting file: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("OpenAPI Specification Converter\n")
	fmt.Printf("===============================\n\n")
	fmt.Printf("Specification: %s\n", specPath)
	fmt.Printf("Source Version: %s\n", result.SourceVersion)
	fmt.Printf("Target Version: %s\n", result.TargetVersion)
	fmt.Printf("Source Size: %s\n", parser.FormatBytes(result.SourceSize))
	fmt.Printf("Load Time: %v\n", result.LoadTime)
	fmt.Printf("Total Time: %v\n\n", totalTime)

	// Print issues
	if len(result.Issues) > 0 {
		fmt.Printf("Conversion Issues (%d):\n", len(result.Issues))
		for _, issue := range result.Issues {
			fmt.Printf("  %s\n", issue.String())
		}
		fmt.Println()
	}

	// Print summary
	if result.Success {
		fmt.Printf("✓ Conversion successful")
		if result.InfoCount > 0 || result.WarningCount > 0 {
			fmt.Printf(" (%d info, %d warnings)", result.InfoCount, result.WarningCount)
		}
		fmt.Println()
	} else {
		fmt.Printf("✗ Conversion completed with %d critical issue(s)", result.CriticalCount)
		if result.WarningCount > 0 {
			fmt.Printf(", %d warning(s)", result.WarningCount)
		}
		fmt.Println()
	}

	// Write output
	data, err := marshalDocument(result.Document, result.SourceFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling converted document: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" {
		if err := os.WriteFile(outputPath, data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nOutput written to: %s\n", outputPath)
	} else {
		// Write to stdout
		if _, err = os.Stdout.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing converted document to stdout: %v\n", err)
			os.Exit(1)
		}
	}

	// Exit with error if conversion failed
	if !result.Success {
		os.Exit(1)
	}
}

func printConvertUsage() {
	fmt.Println(`Usage: oastools convert [options] <file|url>

Convert an OpenAPI specification from one version to another.

Required Options:
  -t, --target <version>  Target OAS version (e.g., "3.0.3", "2.0", "3.1.0")

Optional:
  -o, --output <file>     Output file path (default: stdout)
  --strict                Fail on any conversion issues (even warnings)
  --no-warnings           Suppress warning and info messages
  -h, --help              Show this help message

Supported Conversions:
  - OAS 2.0 → OAS 3.x (3.0.0 through 3.2.0)
  - OAS 3.x → OAS 2.0
  - OAS 3.x → OAS 3.y (version updates)

Examples:
  oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml
  oastools convert -t 3.0.3 https://example.com/swagger.yaml -o openapi.yaml
  oastools convert -t 2.0 openapi-v3.yaml
  oastools convert --strict -t 3.1.0 swagger.yaml -o openapi-v3.yaml

Notes:
  - Critical issues indicate features that cannot be converted (data loss)
  - Warnings indicate lossy conversions or best-effort transformations
  - Info messages provide context about conversion choices
  - Always validate converted documents before deployment`)
}

func handleDiff(args []string) {
	// Parse flags
	var breaking bool
	var noInfo bool
	var sourcePath string
	var targetPath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--breaking":
			breaking = true
		case "--no-info":
			noInfo = true
		case "-h", "--help":
			printDiffUsage()
			return
		default:
			if sourcePath == "" {
				sourcePath = arg
			} else if targetPath == "" {
				targetPath = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument '%s'\n", arg)
				os.Exit(1)
			}
		}
	}

	if sourcePath == "" || targetPath == "" {
		fmt.Fprintf(os.Stderr, "Error: diff command requires two file paths or URLs\n\n")
		printDiffUsage()
		os.Exit(1)
	}

	// Create differ with options
	d := differ.New()
	if breaking {
		d.Mode = differ.ModeBreaking
	} else {
		d.Mode = differ.ModeSimple
	}
	d.IncludeInfo = !noInfo

	// Diff the files with timing
	startTime := time.Now()
	result, err := d.Diff(sourcePath, targetPath)
	totalTime := time.Since(startTime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error comparing specifications: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("OpenAPI Specification Diff\n")
	fmt.Printf("==========================\n\n")
	fmt.Printf("Source: %s (%s)\n", sourcePath, result.SourceVersion)
	fmt.Printf("Target: %s (%s)\n", targetPath, result.TargetVersion)
	fmt.Printf("Total Time: %v\n\n", totalTime)

	if len(result.Changes) == 0 {
		fmt.Println("✓ No differences found - specifications are identical")
		return
	}

	// Print changes grouped by category if in breaking mode
	if breaking {
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
}

func printDiffUsage() {
	fmt.Println(`Usage: oastools diff [options] <source> <target>

Compare two OpenAPI specification files or URLs and report differences.

Options:
  --breaking      Enable breaking change detection and categorization
  --no-info       Exclude informational changes from output
  -h, --help      Show this help message

Modes:
  Default (Simple):
    Reports all semantic differences between specifications without
    categorizing them by severity or breaking change impact.

  --breaking (Breaking Change Detection):
    Categorizes changes by severity and identifies breaking API changes:
    - Critical: Removed endpoints or operations
    - Error:    Removed required parameters, incompatible type changes
    - Warning:  Deprecated operations, added required fields
    - Info:     Additions, relaxed constraints, documentation updates

Examples:
  oastools diff api-v1.yaml api-v2.yaml
  oastools diff --breaking api-v1.yaml api-v2.yaml
  oastools diff --breaking --no-info old.yaml new.yaml
  oastools diff https://example.com/api/v1.yaml https://example.com/api/v2.yaml

Exit Status:
  0    No differences found (or no breaking changes in --breaking mode)
  1    Differences found (or breaking changes found in --breaking mode)

Notes:
  - Both specifications must be valid OpenAPI documents
  - Cross-version comparison (2.0 vs 3.x) is supported with limitations
  - Breaking change detection helps identify backward compatibility issues`)
}

func printUsage() {
	fmt.Println(`oastools - OpenAPI Specification Tools

Usage:
  oastools <command> [options]

Commands:
  validate    Validate an OpenAPI specification file or URL
  convert     Convert between OpenAPI specification versions
  diff        Compare two OpenAPI specifications and detect changes
  join        Join multiple OpenAPI specification files
  parse       Parse and display an OpenAPI specification file or URL
  version     Show version information
  help        Show this help message

Examples:
  oastools validate openapi.yaml
  oastools validate https://example.com/api/openapi.yaml
  oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml
  oastools diff --breaking api-v1.yaml api-v2.yaml
  oastools join -o merged.yaml base.yaml extensions.yaml
  oastools parse https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/examples/v3.0/petstore.yaml

Run 'oastools <command> --help' for more information on a command.`)
}
