package validator

import (
	"github.com/erraggy/oastools/internal/options"
	"github.com/erraggy/oastools/parser"
)

// Option is a function that configures a validation operation
type Option func(*validateConfig) error

// validateConfig holds configuration for a validation operation
type validateConfig struct {
	// Input source (exactly one must be set)
	filePath *string
	parsed   *parser.ParseResult

	// Configuration options
	includeWarnings   bool
	strictMode        bool
	validateStructure bool
	userAgent         string
	sourceMap         *parser.SourceMap
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*validateConfig, error) {
	cfg := &validateConfig{
		// Set defaults to match existing behavior
		includeWarnings:   true,
		strictMode:        false,
		validateStructure: true,
		userAgent:         "",
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate exactly one input source is specified
	if err := options.ValidateSingleInputSource(
		"must specify an input source (use WithFilePath or WithParsed)",
		"must specify exactly one input source",
		cfg.filePath != nil, cfg.parsed != nil,
	); err != nil {
		return nil, err
	}

	return cfg, nil
}

// WithFilePath specifies a file path or URL as the input source
func WithFilePath(path string) Option {
	return func(cfg *validateConfig) error {
		cfg.filePath = &path
		return nil
	}
}

// WithParsed specifies a parsed ParseResult as the input source
func WithParsed(result parser.ParseResult) Option {
	return func(cfg *validateConfig) error {
		cfg.parsed = &result
		return nil
	}
}

// WithIncludeWarnings enables or disables best practice warnings
// Default: true
func WithIncludeWarnings(enabled bool) Option {
	return func(cfg *validateConfig) error {
		cfg.includeWarnings = enabled
		return nil
	}
}

// WithStrictMode enables or disables strict validation beyond spec requirements
// Default: false
func WithStrictMode(enabled bool) Option {
	return func(cfg *validateConfig) error {
		cfg.strictMode = enabled
		return nil
	}
}

// WithValidateStructure enables or disables parser structure validation.
// When enabled (default), the parser validates required fields and correct types.
// When disabled, parsing is more lenient and skips structure validation.
// Default: true
func WithValidateStructure(enabled bool) Option {
	return func(cfg *validateConfig) error {
		cfg.validateStructure = enabled
		return nil
	}
}

// WithUserAgent sets the User-Agent string for HTTP requests
// Default: "" (uses parser default)
func WithUserAgent(ua string) Option {
	return func(cfg *validateConfig) error {
		cfg.userAgent = ua
		return nil
	}
}

// WithSourceMap provides a SourceMap for populating line/column information
// in validation errors. The SourceMap is typically obtained from parsing
// with parser.WithSourceMap(true).
func WithSourceMap(sm *parser.SourceMap) Option {
	return func(cfg *validateConfig) error {
		cfg.sourceMap = sm
		return nil
	}
}
