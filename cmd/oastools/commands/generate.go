package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/generator"
	"github.com/erraggy/oastools/internal/cliutil"
	"github.com/erraggy/oastools/parser"
)

// GenerateFlags contains flags for the generate command
type GenerateFlags struct {
	Output       string
	PackageName  string
	Client       bool
	Server       bool
	Types        bool
	NoPointers   bool
	NoValidation bool
	Strict       bool
	NoWarnings   bool
	SourceMap    bool

	// Security generation options
	NoSecurity      bool
	OAuth2Flows     bool
	CredentialMgmt  bool
	SecurityEnforce bool
	OIDCDiscovery   bool
	NoReadme        bool

	// File splitting options
	MaxLinesPerFile int
	MaxTypesPerFile int
	MaxOpsPerFile   int
	SplitByTag      bool
	NoSplitByTag    bool
	SplitByPath     bool
	NoSplitByPath   bool

	// Server generation options
	ServerRouter     string
	ServerMiddleware bool
	ServerBinder     bool
	ServerResponses  bool
	ServerStubs      bool
	ServerEmbedSpec  bool
	ServerAll        bool
}

// SetupGenerateFlags creates and configures a FlagSet for the generate command.
// Returns the FlagSet and a GenerateFlags struct with bound flag variables.
func SetupGenerateFlags() (*flag.FlagSet, *GenerateFlags) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	flags := &GenerateFlags{}

	fs.StringVar(&flags.Output, "o", "", "output directory for generated files (required)")
	fs.StringVar(&flags.Output, "output", "", "output directory for generated files (required)")
	fs.StringVar(&flags.PackageName, "p", "api", "Go package name for generated code")
	fs.StringVar(&flags.PackageName, "package", "api", "Go package name for generated code")
	fs.BoolVar(&flags.Client, "client", false, "generate HTTP client code")
	fs.BoolVar(&flags.Server, "server", false, "generate server interface code")
	fs.BoolVar(&flags.Types, "types", true, "generate type definitions from schemas")
	fs.BoolVar(&flags.NoPointers, "no-pointers", false, "don't use pointer types for optional fields")
	fs.BoolVar(&flags.NoValidation, "no-validation", false, "don't include validation tags")
	fs.BoolVar(&flags.Strict, "strict", false, "fail on any generation issues (even warnings)")
	fs.BoolVar(&flags.NoWarnings, "no-warnings", false, "suppress warning and info messages")
	fs.BoolVar(&flags.SourceMap, "source-map", false, "include line numbers in generation issues (IDE-friendly format)")
	fs.BoolVar(&flags.SourceMap, "s", false, "include line numbers in generation issues (IDE-friendly format)")

	// Security generation flags
	fs.BoolVar(&flags.NoSecurity, "no-security", false, "don't generate security helper functions")
	fs.BoolVar(&flags.OAuth2Flows, "oauth2-flows", false, "generate OAuth2 token flow helpers")
	fs.BoolVar(&flags.CredentialMgmt, "credential-mgmt", false, "generate credential management interfaces")
	fs.BoolVar(&flags.SecurityEnforce, "security-enforce", false, "generate security enforcement middleware")
	fs.BoolVar(&flags.OIDCDiscovery, "oidc-discovery", false, "generate OpenID Connect discovery client")
	fs.BoolVar(&flags.NoReadme, "no-readme", false, "don't generate README.md file")

	// File splitting flags
	fs.IntVar(&flags.MaxLinesPerFile, "max-lines-per-file", 2000, "maximum lines per generated file (0 = no limit)")
	fs.IntVar(&flags.MaxTypesPerFile, "max-types-per-file", 200, "maximum types per generated file (0 = no limit)")
	fs.IntVar(&flags.MaxOpsPerFile, "max-ops-per-file", 100, "maximum operations per generated file (0 = no limit)")
	fs.BoolVar(&flags.NoSplitByTag, "no-split-by-tag", false, "don't split files by operation tag")
	fs.BoolVar(&flags.NoSplitByPath, "no-split-by-path", false, "don't split files by path prefix")

	// Server generation flags
	fs.StringVar(&flags.ServerRouter, "server-router", "", "generate HTTP router (stdlib)")
	fs.BoolVar(&flags.ServerMiddleware, "server-middleware", false, "generate validation middleware using httpvalidator")
	fs.BoolVar(&flags.ServerBinder, "server-binder", false, "generate parameter binding from validation results")
	fs.BoolVar(&flags.ServerResponses, "server-responses", false, "generate typed response writers and error types")
	fs.BoolVar(&flags.ServerStubs, "server-stubs", false, "generate stub implementations for testing")
	fs.BoolVar(&flags.ServerEmbedSpec, "server-embed-spec", false, "embed OpenAPI spec in generated code")
	fs.BoolVar(&flags.ServerAll, "server-all", false, "enable all server generation options")

	fs.Usage = func() {
		cliutil.Writef(fs.Output(), "Usage: oastools generate [flags] <file|url|->\n\n")
		cliutil.Writef(fs.Output(), "Generate Go code from an OpenAPI specification.\n\n")
		cliutil.Writef(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
		cliutil.Writef(fs.Output(), "\nExamples:\n")
		cliutil.Writef(fs.Output(), "  oastools generate --client -o ./client openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --server -o ./server -p myapi openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --client --server -o ./api petstore.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --types -o ./models https://example.com/api/openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --client --oauth2-flows -o ./client openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --client --credential-mgmt --security-enforce -o ./api openapi.yaml\n")
		cliutil.Writef(fs.Output(), "  oastools generate --client --max-lines-per-file 1500 -o ./client large-api.yaml\n")
		cliutil.Writef(fs.Output(), "  cat openapi.yaml | oastools generate --client -o ./client -\n")
		cliutil.Writef(fs.Output(), "  oastools generate -s --client -o ./client openapi.yaml  # Include line numbers in issues\n")
		cliutil.Writef(fs.Output(), "\nServer Generation Examples:\n")
		cliutil.Writef(fs.Output(), "  oastools generate --server --server-all -o ./server openapi.yaml  # Full server with validation\n")
		cliutil.Writef(fs.Output(), "  oastools generate --server --server-router=stdlib -o ./server openapi.yaml  # With router\n")
		cliutil.Writef(fs.Output(), "  oastools generate --server --server-middleware --server-binder -o ./server openapi.yaml\n")
		cliutil.Writef(fs.Output(), "\nPipelining:\n")
		cliutil.Writef(fs.Output(), "  Use '-' as the file path to read the OpenAPI specification from stdin.\n")
		cliutil.Writef(fs.Output(), "  Example: oastools convert -t 3.0.3 swagger.yaml | oastools generate --client -o ./client -\n")
		cliutil.Writef(fs.Output(), "\nNotes:\n")
		cliutil.Writef(fs.Output(), "  - At least one of --client, --server, or --types must be enabled\n")
		cliutil.Writef(fs.Output(), "  - Types are always generated when --client or --server is enabled\n")
		cliutil.Writef(fs.Output(), "  - Security helpers are generated by default when --client is enabled\n")
		cliutil.Writef(fs.Output(), "  - Generated code uses Go idioms and best practices\n")
		cliutil.Writef(fs.Output(), "  - Server interface is framework-agnostic\n")
	}

	return fs, flags
}

// HandleGenerate executes the generate command
func HandleGenerate(args []string) error {
	fs, flags := SetupGenerateFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("generate command requires exactly one file path, URL, or '-' for stdin")
	}

	specPath := fs.Arg(0)

	if flags.Output == "" {
		fs.Usage()
		return fmt.Errorf("output directory is required (use -o or --output)")
	}

	// Ensure at least one generation mode is enabled
	if !flags.Client && !flags.Server && !flags.Types {
		fs.Usage()
		return fmt.Errorf("at least one of --client, --server, or --types must be enabled")
	}

	// Generate the code with timing
	startTime := time.Now()
	var result *generator.GenerateResult
	var err error

	if specPath == StdinFilePath {
		// Read from stdin - source map not supported for stdin
		p := parser.New()
		parseResult, parseErr := p.ParseReader(os.Stdin)
		if parseErr != nil {
			return fmt.Errorf("parsing stdin: %w", parseErr)
		}
		g := generator.New()
		g.PackageName = flags.PackageName
		g.GenerateClient = flags.Client
		g.GenerateServer = flags.Server
		g.GenerateTypes = flags.Types || flags.Client || flags.Server
		g.UsePointers = !flags.NoPointers
		g.IncludeValidation = !flags.NoValidation
		g.StrictMode = flags.Strict
		g.IncludeInfo = !flags.NoWarnings
		g.GenerateSecurity = !flags.NoSecurity
		g.GenerateOAuth2Flows = flags.OAuth2Flows
		g.GenerateCredentialMgmt = flags.CredentialMgmt
		g.GenerateSecurityEnforce = flags.SecurityEnforce
		g.GenerateOIDCDiscovery = flags.OIDCDiscovery
		g.GenerateReadme = !flags.NoReadme
		g.MaxLinesPerFile = flags.MaxLinesPerFile
		g.MaxTypesPerFile = flags.MaxTypesPerFile
		g.MaxOperationsPerFile = flags.MaxOpsPerFile
		g.SplitByTag = !flags.NoSplitByTag
		g.SplitByPathPrefix = !flags.NoSplitByPath
		// Server generation options
		if flags.ServerAll {
			g.ServerRouter = "stdlib"
			g.ServerMiddleware = true
			g.ServerBinder = true
			g.ServerResponses = true
			g.ServerStubs = true
		} else {
			g.ServerRouter = flags.ServerRouter
			g.ServerMiddleware = flags.ServerMiddleware
			g.ServerBinder = flags.ServerBinder
			g.ServerResponses = flags.ServerResponses
			g.ServerStubs = flags.ServerStubs
		}
		g.ServerEmbedSpec = flags.ServerEmbedSpec
		result, err = g.GenerateParsed(*parseResult)
	} else {
		// Build generator options
		genOpts := []generator.Option{
			generator.WithFilePath(specPath),
			generator.WithPackageName(flags.PackageName),
			generator.WithClient(flags.Client),
			generator.WithServer(flags.Server),
			generator.WithTypes(flags.Types || flags.Client || flags.Server),
			generator.WithPointers(!flags.NoPointers),
			generator.WithValidation(!flags.NoValidation),
			generator.WithStrictMode(flags.Strict),
			generator.WithIncludeInfo(!flags.NoWarnings),
			// Security options
			generator.WithSecurity(!flags.NoSecurity),
			generator.WithOAuth2Flows(flags.OAuth2Flows),
			generator.WithCredentialMgmt(flags.CredentialMgmt),
			generator.WithSecurityEnforce(flags.SecurityEnforce),
			generator.WithOIDCDiscovery(flags.OIDCDiscovery),
			generator.WithReadme(!flags.NoReadme),
			// File splitting options
			generator.WithMaxLinesPerFile(flags.MaxLinesPerFile),
			generator.WithMaxTypesPerFile(flags.MaxTypesPerFile),
			generator.WithMaxOperationsPerFile(flags.MaxOpsPerFile),
			generator.WithSplitByTag(!flags.NoSplitByTag),
			generator.WithSplitByPathPrefix(!flags.NoSplitByPath),
		}

		// Add server generation options
		if flags.ServerAll {
			genOpts = append(genOpts, generator.WithServerAll())
		} else {
			if flags.ServerRouter != "" {
				genOpts = append(genOpts, generator.WithServerRouter(flags.ServerRouter))
			}
			if flags.ServerMiddleware {
				genOpts = append(genOpts, generator.WithServerMiddleware(true))
			}
			if flags.ServerBinder {
				genOpts = append(genOpts, generator.WithServerBinder(true))
			}
			if flags.ServerResponses {
				genOpts = append(genOpts, generator.WithServerResponses(true))
			}
			if flags.ServerStubs {
				genOpts = append(genOpts, generator.WithServerStubs(true))
			}
		}
		if flags.ServerEmbedSpec {
			genOpts = append(genOpts, generator.WithServerEmbedSpec(true))
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
			genOpts = []generator.Option{
				generator.WithParsed(*parseResult),
				generator.WithPackageName(flags.PackageName),
				generator.WithClient(flags.Client),
				generator.WithServer(flags.Server),
				generator.WithTypes(flags.Types || flags.Client || flags.Server),
				generator.WithPointers(!flags.NoPointers),
				generator.WithValidation(!flags.NoValidation),
				generator.WithStrictMode(flags.Strict),
				generator.WithIncludeInfo(!flags.NoWarnings),
				// Security options
				generator.WithSecurity(!flags.NoSecurity),
				generator.WithOAuth2Flows(flags.OAuth2Flows),
				generator.WithCredentialMgmt(flags.CredentialMgmt),
				generator.WithSecurityEnforce(flags.SecurityEnforce),
				generator.WithOIDCDiscovery(flags.OIDCDiscovery),
				generator.WithReadme(!flags.NoReadme),
				// File splitting options
				generator.WithMaxLinesPerFile(flags.MaxLinesPerFile),
				generator.WithMaxTypesPerFile(flags.MaxTypesPerFile),
				generator.WithMaxOperationsPerFile(flags.MaxOpsPerFile),
				generator.WithSplitByTag(!flags.NoSplitByTag),
				generator.WithSplitByPathPrefix(!flags.NoSplitByPath),
			}
			// Add server generation options
			if flags.ServerAll {
				genOpts = append(genOpts, generator.WithServerAll())
			} else {
				if flags.ServerRouter != "" {
					genOpts = append(genOpts, generator.WithServerRouter(flags.ServerRouter))
				}
				if flags.ServerMiddleware {
					genOpts = append(genOpts, generator.WithServerMiddleware(true))
				}
				if flags.ServerBinder {
					genOpts = append(genOpts, generator.WithServerBinder(true))
				}
				if flags.ServerResponses {
					genOpts = append(genOpts, generator.WithServerResponses(true))
				}
				if flags.ServerStubs {
					genOpts = append(genOpts, generator.WithServerStubs(true))
				}
			}
			if flags.ServerEmbedSpec {
				genOpts = append(genOpts, generator.WithServerEmbedSpec(true))
			}
			if parseResult.SourceMap != nil {
				genOpts = append(genOpts, generator.WithSourceMap(parseResult.SourceMap))
			}
		}

		result, err = generator.GenerateWithOptions(genOpts...)
	}
	totalTime := time.Since(startTime)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Print results
	fmt.Printf("OpenAPI Code Generator\n")
	fmt.Printf("=====================\n\n")
	fmt.Printf("oastools version: %s\n", oastools.Version())
	if specPath == StdinFilePath {
		fmt.Printf("Specification: <stdin>\n")
	} else {
		fmt.Printf("Specification: %s\n", specPath)
	}
	fmt.Printf("OAS Version: %s\n", result.SourceVersion)
	fmt.Printf("Source Size: %s\n", parser.FormatBytes(result.SourceSize))
	fmt.Printf("Package: %s\n", result.PackageName)
	fmt.Printf("Types: %d\n", result.GeneratedTypes)
	fmt.Printf("Operations: %d\n", result.GeneratedOperations)
	fmt.Printf("Total Time: %v\n\n", totalTime)

	// Print issues
	if len(result.Issues) > 0 {
		fmt.Printf("Generation Issues (%d):\n", len(result.Issues))
		for _, issue := range result.Issues {
			if flags.SourceMap && issue.HasLocation() {
				// IDE-friendly format: file:line:column: path: message
				fmt.Printf("  %s: %s: %s\n", issue.Location(), issue.Path, issue.Message)
			} else {
				fmt.Printf("  %s\n", issue.String())
			}
		}
		fmt.Println()
	}

	// Write files
	if err := result.WriteFiles(flags.Output); err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	// Print generated files
	fmt.Printf("Generated Files (%d):\n", len(result.Files))
	for _, file := range result.Files {
		fmt.Printf("  - %s/%s (%d bytes)\n", flags.Output, file.Name, len(file.Content))
	}
	fmt.Println()

	// Print summary
	if result.Success {
		fmt.Printf("✓ Generation successful")
		if result.InfoCount > 0 || result.WarningCount > 0 {
			fmt.Printf(" (%d info, %d warnings)", result.InfoCount, result.WarningCount)
		}
		fmt.Println()
	} else {
		fmt.Printf("✗ Generation completed with %d critical issue(s)", result.CriticalCount)
		if result.WarningCount > 0 {
			fmt.Printf(", %d warning(s)", result.WarningCount)
		}
		fmt.Println()
		return fmt.Errorf("generation failed with %d critical issue(s)", result.CriticalCount)
	}

	return nil
}
