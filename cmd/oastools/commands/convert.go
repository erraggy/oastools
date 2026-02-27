package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/internal/fileutil"
	"github.com/erraggy/oastools/parser"
)

// ConvertFlags contains flags for the convert command
type ConvertFlags struct {
	Target     string
	Output     string
	Strict     bool
	NoWarnings bool
	Quiet      bool
	SourceMap  bool
}

// SetupConvertFlags creates and configures a FlagSet for the convert command.
// Returns the FlagSet and a ConvertFlags struct with bound flag variables.
func SetupConvertFlags() (*flag.FlagSet, *ConvertFlags) {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	flags := &ConvertFlags{}

	fs.StringVar(&flags.Target, "t", "", "target OAS version (e.g., \"3.0.3\", \"2.0\", \"3.1.0\") (required)")
	fs.StringVar(&flags.Target, "target", "", "target OAS version (e.g., \"3.0.3\", \"2.0\", \"3.1.0\") (required)")
	fs.StringVar(&flags.Output, "o", "", "output file path (default: stdout)")
	fs.StringVar(&flags.Output, "output", "", "output file path (default: stdout)")
	fs.BoolVar(&flags.Strict, "strict", false, "fail on any conversion issues (even warnings)")
	fs.BoolVar(&flags.NoWarnings, "no-warnings", false, "suppress warning and info messages")
	fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.SourceMap, "source-map", false, "include line numbers in conversion issues (IDE-friendly format)")
	fs.BoolVar(&flags.SourceMap, "s", false, "include line numbers in conversion issues (IDE-friendly format)")

	fs.Usage = func() {
		Writef(fs.Output(), "Usage: oastools convert [flags] <file|url|->\n\n")
		Writef(fs.Output(), "Convert an OpenAPI specification from one version to another.\n\n")
		Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		Writef(fs.Output(), "\nSupported Conversions:\n")
		Writef(fs.Output(), "  - OAS 2.0 → OAS 3.x (3.0.0 through 3.2.0)\n")
		Writef(fs.Output(), "  - OAS 3.x → OAS 2.0\n")
		Writef(fs.Output(), "  - OAS 3.x → OAS 3.y (version updates)\n")
		Writef(fs.Output(), "\nExamples:\n")
		Writef(fs.Output(), "  oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml\n")
		Writef(fs.Output(), "  oastools convert -t 3.0.3 https://example.com/swagger.yaml -o openapi.yaml\n")
		Writef(fs.Output(), "  oastools convert -t 2.0 openapi-v3.yaml\n")
		Writef(fs.Output(), "  oastools convert --strict -t 3.1.0 swagger.yaml -o openapi-v3.yaml\n")
		Writef(fs.Output(), "  cat swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml\n")
		Writef(fs.Output(), "  oastools convert -s -t 3.0.3 swagger.yaml  # Include line numbers in issues\n")
		Writef(fs.Output(), "\nPipelining:\n")
		Writef(fs.Output(), "  - Use '-' as the file path to read from stdin\n")
		Writef(fs.Output(), "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		Writef(fs.Output(), "\nNotes:\n")
		Writef(fs.Output(), "  - Critical issues indicate features that cannot be converted (data loss)\n")
		Writef(fs.Output(), "  - Warnings indicate lossy conversions or best-effort transformations\n")
		Writef(fs.Output(), "  - Info messages provide context about conversion choices\n")
		Writef(fs.Output(), "  - Always validate converted documents before deployment\n")
		Writef(fs.Output(), "\nExit Codes:\n")
		Writef(fs.Output(), "  0    Conversion successful\n")
		Writef(fs.Output(), "  1    Conversion failed or critical issues found (in --strict mode)\n")
	}

	return fs, flags
}

// HandleConvert executes the convert command
func HandleConvert(args []string) error {
	fs, flags := SetupConvertFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("convert command requires exactly one file path, URL, or '-' for stdin")
	}

	specPath := fs.Arg(0)

	if flags.Target == "" {
		fs.Usage()
		return fmt.Errorf("target version is required (use -t or --target)")
	}

	// Convert the file, URL, or stdin with timing
	startTime := time.Now()
	var result *converter.ConversionResult
	var err error

	if specPath == StdinFilePath {
		// Read from stdin - source map not supported for stdin
		p := parser.New()
		parseResult, parseErr := p.ParseReader(os.Stdin)
		if parseErr != nil {
			return fmt.Errorf("parsing stdin: %w", parseErr)
		}
		c := converter.New()
		c.StrictMode = flags.Strict
		c.IncludeInfo = !flags.NoWarnings
		result, err = c.ConvertParsed(*parseResult, flags.Target)
		if err != nil {
			return fmt.Errorf("converting from stdin: %w", err)
		}
	} else {
		// Build converter options
		convertOpts := []converter.Option{
			converter.WithFilePath(specPath),
			converter.WithTargetVersion(flags.Target),
			converter.WithStrictMode(flags.Strict),
			converter.WithIncludeInfo(!flags.NoWarnings),
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
			convertOpts = []converter.Option{
				converter.WithParsed(*parseResult),
				converter.WithTargetVersion(flags.Target),
				converter.WithStrictMode(flags.Strict),
				converter.WithIncludeInfo(!flags.NoWarnings),
			}
			if parseResult.SourceMap != nil {
				convertOpts = append(convertOpts, converter.WithSourceMap(parseResult.SourceMap))
			}
		}

		result, err = converter.ConvertWithOptions(convertOpts...)
		if err != nil {
			return fmt.Errorf("converting file: %w", err)
		}
	}
	totalTime := time.Since(startTime)

	// Print results (to stderr in quiet mode)
	if !flags.Quiet {
		Writef(os.Stderr, "OpenAPI Specification Converter\n")
		Writef(os.Stderr, "===============================\n\n")
		Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		Writef(os.Stderr, "Specification: %s\n", FormatSpecPath(specPath))
		Writef(os.Stderr, "Source Version: %s\n", result.SourceVersion)
		Writef(os.Stderr, "Target Version: %s\n", result.TargetVersion)
		OutputSpecStats(result.SourceSize, result.Stats, result.LoadTime)
		Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		// Print issues
		if len(result.Issues) > 0 {
			Writef(os.Stderr, "Conversion Issues (%d):\n", len(result.Issues))
			for _, issue := range result.Issues {
				if flags.SourceMap && issue.HasLocation() {
					// IDE-friendly format: file:line:column: path: message
					Writef(os.Stderr, "  %s: %s: %s\n", issue.Location(), issue.Path, issue.Message)
				} else {
					Writef(os.Stderr, "  %s\n", issue.String())
				}
			}
			Writef(os.Stderr, "\n")
		}

		// Print summary
		if result.Success {
			Writef(os.Stderr, "✓ Conversion successful")
			if result.InfoCount > 0 || result.WarningCount > 0 {
				Writef(os.Stderr, " (%d info, %d warnings)", result.InfoCount, result.WarningCount)
			}
			Writef(os.Stderr, "\n")
		} else {
			Writef(os.Stderr, "✗ Conversion completed with %d critical issue(s)", result.CriticalCount)
			if result.WarningCount > 0 {
				Writef(os.Stderr, ", %d warning(s)", result.WarningCount)
			}
			Writef(os.Stderr, "\n")
		}
	}

	// Write output
	data, err := MarshalDocument(result.Document, result.SourceFormat)
	if err != nil {
		return fmt.Errorf("marshaling converted document: %w", err)
	}

	if flags.Output != "" {
		cleanedOutput := filepath.Clean(flags.Output)
		// Reject symlinks to prevent symlink attacks
		if err := RejectSymlinkOutput(cleanedOutput); err != nil {
			return err
		}
		if err := os.WriteFile(cleanedOutput, data, fileutil.OwnerReadWrite); err != nil { //nolint:gosec // G703 - output path is user-provided CLI flag
			return fmt.Errorf("writing output file: %w", err)
		}
		if !flags.Quiet {
			Writef(os.Stderr, "\nOutput written to: %s\n", cleanedOutput)
		}
	} else {
		// Write to stdout
		if _, err = os.Stdout.Write(data); err != nil {
			return fmt.Errorf("writing converted document to stdout: %w", err)
		}
	}

	// Exit with error if conversion failed
	if !result.Success {
		os.Exit(1)
	}

	return nil
}
