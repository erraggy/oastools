package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/internal/cliutil"
	"github.com/erraggy/oastools/parser"
)

// ConvertFlags contains flags for the convert command
type ConvertFlags struct {
	Target     string
	Output     string
	Strict     bool
	NoWarnings bool
	Quiet      bool
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

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools convert [flags] <file|url|->\n\n")
		cliutil.Writef(fs.Output(), "Convert an OpenAPI specification from one version to another.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nSupported Conversions:\n")
		cliutil.Writef(fs.Output(), "  - OAS 2.0 → OAS 3.x (3.0.0 through 3.2.0)\n")
		cliutil.Writef(fs.Output(), "  - OAS 3.x → OAS 2.0\n")
		cliutil.Writef(fs.Output(), "  - OAS 3.x → OAS 3.y (version updates)\n")
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools convert -t 3.0.3 https://example.com/swagger.yaml -o openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools convert -t 2.0 openapi-v3.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools convert --strict -t 3.1.0 swagger.yaml -o openapi-v3.yaml\n")
		cliutil.Writef(fs.Output(), "  cat swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml\n")
		cliutil.Writef(fs.Output(), "\nPipelining:\n")
		cliutil.Writef(fs.Output(), "  - Use '-' as the file path to read from stdin\n")
		cliutil.Writef(fs.Output(), "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		cliutil.Writef(fs.Output(), "\nNotes:\n")
		cliutil.Writef(fs.Output(), "  - Critical issues indicate features that cannot be converted (data loss)\n")
		cliutil.Writef(fs.Output(), "  - Warnings indicate lossy conversions or best-effort transformations\n")
		cliutil.Writef(fs.Output(), "  - Info messages provide context about conversion choices\n")
		cliutil.Writef(fs.Output(), "  - Always validate converted documents before deployment\n")
		cliutil.Writef(fs.Output(), "\nExit Codes:\n")
		cliutil.Writef(fs.Output(), "  0    Conversion successful\n")
		cliutil.Writef(fs.Output(), "  1    Conversion failed or critical issues found (in --strict mode)\n")
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

	// Create converter with options
	c := converter.New()
	c.StrictMode = flags.Strict
	c.IncludeInfo = !flags.NoWarnings

	// Convert the file, URL, or stdin with timing
	startTime := time.Now()
	var result *converter.ConversionResult
	var err error

	if specPath == StdinFilePath {
		// Read from stdin
		p := parser.New()
		parseResult, err := p.ParseReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("parsing stdin: %w", err)
		}
		result, err = c.ConvertParsed(*parseResult, flags.Target)
		if err != nil {
			return fmt.Errorf("converting from stdin: %w", err)
		}
	} else {
		result, err = c.Convert(specPath, flags.Target)
		if err != nil {
			return fmt.Errorf("converting file: %w", err)
		}
	}
	totalTime := time.Since(startTime)

	// Print results (to stderr in quiet mode)
	if !flags.Quiet {
		cliutil.Writef(os.Stderr, "OpenAPI Specification Converter\n")
		cliutil.Writef(os.Stderr, "===============================\n\n")
		cliutil.Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
		if specPath == StdinFilePath {
			cliutil.Writef(os.Stderr, "Specification: <stdin>\n")
		} else {
			cliutil.Writef(os.Stderr, "Specification: %s\n", specPath)
		}
		cliutil.Writef(os.Stderr, "Source Version: %s\n", result.SourceVersion)
		cliutil.Writef(os.Stderr, "Target Version: %s\n", result.TargetVersion)
		cliutil.Writef(os.Stderr, "Source Size: %s\n", parser.FormatBytes(result.SourceSize))
		cliutil.Writef(os.Stderr, "Paths: %d\n", result.Stats.PathCount)
		cliutil.Writef(os.Stderr, "Operations: %d\n", result.Stats.OperationCount)
		cliutil.Writef(os.Stderr, "Schemas: %d\n", result.Stats.SchemaCount)
		cliutil.Writef(os.Stderr, "Load Time: %v\n", result.LoadTime)
		cliutil.Writef(os.Stderr, "Total Time: %v\n\n", totalTime)

		// Print issues
		if len(result.Issues) > 0 {
			cliutil.Writef(os.Stderr, "Conversion Issues (%d):\n", len(result.Issues))
			for _, issue := range result.Issues {
				cliutil.Writef(os.Stderr, "  %s\n", issue.String())
			}
			cliutil.Writef(os.Stderr, "\n")
		}

		// Print summary
		if result.Success {
			cliutil.Writef(os.Stderr, "✓ Conversion successful")
			if result.InfoCount > 0 || result.WarningCount > 0 {
				cliutil.Writef(os.Stderr, " (%d info, %d warnings)", result.InfoCount, result.WarningCount)
			}
			cliutil.Writef(os.Stderr, "\n")
		} else {
			cliutil.Writef(os.Stderr, "✗ Conversion completed with %d critical issue(s)", result.CriticalCount)
			if result.WarningCount > 0 {
				cliutil.Writef(os.Stderr, ", %d warning(s)", result.WarningCount)
			}
			cliutil.Writef(os.Stderr, "\n")
		}
	}

	// Write output
	data, err := MarshalDocument(result.Document, result.SourceFormat)
	if err != nil {
		return fmt.Errorf("marshaling converted document: %w", err)
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
			return fmt.Errorf("writing converted document to stdout: %w", err)
		}
	}

	// Exit with error if conversion failed
	if !result.Success {
		os.Exit(1)
	}

	return nil
}
