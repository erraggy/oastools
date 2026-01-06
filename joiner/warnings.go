package joiner

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/severity"
)

// WarningCategory identifies the type of warning.
type WarningCategory string

const (
	// WarnVersionMismatch indicates documents have different minor versions.
	WarnVersionMismatch WarningCategory = "version_mismatch"
	// WarnPathCollision indicates a path collision was resolved.
	WarnPathCollision WarningCategory = "path_collision"
	// WarnSchemaCollision indicates a schema/definition collision was resolved.
	WarnSchemaCollision WarningCategory = "schema_collision"
	// WarnWebhookCollision indicates a webhook collision was resolved.
	WarnWebhookCollision WarningCategory = "webhook_collision"
	// WarnSchemaRenamed indicates a schema was renamed due to collision.
	WarnSchemaRenamed WarningCategory = "schema_renamed"
	// WarnSchemaDeduplicated indicates a schema was deduplicated.
	WarnSchemaDeduplicated WarningCategory = "schema_deduplicated"
	// WarnNamespacePrefixed indicates a namespace prefix was applied.
	WarnNamespacePrefixed WarningCategory = "namespace_prefixed"
	// WarnMetadataOverride indicates metadata was overridden (host, basePath).
	WarnMetadataOverride WarningCategory = "metadata_override"
	// WarnSemanticDedup indicates semantic deduplication summary.
	WarnSemanticDedup WarningCategory = "semantic_dedup"
	// WarnGenericSourceName indicates a document has a generic or empty source name.
	// This makes collision reports less useful for identifying which document caused the collision.
	WarnGenericSourceName WarningCategory = "generic_source_name"
)

// JoinWarning represents a structured warning from the joiner package.
// It provides detailed context about non-fatal issues encountered during document joining.
type JoinWarning struct {
	// Category identifies the type of warning.
	Category WarningCategory
	// Path is the JSON path to the affected element.
	Path string
	// Message is a human-readable description.
	Message string
	// SourceFile is the file that triggered the warning.
	SourceFile string
	// Line is the 1-based line number (0 if unknown).
	Line int
	// Column is the 1-based column number (0 if unknown).
	Column int
	// Severity indicates warning severity (default: SeverityWarning).
	Severity severity.Severity
	// Context provides additional details.
	Context map[string]any
}

// String returns a formatted warning message.
// For most warnings, Context is included in Message via the constructor functions.
// This method returns just the Message for simplicity and backward compatibility.
func (w *JoinWarning) String() string {
	return w.Message
}

// HasLocation returns true if source location information is available.
func (w *JoinWarning) HasLocation() bool {
	return w.Line > 0
}

// Location returns an IDE-friendly location string.
func (w *JoinWarning) Location() string {
	if w.Line == 0 {
		if w.Path != "" {
			return w.Path
		}
		return w.SourceFile
	}
	if w.SourceFile != "" {
		if w.Column > 0 {
			return fmt.Sprintf("%s:%d:%d", w.SourceFile, w.Line, w.Column)
		}
		return fmt.Sprintf("%s:%d", w.SourceFile, w.Line)
	}
	if w.Column > 0 {
		return fmt.Sprintf("%d:%d", w.Line, w.Column)
	}
	return fmt.Sprintf("%d", w.Line)
}

// NewPathCollisionWarning creates a warning for path collisions.
func NewPathCollisionWarning(path, resolution, firstFile, secondFile string, line, col int) *JoinWarning {
	return &JoinWarning{
		Category:   WarnPathCollision,
		Path:       fmt.Sprintf("paths.%s", path),
		Message:    fmt.Sprintf("path '%s' %s: %s -> %s", path, resolution, firstFile, secondFile),
		SourceFile: secondFile,
		Line:       line,
		Column:     col,
		Severity:   severity.SeverityWarning,
		Context: map[string]any{
			"first_file":  firstFile,
			"second_file": secondFile,
			"resolution":  resolution,
		},
	}
}

// NewWebhookCollisionWarning creates a warning for webhook collisions.
func NewWebhookCollisionWarning(webhookName, resolution, firstFile, secondFile string, line, col int) *JoinWarning {
	return &JoinWarning{
		Category:   WarnWebhookCollision,
		Path:       fmt.Sprintf("webhooks.%s", webhookName),
		Message:    fmt.Sprintf("webhook '%s' %s: %s -> %s", webhookName, resolution, firstFile, secondFile),
		SourceFile: secondFile,
		Line:       line,
		Column:     col,
		Severity:   severity.SeverityWarning,
		Context: map[string]any{
			"first_file":  firstFile,
			"second_file": secondFile,
			"resolution":  resolution,
		},
	}
}

// NewSchemaCollisionWarning creates a warning for schema/definition collisions.
func NewSchemaCollisionWarning(schemaName, resolution, section, firstFile, secondFile string, line, col int) *JoinWarning {
	return &JoinWarning{
		Category:   WarnSchemaCollision,
		Path:       fmt.Sprintf("%s.%s", section, schemaName),
		Message:    fmt.Sprintf("%s '%s' %s: source %s", section, schemaName, resolution, secondFile),
		SourceFile: secondFile,
		Line:       line,
		Column:     col,
		Severity:   severity.SeverityWarning,
		Context: map[string]any{
			"section":     section,
			"first_file":  firstFile,
			"second_file": secondFile,
			"resolution":  resolution,
		},
	}
}

// NewSchemaRenamedWarning creates a warning when a schema is renamed.
func NewSchemaRenamedWarning(originalName, newName, section, sourceFile string, line, col int, keptOriginal bool) *JoinWarning {
	var msg string
	if keptOriginal {
		msg = fmt.Sprintf("%s '%s' renamed to '%s' (kept from first document)", section, originalName, newName)
	} else {
		msg = fmt.Sprintf("%s '%s' from %s renamed to '%s'", section, originalName, sourceFile, newName)
	}
	return &JoinWarning{
		Category:   WarnSchemaRenamed,
		Path:       fmt.Sprintf("%s.%s", section, originalName),
		Message:    msg,
		SourceFile: sourceFile,
		Line:       line,
		Column:     col,
		Severity:   severity.SeverityInfo,
		Context: map[string]any{
			"original_name": originalName,
			"new_name":      newName,
			"section":       section,
			"kept_original": keptOriginal,
		},
	}
}

// NewSchemaDedupWarning creates a warning for schema deduplication.
func NewSchemaDedupWarning(schemaName, section, sourceFile string, line, col int) *JoinWarning {
	return &JoinWarning{
		Category:   WarnSchemaDeduplicated,
		Path:       fmt.Sprintf("%s.%s", section, schemaName),
		Message:    fmt.Sprintf("%s '%s' deduplicated (structurally equivalent): %s", section, schemaName, sourceFile),
		SourceFile: sourceFile,
		Line:       line,
		Column:     col,
		Severity:   severity.SeverityInfo,
		Context: map[string]any{
			"section": section,
		},
	}
}

// NewNamespacePrefixWarning creates a warning when a namespace prefix is applied.
func NewNamespacePrefixWarning(originalName, newName, section, sourceFile string, line, col int) *JoinWarning {
	return &JoinWarning{
		Category:   WarnNamespacePrefixed,
		Path:       fmt.Sprintf("%s.%s", section, originalName),
		Message:    fmt.Sprintf("%s '%s' prefixed to '%s' (namespace prefix from %s)", section, originalName, newName, sourceFile),
		SourceFile: sourceFile,
		Line:       line,
		Column:     col,
		Severity:   severity.SeverityInfo,
		Context: map[string]any{
			"original_name": originalName,
			"new_name":      newName,
			"section":       section,
		},
	}
}

// NewVersionMismatchWarning creates a warning for version mismatches.
func NewVersionMismatchWarning(file1, version1, file2, version2, targetVersion string) *JoinWarning {
	return &JoinWarning{
		Category: WarnVersionMismatch,
		Message: fmt.Sprintf(
			"joining documents with different minor versions: %s (%s) and %s (%s). Result will use version %s",
			file1, version1, file2, version2, targetVersion),
		Severity: severity.SeverityWarning,
		Context: map[string]any{
			"file1":          file1,
			"version1":       version1,
			"file2":          file2,
			"version2":       version2,
			"target_version": targetVersion,
		},
	}
}

// NewMetadataOverrideWarning creates a warning when metadata is overridden.
func NewMetadataOverrideWarning(field, firstValue, secondValue, secondFile string) *JoinWarning {
	return &JoinWarning{
		Category:   WarnMetadataOverride,
		Path:       field,
		Message:    fmt.Sprintf("%s '%s' ignored (kept '%s' from first document)", field, secondValue, firstValue),
		SourceFile: secondFile,
		Severity:   severity.SeverityInfo,
		Context: map[string]any{
			"field":        field,
			"first_value":  firstValue,
			"second_value": secondValue,
		},
	}
}

// NewSemanticDedupSummaryWarning creates a summary warning for semantic deduplication.
func NewSemanticDedupSummaryWarning(count int, section string) *JoinWarning {
	return &JoinWarning{
		Category: WarnSemanticDedup,
		Message:  fmt.Sprintf("semantic deduplication: consolidated %d duplicate %s(s)", count, section),
		Severity: severity.SeverityInfo,
		Context: map[string]any{
			"count":   count,
			"section": section,
		},
	}
}

// IsGenericSourceName returns true if the source path appears to be a generic
// parser-generated name rather than a meaningful identifier.
// Generic names include empty strings and default names like "ParseBytes.yaml".
func IsGenericSourceName(sourcePath string) bool {
	if sourcePath == "" {
		return true
	}
	genericPrefixes := []string{
		"ParseBytes.",
		"ParseReader.",
	}
	for _, prefix := range genericPrefixes {
		if strings.HasPrefix(sourcePath, prefix) {
			return true
		}
	}
	return false
}

// NewGenericSourceNameWarning creates a warning when a document has a generic source name.
// This helps users identify that collision reports may be unclear and guides them to set
// meaningful source names using ParseResult.SourcePath.
func NewGenericSourceNameWarning(sourcePath string, docIndex int) *JoinWarning {
	var msg string
	if sourcePath == "" {
		msg = fmt.Sprintf("document %d has empty source name - collision reports will be unclear. "+
			"Set ParseResult.SourcePath to a meaningful identifier before joining", docIndex)
	} else {
		msg = fmt.Sprintf("document %d has generic source name '%s' - collision reports may be unclear. "+
			"Set ParseResult.SourcePath to a meaningful identifier (e.g., service name) before joining", docIndex, sourcePath)
	}
	return &JoinWarning{
		Category:   WarnGenericSourceName,
		Message:    msg,
		SourceFile: sourcePath,
		Severity:   severity.SeverityInfo,
		Context: map[string]any{
			"doc_index":   docIndex,
			"source_path": sourcePath,
		},
	}
}

// JoinWarnings is a collection of JoinWarning.
type JoinWarnings []*JoinWarning

// Strings returns warning messages for backward compatibility.
func (ws JoinWarnings) Strings() []string {
	result := make([]string, len(ws))
	for i, w := range ws {
		if w == nil {
			continue
		}
		result[i] = w.String()
	}
	return result
}

// ByCategory filters warnings by category.
func (ws JoinWarnings) ByCategory(cat WarningCategory) JoinWarnings {
	var result JoinWarnings
	for _, w := range ws {
		if w.Category == cat {
			result = append(result, w)
		}
	}
	return result
}

// BySeverity filters warnings by severity.
func (ws JoinWarnings) BySeverity(sev severity.Severity) JoinWarnings {
	var result JoinWarnings
	for _, w := range ws {
		if w.Severity == sev {
			result = append(result, w)
		}
	}
	return result
}

// Summary returns a formatted summary of warnings.
func (ws JoinWarnings) Summary() string {
	if len(ws) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d warning(s):\n", len(ws)))
	for _, w := range ws {
		sb.WriteString("  - ")
		sb.WriteString(w.String())
		sb.WriteString("\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}
