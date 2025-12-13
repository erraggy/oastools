package overlay

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// Option is a function that configures an overlay application operation.
type Option func(*applyConfig) error

// applyConfig holds configuration for an overlay application operation.
type applyConfig struct {
	// Input source for specification (exactly one must be set)
	specFilePath *string
	specParsed   *parser.ParseResult

	// Input source for overlay (exactly one must be set)
	overlayFilePath *string
	overlayParsed   *Overlay

	// Configuration options
	strictTargets bool
}

// WithSpecFilePath specifies a file path or URL as the specification input source.
func WithSpecFilePath(path string) Option {
	return func(cfg *applyConfig) error {
		if path == "" {
			return fmt.Errorf("specification path cannot be empty")
		}
		cfg.specFilePath = &path
		return nil
	}
}

// WithSpecParsed specifies an already-parsed specification as the input source.
func WithSpecParsed(result parser.ParseResult) Option {
	return func(cfg *applyConfig) error {
		cfg.specParsed = &result
		return nil
	}
}

// WithOverlayFilePath specifies a file path as the overlay input source.
func WithOverlayFilePath(path string) Option {
	return func(cfg *applyConfig) error {
		if path == "" {
			return fmt.Errorf("overlay path cannot be empty")
		}
		cfg.overlayFilePath = &path
		return nil
	}
}

// WithOverlayParsed specifies an already-parsed overlay as the input source.
func WithOverlayParsed(o *Overlay) Option {
	return func(cfg *applyConfig) error {
		if o == nil {
			return fmt.Errorf("overlay cannot be nil")
		}
		cfg.overlayParsed = o
		return nil
	}
}

// WithStrictTargets enables strict mode where unmatched targets cause errors.
//
// By default, actions with targets that match no nodes are skipped with a warning.
// When strict mode is enabled, such actions cause an error instead.
func WithStrictTargets(strict bool) Option {
	return func(cfg *applyConfig) error {
		cfg.strictTargets = strict
		return nil
	}
}

// applyOptions applies all options and returns the configuration.
func applyOptions(opts ...Option) (*applyConfig, error) {
	cfg := &applyConfig{
		strictTargets: false, // Default: non-strict
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate exactly one specification source
	specSourceCount := 0
	if cfg.specFilePath != nil {
		specSourceCount++
	}
	if cfg.specParsed != nil {
		specSourceCount++
	}

	if specSourceCount == 0 {
		return nil, fmt.Errorf("must specify a specification source (use WithSpecFilePath or WithSpecParsed)")
	}
	if specSourceCount > 1 {
		return nil, fmt.Errorf("must specify exactly one specification source")
	}

	// Validate exactly one overlay source
	overlaySourceCount := 0
	if cfg.overlayFilePath != nil {
		overlaySourceCount++
	}
	if cfg.overlayParsed != nil {
		overlaySourceCount++
	}

	if overlaySourceCount == 0 {
		return nil, fmt.Errorf("must specify an overlay source (use WithOverlayFilePath or WithOverlayParsed)")
	}
	if overlaySourceCount > 1 {
		return nil, fmt.Errorf("must specify exactly one overlay source")
	}

	return cfg, nil
}

// loadInputs parses the specification and overlay from the configuration.
func loadInputs(cfg *applyConfig) (*parser.ParseResult, *Overlay, error) {
	var spec *parser.ParseResult
	var o *Overlay
	var err error

	// Get specification
	if cfg.specFilePath != nil {
		p := parser.New()
		spec, err = p.Parse(*cfg.specFilePath)
		if err != nil {
			return nil, nil, fmt.Errorf("overlay: failed to parse specification: %w", err)
		}
	} else {
		spec = cfg.specParsed
	}

	// Get overlay
	if cfg.overlayFilePath != nil {
		o, err = ParseOverlayFile(*cfg.overlayFilePath)
		if err != nil {
			return nil, nil, err
		}
	} else {
		o = cfg.overlayParsed
	}

	return spec, o, nil
}

// ApplyWithOptions applies an overlay to a specification using functional options.
//
// This is the recommended API for most use cases. It provides a clean, fluent
// interface for configuring overlay application.
//
// Example:
//
//	result, err := overlay.ApplyWithOptions(
//	    overlay.WithSpecFilePath("openapi.yaml"),
//	    overlay.WithOverlayFilePath("changes.yaml"),
//	    overlay.WithStrictTargets(true),
//	)
func ApplyWithOptions(opts ...Option) (*ApplyResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("overlay: invalid options: %w", err)
	}

	spec, o, err := loadInputs(cfg)
	if err != nil {
		return nil, err
	}

	a := &Applier{StrictTargets: cfg.strictTargets}
	return a.ApplyParsed(spec, o)
}
