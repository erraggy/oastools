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

	// Schema name fixing flags
	FixSchemaNames        bool
	GenericNaming         string
	GenericSeparator      string
	GenericParamSeparator string
	PreserveCasing        bool

	// Pruning flags
	PruneSchemas bool
	PrunePaths   bool
	PruneAll     bool

	// Dry run flag
	DryRun bool
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

	// Schema name fixing flags
	fs.BoolVar(&flags.FixSchemaNames, "fix-schema-names", false, "fix invalid schema names (brackets, special characters)")
	fs.StringVar(&flags.GenericNaming, "generic-naming", "underscore", "strategy for renaming generic types: underscore, of, for, flat, dot")
	fs.StringVar(&flags.GenericSeparator, "generic-separator", "_", "separator for underscore strategy")
	fs.StringVar(&flags.GenericParamSeparator, "generic-param-separator", "_", "separator between multiple type parameters")
	fs.BoolVar(&flags.PreserveCasing, "preserve-casing", false, "preserve original casing of type parameters")

	// Pruning flags
	fs.BoolVar(&flags.PruneSchemas, "prune-schemas", false, "remove unreferenced schema definitions")
	fs.BoolVar(&flags.PrunePaths, "prune-paths", false, "remove paths with no operations")
	fs.BoolVar(&flags.PruneAll, "prune-all", false, "apply all pruning fixes (schemas, paths)")
	fs.BoolVar(&flags.PruneAll, "prune", false, "apply all pruning fixes (alias for --prune-all)")

	// Dry run flag
	fs.BoolVar(&flags.DryRun, "dry-run", false, "preview changes without modifying the document")

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools fix [flags] <file|url|->\n\n")
		cliutil.Writef(fs.Output(), "Apply automatic fixes to common OpenAPI specification issues.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nSupported Fixes:\n")
		cliutil.Writef(fs.Output(), "  - Missing path parameters: Adds Parameter objects for path template\n")
		cliutil.Writef(fs.Output(), "    variables that are not declared in the operation's parameters list.\n")
		cliutil.Writef(fs.Output(), "    Default type is 'string'. Use --infer for smart type inference.\n")
		cliutil.Writef(fs.Output(), "  - Invalid schema names (--fix-schema-names): Renames schemas with\n")
		cliutil.Writef(fs.Output(), "    invalid characters (brackets, etc.) using configurable strategies.\n")
		cliutil.Writef(fs.Output(), "  - Prune schemas (--prune-schemas): Removes unreferenced schemas.\n")
		cliutil.Writef(fs.Output(), "  - Prune paths (--prune-paths): Removes paths with no operations.\n")
		cliutil.Writef(fs.Output(), "\nType Inference (--infer):\n")
		cliutil.Writef(fs.Output(), "  - Names ending in 'id', 'Id', 'ID' -> integer\n")
		cliutil.Writef(fs.Output(), "  - Names containing 'uuid', 'guid' -> string with format uuid\n")
		cliutil.Writef(fs.Output(), "  - All other names -> string\n")
		cliutil.Writef(fs.Output(), "\nGeneric Naming Strategies (--generic-naming):\n")
		cliutil.Writef(fs.Output(), "  underscore: Response[User] → Response_User_\n")
		cliutil.Writef(fs.Output(), "  of:         Response[User] → ResponseOfUser\n")
		cliutil.Writef(fs.Output(), "  for:        Response[User] → ResponseForUser\n")
		cliutil.Writef(fs.Output(), "  flat:       Response[User] → ResponseUser\n")
		cliutil.Writef(fs.Output(), "  dot:        Response[User] → Response.User\n")
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools fix openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix -o fixed.yaml openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix --infer openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix --fix-schema-names --generic-naming of api.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix --prune-all api.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix --dry-run --prune-schemas api.yaml\n")
		cliutil.Writef(fs.Output(), "  cat openapi.yaml | oastools fix -q - > fixed.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools fix -s openapi.yaml  # Include line numbers in fixes\n")
		cliutil.Writef(fs.Output(), "\nPipelining:\n")
		cliutil.Writef(fs.Output(), "  oastools fix -q api.yaml | oastools validate -q -\n")
		cliutil.Writef(fs.Output(), "  oastools fix -q --infer api.yaml | oastools convert -q -t 3.1.0 -\n")
		cliutil.Writef(fs.Output(), "\nNotes:\n")
		cliutil.Writef(fs.Output(), "  - Use 'oastools validate' to see validation errors before fixing\n")
		cliutil.Writef(fs.Output(), "  - Pruning fixes only run when explicitly requested via flags\n")
		cliutil.Writef(fs.Output(), "  - Use --dry-run to preview what would be changed\n")
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

	// Parse generic naming strategy
	strategy, stratErr := fixer.ParseGenericNamingStrategy(flags.GenericNaming)
	if stratErr != nil {
		return fmt.Errorf("invalid generic naming strategy: %w", stratErr)
	}

	// Build generic naming config
	genericConfig := fixer.GenericNamingConfig{
		Strategy:       strategy,
		Separator:      flags.GenericSeparator,
		ParamSeparator: flags.GenericParamSeparator,
		PreserveCasing: flags.PreserveCasing,
	}

	// Build enabled fixes list based on flags
	var enabledFixes []fixer.FixType

	// Always enable missing path parameters (default behavior)
	enabledFixes = append(enabledFixes, fixer.FixTypeMissingPathParameter)

	// Add explicit fixes based on flags
	if flags.FixSchemaNames {
		enabledFixes = append(enabledFixes, fixer.FixTypeRenamedGenericSchema)
	}
	if flags.PruneSchemas || flags.PruneAll {
		enabledFixes = append(enabledFixes, fixer.FixTypePrunedUnusedSchema)
	}
	if flags.PrunePaths || flags.PruneAll {
		enabledFixes = append(enabledFixes, fixer.FixTypePrunedEmptyPath)
	}

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
		f.EnabledFixes = enabledFixes
		f.GenericNamingConfig = genericConfig
		f.DryRun = flags.DryRun
		result, err = f.FixParsed(*parseResult)
		if err != nil {
			return fmt.Errorf("fixing from stdin: %w", err)
		}
	} else {
		// Build fixer options
		fixOpts := []fixer.Option{
			fixer.WithFilePath(specPath),
			fixer.WithInferTypes(flags.Infer),
			fixer.WithEnabledFixes(enabledFixes...),
			fixer.WithGenericNamingConfig(genericConfig),
			fixer.WithDryRun(flags.DryRun),
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
				fixer.WithEnabledFixes(enabledFixes...),
				fixer.WithGenericNamingConfig(genericConfig),
				fixer.WithDryRun(flags.DryRun),
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
			if flags.DryRun {
				cliutil.Writef(os.Stderr, "Fixes That Would Be Applied (%d):\n", result.FixCount)
			} else {
				cliutil.Writef(os.Stderr, "Fixes Applied (%d):\n", result.FixCount)
			}
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
			if flags.DryRun {
				cliutil.Writef(os.Stderr, "⚡ Would apply %d fix(es) (dry-run mode)\n", result.FixCount)
			} else {
				cliutil.Writef(os.Stderr, "✓ Applied %d fix(es)\n", result.FixCount)
			}
		} else {
			cliutil.Writef(os.Stderr, "✓ No fixes needed - specification is already valid\n")
		}
	}

	// In dry-run mode, don't output the document
	if flags.DryRun {
		return nil
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
