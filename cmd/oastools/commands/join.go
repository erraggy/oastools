package commands

import (
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/internal/cliutil"
	"github.com/erraggy/oastools/joiner"
)

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
	// Advanced collision strategies
	RenameTemplate  string
	EquivalenceMode string
	CollisionReport bool
	SemanticDedup   bool
	// Namespace prefix configuration
	NamespacePrefix namespacePrefixFlag
	AlwaysPrefix    bool
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

	// Advanced collision strategies
	fs.StringVar(&flags.RenameTemplate, "rename-template", "{{.Name}}_{{.Source}}", "template for renamed schema names")
	fs.StringVar(&flags.EquivalenceMode, "equivalence-mode", "none", "schema comparison mode for deduplication (none, shallow, deep)")
	fs.BoolVar(&flags.CollisionReport, "collision-report", false, "generate detailed collision analysis report")
	fs.BoolVar(&flags.SemanticDedup, "semantic-dedup", false, "enable semantic deduplication to consolidate identical schemas")

	// Namespace prefix configuration
	fs.Var(flags.NamespacePrefix, "namespace-prefix", "namespace prefix for source file (format: source=prefix, can be repeated)")
	fs.BoolVar(&flags.AlwaysPrefix, "always-prefix", false, "apply namespace prefix to all schemas, not just on collision")

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools join [flags] <file1> <file2> [file3...]\n\n")
		cliutil.Writef(fs.Output(), "Join multiple OpenAPI specification files into a single document.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nCollision Strategies:\n")
		cliutil.Writef(fs.Output(), "  accept-left      Keep the first value when collisions occur\n")
		cliutil.Writef(fs.Output(), "  accept-right     Keep the last value when collisions occur (overwrite)\n")
		cliutil.Writef(fs.Output(), "  rename-left      Rename left schema, keep right under original name\n")
		cliutil.Writef(fs.Output(), "  rename-right     Rename right schema, keep left under original name\n")
		cliutil.Writef(fs.Output(), "  deduplicate      Merge structurally identical schemas (requires equivalence-mode)\n")
		cliutil.Writef(fs.Output(), "  fail             Fail with an error on any collision\n")
		cliutil.Writef(fs.Output(), "  fail-on-paths    Fail only on path collisions, allow schema collisions\n")
		cliutil.Writef(fs.Output(), "\nNamespace Prefixes:\n")
		cliutil.Writef(fs.Output(), "  Use --namespace-prefix to add source-based prefixes to schema names.\n")
		cliutil.Writef(fs.Output(), "  Format: source=prefix (can be specified multiple times)\n")
		cliutil.Writef(fs.Output(), "  By default, prefix is only applied on collision. Use --always-prefix to\n")
		cliutil.Writef(fs.Output(), "  apply namespace prefixes to all schemas from the configured sources.\n")
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools join -o merged.yaml base.yaml extensions.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools join --path-strategy accept-left -o api.yaml spec1.yaml spec2.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools join --schema-strategy accept-right -o output.yaml api1.yaml api2.yaml api3.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools join --namespace-prefix users.yaml=Users --namespace-prefix billing.yaml=Billing -o merged.yaml users.yaml billing.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools join --namespace-prefix api2.yaml=V2 --always-prefix -o merged.yaml api1.yaml api2.yaml\n")
		cliutil.Writef(fs.Output(), "\nPipelining:\n")
		cliutil.Writef(fs.Output(), "  oastools join -q base.yaml ext.yaml | oastools validate -q -\n")
		cliutil.Writef(fs.Output(), "  oastools join -q spec1.yaml spec2.yaml | oastools convert -q -t 3.1.0 -\n")
		cliutil.Writef(fs.Output(), "\nNotes:\n")
		cliutil.Writef(fs.Output(), "  - All input files must be the same major OAS version (2.0 or 3.x)\n")
		cliutil.Writef(fs.Output(), "  - The output will use the version of the first input file\n")
		cliutil.Writef(fs.Output(), "  - Info section is taken from the first document by default\n")
		cliutil.Writef(fs.Output(), "  - When -o is specified, file is written with restrictive permissions (0600)\n")
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
	j := joiner.New(config)
	result, err := j.Join(filePaths)
	if err != nil {
		return fmt.Errorf("joining specifications: %w", err)
	}
	totalTime := time.Since(startTime)

	// Print diagnostic messages (to stderr to keep stdout clean for pipelining)
	if !flags.Quiet {
		cliutil.Writef(os.Stderr, "OpenAPI Specification Joiner\n")
		cliutil.Writef(os.Stderr, "============================\n\n")
		cliutil.Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		cliutil.Writef(os.Stderr, "Successfully joined %d specification files\n", len(filePaths))
		if flags.Output != "" {
			cliutil.Writef(os.Stderr, "Output: %s\n", flags.Output)
		} else {
			cliutil.Writef(os.Stderr, "Output: <stdout>\n")
		}
		cliutil.Writef(os.Stderr, "OAS Version: %s\n", result.Version)
		cliutil.Writef(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		cliutil.Writef(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		cliutil.Writef(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		cliutil.Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		if result.CollisionCount > 0 {
			cliutil.Writef(os.Stderr, "Collisions resolved: %d\n\n", result.CollisionCount)
		}

		if len(result.Warnings) > 0 {
			cliutil.Writef(os.Stderr, "Warnings (%d):\n", len(result.Warnings))
			for _, warning := range result.Warnings {
				cliutil.Writef(os.Stderr, "  - %s\n", warning)
			}
			cliutil.Writef(os.Stderr, "\n")
		}

		cliutil.Writef(os.Stderr, "✓ Join completed successfully!\n")
	}

	// Write output
	if flags.Output != "" {
		// Write to file
		err = j.WriteResult(result, flags.Output)
		if err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		if !flags.Quiet {
			cliutil.Writef(os.Stderr, "\nOutput written to: %s\n", flags.Output)
		}
	} else {
		// Write to stdout
		data, err := MarshalDocument(result.Document, result.SourceFormat)
		if err != nil {
			return fmt.Errorf("marshaling joined document: %w", err)
		}
		if _, err = os.Stdout.Write(data); err != nil {
			return fmt.Errorf("writing joined document to stdout: %w", err)
		}
	}

	return nil
}
