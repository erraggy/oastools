package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

// OverlayApplyFlags contains flags for the overlay apply command
type OverlayApplyFlags struct {
	Spec   string
	Output string
	Strict bool
	Quiet  bool
	DryRun bool
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
	fs.BoolVar(&flags.DryRun, "dry-run", false, "preview changes without applying")
	fs.BoolVar(&flags.DryRun, "n", false, "preview changes without applying")
	fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: only output the document, no diagnostic messages")

	fs.Usage = func() {
		Writef(fs.Output(), "Usage: oastools overlay apply [flags] <overlay-file>\n\n")
		Writef(fs.Output(), "Apply an overlay document to an OpenAPI specification.\n\n")
		Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		Writef(fs.Output(), "\nExamples:\n")
		Writef(fs.Output(), "  oastools overlay apply --spec openapi.yaml changes.yaml\n")
		Writef(fs.Output(), "  oastools overlay apply -s openapi.yaml -o production.yaml changes.yaml\n")
		Writef(fs.Output(), "  oastools overlay apply --dry-run -s api.yaml changes.yaml\n")
		Writef(fs.Output(), "  oastools overlay apply --strict -s api.yaml changes.yaml\n")
		Writef(fs.Output(), "  cat openapi.yaml | oastools overlay apply -s - changes.yaml\n")
		Writef(fs.Output(), "\nPipelining:\n")
		Writef(fs.Output(), "  - Use '-' as the spec path to read from stdin\n")
		Writef(fs.Output(), "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		Writef(fs.Output(), "\nNotes:\n")
		Writef(fs.Output(), "  - Actions are applied sequentially in order\n")
		Writef(fs.Output(), "  - Update actions merge content, remove actions delete matched nodes\n")
		Writef(fs.Output(), "  - When both update and remove are specified, remove takes precedence\n")
		Writef(fs.Output(), "  - Use --strict to fail if any target matches nothing\n")
		Writef(fs.Output(), "\nExit Codes:\n")
		Writef(fs.Output(), "  0    Overlay applied successfully\n")
		Writef(fs.Output(), "  1    Overlay application failed\n")
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
		Writef(fs.Output(), "Usage: oastools overlay validate [flags] <overlay-file>\n\n")
		Writef(fs.Output(), "Validate an OpenAPI overlay document.\n\n")
		Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		Writef(fs.Output(), "\nExamples:\n")
		Writef(fs.Output(), "  oastools overlay validate changes.yaml\n")
		Writef(fs.Output(), "  oastools overlay validate --quiet production-overlay.yaml\n")
		Writef(fs.Output(), "\nValidation Checks:\n")
		Writef(fs.Output(), "  - overlay version is present and supported (1.0.0)\n")
		Writef(fs.Output(), "  - info.title and info.version are present\n")
		Writef(fs.Output(), "  - at least one action is defined\n")
		Writef(fs.Output(), "  - each action has a target with valid JSONPath syntax\n")
		Writef(fs.Output(), "  - each action has update or remove (or both)\n")
		Writef(fs.Output(), "\nExit Codes:\n")
		Writef(fs.Output(), "  0    Overlay is valid\n")
		Writef(fs.Output(), "  1    Overlay has validation errors\n")
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
	Writef(os.Stderr, `Usage: oastools overlay <subcommand> [options]

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

	// Build common options
	startTime := time.Now()
	var opts []overlay.Option
	if flags.Spec == StdinFilePath {
		p := parser.New()
		parseResult, err := p.ParseReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("parsing stdin: %w", err)
		}
		opts = append(opts, overlay.WithSpecParsed(*parseResult))
	} else {
		opts = append(opts, overlay.WithSpecFilePath(flags.Spec))
	}
	opts = append(opts,
		overlay.WithOverlayFilePath(overlayPath),
		overlay.WithStrictTargets(flags.Strict),
	)

	// Dry-run mode: preview changes without applying
	if flags.DryRun {
		return handleOverlayDryRun(opts, flags, overlayPath, startTime)
	}

	// Apply overlay
	result, err := overlay.ApplyWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("applying overlay: %w", err)
	}
	totalTime := time.Since(startTime)

	// Print results to stderr
	if !flags.Quiet {
		Writef(os.Stderr, "OpenAPI Overlay Application\n")
		Writef(os.Stderr, "============================\n\n")
		Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		if flags.Spec == StdinFilePath {
			Writef(os.Stderr, "Specification: <stdin>\n")
		} else {
			Writef(os.Stderr, "Specification: %s\n", flags.Spec)
		}
		Writef(os.Stderr, "Overlay: %s\n", overlayPath)
		Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		Writef(os.Stderr, "Actions applied: %d\n", result.ActionsApplied)
		Writef(os.Stderr, "Actions skipped: %d\n", result.ActionsSkipped)

		// Print warnings
		if len(result.Warnings) > 0 {
			Writef(os.Stderr, "\nWarnings:\n")
			for _, warning := range result.Warnings {
				Writef(os.Stderr, "  - %s\n", warning)
			}
		}

		// Print changes
		if len(result.Changes) > 0 {
			Writef(os.Stderr, "\nChanges:\n")
			for _, change := range result.Changes {
				Writef(os.Stderr, "  [%d] %s: %s (%d match(es))\n",
					change.ActionIndex, change.Operation, change.Target, change.MatchCount)
			}
		}

		Writef(os.Stderr, "\n")
		if result.ActionsSkipped == 0 {
			Writef(os.Stderr, "✓ Overlay applied successfully\n")
		} else {
			Writef(os.Stderr, "✓ Overlay applied with %d skipped action(s)\n", result.ActionsSkipped)
		}
	}

	// Write output
	data, err := MarshalDocument(result.Document, result.SourceFormat)
	if err != nil {
		return fmt.Errorf("marshaling result document: %w", err)
	}

	if flags.Output != "" {
		cleanedOutput := filepath.Clean(flags.Output)
		// Reject symlinks to prevent symlink attacks
		if err := RejectSymlinkOutput(cleanedOutput); err != nil {
			return err
		}
		if err := os.WriteFile(cleanedOutput, data, 0600); err != nil { //nolint:gosec // G703 - output path is user-provided CLI flag
			return fmt.Errorf("writing output file: %w", err)
		}
		if !flags.Quiet {
			Writef(os.Stderr, "\nOutput written to: %s\n", cleanedOutput)
		}
	} else {
		// Write to stdout
		if _, err = os.Stdout.Write(data); err != nil {
			return fmt.Errorf("writing result to stdout: %w", err)
		}
	}

	return nil
}

func handleOverlayDryRun(opts []overlay.Option, flags *OverlayApplyFlags, overlayPath string, startTime time.Time) error {
	dryResult, err := overlay.DryRunWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("dry-run overlay: %w", err)
	}
	totalTime := time.Since(startTime)

	if !flags.Quiet {
		Writef(os.Stderr, "OpenAPI Overlay Dry Run\n")
		Writef(os.Stderr, "=======================\n\n")
		Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		if flags.Spec == StdinFilePath {
			Writef(os.Stderr, "Specification: <stdin>\n")
		} else {
			Writef(os.Stderr, "Specification: %s\n", flags.Spec)
		}
		Writef(os.Stderr, "Overlay: %s\n", overlayPath)
		Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		Writef(os.Stderr, "Would apply: %d action(s)\n", dryResult.WouldApply)
		Writef(os.Stderr, "Would skip:  %d action(s)\n", dryResult.WouldSkip)

		if len(dryResult.Changes) > 0 {
			Writef(os.Stderr, "\nProposed Changes:\n")
			for _, change := range dryResult.Changes {
				desc := change.Description
				if desc == "" {
					desc = change.Target
				}
				Writef(os.Stderr, "  [%d] %s: %s (%d match(es))\n",
					change.ActionIndex, change.Operation, desc, change.MatchCount)
				for _, path := range change.MatchedPaths {
					Writef(os.Stderr, "       → %s\n", path)
				}
			}
		}

		if len(dryResult.Warnings) > 0 {
			Writef(os.Stderr, "\nWarnings:\n")
			for _, warning := range dryResult.Warnings {
				Writef(os.Stderr, "  - %s\n", warning)
			}
		}

		Writef(os.Stderr, "\n")
		if dryResult.HasChanges() {
			Writef(os.Stderr, "ℹ️  No changes were made (dry-run mode)\n")
		} else {
			Writef(os.Stderr, "ℹ️  No changes would be made\n")
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
		Writef(os.Stderr, "OpenAPI Overlay Validation\n")
		Writef(os.Stderr, "===========================\n\n")
		Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		Writef(os.Stderr, "Overlay: %s\n", overlayPath)
		Writef(os.Stderr, "Parse Time: %v\n", parseTime)
		Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		Writef(os.Stderr, "Overlay Version: %s\n", o.Version)
		Writef(os.Stderr, "Overlay Title: %s\n", o.Info.Title)
		Writef(os.Stderr, "Actions: %d\n", len(o.Actions))
		if o.Extends != "" {
			Writef(os.Stderr, "Extends: %s\n", o.Extends)
		}
		Writef(os.Stderr, "\n")
	}

	if len(errs) > 0 {
		if !flags.Quiet {
			Writef(os.Stderr, "Validation Errors (%d):\n", len(errs))
			for _, ve := range errs {
				Writef(os.Stderr, "  - %s\n", ve.Message)
			}
			Writef(os.Stderr, "\n✗ Overlay is invalid\n")
		}
		os.Exit(1)
	}

	if !flags.Quiet {
		Writef(os.Stderr, "✓ Overlay is valid\n")
	}

	return nil
}
