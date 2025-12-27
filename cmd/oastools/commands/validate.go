package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

// ValidateFlags contains flags for the validate command
type ValidateFlags struct {
	Strict     bool
	NoWarnings bool
	Quiet      bool
	Format     string
	SourceMap  bool
}

// SetupValidateFlags creates and configures a FlagSet for the validate command.
// Returns the FlagSet and a ValidateFlags struct with bound flag variables.
func SetupValidateFlags() (*flag.FlagSet, *ValidateFlags) {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags := &ValidateFlags{}

	fs.BoolVar(&flags.Strict, "strict", false, "enable stricter validation beyond spec requirements")
	fs.BoolVar(&flags.NoWarnings, "no-warnings", false, "suppress warning messages (only show errors)")
	fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: only output validation result, no diagnostic messages")
	fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: only output validation result, no diagnostic messages")
	fs.StringVar(&flags.Format, "format", FormatText, "output format: text, json, or yaml")
	fs.BoolVar(&flags.SourceMap, "source-map", false, "include line numbers in validation errors (IDE-friendly format)")
	fs.BoolVar(&flags.SourceMap, "s", false, "include line numbers in validation errors (IDE-friendly format)")

	fs.Usage = func() {
		Writef(fs.Output(), "Usage: oastools validate [flags] <file|url|->\n\n")
		Writef(fs.Output(), "Validate an OpenAPI specification file, URL, or stdin against the specification version it declares.\n\n")
		Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		Writef(fs.Output(), "\nOutput Formats:\n")
		Writef(fs.Output(), "  text (default)  Human-readable text output\n")
		Writef(fs.Output(), "  json            JSON format for programmatic processing\n")
		Writef(fs.Output(), "  yaml            YAML format for programmatic processing\n")
		Writef(fs.Output(), "\nExamples:\n")
		Writef(fs.Output(), "  oastools validate openapi.yaml\n")
		Writef(fs.Output(), "  oastools validate https://example.com/api/openapi.yaml\n")
		Writef(fs.Output(), "  oastools validate --strict api-spec.yaml\n")
		Writef(fs.Output(), "  oastools validate --no-warnings openapi.json\n")
		Writef(fs.Output(), "  cat openapi.yaml | oastools validate -q -\n")
		Writef(fs.Output(), "  oastools validate --format json openapi.yaml | jq '.valid'\n")
		Writef(fs.Output(), "  oastools validate -s openapi.yaml  # Include line numbers in errors\n")
		Writef(fs.Output(), "\nPipelining:\n")
		Writef(fs.Output(), "  - Use '-' as the file path to read from stdin\n")
		Writef(fs.Output(), "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		Writef(fs.Output(), "  - Use --format json/yaml for structured output that can be parsed\n")
		Writef(fs.Output(), "\nExit Codes:\n")
		Writef(fs.Output(), "  0    Validation successful\n")
		Writef(fs.Output(), "  1    Validation failed with errors\n")
	}

	return fs, flags
}

// HandleValidate executes the validate command
func HandleValidate(args []string) error {
	fs, flags := SetupValidateFlags()

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

	// Validate format flag early to fail fast before expensive operations
	if err := ValidateOutputFormat(flags.Format); err != nil {
		return err
	}

	// Validate the file, URL, or stdin with timing
	startTime := time.Now()
	var result *validator.ValidationResult
	var err error

	if specPath == StdinFilePath {
		// Read from stdin - source map not supported for stdin
		p := parser.New()
		parseResult, parseErr := p.ParseReader(os.Stdin)
		if parseErr != nil {
			return fmt.Errorf("parsing stdin: %w", parseErr)
		}
		v := validator.New()
		v.StrictMode = flags.Strict
		v.IncludeWarnings = !flags.NoWarnings
		result, err = v.ValidateParsed(*parseResult)
		if err != nil {
			return fmt.Errorf("validating from stdin: %w", err)
		}
	} else {
		// Build validation options
		validateOpts := []validator.Option{
			validator.WithFilePath(specPath),
			validator.WithStrictMode(flags.Strict),
			validator.WithIncludeWarnings(!flags.NoWarnings),
		}

		// If source map requested, parse with source map first
		if flags.SourceMap {
			parseResult, parseErr := parser.ParseWithOptions(
				parser.WithFilePath(specPath),
				parser.WithSourceMap(true),
			)
			if parseErr != nil {
				return fmt.Errorf("parsing file: %w", parseErr)
			}
			validateOpts = []validator.Option{
				validator.WithParsed(*parseResult),
				validator.WithStrictMode(flags.Strict),
				validator.WithIncludeWarnings(!flags.NoWarnings),
			}
			if parseResult.SourceMap != nil {
				validateOpts = append(validateOpts, validator.WithSourceMap(parseResult.SourceMap))
			}
		}

		result, err = validator.ValidateWithOptions(validateOpts...)
		if err != nil {
			return fmt.Errorf("validating file: %w", err)
		}
	}
	totalTime := time.Since(startTime)

	// Handle structured output formats
	if flags.Format == FormatJSON || flags.Format == FormatYAML {
		if err := OutputStructured(result, flags.Format); err != nil {
			return err
		}

		// Exit with error if validation failed
		if !result.Valid {
			os.Exit(1)
		}

		return nil
	}

	// Text format output (original behavior)
	// Print results (always to stderr to be consistent with parse and convert)
	if !flags.Quiet {
		Writef(os.Stderr, "OpenAPI Specification Validator\n")
		Writef(os.Stderr, "================================\n\n")
		Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		Writef(os.Stderr, "Specification: %s\n", FormatSpecPath(specPath))
		Writef(os.Stderr, "OAS Version: %s\n", result.Version)
		Writef(os.Stderr, "Source Size: %s\n", parser.FormatBytes(result.SourceSize))
		Writef(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		Writef(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		Writef(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		Writef(os.Stderr, "Load Time: %v\n", result.LoadTime)
		Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		// Print errors
		if len(result.Errors) > 0 {
			Writef(os.Stderr, "Errors (%d):\n", result.ErrorCount)
			for _, e := range result.Errors {
				if flags.SourceMap && e.HasLocation() {
					// IDE-friendly format: file:line:column: path: message
					Writef(os.Stderr, "  %s: %s: %s\n", e.Location(), e.Path, e.Message)
				} else {
					Writef(os.Stderr, "  %s\n", e.String())
				}
			}
			Writef(os.Stderr, "\n")
		}

		// Print warnings
		if len(result.Warnings) > 0 {
			Writef(os.Stderr, "Warnings (%d):\n", result.WarningCount)
			for _, warning := range result.Warnings {
				if flags.SourceMap && warning.HasLocation() {
					// IDE-friendly format: file:line:column: path: message
					Writef(os.Stderr, "  %s: %s: %s\n", warning.Location(), warning.Path, warning.Message)
				} else {
					Writef(os.Stderr, "  %s\n", warning.String())
				}
			}
			Writef(os.Stderr, "\n")
		}
	}

	// Print summary (only in non-quiet mode to respect --quiet flag)
	if !flags.Quiet {
		if result.Valid {
			Writef(os.Stderr, "✓ Validation passed")
			if result.WarningCount > 0 {
				Writef(os.Stderr, " with %d warning(s)", result.WarningCount)
			}
			Writef(os.Stderr, "\n")
		} else {
			Writef(os.Stderr, "✗ Validation failed: %d error(s)", result.ErrorCount)
			if result.WarningCount > 0 {
				Writef(os.Stderr, ", %d warning(s)", result.WarningCount)
			}
			Writef(os.Stderr, "\n")
		}
	}

	// Exit with error if validation failed
	if !result.Valid {
		os.Exit(1)
	}

	return nil
}
