package converter

import (
	"fmt"

	"github.com/erraggy/oastools/internal/issues"
	"github.com/erraggy/oastools/internal/severity"
	"github.com/erraggy/oastools/parser"
)

// Severity indicates the severity level of a conversion issue
type Severity = severity.Severity

const (
	// SeverityInfo indicates informational messages about conversion choices
	SeverityInfo = severity.SeverityInfo
	// SeverityWarning indicates lossy conversions or best-effort transformations
	SeverityWarning = severity.SeverityWarning
	// SeverityCritical indicates features that cannot be converted (data loss)
	SeverityCritical = severity.SeverityCritical
)

// ConversionIssue represents a single conversion issue or limitation
type ConversionIssue = issues.Issue

// ConversionResult contains the results of converting an OpenAPI specification
type ConversionResult struct {
	// Document contains the converted document (*parser.OAS2Document or *parser.OAS3Document)
	Document any
	// SourceVersion is the detected source OAS version string
	SourceVersion string
	// SourceOASVersion is the enumerated source OAS version
	SourceOASVersion parser.OASVersion
	// SourceFormat is the format of the source file (JSON or YAML)
	SourceFormat parser.SourceFormat
	// TargetVersion is the target OAS version string
	TargetVersion string
	// TargetOASVersion is the enumerated target OAS version
	TargetOASVersion parser.OASVersion
	// Issues contains all conversion issues grouped by severity
	Issues []ConversionIssue
	// InfoCount is the total number of info messages
	InfoCount int
	// WarningCount is the total number of warnings
	WarningCount int
	// CriticalCount is the total number of critical issues
	CriticalCount int
	// Success is true if conversion completed without critical issues
	Success bool
}

// HasCriticalIssues returns true if there are any critical issues
func (r *ConversionResult) HasCriticalIssues() bool {
	return r.CriticalCount > 0
}

// HasWarnings returns true if there are any warnings
func (r *ConversionResult) HasWarnings() bool {
	return r.WarningCount > 0
}

// Converter handles OpenAPI specification version conversion
type Converter struct {
	// StrictMode causes conversion to fail on any issues (even warnings)
	StrictMode bool
	// IncludeInfo determines whether to include informational messages
	IncludeInfo bool
	// UserAgent is the User-Agent string used when fetching URLs
	// Defaults to "oastools" if not set
	UserAgent string
}

// New creates a new Converter instance with default settings
func New() *Converter {
	return &Converter{
		StrictMode:  false,
		IncludeInfo: true,
	}
}

// Convert is a convenience function that converts an OpenAPI specification file
// to a target version with the specified options. It's equivalent to creating a
// Converter with New() and calling Convert().
//
// For one-off conversion operations, this function provides a simpler API.
// For converting multiple files with the same configuration, create a Converter
// instance and reuse it.
//
// Example:
//
//	result, err := converter.Convert("swagger.yaml", "3.0.3")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if result.HasCriticalIssues() {
//	    // Handle critical issues
//	}
func Convert(specPath string, targetVersion string) (*ConversionResult, error) {
	c := New()
	return c.Convert(specPath, targetVersion)
}

// ConvertParsed is a convenience function that converts an already-parsed
// OpenAPI specification to a target version.
//
// Example:
//
//	parseResult, _ := parser.Parse("swagger.yaml", false, true)
//	result, err := converter.ConvertParsed(*parseResult, "3.0.3")
func ConvertParsed(parseResult parser.ParseResult, targetVersion string) (*ConversionResult, error) {
	c := New()
	return c.ConvertParsed(parseResult, targetVersion)
}

// Convert converts an OpenAPI specification file to a target version
func (c *Converter) Convert(specPath string, targetVersion string) (*ConversionResult, error) {
	// Create parser and set UserAgent if specified
	p := parser.New()
	if c.UserAgent != "" {
		p.UserAgent = c.UserAgent
	}

	// Parse the source document
	parseResult, err := p.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse specification: %w", err)
	}

	// Check for parse errors
	if len(parseResult.Errors) > 0 {
		return nil, fmt.Errorf("source document has %d parse error(s), cannot convert", len(parseResult.Errors))
	}

	return c.ConvertParsed(*parseResult, targetVersion)
}

// ConvertParsed converts an already-parsed OpenAPI specification to a target version
func (c *Converter) ConvertParsed(parseResult parser.ParseResult, targetVersionStr string) (*ConversionResult, error) {
	// Parse target version
	targetVersion, ok := parser.ParseVersion(targetVersionStr)
	if !ok {
		return nil, fmt.Errorf("invalid target version: %s", targetVersionStr)
	}

	// Initialize result
	result := &ConversionResult{
		SourceVersion:    parseResult.Version,
		SourceOASVersion: parseResult.OASVersion,
		SourceFormat:     parseResult.SourceFormat,
		TargetVersion:    targetVersionStr,
		TargetOASVersion: targetVersion,
		Issues:           make([]ConversionIssue, 0),
	}

	// Check if conversion is needed
	if parseResult.OASVersion == targetVersion {
		// No conversion needed, just copy the document
		result.Document = parseResult.Document
		result.Success = true
		result.Issues = append(result.Issues, ConversionIssue{
			Path:     "document",
			Message:  fmt.Sprintf("Source and target versions are the same (%s), no conversion needed", targetVersionStr),
			Severity: SeverityInfo,
		})
		c.updateCounts(result)
		return result, nil
	}

	// Determine conversion direction
	sourceIsOAS2 := parseResult.OASVersion == parser.OASVersion20
	targetIsOAS2 := targetVersion == parser.OASVersion20

	var err error
	switch {
	case sourceIsOAS2 && !targetIsOAS2:
		// OAS 2.0 → OAS 3.x
		err = c.convertOAS2ToOAS3(parseResult, targetVersion, result)
	case !sourceIsOAS2 && targetIsOAS2:
		// OAS 3.x → OAS 2.0
		err = c.convertOAS3ToOAS2(parseResult, result)
	case !sourceIsOAS2 && !targetIsOAS2:
		// OAS 3.x → OAS 3.y (version update)
		err = c.convertOAS3ToOAS3(parseResult, targetVersion, result)
	default:
		return nil, fmt.Errorf("unsupported conversion: %s → %s", parseResult.Version, targetVersionStr)
	}

	if err != nil {
		return nil, err
	}

	// Update counts and success status
	c.updateCounts(result)
	result.Success = result.CriticalCount == 0

	// In strict mode, fail on any issues
	if c.StrictMode && (result.CriticalCount > 0 || result.WarningCount > 0) {
		return result, fmt.Errorf("conversion failed in strict mode: %d critical issue(s), %d warning(s)",
			result.CriticalCount, result.WarningCount)
	}

	// Filter info messages if not included
	if !c.IncludeInfo {
		filtered := make([]ConversionIssue, 0, len(result.Issues))
		for _, issue := range result.Issues {
			if issue.Severity != SeverityInfo {
				filtered = append(filtered, issue)
			}
		}
		result.Issues = filtered
		result.InfoCount = 0
	}

	return result, nil
}

// updateCounts updates the issue counts in the result
func (c *Converter) updateCounts(result *ConversionResult) {
	result.InfoCount = 0
	result.WarningCount = 0
	result.CriticalCount = 0

	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityInfo:
			result.InfoCount++
		case SeverityWarning:
			result.WarningCount++
		case SeverityCritical:
			result.CriticalCount++
		}
	}
}

// convertOAS3ToOAS3 handles version updates within OAS 3.x (e.g., 3.0.3 → 3.1.0)
func (c *Converter) convertOAS3ToOAS3(parseResult parser.ParseResult, targetVersion parser.OASVersion, result *ConversionResult) error {
	// For OAS 3.x to 3.y conversions, we primarily just update the version string
	doc, ok := parseResult.Document.(*parser.OAS3Document)
	if !ok {
		return fmt.Errorf("source document is not an OAS3Document")
	}

	// Create a deep copy of the document
	converted, err := c.deepCopyOAS3Document(doc)
	if err != nil {
		return err
	}

	// Update the version string
	converted.OpenAPI = result.TargetVersion
	converted.OASVersion = targetVersion

	result.Document = converted

	// Add informational message about version update
	result.Issues = append(result.Issues, ConversionIssue{
		Path:     "openapi",
		Message:  fmt.Sprintf("Updated version from %s to %s", parseResult.Version, result.TargetVersion),
		Severity: SeverityInfo,
		Context:  "OAS 3.x versions are generally compatible, but verify features are supported",
	})

	return nil
}

// addIssue is a helper to add a conversion issue to the result
func (c *Converter) addIssue(result *ConversionResult, path, message string, severity Severity) {
	result.Issues = append(result.Issues, ConversionIssue{
		Path:     path,
		Message:  message,
		Severity: severity,
	})
}

// addIssueWithContext is a helper to add a conversion issue with context
func (c *Converter) addIssueWithContext(result *ConversionResult, path, message, context string) {
	result.Issues = append(result.Issues, ConversionIssue{
		Path:     path,
		Message:  message,
		Severity: SeverityWarning,
		Context:  context,
	})
}
