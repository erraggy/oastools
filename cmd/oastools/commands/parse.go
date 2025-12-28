package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/erraggy/oastools/parser"
)

// ParseFlags contains flags for the parse command
type ParseFlags struct {
	ResolveRefs       bool
	ResolveHTTPRefs   bool
	Insecure          bool
	ValidateStructure bool
	Quiet             bool
}

// SetupParseFlags creates and configures a FlagSet for the parse command.
// Returns the FlagSet and a ParseFlags struct with bound flag variables.
func SetupParseFlags() (*flag.FlagSet, *ParseFlags) {
	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	flags := &ParseFlags{}

	fs.BoolVar(&flags.ResolveRefs, "resolve-refs", false, "resolve external $ref references")
	fs.BoolVar(&flags.ResolveHTTPRefs, "resolve-http-refs", false, "resolve HTTP/HTTPS $ref URLs (requires --resolve-refs)")
	fs.BoolVar(&flags.Insecure, "insecure", false, "disable TLS certificate verification for HTTPS refs")
	fs.BoolVar(&flags.ValidateStructure, "validate-structure", false, "validate document structure during parsing")
	fs.BoolVar(&flags.Quiet, "q", false, "quiet mode: only output the document, no diagnostic messages")
	fs.BoolVar(&flags.Quiet, "quiet", false, "quiet mode: only output the document, no diagnostic messages")

	fs.Usage = func() {
		output := fs.Output()
		Writef(output, "Usage: oastools parse [flags] <file|url|->\n\n")
		Writef(output, "Parse and output OpenAPI document structure.\n\n")
		Writef(output, "Flags:\n")
		fs.PrintDefaults()
		Writef(output, "\nExamples:\n")
		Writef(output, "  oastools parse openapi.yaml\n")
		Writef(output, "  oastools parse --resolve-refs openapi.yaml\n")
		Writef(output, "  oastools parse --resolve-refs --resolve-http-refs https://example.com/api/openapi.yaml\n")
		Writef(output, "  oastools parse --resolve-refs --resolve-http-refs --insecure https://internal-server/api.yaml\n")
		Writef(output, "  oastools parse --validate-structure https://example.com/api/openapi.yaml\n")
		Writef(output, "  cat openapi.yaml | oastools parse -q -\n")
		Writef(output, "\nHTTP Reference Resolution:\n")
		Writef(output, "  --resolve-http-refs enables fetching and resolving $refs that point to HTTP/HTTPS URLs.\n")
		Writef(output, "  This is disabled by default for security (SSRF protection).\n")
		Writef(output, "  Use --insecure to skip TLS certificate verification for self-signed certs.\n")
		Writef(output, "\nPipelining:\n")
		Writef(output, "  - Use '-' as the file path to read from stdin\n")
		Writef(output, "  - Use --quiet/-q to suppress diagnostic output for pipelining\n")
		Writef(output, "\nExit Codes:\n")
		Writef(output, "  0    Parsing successful\n")
		Writef(output, "  1    Parsing failed or validation errors found (with --validate-structure)\n")
	}

	return fs, flags
}

// HandleParse executes the parse command
func HandleParse(args []string) error {
	fs, flags := SetupParseFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("parse command requires exactly one file path, URL, or '-' for stdin")
	}

	specPath := fs.Arg(0)

	// Create parser with options
	p := parser.New()
	p.ResolveRefs = flags.ResolveRefs
	p.ResolveHTTPRefs = flags.ResolveHTTPRefs
	p.InsecureSkipVerify = flags.Insecure
	p.ValidateStructure = flags.ValidateStructure

	// Parse the file, URL, or stdin
	var result *parser.ParseResult
	var err error

	if specPath == StdinFilePath {
		result, err = p.ParseReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("parsing stdin: %w", err)
		}
	} else {
		result, err = p.Parse(specPath)
		if err != nil {
			return fmt.Errorf("parsing file: %w", err)
		}
	}

	// Always print errors to stderr, even in quiet mode (critical for debugging)
	if len(result.Errors) > 0 {
		Writef(os.Stderr, "Validation Errors:\n")
		for _, err := range result.Errors {
			Writef(os.Stderr, "  - %s\n", err)
		}
		Writef(os.Stderr, "\n")
		os.Exit(1)
	}

	// Print results (always to stderr to keep stdout clean for JSON output)
	if !flags.Quiet {
		Writef(os.Stderr, "OpenAPI Specification Parser\n")
		Writef(os.Stderr, "============================\n\n")
		OutputSpecHeader(specPath, result.Version)
		OutputSpecStats(result.SourceSize, result.Stats, result.LoadTime)
		Writef(os.Stderr, "\n")

		// Print warnings
		if len(result.Warnings) > 0 {
			Writef(os.Stderr, "Warnings:\n")
			for _, warning := range result.Warnings {
				Writef(os.Stderr, "  - %s\n", warning)
			}
			Writef(os.Stderr, "\n")
		}

		// Print document info
		if result.Document != nil {
			switch doc := result.Document.(type) {
			case *parser.OAS2Document:
				Writef(os.Stderr, "Document Type: OpenAPI 2.0 (Swagger)\n")
				if doc.Info != nil {
					Writef(os.Stderr, "Title: %s\n", doc.Info.Title)
					Writef(os.Stderr, "Description: %s\n", doc.Info.Description)
					Writef(os.Stderr, "Version: %s\n", doc.Info.Version)
				}
				Writef(os.Stderr, "Paths: %d\n", len(doc.Paths))

			case *parser.OAS3Document:
				Writef(os.Stderr, "Document Type: OpenAPI 3.x\n")
				if doc.Info != nil {
					Writef(os.Stderr, "Title: %s\n", doc.Info.Title)
					if doc.Info.Summary != "" {
						Writef(os.Stderr, "Summary: %s\n", doc.Info.Summary)
					}
					Writef(os.Stderr, "Description: %s\n", doc.Info.Description)
					Writef(os.Stderr, "Version: %s\n", doc.Info.Version)
				}
				Writef(os.Stderr, "Servers: %d\n", len(doc.Servers))
				Writef(os.Stderr, "Paths: %d\n", len(doc.Paths))
				if len(doc.Webhooks) > 0 {
					Writef(os.Stderr, "Webhooks: %d\n", len(doc.Webhooks))
				}
			}
		}

		Writef(os.Stderr, "\n")
		Writef(os.Stderr, "Raw Data (JSON):\n")
	}
	jsonData, err := json.MarshalIndent(result.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}
	fmt.Println(string(jsonData))

	if !flags.Quiet {
		Writef(os.Stderr, "\nParsing completed successfully!\n")
	}
	return nil
}
