package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/internal/cliutil"
	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

// OverlayApplyFlags contains flags for the overlay apply command
type OverlayApplyFlags struct {
	Spec   string
	Output string
	Strict bool
	Quiet  bool
}

// OverlayValidateFlags contains flags for the overlay validate command
type OverlayValidateFlags struct {
	Quiet bool
}

// SetupOverlayApplyFlags creates and configures a FlagSet for the overlay apply command.
// Returns the FlagSet and an OverlayApplyFlags struct with bound flag variables.
func SetupOverlayApplyFlags() (*flag.FlagSet, *OverlayApplyFlags) {
	fs := flag.NewFlagSet("overlay apply", flag.ContinueOnError)
	flags := &OverlayApplyFlags{}

	fs.StringVar(&flags.Spec, "s", "", "base OpenAPI specification file or URL (required)")
	fs.StringVar(&flags.Spec, "spec", "", "base OpenAPI specification file or URL (required)")
	fs.StringVar(&flags.Output, "o", "", "output file path (default: stdout)")
	fs.StringVar(&flags.Output, "output", "", "output file path (default: stdout)")
	fs.BoolVar(&flags.Strict, "strict", false, "fail if any action target matches no nodes")
	fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: only output the document, no diagnostic messages")

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools overlay apply [flags] <overlay-file>\n\n")
		cliutil.Writef(fs.Output(), "Apply an overlay document to an OpenAPI specification.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools overlay apply --spec openapi.yaml changes.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools overlay apply -s openapi.yaml -o production.yaml changes.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools overlay apply --strict -s api.yaml changes.yaml\n")
		cliutil.Writef(fs.Output(), "  cat openapi.yaml | oastools overlay apply -s - changes.yaml\n")
		cliutil.Writef(fs.Output(), "\nPipelining:\n")
		cliutil.Writef(fs.Output(), "  - Use '-' as the spec path to read from stdin\n")
		cliutil.Writef(fs.Output(), "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		cliutil.Writef(fs.Output(), "\nNotes:\n")
		cliutil.Writef(fs.Output(), "  - Actions are applied sequentially in order\n")
		cliutil.Writef(fs.Output(), "  - Update actions merge content, remove actions delete matched nodes\n")
		cliutil.Writef(fs.Output(), "  - When both update and remove are specified, remove takes precedence\n")
		cliutil.Writef(fs.Output(), "  - Use --strict to fail if any target matches nothing\n")
		cliutil.Writef(fs.Output(), "\nExit Codes:\n")
		cliutil.Writef(fs.Output(), "  0    Overlay applied successfully\n")
		cliutil.Writef(fs.Output(), "  1    Overlay application failed\n")
	}

	return fs, flags
}

// SetupOverlayValidateFlags creates and configures a FlagSet for the overlay validate command.
// Returns the FlagSet and an OverlayValidateFlags struct with bound flag variables.
func SetupOverlayValidateFlags() (*flag.FlagSet, *OverlayValidateFlags) {
	fs := flag.NewFlagSet("overlay validate", flag.ContinueOnError)
	flags := &OverlayValidateFlags{}

	fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: only output validation result, no diagnostic messages")
	fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: only output validation result, no diagnostic messages")

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools overlay validate [flags] <overlay-file>\n\n")
		cliutil.Writef(fs.Output(), "Validate an OpenAPI overlay document.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools overlay validate changes.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools overlay validate --quiet production-overlay.yaml\n")
		cliutil.Writef(fs.Output(), "\nValidation Checks:\n")
		cliutil.Writef(fs.Output(), "  - overlay version is present and supported (1.0.0)\n")
		cliutil.Writef(fs.Output(), "  - info.title and info.version are present\n")
		cliutil.Writef(fs.Output(), "  - at least one action is defined\n")
		cliutil.Writef(fs.Output(), "  - each action has a target with valid JSONPath syntax\n")
		cliutil.Writef(fs.Output(), "  - each action has update or remove (or both)\n")
		cliutil.Writef(fs.Output(), "\nExit Codes:\n")
		cliutil.Writef(fs.Output(), "  0    Overlay is valid\n")
		cliutil.Writef(fs.Output(), "  1    Overlay has validation errors\n")
	}

	return fs, flags
}

// HandleOverlay executes the overlay command
func HandleOverlay(args []string) error {
	if len(args) < 1 {
		printOverlayUsage()
		return fmt.Errorf("overlay command requires a subcommand: apply, validate")
	}

	switch args[0] {
	case "apply":
		return handleOverlayApply(args[1:])
	case "validate":
		return handleOverlayValidate(args[1:])
	case "-h", "--help", "help":
		printOverlayUsage()
		return nil
	default:
		printOverlayUsage()
		return fmt.Errorf("unknown overlay subcommand: %s", args[0])
	}
}

func printOverlayUsage() {
	cliutil.Writef(os.Stderr, `Usage: oastools overlay <subcommand> [options]

Apply or validate OpenAPI Overlay documents (v1.0.0).

Subcommands:
  apply       Apply an overlay to an OpenAPI specification
  validate    Validate an overlay document

Run 'oastools overlay <subcommand> --help' for more information.
`)
}

func handleOverlayApply(args []string) error {
	fs, flags := SetupOverlayApplyFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Overlay file is a positional argument
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("overlay apply requires exactly one overlay file")
	}
	overlayPath := fs.Arg(0)

	// Spec is required
	if flags.Spec == "" {
		fs.Usage()
		return fmt.Errorf("specification is required (use -s or --spec)")
	}

	// Apply overlay with timing
	startTime := time.Now()
	var result *overlay.ApplyResult
	var err error

	if flags.Spec == StdinFilePath {
		// Read spec from stdin
		p := parser.New()
		parseResult, err := p.ParseReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("parsing stdin: %w", err)
		}
		result, err = overlay.ApplyWithOptions(
			overlay.WithSpecParsed(*parseResult),
			overlay.WithOverlayFilePath(overlayPath),
			overlay.WithStrictTargets(flags.Strict),
		)
		if err != nil {
			return fmt.Errorf("applying overlay: %w", err)
		}
	} else {
		result, err = overlay.ApplyWithOptions(
			overlay.WithSpecFilePath(flags.Spec),
			overlay.WithOverlayFilePath(overlayPath),
			overlay.WithStrictTargets(flags.Strict),
		)
		if err != nil {
			return fmt.Errorf("applying overlay: %w", err)
		}
	}
	totalTime := time.Since(startTime)

	// Print results to stderr
	if !flags.Quiet {
		cliutil.Writef(os.Stderr, "OpenAPI Overlay Application\n")
		cliutil.Writef(os.Stderr, "============================\n\n")
		cliutil.Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		if flags.Spec == StdinFilePath {
			cliutil.Writef(os.Stderr, "Specification: <stdin>\n")
		} else {
			cliutil.Writef(os.Stderr, "Specification: %s\n", flags.Spec)
		}
		cliutil.Writef(os.Stderr, "Overlay: %s\n", overlayPath)
		cliutil.Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		cliutil.Writef(os.Stderr, "Actions applied: %d\n", result.ActionsApplied)
		cliutil.Writef(os.Stderr, "Actions skipped: %d\n", result.ActionsSkipped)

		// Print warnings
		if len(result.Warnings) > 0 {
			cliutil.Writef(os.Stderr, "\nWarnings:\n")
			for _, warning := range result.Warnings {
				cliutil.Writef(os.Stderr, "  - %s\n", warning)
			}
		}

		// Print changes
		if len(result.Changes) > 0 {
			cliutil.Writef(os.Stderr, "\nChanges:\n")
			for _, change := range result.Changes {
				cliutil.Writef(os.Stderr, "  [%d] %s: %s (%d match(es))\n",
					change.ActionIndex, change.Operation, change.Target, change.MatchCount)
			}
		}

		cliutil.Writef(os.Stderr, "\n")
		if result.ActionsSkipped == 0 {
			cliutil.Writef(os.Stderr, "✓ Overlay applied successfully\n")
		} else {
			cliutil.Writef(os.Stderr, "✓ Overlay applied with %d skipped action(s)\n", result.ActionsSkipped)
		}
	}

	// Write output
	data, err := MarshalDocument(result.Document, result.SourceFormat)
	if err != nil {
		return fmt.Errorf("marshaling result document: %w", err)
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
			return fmt.Errorf("writing result to stdout: %w", err)
		}
	}

	return nil
}

func handleOverlayValidate(args []string) error {
	fs, flags := SetupOverlayValidateFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("overlay validate requires exactly one overlay file")
	}
	overlayPath := fs.Arg(0)

	// Parse the overlay
	startTime := time.Now()
	o, err := overlay.ParseOverlayFile(overlayPath)
	if err != nil {
		return fmt.Errorf("parsing overlay: %w", err)
	}
	parseTime := time.Since(startTime)

	// Validate the overlay
	errs := overlay.Validate(o)
	totalTime := time.Since(startTime)

	if !flags.Quiet {
		cliutil.Writef(os.Stderr, "OpenAPI Overlay Validation\n")
		cliutil.Writef(os.Stderr, "===========================\n\n")
		cliutil.Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		cliutil.Writef(os.Stderr, "Overlay: %s\n", overlayPath)
		cliutil.Writef(os.Stderr, "Parse Time: %v\n", parseTime)
		cliutil.Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		cliutil.Writef(os.Stderr, "Overlay Version: %s\n", o.Version)
		cliutil.Writef(os.Stderr, "Overlay Title: %s\n", o.Info.Title)
		cliutil.Writef(os.Stderr, "Actions: %d\n", len(o.Actions))
		if o.Extends != "" {
			cliutil.Writef(os.Stderr, "Extends: %s\n", o.Extends)
		}
		cliutil.Writef(os.Stderr, "\n")
	}

	if len(errs) > 0 {
		if !flags.Quiet {
			cliutil.Writef(os.Stderr, "Validation Errors (%d):\n", len(errs))
			for _, ve := range errs {
				cliutil.Writef(os.Stderr, "  - %s\n", ve.Message)
			}
			cliutil.Writef(os.Stderr, "\n✗ Overlay is invalid\n")
		}
		os.Exit(1)
	}

	if !flags.Quiet {
		cliutil.Writef(os.Stderr, "✓ Overlay is valid\n")
	}

	return nil
}
