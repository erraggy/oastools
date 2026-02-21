// Package commands provides CLI command handlers for oastools.
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	oastools "github.com/erraggy/oastools"
	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
	"go.yaml.in/yaml/v4"
)

// Output format constants
const (
	FormatText = "text"
	FormatJSON = "json"
	FormatYAML = "yaml"
)

// StdinFilePath is the special file path used to indicate reading from stdin.
const StdinFilePath = "-"

// ValidateOutputFormat validates an output format and returns an error if invalid.
func ValidateOutputFormat(format string) error {
	if format != FormatText && format != FormatJSON && format != FormatYAML {
		return fmt.Errorf("invalid format '%s'. Valid formats: %s, %s, %s", format, FormatText, FormatJSON, FormatYAML)
	}
	return nil
}

// OutputStructured outputs data in the specified format (json or yaml) to stdout.
// Returns an error if marshaling fails.
func OutputStructured(data any, format string) error {
	var bytes []byte
	var err error

	switch format {
	case FormatJSON:
		bytes, err = json.MarshalIndent(data, "", "  ")
	case FormatYAML:
		bytes, err = yaml.Marshal(data)
	default:
		return fmt.Errorf("invalid format for structured output: %s", format)
	}

	if err != nil {
		return fmt.Errorf("marshaling to %s: %w", format, err)
	}

	fmt.Println(string(bytes))
	return nil
}

// ValidateCollisionStrategy validates a collision strategy name and returns an error if invalid.
// The strategyName parameter is used in the error message (e.g., "path-strategy").
func ValidateCollisionStrategy(strategyName, value string) error {
	if value != "" && !joiner.IsValidStrategy(value) {
		return fmt.Errorf("invalid %s '%s'. Valid strategies: %v", strategyName, value, joiner.ValidStrategies())
	}
	return nil
}

// ValidateEquivalenceMode validates an equivalence mode and returns an error if invalid.
func ValidateEquivalenceMode(value string) error {
	if value != "" && !joiner.IsValidEquivalenceMode(value) {
		return fmt.Errorf("invalid equivalence-mode '%s'. Valid modes: %v", value, joiner.ValidEquivalenceModes())
	}
	return nil
}

// ValidatePrimaryOperationPolicy validates the primary operation policy flag value.
func ValidatePrimaryOperationPolicy(policy string) error {
	if policy == "" {
		return nil
	}
	valid := []string{"first", "most-specific", "alphabetical"}
	if slices.Contains(valid, policy) {
		return nil
	}
	return fmt.Errorf("commands: invalid primary-operation-policy %q: must be one of: first, most-specific, alphabetical", policy)
}

// MapPrimaryOperationPolicy maps a string policy to the joiner enum.
func MapPrimaryOperationPolicy(policy string) joiner.PrimaryOperationPolicy {
	switch policy {
	case "most-specific":
		return joiner.PolicyMostSpecific
	case "alphabetical":
		return joiner.PolicyAlphabetical
	default:
		return joiner.PolicyFirstEncountered
	}
}

// ValidateOutputPath checks if the output path is safe to write to
func ValidateOutputPath(outputPath string, inputPaths []string) error {
	// Get absolute path of output file
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Check if output file would overwrite any input files
	for _, inputPath := range inputPaths {
		absInputPath, err := filepath.Abs(inputPath)
		if err != nil {
			return fmt.Errorf("invalid input path %s: %w", inputPath, err)
		}

		if absOutputPath == absInputPath {
			return fmt.Errorf("output file %s would overwrite input file %s", outputPath, inputPath)
		}
	}

	// Check if output file already exists and warn (but don't error)
	if _, err := os.Stat(outputPath); err == nil {
		Writef(os.Stderr, "Warning: output file %s already exists and will be overwritten\n", outputPath)
	}

	return nil
}

// RejectSymlinkOutput checks if the output path is a symlink and returns an error if so.
// This prevents symlink attacks where a symlink could redirect output to an unintended location.
func RejectSymlinkOutput(cleanedPath string) error {
	info, err := os.Lstat(cleanedPath)
	if os.IsNotExist(err) {
		// File doesn't exist yet — safe to write.
		return nil
	}
	if err != nil {
		return fmt.Errorf("commands: checking output path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("commands: refusing to write to symlink: %s", cleanedPath)
	}
	return nil
}

// MarshalDocument marshals a document to bytes in the specified format
func MarshalDocument(doc any, format parser.SourceFormat) ([]byte, error) {
	if format == parser.SourceFormatJSON {
		return json.MarshalIndent(doc, "", "  ")
	}
	return yaml.Marshal(doc)
}

// FormatSpecPath returns a display-friendly path for the specification.
// Returns "<stdin>" if the path is StdinFilePath, otherwise returns the path as-is.
func FormatSpecPath(specPath string) string {
	if specPath == StdinFilePath {
		return "<stdin>"
	}
	return specPath
}

// Writef writes formatted output to the writer.
// If the write fails, it logs to stderr (useful for debugging).
func Writef(w io.Writer, format string, args ...any) {
	if _, err := fmt.Fprintf(w, format, args...); err != nil { //nolint:gosec // G705 - CLI tool, not a web server
		_, _ = fmt.Fprintf(os.Stderr, "write error: %v\n", err)
	}
}

// OutputSpecHeader outputs the common specification header to stderr.
// This includes oastools version, specification path, and OAS version.
func OutputSpecHeader(specPath, version string) {
	Writef(os.Stderr, "oastools version: %s\n", oastools.Version())
	Writef(os.Stderr, "Specification: %s\n", FormatSpecPath(specPath))
	Writef(os.Stderr, "OAS Version: %s\n", version)
}

// OutputSpecStats outputs the common specification statistics to stderr.
// This includes source size, path count, operation count, schema count, and load time.
func OutputSpecStats(sourceSize int64, stats parser.DocumentStats, loadTime any) {
	Writef(os.Stderr, "Source Size: %s\n", parser.FormatBytes(sourceSize))
	Writef(os.Stderr, "Paths: %d\n", stats.PathCount)
	Writef(os.Stderr, "Operations: %d\n", stats.OperationCount)
	Writef(os.Stderr, "Schemas: %d\n", stats.SchemaCount)
	Writef(os.Stderr, "Load Time: %v\n", loadTime)
}
