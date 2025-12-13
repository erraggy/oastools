package joiner

import (
	"github.com/erraggy/oastools/overlay"
)

// Overlay integration options for the joiner package.
//
// These options allow applying OpenAPI Overlays during the join process:
//   - Pre-join overlays: Applied to each input spec before merging
//   - Post-join overlay: Applied to the merged result after joining
//   - Per-spec overlays: Apply different overlays to specific input specs

// WithPreJoinOverlay adds an overlay to be applied to all input specs before joining.
//
// Multiple pre-join overlays can be specified and are applied in order.
// This is useful for normalizing specs before merge (e.g., adding required fields,
// standardizing naming conventions).
//
// Example:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths("api1.yaml", "api2.yaml"),
//	    joiner.WithPreJoinOverlay(normalizeOverlay),
//	)
func WithPreJoinOverlay(o *overlay.Overlay) Option {
	return func(cfg *joinConfig) error {
		if o == nil {
			return nil
		}
		cfg.preJoinOverlays = append(cfg.preJoinOverlays, o)
		return nil
	}
}

// WithPostJoinOverlay sets an overlay to be applied after joining is complete.
//
// Only one post-join overlay can be specified (last one wins).
// This is useful for adding unified metadata, removing internal extensions,
// or applying final transformations to the merged document.
//
// Example:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths("api1.yaml", "api2.yaml"),
//	    joiner.WithPostJoinOverlay(enhanceOverlay),
//	)
func WithPostJoinOverlay(o *overlay.Overlay) Option {
	return func(cfg *joinConfig) error {
		cfg.postJoinOverlay = o
		return nil
	}
}

// WithPreJoinOverlayFile adds an overlay file to be applied to all input specs before joining.
//
// This is a convenience wrapper around WithPreJoinOverlay that parses the overlay file.
// Multiple pre-join overlay files can be specified.
//
// Example:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths("api1.yaml", "api2.yaml"),
//	    joiner.WithPreJoinOverlayFile("normalize.yaml"),
//	)
func WithPreJoinOverlayFile(path string) Option {
	return func(cfg *joinConfig) error {
		if path == "" {
			return nil
		}
		cfg.preJoinOverlayFiles = append(cfg.preJoinOverlayFiles, path)
		return nil
	}
}

// WithPostJoinOverlayFile sets an overlay file to be applied after joining is complete.
//
// This is a convenience wrapper around WithPostJoinOverlay that parses the overlay file.
// Only one post-join overlay file can be specified (last one wins).
//
// Example:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths("api1.yaml", "api2.yaml"),
//	    joiner.WithPostJoinOverlayFile("enhance.yaml"),
//	)
func WithPostJoinOverlayFile(path string) Option {
	return func(cfg *joinConfig) error {
		cfg.postJoinOverlayFile = &path
		return nil
	}
}

// WithSpecOverlay maps a specific overlay to a specific input spec.
//
// The specIdentifier should match either:
//   - A file path from WithFilePaths (e.g., "api1.yaml")
//   - An index like "0", "1", etc. for WithParsed documents
//
// This allows applying different transformations to different input specs.
//
// Example:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths("users-api.yaml", "billing-api.yaml"),
//	    joiner.WithSpecOverlay("users-api.yaml", usersOverlay),
//	    joiner.WithSpecOverlay("billing-api.yaml", billingOverlay),
//	)
func WithSpecOverlay(specIdentifier string, o *overlay.Overlay) Option {
	return func(cfg *joinConfig) error {
		if cfg.specOverlays == nil {
			cfg.specOverlays = make(map[string]*overlay.Overlay)
		}
		cfg.specOverlays[specIdentifier] = o
		return nil
	}
}

// WithSpecOverlayFile maps a specific overlay file to a specific input spec.
//
// This is a convenience wrapper around WithSpecOverlay that parses the overlay file.
//
// Example:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths("users-api.yaml", "billing-api.yaml"),
//	    joiner.WithSpecOverlayFile("users-api.yaml", "users-overlay.yaml"),
//	)
func WithSpecOverlayFile(specIdentifier, overlayPath string) Option {
	return func(cfg *joinConfig) error {
		if cfg.specOverlayFiles == nil {
			cfg.specOverlayFiles = make(map[string]string)
		}
		cfg.specOverlayFiles[specIdentifier] = overlayPath
		return nil
	}
}
