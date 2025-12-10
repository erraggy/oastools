package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/differ"
	"github.com/erraggy/oastools/internal/cliutil"
	"github.com/erraggy/oastools/parser"
)

// DiffFlags contains flags for the diff command
type DiffFlags struct {
	Breaking bool
	NoInfo   bool
	Format   string
}

// SetupDiffFlags creates and configures a FlagSet for the diff command.
// Returns the FlagSet and a DiffFlags struct with bound flag variables.
func SetupDiffFlags() (*flag.FlagSet, *DiffFlags) {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	flags := &DiffFlags{}

	fs.BoolVar(&flags.Breaking, "breaking", false, "enable breaking change detection and categorization")
	fs.BoolVar(&flags.NoInfo, "no-info", false, "exclude informational changes from output")
	fs.StringVar(&flags.Format, "format", FormatText, "output format: text, json, or yaml")

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools diff [flags] <source> <target>\n\n")
		cliutil.Writef(fs.Output(), "Compare two OpenAPI specification files or URLs and report differences.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nOutput Formats:\n")
		cliutil.Writef(fs.Output(), "  text (default)  Human-readable text output\n")
		cliutil.Writef(fs.Output(), "  json            JSON format for programmatic processing\n")
		cliutil.Writef(fs.Output(), "  yaml            YAML format for programmatic processing\n")
		cliutil.Writef(fs.Output(), "\nModes:\n")
		cliutil.Writef(fs.Output(), "  Default (Simple):\n")
		cliutil.Writef(fs.Output(), "    Reports all semantic differences between specifications without\n")
		cliutil.Writef(fs.Output(), "    categorizing them by severity or breaking change impact.\n\n")
		cliutil.Writef(fs.Output(), "  --breaking (Breaking Change Detection):\n")
		cliutil.Writef(fs.Output(), "    Categorizes changes by severity and identifies breaking API changes:\n")
		cliutil.Writef(fs.Output(), "    - Critical: Removed endpoints or operations\n")
		cliutil.Writef(fs.Output(), "    - Error:    Removed required parameters, incompatible type changes\n")
		cliutil.Writef(fs.Output(), "    - Warning:  Deprecated operations, added required fields\n")
		cliutil.Writef(fs.Output(), "    - Info:     Additions, relaxed constraints, documentation updates\n")
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools diff api-v1.yaml api-v2.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools diff --breaking api-v1.yaml api-v2.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools diff --breaking --no-info old.yaml new.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools diff --format json --breaking api-v1.yaml api-v2.yaml | jq '.HasBreakingChanges'\n")
		cliutil.Writef(fs.Output(), "  oastools diff https://example.com/api/v1.yaml https://example.com/api/v2.yaml\n")
		cliutil.Writef(fs.Output(), "\nExit Status:\n")
		cliutil.Writef(fs.Output(), "  0    No differences found (or no breaking changes in --breaking mode)\n")
		cliutil.Writef(fs.Output(), "  1    Differences found (or breaking changes found in --breaking mode)\n")
		cliutil.Writef(fs.Output(), "\nNotes:\n")
		cliutil.Writef(fs.Output(), "  - Both specifications must be valid OpenAPI documents\n")
		cliutil.Writef(fs.Output(), "  - Cross-version comparison (2.0 vs 3.x) is supported with limitations\n")
		cliutil.Writef(fs.Output(), "  - Breaking change detection helps identify backward compatibility issues\n")
	}

	return fs, flags
}

// HandleDiff executes the diff command
func HandleDiff(args []string) error {
	fs, flags := SetupDiffFlags()

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
	if err := ValidateOutputFormat(flags.Format); err != nil {
		return err
	}

	// Create differ with options
	d := differ.New()
	if flags.Breaking {
		d.Mode = differ.ModeBreaking
	} else {
		d.Mode = differ.ModeSimple
	}
	d.IncludeInfo = !flags.NoInfo

	// Diff the files with timing
	startTime := time.Now()
	result, err := d.Diff(sourcePath, targetPath)
	totalTime := time.Since(startTime)
	if err != nil {
		return fmt.Errorf("comparing specifications: %w", err)
	}

	// Handle structured output formats
	if flags.Format == FormatJSON || flags.Format == FormatYAML {
		if err := OutputStructured(result, flags.Format); err != nil {
			return err
		}

		// Exit with error if breaking changes found (in breaking mode)
		if flags.Breaking && result.HasBreakingChanges {
			os.Exit(1)
		}

		return nil
	}

	// Text format output (original behavior)
	// Print results
	fmt.Printf("OpenAPI Specification Diff\n")
	fmt.Printf("==========================\n\n")
	fmt.Printf("oastools version: %s\n\n", oastools.Version())

	// Determine if we should use single or 2-column layout
	// Use single column if either path is too long to fit comfortably in 2 columns
	// For 80-char terminal: leave room for labels, spacing, and both paths
	const maxPathLengthForTwoColumns = 35 // "Source: " (8 chars) + path should fit in ~40 chars
	useSingleColumn := len(sourcePath) > maxPathLengthForTwoColumns || len(targetPath) > maxPathLengthForTwoColumns

	if useSingleColumn {
		// Single column layout for long paths
		fmt.Printf("Source: %s\n", sourcePath)
		fmt.Printf("  OAS Version: %s\n", result.SourceVersion)
		fmt.Printf("  Source Size: %s\n", parser.FormatBytes(result.SourceSize))
		fmt.Printf("  Paths: %d\n", result.SourceStats.PathCount)
		fmt.Printf("  Operations: %d\n", result.SourceStats.OperationCount)
		fmt.Printf("  Schemas: %d\n\n", result.SourceStats.SchemaCount)

		fmt.Printf("Target: %s\n", targetPath)
		fmt.Printf("  OAS Version: %s\n", result.TargetVersion)
		fmt.Printf("  Target Size: %s\n", parser.FormatBytes(result.TargetSize))
		fmt.Printf("  Paths: %d\n", result.TargetStats.PathCount)
		fmt.Printf("  Operations: %d\n", result.TargetStats.OperationCount)
		fmt.Printf("  Schemas: %d\n", result.TargetStats.SchemaCount)
	} else {
		// 2-column layout for short paths (side-by-side comparison)
		fmt.Printf("%-40s %s\n", "Source: "+sourcePath, "Target: "+targetPath)
		fmt.Printf("%-40s %s\n", "  OAS Version: "+result.SourceVersion, "  OAS Version: "+result.TargetVersion)
		fmt.Printf("%-40s %s\n",
			"  Source Size: "+parser.FormatBytes(result.SourceSize),
			"  Target Size: "+parser.FormatBytes(result.TargetSize))
		fmt.Printf("%-40s %s\n",
			fmt.Sprintf("  Paths: %d", result.SourceStats.PathCount),
			fmt.Sprintf("  Paths: %d", result.TargetStats.PathCount))
		fmt.Printf("%-40s %s\n",
			fmt.Sprintf("  Operations: %d", result.SourceStats.OperationCount),
			fmt.Sprintf("  Operations: %d", result.TargetStats.OperationCount))
		fmt.Printf("%-40s %s\n",
			fmt.Sprintf("  Schemas: %d", result.SourceStats.SchemaCount),
			fmt.Sprintf("  Schemas: %d", result.TargetStats.SchemaCount))
	}
	fmt.Printf("\nTotal Time: %v\n\n", totalTime)

	if len(result.Changes) == 0 {
		fmt.Println("✓ No differences found - specifications are identical")
		return nil
	}

	// Print changes grouped by category if in breaking mode
	if flags.Breaking {
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
