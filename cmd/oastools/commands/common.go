// Package commands provides CLI command handlers for oastools.
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/erraggy/oastools/internal/cliutil"
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
		cliutil.Writef(os.Stderr, "Warning: output file %s already exists and will be overwritten\n", outputPath)
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
