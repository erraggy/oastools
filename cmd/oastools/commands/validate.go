package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/internal/cliutil"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

// ValidateFlags contains flags for the validate command
type ValidateFlags struct {
	Strict     bool
	NoWarnings bool
	Quiet      bool
	Format     string
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

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools validate [flags] <file|url|->\n\n")
		cliutil.Writef(fs.Output(), "Validate an OpenAPI specification file, URL, or stdin against the specification version it declares.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nOutput Formats:\n")
		cliutil.Writef(fs.Output(), "  text (default)  Human-readable text output\n")
		cliutil.Writef(fs.Output(), "  json            JSON format for programmatic processing\n")
		cliutil.Writef(fs.Output(), "  yaml            YAML format for programmatic processing\n")
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools validate openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools validate https://example.com/api/openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools validate --strict api-spec.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools validate --no-warnings openapi.json\n")
		cliutil.Writef(fs.Output(), "  cat openapi.yaml | oastools validate -q -\n")
		cliutil.Writef(fs.Output(), "  oastools validate --format json openapi.yaml | jq '.valid'\n")
		cliutil.Writef(fs.Output(), "\nPipelining:\n")
		cliutil.Writef(fs.Output(), "  - Use '-' as the file path to read from stdin\n")
		cliutil.Writef(fs.Output(), "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		cliutil.Writef(fs.Output(), "  - Use --format json/yaml for structured output that can be parsed\n")
		cliutil.Writef(fs.Output(), "\nExit Codes:\n")
		cliutil.Writef(fs.Output(), "  0    Validation successful\n")
		cliutil.Writef(fs.Output(), "  1    Validation failed with errors\n")
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

	// Create validator with options
	v := validator.New()
	v.StrictMode = flags.Strict
	v.IncludeWarnings = !flags.NoWarnings

	// Validate the file, URL, or stdin with timing
	startTime := time.Now()
	var result *validator.ValidationResult
	var err error

	if specPath == StdinFilePath {
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
		cliutil.Writef(os.Stderr, "OpenAPI Specification Validator\n")
		cliutil.Writef(os.Stderr, "================================\n\n")
		cliutil.Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		if specPath == StdinFilePath {
			cliutil.Writef(os.Stderr, "Specification: <stdin>\n")
		} else {
			cliutil.Writef(os.Stderr, "Specification: %s\n", specPath)
		}
		cliutil.Writef(os.Stderr, "OAS Version: %s\n", result.Version)
		cliutil.Writef(os.Stderr, "Source Size: %s\n", parser.FormatBytes(result.SourceSize))
		cliutil.Writef(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		cliutil.Writef(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		cliutil.Writef(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		cliutil.Writef(os.Stderr, "Load Time: %v\n", result.LoadTime)
		cliutil.Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		// Print errors
		if len(result.Errors) > 0 {
			cliutil.Writef(os.Stderr, "Errors (%d):\n", result.ErrorCount)
			for _, err := range result.Errors {
				cliutil.Writef(os.Stderr, "  %s\n", err.String())
			}
			cliutil.Writef(os.Stderr, "\n")
		}

		// Print warnings
		if len(result.Warnings) > 0 {
			cliutil.Writef(os.Stderr, "Warnings (%d):\n", result.WarningCount)
			for _, warning := range result.Warnings {
				cliutil.Writef(os.Stderr, "  %s\n", warning.String())
			}
			cliutil.Writef(os.Stderr, "\n")
		}
	}

	// Print summary (only in non-quiet mode to respect --quiet flag)
	if !flags.Quiet {
		if result.Valid {
			cliutil.Writef(os.Stderr, "✓ Validation passed")
			if result.WarningCount > 0 {
				cliutil.Writef(os.Stderr, " with %d warning(s)", result.WarningCount)
			}
			cliutil.Writef(os.Stderr, "\n")
		} else {
			cliutil.Writef(os.Stderr, "✗ Validation failed: %d error(s)", result.ErrorCount)
			if result.WarningCount > 0 {
				cliutil.Writef(os.Stderr, ", %d warning(s)", result.WarningCount)
			}
			cliutil.Writef(os.Stderr, "\n")
		}
	}

	// Exit with error if validation failed
	if !result.Valid {
		os.Exit(1)
	}

	return nil
}
