package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/fixer"
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

	// Duplicate operationId fixing flags
	FixDuplicateOperationIds bool
	OperationIdTemplate      string
	OperationIdPathSep       string
	OperationIdTagSep        string

	// Stub missing refs flags
	StubMissingRefs  bool
	StubResponseDesc string

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

	// Duplicate operationId fixing flags
	fs.BoolVar(&flags.FixDuplicateOperationIds, "fix-duplicate-operationids", false, "fix duplicate operationId values by renaming")
	fs.StringVar(&flags.OperationIdTemplate, "operationid-template", "{operationId}{n}", "template for renaming duplicate operationIds")
	fs.StringVar(&flags.OperationIdPathSep, "operationid-path-sep", "_", "separator for path segments in operationId template")
	fs.StringVar(&flags.OperationIdTagSep, "operationid-tag-sep", "_", "separator for tags in operationId template")

	// Stub missing refs flags
	fs.BoolVar(&flags.StubMissingRefs, "stub-missing-refs", false, "create stubs for unresolved local $ref pointers")
	fs.StringVar(&flags.StubResponseDesc, "stub-response-desc", "", "description text for stub responses (default: auto-generated message)")

	// Dry run flag
	fs.BoolVar(&flags.DryRun, "dry-run", false, "preview changes without modifying the document")

	fs.Usage = func() {
		Writef(fs.Output(), "Usage: oastools fix [flags] <file|url|->\n\n")
		Writef(fs.Output(), "Apply automatic fixes to common OpenAPI specification issues.\n\n")
		Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		Writef(fs.Output(), "\nSupported Fixes:\n")
		Writef(fs.Output(), "  - Missing path parameters: Adds Parameter objects for path template\n")
		Writef(fs.Output(), "    variables that are not declared in the operation's parameters list.\n")
		Writef(fs.Output(), "    Default type is 'string'. Use --infer for smart type inference.\n")
		Writef(fs.Output(), "  - Invalid schema names (--fix-schema-names): Renames schemas with\n")
		Writef(fs.Output(), "    invalid characters (brackets, etc.) using configurable strategies.\n")
		Writef(fs.Output(), "  - Duplicate operationIds (--fix-duplicate-operationids): Renames\n")
		Writef(fs.Output(), "    duplicate operationId values using configurable templates.\n")
		Writef(fs.Output(), "  - Stub missing refs (--stub-missing-refs): Creates stub definitions\n")
		Writef(fs.Output(), "    for unresolved local $ref pointers. Schemas get empty {} stubs,\n")
		Writef(fs.Output(), "    responses get stubs with configurable descriptions.\n")
		Writef(fs.Output(), "  - Prune schemas (--prune-schemas): Removes unreferenced schemas.\n")
		Writef(fs.Output(), "  - Prune paths (--prune-paths): Removes paths with no operations.\n")
		Writef(fs.Output(), "\nType Inference (--infer):\n")
		Writef(fs.Output(), "  - Names ending in 'id', 'Id', 'ID' -> integer\n")
		Writef(fs.Output(), "  - Names containing 'uuid', 'guid' -> string with format uuid\n")
		Writef(fs.Output(), "  - All other names -> string\n")
		Writef(fs.Output(), "\nGeneric Naming Strategies (--generic-naming):\n")
		Writef(fs.Output(), "  underscore: Response[User] → Response_User_\n")
		Writef(fs.Output(), "  of:         Response[User] → ResponseOfUser\n")
		Writef(fs.Output(), "  for:        Response[User] → ResponseForUser\n")
		Writef(fs.Output(), "  flat:       Response[User] → ResponseUser\n")
		Writef(fs.Output(), "  dot:        Response[User] → Response.User\n")
		Writef(fs.Output(), "\nOperationId Templates (--operationid-template):\n")
		Writef(fs.Output(), "  Placeholders: {operationId}, {method}, {path}, {tag}, {tags}, {n}\n")
		Writef(fs.Output(), "  Modifiers: :pascal, :camel, :snake, :kebab, :upper, :lower\n")
		Writef(fs.Output(), "  {operationId}{n}             First: getUser, duplicates: getUser2, getUser3 (default)\n")
		Writef(fs.Output(), "  {operationId}_{method}       getUser -> getUser, getUser_post\n")
		Writef(fs.Output(), "  {operationId:pascal}_{method:upper} get_user -> GetUser_POST\n")
		Writef(fs.Output(), "  {method}_{path:snake}        /users/{id} -> get_users_id\n")
		Writef(fs.Output(), "\nExamples:\n")
		Writef(fs.Output(), "  oastools fix openapi.yaml\n")
		Writef(fs.Output(), "  oastools fix -o fixed.yaml openapi.yaml\n")
		Writef(fs.Output(), "  oastools fix --infer openapi.yaml\n")
		Writef(fs.Output(), "  oastools fix --fix-schema-names --generic-naming of api.yaml\n")
		Writef(fs.Output(), "  oastools fix --fix-duplicate-operationids api.yaml\n")
		Writef(fs.Output(), "  oastools fix --fix-duplicate-operationids --operationid-template '{operationId:pascal}_{method:upper}' api.yaml\n")
		Writef(fs.Output(), "  oastools fix --stub-missing-refs api.yaml\n")
		Writef(fs.Output(), "  oastools fix --stub-missing-refs --stub-response-desc 'TODO: implement' api.yaml\n")
		Writef(fs.Output(), "  oastools fix --prune-all api.yaml\n")
		Writef(fs.Output(), "  oastools fix --dry-run --prune-schemas api.yaml\n")
		Writef(fs.Output(), "  cat openapi.yaml | oastools fix -q - > fixed.yaml\n")
		Writef(fs.Output(), "  oastools fix -s openapi.yaml  # Include line numbers in fixes\n")
		Writef(fs.Output(), "\nPipelining:\n")
		Writef(fs.Output(), "  oastools fix -q api.yaml | oastools validate -q -\n")
		Writef(fs.Output(), "  oastools fix -q --infer api.yaml | oastools convert -q -t 3.1.0 -\n")
		Writef(fs.Output(), "\nNotes:\n")
		Writef(fs.Output(), "  - Use 'oastools validate' to see validation errors before fixing\n")
		Writef(fs.Output(), "  - Pruning fixes only run when explicitly requested via flags\n")
		Writef(fs.Output(), "  - Use --dry-run to preview what would be changed\n")
		Writef(fs.Output(), "  - Output preserves the original format (JSON or YAML)\n")
		Writef(fs.Output(), "\nExit Codes:\n")
		Writef(fs.Output(), "  0    Fixes applied successfully (or no fixes needed)\n")
		Writef(fs.Output(), "  1    Failed to parse or fix the specification\n")
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

	// Build operationId naming config
	// Note: Template validation happens in WithOperationIdNamingConfig option
	operationIdConfig := fixer.OperationIdNamingConfig{
		Template:      flags.OperationIdTemplate,
		PathSeparator: flags.OperationIdPathSep,
		TagSeparator:  flags.OperationIdTagSep,
	}

	// Build enabled fixes list based on flags
	var enabledFixes []fixer.FixType

	// Always enable missing path parameters (default behavior)
	enabledFixes = append(enabledFixes, fixer.FixTypeMissingPathParameter)

	// Add explicit fixes based on flags
	if flags.FixSchemaNames {
		enabledFixes = append(enabledFixes, fixer.FixTypeRenamedGenericSchema)
	}
	if flags.FixDuplicateOperationIds {
		enabledFixes = append(enabledFixes, fixer.FixTypeDuplicateOperationId)
	}
	if flags.StubMissingRefs {
		enabledFixes = append(enabledFixes, fixer.FixTypeStubMissingRef)
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
		f.OperationIdNamingConfig = operationIdConfig
		f.DryRun = flags.DryRun
		f.MutableInput = true
		if flags.StubResponseDesc != "" {
			f.StubConfig.ResponseDescription = flags.StubResponseDesc
		}
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
			fixer.WithOperationIdNamingConfig(operationIdConfig),
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
				fixer.WithOperationIdNamingConfig(operationIdConfig),
				fixer.WithDryRun(flags.DryRun),
				fixer.WithMutableInput(true),
			}
			if parseResult.SourceMap != nil {
				fixOpts = append(fixOpts, fixer.WithSourceMap(parseResult.SourceMap))
			}
		}

		// Add stub response description if specified (after potential SourceMap rebuild)
		if flags.StubResponseDesc != "" {
			fixOpts = append(fixOpts, fixer.WithStubResponseDescription(flags.StubResponseDesc))
		}

		result, err = fixer.FixWithOptions(fixOpts...)
		if err != nil {
			return fmt.Errorf("fixing file: %w", err)
		}
	}
	totalTime := time.Since(startTime)

	// Print diagnostic messages (to stderr to keep stdout clean for pipelining)
	if !flags.Quiet {
		Writef(os.Stderr, "OpenAPI Specification Fixer\n")
		Writef(os.Stderr, "===========================\n\n")
		Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		if specPath == StdinFilePath {
			Writef(os.Stderr, "Specification: <stdin>\n")
		} else {
			Writef(os.Stderr, "Specification: %s\n", specPath)
		}
		Writef(os.Stderr, "OAS Version: %s\n", result.SourceVersion)
		Writef(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		Writef(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		Writef(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		// Print fixes applied
		if result.HasFixes() {
			if flags.DryRun {
				Writef(os.Stderr, "Fixes That Would Be Applied (%d):\n", result.FixCount)
			} else {
				Writef(os.Stderr, "Fixes Applied (%d):\n", result.FixCount)
			}
			for _, fix := range result.Fixes {
				if flags.SourceMap && fix.HasLocation() {
					// IDE-friendly format: file:line:column: path: description
					Writef(os.Stderr, "  - %s: [%s] %s: %s\n", fix.Location(), fix.Type, fix.Path, fix.Description)
				} else {
					Writef(os.Stderr, "  - [%s] %s: %s\n", fix.Type, fix.Path, fix.Description)
				}
			}
			Writef(os.Stderr, "\n")
		}

		// Print summary
		if result.HasFixes() {
			if flags.DryRun {
				Writef(os.Stderr, "⚡ Would apply %d fix(es) (dry-run mode)\n", result.FixCount)
			} else {
				Writef(os.Stderr, "✓ Applied %d fix(es)\n", result.FixCount)
			}
		} else {
			Writef(os.Stderr, "✓ No fixes needed - specification is already valid\n")
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
			Writef(os.Stderr, "\nOutput written to: %s\n", flags.Output)
		}
	} else {
		// Write to stdout
		if _, err = os.Stdout.Write(data); err != nil {
			return fmt.Errorf("writing fixed document to stdout: %w", err)
		}
	}

	return nil
}
