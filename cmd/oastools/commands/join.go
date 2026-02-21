package commands

import (
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
)

// stringSliceFlag is a custom flag type for collecting multiple string values.
// It allows the flag to be specified multiple times, each adding to the slice.
type stringSliceFlag []string

// String returns the string representation of the flag value.
func (s *stringSliceFlag) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

// Set parses a value and adds it to the slice.
func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// namespacePrefixFlag is a custom flag type for collecting namespace prefix mappings.
// It allows the flag to be specified multiple times, each with "source=prefix" format.
type namespacePrefixFlag map[string]string

// String returns the string representation of the flag value
func (n namespacePrefixFlag) String() string {
	if n == nil {
		return ""
	}
	pairs := make([]string, 0, len(n))
	for k, v := range n {
		pairs = append(pairs, k+"="+v)
	}
	return strings.Join(pairs, ",")
}

// Set parses a "source=prefix" value and adds it to the map
func (n namespacePrefixFlag) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid namespace prefix format: %q (expected source=prefix)", value)
	}
	source := strings.TrimSpace(parts[0])
	prefix := strings.TrimSpace(parts[1])
	if source == "" || prefix == "" {
		return fmt.Errorf("namespace prefix requires non-empty source and prefix: %q", value)
	}
	n[source] = prefix
	return nil
}

// JoinFlags contains flags for the join command
type JoinFlags struct {
	Output            string
	PathStrategy      string
	SchemaStrategy    string
	ComponentStrategy string
	NoMergeArrays     bool
	NoDedupTags       bool
	Quiet             bool
	SourceMap         bool
	// Advanced collision strategies
	RenameTemplate  string
	EquivalenceMode string
	CollisionReport bool
	SemanticDedup   bool
	// Namespace prefix configuration
	NamespacePrefix namespacePrefixFlag
	AlwaysPrefix    bool
	// Operation context configuration
	OperationContext       bool
	PrimaryOperationPolicy string
	// Overlay configuration
	PreOverlays stringSliceFlag
	PostOverlay string
}

// SetupJoinFlags creates and configures a FlagSet for the join command.
// Returns the FlagSet and a JoinFlags struct with bound flag variables.
func SetupJoinFlags() (*flag.FlagSet, *JoinFlags) {
	fs := flag.NewFlagSet("join", flag.ContinueOnError)
	flags := &JoinFlags{
		NamespacePrefix: make(namespacePrefixFlag),
	}

	fs.StringVar(&flags.Output, "o", "", "output file path (default: stdout)")
	fs.StringVar(&flags.Output, "output", "", "output file path (default: stdout)")
	fs.StringVar(&flags.PathStrategy, "path-strategy", "", "collision strategy for paths (accept-left, accept-right, fail, fail-on-paths)")
	fs.StringVar(&flags.SchemaStrategy, "schema-strategy", "", "collision strategy for schemas (accept-left, accept-right, rename-left, rename-right, deduplicate, fail)")
	fs.StringVar(&flags.ComponentStrategy, "component-strategy", "", "collision strategy for other components")
	fs.BoolVar(&flags.NoMergeArrays, "no-merge-arrays", false, "don't merge arrays (servers, security, etc.)")
	fs.BoolVar(&flags.NoDedupTags, "no-dedup-tags", false, "don't deduplicate tags by name")
	fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: suppress diagnostic messages (for pipelining)")
	fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: suppress diagnostic messages (for pipelining)")
	fs.BoolVar(&flags.SourceMap, "source-map", false, "include line numbers in collision warnings (IDE-friendly format)")
	fs.BoolVar(&flags.SourceMap, "s", false, "include line numbers in collision warnings (IDE-friendly format)")

	// Advanced collision strategies
	fs.StringVar(&flags.RenameTemplate, "rename-template", "{{.Name}}_{{.Source}}", "template for renamed schema names")
	fs.StringVar(&flags.EquivalenceMode, "equivalence-mode", "none", "schema comparison mode for deduplication (none, shallow, deep)")
	fs.BoolVar(&flags.CollisionReport, "collision-report", false, "generate detailed collision analysis report")
	fs.BoolVar(&flags.SemanticDedup, "semantic-dedup", false, "enable semantic deduplication to consolidate identical schemas")

	// Namespace prefix configuration
	fs.Var(flags.NamespacePrefix, "namespace-prefix", "namespace prefix for source file (format: source=prefix, can be repeated)")
	fs.BoolVar(&flags.AlwaysPrefix, "always-prefix", false, "apply namespace prefix to all schemas, not just on collision")

	// Operation context configuration
	fs.BoolVar(&flags.OperationContext, "operation-context", false,
		"enable operation-aware schema renaming (adds Path, Method, OperationID, Tags to templates)")
	fs.StringVar(&flags.PrimaryOperationPolicy, "primary-operation-policy", "",
		"policy for selecting primary operation context: first (default), most-specific, alphabetical")

	// Overlay configuration
	fs.Var(&flags.PreOverlays, "pre-overlay",
		"overlay file to apply before joining (can be repeated)")
	fs.StringVar(&flags.PostOverlay, "post-overlay", "",
		"overlay file to apply to the merged result")

	fs.Usage = func() {
		Writef(fs.Output(), "Usage: oastools join [flags] <file1> <file2> [file3...]\n\n")
		Writef(fs.Output(), "Join multiple OpenAPI specification files into a single document.\n\n")
		Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		Writef(fs.Output(), "\nCollision Strategies:\n")
		Writef(fs.Output(), "  accept-left      Keep the first value when collisions occur\n")
		Writef(fs.Output(), "  accept-right     Keep the last value when collisions occur (overwrite)\n")
		Writef(fs.Output(), "  rename-left      Rename left schema, keep right under original name\n")
		Writef(fs.Output(), "  rename-right     Rename right schema, keep left under original name\n")
		Writef(fs.Output(), "  deduplicate      Merge structurally identical schemas (requires equivalence-mode)\n")
		Writef(fs.Output(), "  fail             Fail with an error on any collision\n")
		Writef(fs.Output(), "  fail-on-paths    Fail only on path collisions, allow schema collisions\n")
		Writef(fs.Output(), "\nNamespace Prefixes:\n")
		Writef(fs.Output(), "  Use --namespace-prefix to add source-based prefixes to schema names.\n")
		Writef(fs.Output(), "  Format: source=prefix (can be specified multiple times)\n")
		Writef(fs.Output(), "  By default, prefix is only applied on collision. Use --always-prefix to\n")
		Writef(fs.Output(), "  apply namespace prefixes to all schemas from the configured sources.\n")
		Writef(fs.Output(), "\nOperation Context:\n")
		Writef(fs.Output(), "  When --operation-context is enabled, rename templates gain access to:\n")
		Writef(fs.Output(), "  - {{.Path}}, {{.Method}}, {{.OperationID}}, {{.Tags}}\n")
		Writef(fs.Output(), "  - {{.UsageType}}, {{.StatusCode}}, {{.ParamName}}, {{.MediaType}}\n")
		Writef(fs.Output(), "  - Aggregate: {{.AllPaths}}, {{.AllMethods}}, {{.IsShared}}, {{.RefCount}}\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  Note: Only affects RIGHT (incoming) document schemas.\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  Primary operation policies:\n")
		Writef(fs.Output(), "    first         Use the first operation found (default)\n")
		Writef(fs.Output(), "    most-specific Prefer operations with operationId, then tags\n")
		Writef(fs.Output(), "    alphabetical  Sort by path+method, use alphabetically first\n")
		Writef(fs.Output(), "\nTemplate Functions (for use with --rename-template):\n")
		Writef(fs.Output(), "  Path:    pathSegment, pathResource, pathLast, pathClean\n")
		Writef(fs.Output(), "  Case:    pascalCase, camelCase, snakeCase, kebabCase\n")
		Writef(fs.Output(), "  Tags:    firstTag, joinTags, hasTag\n")
		Writef(fs.Output(), "  Logic:   default, coalesce\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  See joiner package documentation for full details.\n")
		Writef(fs.Output(), "\nOverlays:\n")
		Writef(fs.Output(), "  Use --pre-overlay to apply overlays to input specs before joining.\n")
		Writef(fs.Output(), "  Use --post-overlay to apply an overlay to the merged result.\n")
		Writef(fs.Output(), "  Pre-overlays can be specified multiple times.\n")
		Writef(fs.Output(), "\nExamples:\n")
		Writef(fs.Output(), "  # Basic joining\n")
		Writef(fs.Output(), "  oastools join -o merged.yaml base.yaml extensions.yaml\n")
		Writef(fs.Output(), "  oastools join --path-strategy accept-left -o api.yaml spec1.yaml spec2.yaml\n")
		Writef(fs.Output(), "  oastools join --schema-strategy accept-right -o output.yaml api1.yaml api2.yaml\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  # Namespace prefixes\n")
		Writef(fs.Output(), "  oastools join --namespace-prefix users.yaml=Users \\\n")
		Writef(fs.Output(), "    --namespace-prefix billing.yaml=Billing -o merged.yaml users.yaml billing.yaml\n")
		Writef(fs.Output(), "  oastools join --namespace-prefix api2.yaml=V2 --always-prefix \\\n")
		Writef(fs.Output(), "    -o merged.yaml api1.yaml api2.yaml\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  # Operation-aware renaming with OperationID\n")
		Writef(fs.Output(), "  oastools join --schema-strategy rename-right --operation-context \\\n")
		Writef(fs.Output(), "    --rename-template \"{{.OperationID | pascalCase}}{{.Name}}\" api1.yaml api2.yaml\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  # Path-based renaming\n")
		Writef(fs.Output(), "  oastools join --schema-strategy rename-right --operation-context \\\n")
		Writef(fs.Output(), "    --rename-template \"{{pathResource .Path | pascalCase}}{{.Name}}\" api1.yaml api2.yaml\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  # With overlays for pre/post processing\n")
		Writef(fs.Output(), "  oastools join --pre-overlay normalize.yaml --post-overlay enhance.yaml \\\n")
		Writef(fs.Output(), "    -o merged.yaml api1.yaml api2.yaml\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  # Multiple pre-overlays\n")
		Writef(fs.Output(), "  oastools join --pre-overlay step1.yaml --pre-overlay step2.yaml \\\n")
		Writef(fs.Output(), "    --post-overlay final.yaml -o merged.yaml api1.yaml api2.yaml\n")
		Writef(fs.Output(), "\n")
		Writef(fs.Output(), "  # Source mapping for IDE-friendly warnings\n")
		Writef(fs.Output(), "  oastools join -s -o merged.yaml api1.yaml api2.yaml\n")
		Writef(fs.Output(), "\nPipelining:\n")
		Writef(fs.Output(), "  oastools join -q base.yaml ext.yaml | oastools validate -q -\n")
		Writef(fs.Output(), "  oastools join -q spec1.yaml spec2.yaml | oastools convert -q -t 3.1.0 -\n")
		Writef(fs.Output(), "\nNotes:\n")
		Writef(fs.Output(), "  - All input files must be the same major OAS version (2.0 or 3.x)\n")
		Writef(fs.Output(), "  - The output will use the version of the first input file\n")
		Writef(fs.Output(), "  - Info section is taken from the first document by default\n")
		Writef(fs.Output(), "  - When -o is specified, file is written with restrictive permissions (0600)\n")
	}

	return fs, flags
}

// HandleJoin executes the join command
func HandleJoin(args []string) error {
	fs, flags := SetupJoinFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("join command requires at least 2 input files")
	}

	filePaths := fs.Args()

	// Build configuration
	config := joiner.DefaultConfig()
	config.MergeArrays = !flags.NoMergeArrays
	config.DeduplicateTags = !flags.NoDedupTags

	// Apply advanced collision strategy settings
	config.RenameTemplate = flags.RenameTemplate
	config.EquivalenceMode = flags.EquivalenceMode
	config.CollisionReport = flags.CollisionReport
	config.SemanticDeduplication = flags.SemanticDedup

	// Apply namespace prefix configuration
	if len(flags.NamespacePrefix) > 0 {
		config.NamespacePrefix = make(map[string]string)
		maps.Copy(config.NamespacePrefix, flags.NamespacePrefix)
	}
	config.AlwaysApplyPrefix = flags.AlwaysPrefix

	// Validate and parse strategy flags
	if err := ValidateCollisionStrategy("path-strategy", flags.PathStrategy); err != nil {
		return err
	}
	if err := ValidateCollisionStrategy("schema-strategy", flags.SchemaStrategy); err != nil {
		return err
	}
	if err := ValidateCollisionStrategy("component-strategy", flags.ComponentStrategy); err != nil {
		return err
	}
	if err := ValidateEquivalenceMode(flags.EquivalenceMode); err != nil {
		return err
	}
	if err := ValidatePrimaryOperationPolicy(flags.PrimaryOperationPolicy); err != nil {
		return err
	}

	// Apply validated strategies to config
	if flags.PathStrategy != "" {
		config.PathStrategy = joiner.CollisionStrategy(flags.PathStrategy)
	}
	if flags.SchemaStrategy != "" {
		config.SchemaStrategy = joiner.CollisionStrategy(flags.SchemaStrategy)
	}
	if flags.ComponentStrategy != "" {
		config.ComponentStrategy = joiner.CollisionStrategy(flags.ComponentStrategy)
	}

	// Validate output path before joining (only when writing to file)
	if flags.Output != "" {
		if err := ValidateOutputPath(flags.Output, filePaths); err != nil {
			return err
		}
	}

	// Create joiner and execute with timing
	startTime := time.Now()

	// Build joiner options - always use JoinWithOptions for consistency
	joinOpts := []joiner.Option{
		joiner.WithFilePaths(filePaths...),
		joiner.WithConfig(config),
	}

	// Add source map support if requested
	if flags.SourceMap {
		parsedDocs := make([]parser.ParseResult, 0, len(filePaths))
		sourceMaps := make(map[string]*parser.SourceMap)
		for _, path := range filePaths {
			parseResult, parseErr := parser.ParseWithOptions(
				parser.WithFilePath(path),
				parser.WithSourceMap(true),
			)
			if parseErr != nil {
				return fmt.Errorf("parsing %s: %w", path, parseErr)
			}
			parsedDocs = append(parsedDocs, *parseResult)
			if parseResult.SourceMap != nil {
				sourceMaps[path] = parseResult.SourceMap
			}
		}
		// Replace WithFiles with WithParsed when using source maps
		joinOpts = []joiner.Option{
			joiner.WithParsed(parsedDocs...),
			joiner.WithConfig(config),
			joiner.WithSourceMaps(sourceMaps),
		}
	}

	// Add operation context options
	if flags.OperationContext {
		joinOpts = append(joinOpts, joiner.WithOperationContext(true))
	}
	if flags.PrimaryOperationPolicy != "" {
		joinOpts = append(joinOpts, joiner.WithPrimaryOperationPolicy(
			MapPrimaryOperationPolicy(flags.PrimaryOperationPolicy),
		))
	}

	// Add overlay options
	for _, preOverlay := range flags.PreOverlays {
		joinOpts = append(joinOpts, joiner.WithPreJoinOverlayFile(preOverlay))
	}
	if flags.PostOverlay != "" {
		joinOpts = append(joinOpts, joiner.WithPostJoinOverlayFile(flags.PostOverlay))
	}

	result, err := joiner.JoinWithOptions(joinOpts...)
	if err != nil {
		return fmt.Errorf("joining specifications: %w", err)
	}
	totalTime := time.Since(startTime)

	// Print diagnostic messages (to stderr to keep stdout clean for pipelining)
	if !flags.Quiet {
		Writef(os.Stderr, "OpenAPI Specification Joiner\n")
		Writef(os.Stderr, "============================\n\n")
		Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		Writef(os.Stderr, "Successfully joined %d specification files\n", len(filePaths))
		if flags.Output != "" {
			Writef(os.Stderr, "Output: %s\n", flags.Output)
		} else {
			Writef(os.Stderr, "Output: <stdout>\n")
		}
		Writef(os.Stderr, "OAS Version: %s\n", result.Version)
		Writef(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		Writef(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		Writef(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		if result.CollisionCount > 0 {
			Writef(os.Stderr, "Collisions resolved: %d\n\n", result.CollisionCount)
		}

		if len(result.Warnings) > 0 {
			Writef(os.Stderr, "Warnings (%d):\n", len(result.Warnings))
			for _, warning := range result.Warnings {
				Writef(os.Stderr, "  - %s\n", warning)
			}
			Writef(os.Stderr, "\n")
		}

		Writef(os.Stderr, "✓ Join completed successfully!\n")
	}

	// Write output
	if flags.Output != "" {
		// Write to file with restrictive permissions (matching joiner.WriteResult behavior)
		data, dataErr := MarshalDocument(result.Document, result.SourceFormat)
		if dataErr != nil {
			return fmt.Errorf("marshaling joined document: %w", dataErr)
		}
		cleanedOutput := filepath.Clean(flags.Output)
		// Reject symlinks to prevent symlink attacks
		if symlinkErr := RejectSymlinkOutput(cleanedOutput); symlinkErr != nil {
			return symlinkErr
		}
		if writeErr := os.WriteFile(cleanedOutput, data, 0600); writeErr != nil { //nolint:gosec // G703 - output path is user-provided CLI flag
			return fmt.Errorf("writing output file: %w", writeErr)
		}
		// Ensure correct permissions even if file pre-existed with different permissions
		if chmodErr := os.Chmod(cleanedOutput, 0600); chmodErr != nil {
			return fmt.Errorf("setting output file permissions: %w", chmodErr)
		}
		if !flags.Quiet {
			Writef(os.Stderr, "\nOutput written to: %s\n", cleanedOutput)
		}
	} else {
		// Write to stdout
		data, dataErr := MarshalDocument(result.Document, result.SourceFormat)
		if dataErr != nil {
			return fmt.Errorf("marshaling joined document: %w", dataErr)
		}
		if _, writeErr := os.Stdout.Write(data); writeErr != nil {
			return fmt.Errorf("writing joined document to stdout: %w", writeErr)
		}
	}

	return nil
}
