package commands

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/generator"
	"github.com/erraggy/oastools/internal/cliutil"
	"github.com/erraggy/oastools/parser"
)

// GenerateFlags contains flags for the generate command
type GenerateFlags struct {
	Output       string
	PackageName  string
	Client       bool
	Server       bool
	Types        bool
	NoPointers   bool
	NoValidation bool
	Strict       bool
	NoWarnings   bool
}

// SetupGenerateFlags creates and configures a FlagSet for the generate command.
// Returns the FlagSet and a GenerateFlags struct with bound flag variables.
func SetupGenerateFlags() (*flag.FlagSet, *GenerateFlags) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	flags := &GenerateFlags{}

	fs.StringVar(&flags.Output, "o", "", "output directory for generated files (required)")
	fs.StringVar(&flags.Output, "output", "", "output directory for generated files (required)")
	fs.StringVar(&flags.PackageName, "p", "api", "Go package name for generated code")
	fs.StringVar(&flags.PackageName, "package", "api", "Go package name for generated code")
	fs.BoolVar(&flags.Client, "client", false, "generate HTTP client code")
	fs.BoolVar(&flags.Server, "server", false, "generate server interface code")
	fs.BoolVar(&flags.Types, "types", true, "generate type definitions from schemas")
	fs.BoolVar(&flags.NoPointers, "no-pointers", false, "don't use pointer types for optional fields")
	fs.BoolVar(&flags.NoValidation, "no-validation", false, "don't include validation tags")
	fs.BoolVar(&flags.Strict, "strict", false, "fail on any generation issues (even warnings)")
	fs.BoolVar(&flags.NoWarnings, "no-warnings", false, "suppress warning and info messages")

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools generate [flags] <file|url>\n\n")
		cliutil.Writef(fs.Output(), "Generate Go code from an OpenAPI specification.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools generate --client -o ./client openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --server -o ./server -p myapi openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --client --server -o ./api petstore.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --types -o ./models https://example.com/api/openapi.yaml\n")
		cliutil.Writef(fs.Output(), "\nNotes:\n")
		cliutil.Writef(fs.Output(), "  - At least one of --client, --server, or --types must be enabled\n")
		cliutil.Writef(fs.Output(), "  - Types are always generated when --client or --server is enabled\n")
		cliutil.Writef(fs.Output(), "  - Generated code uses Go idioms and best practices\n")
		cliutil.Writef(fs.Output(), "  - Server interface is framework-agnostic\n")
	}

	return fs, flags
}

// HandleGenerate executes the generate command
func HandleGenerate(args []string) error {
	fs, flags := SetupGenerateFlags()

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

	if flags.Output == "" {
		fs.Usage()
		return fmt.Errorf("output directory is required (use -o or --output)")
	}

	// Ensure at least one generation mode is enabled
	if !flags.Client && !flags.Server && !flags.Types {
		fs.Usage()
		return fmt.Errorf("at least one of --client, --server, or --types must be enabled")
	}

	// Create generator with options
	g := generator.New()
	g.PackageName = flags.PackageName
	g.GenerateClient = flags.Client
	g.GenerateServer = flags.Server
	g.GenerateTypes = flags.Types || flags.Client || flags.Server
	g.UsePointers = !flags.NoPointers
	g.IncludeValidation = !flags.NoValidation
	g.StrictMode = flags.Strict
	g.IncludeInfo = !flags.NoWarnings

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
	if err := result.WriteFiles(flags.Output); err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	// Print generated files
	fmt.Printf("Generated Files (%d):\n", len(result.Files))
	for _, file := range result.Files {
		fmt.Printf("  - %s/%s (%d bytes)\n", flags.Output, file.Name, len(file.Content))
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
		return fmt.Errorf("generation failed with %d critical issue(s)", result.CriticalCount)
	}

	return nil
}
