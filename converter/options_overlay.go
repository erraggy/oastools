package converter

import (
	"github.com/erraggy/oastools/overlay"
)

// Overlay integration options for the converter package.
//
// These options allow applying OpenAPI Overlays during the conversion process:
//   - Pre-conversion overlay: Applied to the source spec before conversion
//   - Post-conversion overlay: Applied to the converted result

// WithPreConversionOverlay sets an overlay to be applied before conversion.
//
// This is useful for fixing v2-specific issues before converting to v3,
// or normalizing a spec before version conversion.
//
// Example:
//
//	result, err := converter.ConvertWithOptions(
//	    converter.WithFilePath("swagger.yaml"),
//	    converter.WithTargetVersion("3.0.3"),
//	    converter.WithPreConversionOverlay(fixOverlay),
//	)
func WithPreConversionOverlay(o *overlay.Overlay) Option {
	return func(cfg *convertConfig) error {
		cfg.preConversionOverlay = o
		return nil
	}
}

// WithPostConversionOverlay sets an overlay to be applied after conversion.
//
// This is useful for adding v3-specific extensions or making adjustments
// to the converted document.
//
// Example:
//
//	result, err := converter.ConvertWithOptions(
//	    converter.WithFilePath("swagger.yaml"),
//	    converter.WithTargetVersion("3.0.3"),
//	    converter.WithPostConversionOverlay(enhanceOverlay),
//	)
func WithPostConversionOverlay(o *overlay.Overlay) Option {
	return func(cfg *convertConfig) error {
		cfg.postConversionOverlay = o
		return nil
	}
}

// WithPreConversionOverlayFile sets an overlay file to be applied before conversion.
//
// This is a convenience wrapper around WithPreConversionOverlay that parses the overlay file.
//
// Example:
//
//	result, err := converter.ConvertWithOptions(
//	    converter.WithFilePath("swagger.yaml"),
//	    converter.WithTargetVersion("3.0.3"),
//	    converter.WithPreConversionOverlayFile("fix-v2.yaml"),
//	)
func WithPreConversionOverlayFile(path string) Option {
	return func(cfg *convertConfig) error {
		if path == "" {
			return nil
		}
		cfg.preConversionOverlayFile = &path
		return nil
	}
}

// WithPostConversionOverlayFile sets an overlay file to be applied after conversion.
//
// This is a convenience wrapper around WithPostConversionOverlay that parses the overlay file.
//
// Example:
//
//	result, err := converter.ConvertWithOptions(
//	    converter.WithFilePath("swagger.yaml"),
//	    converter.WithTargetVersion("3.0.3"),
//	    converter.WithPostConversionOverlayFile("enhance-v3.yaml"),
//	)
func WithPostConversionOverlayFile(path string) Option {
	return func(cfg *convertConfig) error {
		if path == "" {
			return nil
		}
		cfg.postConversionOverlayFile = &path
		return nil
	}
}
