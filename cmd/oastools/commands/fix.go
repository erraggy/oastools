package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/internal/cliutil"
	"github.com/erraggy/oastools/parser"
)

// FixFlags contains flags for the fix command
type FixFlags struct {
	Output    string
	Infer     bool
	Quiet     bool
	SourceMap bool
}

// SetupFixFlags creates and configures a FlagSet for the fix command.
// Returns the FlagSet and a FixFlags struct with bound flag variables.
func SetupFixFlags() (*flag.FlagSet, *FixFlags) {
	fs := flag.NewFlagSet("fix", flag.ContinueOnError)
	flags := &FixFlags{}

	fs.StringVar(&flags.Output, "o", "", "output file path (default: stdout)")
	fs.StringVar(&flags.Output, "output", "", "output file path (default: stdout)")
	fs.BoolVar(&flags.Infer, "infer", false, "infer parameter types from naming conventions")
	fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.SourceMap, "source-map", false, "include line numbers in fix output (IDE-friendly format)")
	fs.BoolVar(&flags.SourceMap, "s", false, "include line numbers in fix output (IDE-friendly format)")

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools fix [flags] <file|url|->\n\n")
		cliutil.Writef(fs.Output(), "Apply automatic fixes to common OpenAPI specification issues.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nSupported Fixes:\n")
		cliutil.Writef(fs.Output(), "  - Missing path parameters: Adds Parameter objects for path template\n")
		cliutil.Writef(fs.Output(), "    variables that are not declared in the operation's parameters list.\n")
		cliutil.Writef(fs.Output(), "    Default type is 'string'. Use --infer for smart type inference.\n")
		cliutil.Writef(fs.Output(), "\nType Inference (--infer):\n")
		cliutil.Writef(fs.Output(), "  - Names ending in 'id', 'Id', 'ID' -> integer\n")
		cliutil.Writef(fs.Output(), "  - Names containing 'uuid', 'guid' -> string with format uuid\n")
		cliutil.Writef(fs.Output(), "  - All other names -> string\n")
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools fix openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix -o fixed.yaml openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix --infer openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  cat openapi.yaml | oastools fix -q - > fixed.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix -s openapi.yaml  # Include line numbers in fixes\n")
		cliutil.Writef(fs.Output(), "\nPipelining:\n")
		cliutil.Writef(fs.Output(), "  oastools fix -q api.yaml | oastools validate -q -\n")
		cliutil.Writef(fs.Output(), "  oastools fix -q --infer api.yaml | oastools convert -q -t 3.1.0 -\n")
		cliutil.Writef(fs.Output(), "\nNotes:\n")
		cliutil.Writef(fs.Output(), "  - Use 'oastools validate' to see validation errors before fixing\n")
		cliutil.Writef(fs.Output(), "  - The fix command always applies all available fixes\n")
		cliutil.Writef(fs.Output(), "  - Output preserves the original format (JSON or YAML)\n")
		cliutil.Writef(fs.Output(), "\nExit Codes:\n")
		cliutil.Writef(fs.Output(), "  0    Fixes applied successfully (or no fixes needed)\n")
		cliutil.Writef(fs.Output(), "  1    Failed to parse or fix the specification\n")
	}

	return fs, flags
}

// HandleFix executes the fix command
func HandleFix(args []string) error {
	fs, flags := SetupFixFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("fix command requires exactly one file path, URL, or '-' for stdin")
	}

	specPath := fs.Arg(0)

	// Fix the file, URL, or stdin with timing
	startTime := time.Now()
	var result *fixer.FixResult
	var err error

	if specPath == StdinFilePath {
		// Read from stdin - source map not supported for stdin
		p := parser.New()
		parseResult, parseErr := p.ParseReader(os.Stdin)
		if parseErr != nil {
			return fmt.Errorf("parsing stdin: %w", parseErr)
		}
		f := fixer.New()
		f.InferTypes = flags.Infer
		result, err = f.FixParsed(*parseResult)
		if err != nil {
			return fmt.Errorf("fixing from stdin: %w", err)
		}
	} else {
		// Build fixer options
		fixOpts := []fixer.Option{
			fixer.WithFilePath(specPath),
			fixer.WithInferTypes(flags.Infer),
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
			fixOpts = []fixer.Option{
				fixer.WithParsed(*parseResult),
				fixer.WithInferTypes(flags.Infer),
			}
			if parseResult.SourceMap != nil {
				fixOpts = append(fixOpts, fixer.WithSourceMap(parseResult.SourceMap))
			}
		}

		result, err = fixer.FixWithOptions(fixOpts...)
		if err != nil {
			return fmt.Errorf("fixing file: %w", err)
		}
	}
	totalTime := time.Since(startTime)

	// Print diagnostic messages (to stderr to keep stdout clean for pipelining)
	if !flags.Quiet {
		cliutil.Writef(os.Stderr, "OpenAPI Specification Fixer\n")
		cliutil.Writef(os.Stderr, "===========================\n\n")
		cliutil.Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		if specPath == StdinFilePath {
			cliutil.Writef(os.Stderr, "Specification: <stdin>\n")
		} else {
			cliutil.Writef(os.Stderr, "Specification: %s\n", specPath)
		}
		cliutil.Writef(os.Stderr, "OAS Version: %s\n", result.SourceVersion)
		cliutil.Writef(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		cliutil.Writef(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		cliutil.Writef(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		cliutil.Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		// Print fixes applied
		if result.HasFixes() {
			cliutil.Writef(os.Stderr, "Fixes Applied (%d):\n", result.FixCount)
			for _, fix := range result.Fixes {
				if flags.SourceMap && fix.HasLocation() {
					// IDE-friendly format: file:line:column: path: description
					cliutil.Writef(os.Stderr, "  - %s: [%s] %s: %s\n", fix.Location(), fix.Type, fix.Path, fix.Description)
				} else {
					cliutil.Writef(os.Stderr, "  - [%s] %s: %s\n", fix.Type, fix.Path, fix.Description)
				}
			}
			cliutil.Writef(os.Stderr, "\n")
		}

		// Print summary
		if result.HasFixes() {
			cliutil.Writef(os.Stderr, "✓ Applied %d fix(es)\n", result.FixCount)
		} else {
			cliutil.Writef(os.Stderr, "✓ No fixes needed - specification is already valid\n")
		}
	}

	// Write output
	data, err := MarshalDocument(result.Document, result.SourceFormat)
	if err != nil {
		return fmt.Errorf("marshaling fixed document: %w", err)
	}

	if flags.Output != "" {
		if err := os.WriteFile(flags.Output, data, 0600); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		if !flags.Quiet {
			cliutil.Writef(os.Stderr, "\nOutput written to: %s\n", flags.Output)
		}
	} else {
		// Write to stdout
		if _, err = os.Stdout.Write(data); err != nil {
			return fmt.Errorf("writing fixed document to stdout: %w", err)
		}
	}

	return nil
}
