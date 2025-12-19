package differ

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// isCompatibleTypeChange checks if a type change is compatible (widening)
func isCompatibleTypeChange(oldType, newType string) bool {
	// integer -> number is a widening conversion (compatible)
	if oldType == "integer" && newType == "number" {
		return true
	}
	return false
}

// isSuccessCode checks if a status code is a success code (2xx)
func isSuccessCode(code string) bool {
	if strings.HasPrefix(code, "2") {
		return true
	}
	// Check if it's a numeric 2xx code
	if codeNum, err := strconv.Atoi(code); err == nil {
		return codeNum >= 200 && codeNum < 300
	}
	return false
}

// isErrorCode checks if a status code is an error code (4xx or 5xx)
func isErrorCode(code string) bool {
	if strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5") {
		return true
	}
	// Check if it's a numeric 4xx or 5xx code
	if codeNum, err := strconv.Atoi(code); err == nil {
		return codeNum >= 400
	}
	return false
}

// anyToString converts any value to a string representation
func anyToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprint(val)
	}
}

// This file contains unified diff functions that support both ModeSimple and ModeBreaking.
// The mode is checked at runtime to determine appropriate severity levels.
//
// Migration Status:
// - Phase 1: Infrastructure (this file) - COMPLETE
// - Phase 2: Non-schema functions - IN PROGRESS
// - Phase 3: Response functions - PENDING
// - Phase 4: Schema functions - PENDING
// - Phase 5: Operation functions - PENDING
// - Phase 6: Top-level functions - PENDING
// - Phase 7: Cleanup - PENDING

// severity returns the appropriate severity based on diff mode.
// In simple mode, severity is always 0 (unset).
// In breaking mode, the provided breaking severity is used.
func (d *Differ) severity(breakingSeverity Severity) Severity {
	if d.Mode == ModeBreaking {
		return breakingSeverity
	}
	return 0
}

// severityConditional returns severity based on mode and a condition.
// This is useful when the severity depends on the nature of the change.
func (d *Differ) severityConditional(condition bool, severityIfTrue, severityIfFalse Severity) Severity {
	if d.Mode != ModeBreaking {
		return 0
	}
	if condition {
		return severityIfTrue
	}
	return severityIfFalse
}

// addChange is a helper to append a change with mode-appropriate severity.
func (d *Differ) addChange(result *DiffResult, path string, changeType ChangeType, category ChangeCategory, breakingSeverity Severity, oldValue, newValue any, message string) {
	change := Change{
		Path:     path,
		Type:     changeType,
		Category: category,
		Severity: d.severity(breakingSeverity),
		OldValue: oldValue,
		NewValue: newValue,
		Message:  message,
	}
	d.populateChangeLocation(&change, changeType)
	result.Changes = append(result.Changes, change)
}

// addChangeConditional is a helper that picks severity based on a condition.
func (d *Differ) addChangeConditional(result *DiffResult, path string, changeType ChangeType, category ChangeCategory, condition bool, severityIfTrue, severityIfFalse Severity, oldValue, newValue any, message string) {
	change := Change{
		Path:     path,
		Type:     changeType,
		Category: category,
		Severity: d.severityConditional(condition, severityIfTrue, severityIfFalse),
		OldValue: oldValue,
		NewValue: newValue,
		Message:  message,
	}
	d.populateChangeLocation(&change, changeType)
	result.Changes = append(result.Changes, change)
}

// ============================================================================
// Phase 2: Non-schema functions (unified implementations)
// ============================================================================

// diffExtrasUnified compares Extra maps (specification extensions with x- prefix)
func (d *Differ) diffExtrasUnified(source, target map[string]any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed extensions
	for key, sourceValue := range source {
		targetValue, exists := target[key]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, key), ChangeTypeRemoved, CategoryExtension,
				SeverityInfo, sourceValue, nil, fmt.Sprintf("extension %q removed", key))
			continue
		}

		// Check if value changed
		if !reflect.DeepEqual(sourceValue, targetValue) {
			d.addChange(result, fmt.Sprintf("%s.%s", path, key), ChangeTypeModified, CategoryExtension,
				SeverityInfo, sourceValue, targetValue, fmt.Sprintf("extension %q modified", key))
		}
	}

	// Find added extensions
	for key, targetValue := range target {
		if _, exists := source[key]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, key), ChangeTypeAdded, CategoryExtension,
				SeverityInfo, nil, targetValue, fmt.Sprintf("extension %q added", key))
		}
	}
}

// diffStringSlicesUnified compares string slices with mode-appropriate severity
func (d *Differ) diffStringSlicesUnified(source, target []string, path string, category ChangeCategory, itemName string, result *DiffResult) {
	sourceMap := make(map[string]struct{})
	for _, item := range source {
		sourceMap[item] = struct{}{}
	}

	targetMap := make(map[string]struct{})
	for _, item := range target {
		targetMap[item] = struct{}{}
	}

	// Find removed items
	for item := range sourceMap {
		if _, ok := targetMap[item]; !ok {
			d.addChange(result, path, ChangeTypeRemoved, category,
				SeverityWarning, item, nil, fmt.Sprintf("%s %q removed", itemName, item))
		}
	}

	// Find added items
	for item := range targetMap {
		if _, ok := sourceMap[item]; !ok {
			d.addChange(result, path, ChangeTypeAdded, category,
				SeverityInfo, nil, item, fmt.Sprintf("%s %q added", itemName, item))
		}
	}
}

// diffTagUnified compares individual Tag objects
func (d *Differ) diffTagUnified(source, target *parser.Tag, path string, result *DiffResult) {
	if source.Description != target.Description {
		d.addChange(result, path+".description", ChangeTypeModified, CategoryInfo,
			SeverityInfo, source.Description, target.Description, "tag description changed")
	}

	// Compare Tag extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffTagsUnified compares Tag slices
func (d *Differ) diffTagsUnified(source, target []*parser.Tag, path string, result *DiffResult) {
	sourceMap := make(map[string]*parser.Tag)
	for _, tag := range source {
		sourceMap[tag.Name] = tag
	}

	targetMap := make(map[string]*parser.Tag)
	for _, tag := range target {
		targetMap[tag.Name] = tag
	}

	// Find removed tags
	for name, sourceTag := range sourceMap {
		targetTag, exists := targetMap[name]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s[%s]", path, name), ChangeTypeRemoved, CategoryInfo,
				SeverityInfo, nil, nil, fmt.Sprintf("tag %q removed", name))
			continue
		}

		// Compare tag details
		d.diffTagUnified(sourceTag, targetTag, fmt.Sprintf("%s[%s]", path, name), result)
	}

	// Find added tags
	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s[%s]", path, name), ChangeTypeAdded, CategoryInfo,
				SeverityInfo, nil, nil, fmt.Sprintf("tag %q added", name))
		}
	}
}

// diffServerUnified compares individual Server objects
func (d *Differ) diffServerUnified(source, target *parser.Server, path string, result *DiffResult) {
	// In simple mode, always report description changes
	// In breaking mode, only report if source had a description
	if source.Description != target.Description {
		if d.Mode == ModeSimple || source.Description != "" {
			d.addChange(result, path+".description", ChangeTypeModified, CategoryServer,
				SeverityInfo, source.Description, target.Description, "server description changed")
		}
	}

	// Compare Server extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffServersUnified compares Server slices (OAS 3.x)
func (d *Differ) diffServersUnified(source, target []*parser.Server, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Build maps by URL for easier comparison
	sourceMap := make(map[string]*parser.Server)
	for _, srv := range source {
		sourceMap[srv.URL] = srv
	}

	targetMap := make(map[string]*parser.Server)
	for _, srv := range target {
		targetMap[srv.URL] = srv
	}

	// Find removed servers
	for url, sourceSrv := range sourceMap {
		targetSrv, exists := targetMap[url]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s[%s]", path, url), ChangeTypeRemoved, CategoryServer,
				SeverityWarning, url, nil, fmt.Sprintf("server %q removed", url))
			continue
		}

		// Compare server details if both exist
		d.diffServerUnified(sourceSrv, targetSrv, fmt.Sprintf("%s[%s]", path, url), result)
	}

	// Find added servers
	for url := range targetMap {
		if _, exists := sourceMap[url]; !exists {
			d.addChange(result, fmt.Sprintf("%s[%s]", path, url), ChangeTypeAdded, CategoryServer,
				SeverityInfo, nil, url, fmt.Sprintf("server %q added", url))
		}
	}
}

// diffInfoUnified compares Info objects
func (d *Differ) diffInfoUnified(source, target *parser.Info, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, path, ChangeTypeAdded, CategoryInfo,
			SeverityInfo, nil, target, "info object added")
		return
	}

	if target == nil {
		d.addChange(result, path, ChangeTypeRemoved, CategoryInfo,
			SeverityInfo, source, nil, "info object removed")
		return
	}

	// Compare fields
	if source.Title != target.Title {
		d.addChange(result, path+".title", ChangeTypeModified, CategoryInfo,
			SeverityInfo, source.Title, target.Title, fmt.Sprintf("title changed from %q to %q", source.Title, target.Title))
	}

	if source.Version != target.Version {
		d.addChange(result, path+".version", ChangeTypeModified, CategoryInfo,
			SeverityInfo, source.Version, target.Version, fmt.Sprintf("API version changed from %q to %q", source.Version, target.Version))
	}

	// In simple mode, always report description changes
	// In breaking mode, only report if source had a description
	if source.Description != target.Description {
		if d.Mode == ModeSimple || source.Description != "" {
			d.addChange(result, path+".description", ChangeTypeModified, CategoryInfo,
				SeverityInfo, source.Description, target.Description, "description changed")
		}
	}

	// Compare Info extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffSecuritySchemeUnified compares individual SecurityScheme objects
func (d *Differ) diffSecuritySchemeUnified(source, target *parser.SecurityScheme, path string, result *DiffResult) {
	if source.Type != target.Type {
		d.addChange(result, path+".type", ChangeTypeModified, CategorySecurity,
			SeverityError, source.Type, target.Type, fmt.Sprintf("security scheme type changed from %q to %q", source.Type, target.Type))
	}

	// Compare SecurityScheme extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffSecuritySchemesUnified compares security scheme maps
func (d *Differ) diffSecuritySchemesUnified(source, target map[string]*parser.SecurityScheme, path string, result *DiffResult) {
	// Find removed schemes
	for name, sourceScheme := range source {
		targetScheme, exists := target[name]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeRemoved, CategorySecurity,
				SeverityError, nil, nil, fmt.Sprintf("security scheme %q removed", name))
			continue
		}

		// Compare security scheme details
		d.diffSecuritySchemeUnified(sourceScheme, targetScheme, fmt.Sprintf("%s.%s", path, name), result)
	}

	// Find added schemes
	for name := range target {
		if _, exists := source[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeAdded, CategorySecurity,
				SeverityWarning, nil, nil, fmt.Sprintf("security scheme %q added", name))
		}
	}
}

// diffWebhooksUnified compares webhook maps (OAS 3.1+)
func (d *Differ) diffWebhooksUnified(source, target map[string]*parser.PathItem, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed webhooks
	for name := range source {
		if _, exists := target[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeRemoved, CategoryEndpoint,
				SeverityError, nil, nil, fmt.Sprintf("webhook %q removed", name))
		}
	}

	// Find added webhooks
	for name := range target {
		if _, exists := source[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeAdded, CategoryEndpoint,
				SeverityInfo, nil, nil, fmt.Sprintf("webhook %q added", name))
		}
	}
}

// ============================================================================
// Phase 3: Response functions (unified implementations)
// ============================================================================

// diffLinkUnified compares individual Link objects
func (d *Differ) diffLinkUnified(source, target *parser.Link, path string, result *DiffResult) {
	if source.OperationRef != target.OperationRef {
		d.addChange(result, path+".operationRef", ChangeTypeModified, CategoryResponse,
			SeverityWarning, source.OperationRef, target.OperationRef, "operationRef changed")
	}

	if source.OperationID != target.OperationID {
		d.addChange(result, path+".operationId", ChangeTypeModified, CategoryResponse,
			SeverityWarning, source.OperationID, target.OperationID, "operationId changed")
	}

	if source.Description != target.Description {
		d.addChange(result, path+".description", ChangeTypeModified, CategoryResponse,
			SeverityInfo, source.Description, target.Description, "description changed")
	}

	// Compare Link extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffResponseLinksUnified compares response link maps
func (d *Differ) diffResponseLinksUnified(source, target map[string]*parser.Link, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed links
	for name := range source {
		if _, exists := target[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.links.%s", path, name), ChangeTypeRemoved, CategoryResponse,
				SeverityWarning, nil, nil, fmt.Sprintf("response link %q removed", name))
		}
	}

	// Find added or modified links
	for name, targetLink := range target {
		sourceLink, exists := source[name]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.links.%s", path, name), ChangeTypeAdded, CategoryResponse,
				SeverityInfo, nil, nil, fmt.Sprintf("response link %q added", name))
			continue
		}

		// Compare link details
		d.diffLinkUnified(sourceLink, targetLink, fmt.Sprintf("%s.links.%s", path, name), result)
	}
}

// diffResponseExamplesUnified compares response example maps
func (d *Differ) diffResponseExamplesUnified(source, target map[string]any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed examples
	for name := range source {
		if _, exists := target[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.examples.%s", path, name), ChangeTypeRemoved, CategoryResponse,
				SeverityInfo, nil, nil, fmt.Sprintf("response example %q removed", name))
		}
	}

	// Find added examples (we don't deep-compare example values)
	for name := range target {
		if _, exists := source[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.examples.%s", path, name), ChangeTypeAdded, CategoryResponse,
				SeverityInfo, nil, nil, fmt.Sprintf("response example %q added", name))
		}
	}
}

// diffHeaderUnified compares individual Header objects
func (d *Differ) diffHeaderUnified(source, target *parser.Header, path string, result *DiffResult) {
	// Description change (simple only reports it always, breaking only if source had one)
	if source.Description != target.Description {
		if d.Mode == ModeSimple || source.Description != "" {
			d.addChange(result, path+".description", ChangeTypeModified, CategoryResponse,
				SeverityInfo, source.Description, target.Description, "header description changed")
		}
	}

	// Required changed
	if source.Required != target.Required {
		// Making a header required is an error, making it optional is info
		d.addChangeConditional(result, path+".required", ChangeTypeModified, CategoryResponse,
			!source.Required && target.Required, SeverityError, SeverityWarning,
			source.Required, target.Required, fmt.Sprintf("required changed from %v to %v", source.Required, target.Required))
	}

	// Deprecated changed (simple mode only)
	if d.Mode == ModeSimple && source.Deprecated != target.Deprecated {
		d.addChange(result, path+".deprecated", ChangeTypeModified, CategoryResponse,
			SeverityInfo, source.Deprecated, target.Deprecated, fmt.Sprintf("deprecated changed from %v to %v", source.Deprecated, target.Deprecated))
	}

	// Type changed
	if source.Type != target.Type {
		d.addChange(result, path+".type", ChangeTypeModified, CategoryResponse,
			SeverityWarning, source.Type, target.Type, fmt.Sprintf("type changed from %q to %q", source.Type, target.Type))
	}

	// Style changed (simple mode only)
	if d.Mode == ModeSimple && source.Style != target.Style {
		d.addChange(result, path+".style", ChangeTypeModified, CategoryResponse,
			SeverityInfo, source.Style, target.Style, fmt.Sprintf("style changed from %q to %q", source.Style, target.Style))
	}

	// Compare Header extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffResponseHeadersUnified compares header maps
func (d *Differ) diffResponseHeadersUnified(source, target map[string]*parser.Header, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed headers
	for name := range source {
		if _, exists := target[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.headers.%s", path, name), ChangeTypeRemoved, CategoryResponse,
				SeverityWarning, nil, nil, fmt.Sprintf("response header %q removed", name))
		}
	}

	// Find added or modified headers
	for name, targetHeader := range target {
		sourceHeader, exists := source[name]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.headers.%s", path, name), ChangeTypeAdded, CategoryResponse,
				SeverityInfo, nil, nil, fmt.Sprintf("response header %q added", name))
			continue
		}

		// Compare header details
		d.diffHeaderUnified(sourceHeader, targetHeader, fmt.Sprintf("%s.headers.%s", path, name), result)
	}
}

// diffMediaTypeUnified compares individual MediaType objects
func (d *Differ) diffMediaTypeUnified(source, target *parser.MediaType, path string, result *DiffResult) {
	// Compare schemas if present
	if source.Schema != nil && target.Schema != nil {
		d.diffSchemaUnified(source.Schema, target.Schema, path+".schema", result)
	} else if source.Schema != nil && target.Schema == nil {
		d.addChange(result, path+".schema", ChangeTypeRemoved, CategoryResponse,
			SeverityWarning, nil, nil, "schema removed")
	} else if source.Schema == nil && target.Schema != nil {
		d.addChange(result, path+".schema", ChangeTypeAdded, CategoryResponse,
			SeverityInfo, nil, nil, "schema added")
	}

	// Compare MediaType extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffResponseContentUnified compares response content maps
func (d *Differ) diffResponseContentUnified(source, target map[string]*parser.MediaType, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed media types
	for mediaType := range source {
		if _, exists := target[mediaType]; !exists {
			d.addChange(result, fmt.Sprintf("%s.content.%s", path, mediaType), ChangeTypeRemoved, CategoryResponse,
				SeverityWarning, nil, nil, fmt.Sprintf("response media type %q removed", mediaType))
		}
	}

	// Find added or modified media types
	for mediaType, targetMedia := range target {
		sourceMedia, exists := source[mediaType]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.content.%s", path, mediaType), ChangeTypeAdded, CategoryResponse,
				SeverityInfo, nil, nil, fmt.Sprintf("response media type %q added", mediaType))
			continue
		}

		// Compare media type details
		d.diffMediaTypeUnified(sourceMedia, targetMedia, fmt.Sprintf("%s.content.%s", path, mediaType), result)
	}
}

// diffResponseUnified compares individual Response objects
func (d *Differ) diffResponseUnified(source, target *parser.Response, path string, result *DiffResult) {
	// Description change (simple mode always reports, breaking only if source had one)
	if source.Description != target.Description {
		if d.Mode == ModeSimple || source.Description != "" {
			d.addChange(result, path+".description", ChangeTypeModified, CategoryResponse,
				SeverityInfo, source.Description, target.Description, "response description changed")
		}
	}

	// Compare Response headers
	d.diffResponseHeadersUnified(source.Headers, target.Headers, path, result)

	// Compare Response content
	d.diffResponseContentUnified(source.Content, target.Content, path, result)

	// Compare Response links
	d.diffResponseLinksUnified(source.Links, target.Links, path, result)

	// Compare Response examples
	d.diffResponseExamplesUnified(source.Examples, target.Examples, path, result)

	// Compare Response extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffResponsesUnified compares Responses objects
func (d *Differ) diffResponsesUnified(source, target *parser.Responses, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, path, ChangeTypeAdded, CategoryResponse,
			SeverityInfo, nil, target, "responses added")
		return
	}

	if target == nil {
		d.addChange(result, path, ChangeTypeRemoved, CategoryResponse,
			SeverityError, source, nil, "responses removed")
		return
	}

	// Compare individual response codes
	for code, sourceResp := range source.Codes {
		targetResp, exists := target.Codes[code]
		if !exists {
			// Severity depends on whether this is a success code
			severity := SeverityWarning
			if d.Mode == ModeBreaking && isSuccessCode(code) {
				severity = SeverityError
			}
			d.addChange(result, fmt.Sprintf("%s[%s]", path, code), ChangeTypeRemoved, CategoryResponse,
				severity, sourceResp, nil, fmt.Sprintf("response code %s removed", code))
			continue
		}

		// Compare response details
		d.diffResponseUnified(sourceResp, targetResp, fmt.Sprintf("%s[%s]", path, code), result)
	}

	// Find added response codes
	for code, targetResp := range target.Codes {
		if _, exists := source.Codes[code]; !exists {
			// New error codes might indicate new failure modes
			severity := SeverityInfo
			if d.Mode == ModeBreaking && isErrorCode(code) {
				severity = SeverityWarning
			}
			d.addChange(result, fmt.Sprintf("%s[%s]", path, code), ChangeTypeAdded, CategoryResponse,
				severity, nil, targetResp, fmt.Sprintf("response code %s added", code))
		}
	}
}

// ============================================================================
// Phase 4: Schema functions (unified implementations)
// ============================================================================

// diffSchemasUnified compares schema maps
func (d *Differ) diffSchemasUnified(source, target map[string]*parser.Schema, path string, result *DiffResult) {
	// Find removed schemas
	for name, sourceSchema := range source {
		targetSchema, exists := target[name]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeRemoved, CategorySchema,
				SeverityError, nil, nil, fmt.Sprintf("schema %q removed", name))
			continue
		}

		// Compare schema details
		d.diffSchemaUnified(sourceSchema, targetSchema, fmt.Sprintf("%s.%s", path, name), result)
	}

	// Find added schemas
	for name := range target {
		if _, exists := source[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, nil, fmt.Sprintf("schema %q added", name))
		}
	}
}

// diffSchemaUnified compares individual Schema objects
func (d *Differ) diffSchemaUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// Use recursive diffing with cycle detection
	visited := newSchemaVisited()
	d.diffSchemaRecursiveUnified(source, target, path, visited, result)
}

// diffSchemaRecursiveUnified performs recursive schema comparison with cycle detection
func (d *Differ) diffSchemaRecursiveUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	// Nil handling
	if source == nil && target == nil {
		return
	}
	if source == nil {
		d.addChange(result, path, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, target, "schema added")
		return
	}
	if target == nil {
		d.addChange(result, path, ChangeTypeRemoved, CategorySchema,
			SeverityError, source, nil, "schema removed")
		return
	}

	// Cycle detection
	if visited.enter(source, target, path) {
		return
	}
	defer visited.leave(source, target)

	// Compare metadata
	d.diffSchemaMetadataUnified(source, target, path, result)

	// Compare type and format
	d.diffSchemaTypeUnified(source, target, path, result)

	// Compare constraints
	d.diffSchemaNumericConstraintsUnified(source, target, path, result)
	d.diffSchemaStringConstraintsUnified(source, target, path, result)
	d.diffSchemaArrayConstraintsUnified(source, target, path, result)
	d.diffSchemaObjectConstraintsUnified(source, target, path, result)

	// Compare required fields
	d.diffSchemaRequiredFieldsUnified(source, target, path, result)

	// Compare OAS-specific fields
	d.diffSchemaOASFieldsUnified(source, target, path, result)

	// Compare enum values
	d.diffEnumUnified(source.Enum, target.Enum, path+".enum", result)

	// Compare recursive/complex fields
	d.diffSchemaPropertiesUnified(source.Properties, target.Properties, source.Required, target.Required, path, visited, result)
	d.diffSchemaItemsUnified(source.Items, target.Items, path, visited, result)
	d.diffSchemaAdditionalPropertiesUnified(source.AdditionalProperties, target.AdditionalProperties, path, visited, result)

	// Compare composition fields
	d.diffSchemaAllOfUnified(source.AllOf, target.AllOf, path, visited, result)
	d.diffSchemaAnyOfUnified(source.AnyOf, target.AnyOf, path, visited, result)
	d.diffSchemaOneOfUnified(source.OneOf, target.OneOf, path, visited, result)
	d.diffSchemaNotUnified(source.Not, target.Not, path, visited, result)

	// Compare conditional schemas
	d.diffSchemaConditionalUnified(source.If, source.Then, source.Else, target.If, target.Then, target.Else, path, visited, result)

	// JSON Schema 2020-12 fields
	d.diffSchemaUnevaluatedPropertiesUnified(source.UnevaluatedProperties, target.UnevaluatedProperties, path, visited, result)
	d.diffSchemaUnevaluatedItemsUnified(source.UnevaluatedItems, target.UnevaluatedItems, path, visited, result)
	d.diffSchemaContentFieldsUnified(source, target, path, visited, result)
	d.diffSchemaPrefixItemsUnified(source.PrefixItems, target.PrefixItems, path, visited, result)
	d.diffSchemaContainsUnified(source.Contains, target.Contains, path, visited, result)
	d.diffSchemaPropertyNamesUnified(source.PropertyNames, target.PropertyNames, path, visited, result)
	d.diffSchemaDependentSchemasUnified(source.DependentSchemas, target.DependentSchemas, path, visited, result)

	// Compare extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffSchemaMetadataUnified compares schema metadata fields
func (d *Differ) diffSchemaMetadataUnified(source, target *parser.Schema, path string, result *DiffResult) {
	if source.Title != target.Title {
		d.addChange(result, path+".title", ChangeTypeModified, CategorySchema,
			SeverityInfo, source.Title, target.Title, "schema title changed")
	}

	if source.Description != target.Description {
		d.addChange(result, path+".description", ChangeTypeModified, CategorySchema,
			SeverityInfo, source.Description, target.Description, "schema description changed")
	}
}

// diffSchemaTypeUnified compares schema type and format fields
func (d *Differ) diffSchemaTypeUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// Type can be string or []string in OAS 3.1+
	sourceTypeStr := formatSchemaType(source.Type)
	targetTypeStr := formatSchemaType(target.Type)
	if sourceTypeStr != targetTypeStr {
		d.addChange(result, path+".type", ChangeTypeModified, CategorySchema,
			SeverityError, source.Type, target.Type, "schema type changed")
	}

	if source.Format != target.Format {
		d.addChange(result, path+".format", ChangeTypeModified, CategorySchema,
			SeverityWarning, source.Format, target.Format, "schema format changed")
	}
}

// diffSchemaNumericConstraintsUnified compares numeric validation constraints
func (d *Differ) diffSchemaNumericConstraintsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// MultipleOf
	if source.MultipleOf != nil && target.MultipleOf != nil && *source.MultipleOf != *target.MultipleOf {
		d.addChange(result, path+".multipleOf", ChangeTypeModified, CategorySchema,
			SeverityWarning, *source.MultipleOf, *target.MultipleOf, "multipleOf constraint changed")
	}

	// Maximum
	if source.Maximum != nil && target.Maximum != nil && *source.Maximum != *target.Maximum {
		// Tightening (lowering max) is error, relaxing is warning
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.Maximum < *source.Maximum {
			severity = SeverityError
		}
		d.addChange(result, path+".maximum", ChangeTypeModified, CategorySchema,
			severity, *source.Maximum, *target.Maximum, "maximum constraint changed")
	} else if source.Maximum == nil && target.Maximum != nil {
		d.addChange(result, path+".maximum", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.Maximum, "maximum constraint added")
	}

	// Minimum
	if source.Minimum != nil && target.Minimum != nil && *source.Minimum != *target.Minimum {
		// Tightening (raising min) is error, relaxing is warning
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.Minimum > *source.Minimum {
			severity = SeverityError
		}
		d.addChange(result, path+".minimum", ChangeTypeModified, CategorySchema,
			severity, *source.Minimum, *target.Minimum, "minimum constraint changed")
	} else if source.Minimum == nil && target.Minimum != nil {
		d.addChange(result, path+".minimum", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.Minimum, "minimum constraint added")
	}
}

// diffSchemaStringConstraintsUnified compares string validation constraints
func (d *Differ) diffSchemaStringConstraintsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxLength
	if source.MaxLength != nil && target.MaxLength != nil && *source.MaxLength != *target.MaxLength {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MaxLength < *source.MaxLength {
			severity = SeverityError
		}
		d.addChange(result, path+".maxLength", ChangeTypeModified, CategorySchema,
			severity, *source.MaxLength, *target.MaxLength, "maxLength constraint changed")
	} else if source.MaxLength == nil && target.MaxLength != nil {
		d.addChange(result, path+".maxLength", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MaxLength, "maxLength constraint added")
	}

	// MinLength
	if source.MinLength != nil && target.MinLength != nil && *source.MinLength != *target.MinLength {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MinLength > *source.MinLength {
			severity = SeverityError
		}
		d.addChange(result, path+".minLength", ChangeTypeModified, CategorySchema,
			severity, *source.MinLength, *target.MinLength, "minLength constraint changed")
	} else if source.MinLength == nil && target.MinLength != nil {
		d.addChange(result, path+".minLength", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MinLength, "minLength constraint added")
	}

	// Pattern
	if source.Pattern != target.Pattern {
		if source.Pattern != "" || target.Pattern != "" {
			severity := SeverityWarning
			if d.Mode == ModeBreaking && source.Pattern == "" && target.Pattern != "" {
				severity = SeverityError
			}
			d.addChange(result, path+".pattern", ChangeTypeModified, CategorySchema,
				severity, source.Pattern, target.Pattern, "pattern constraint changed")
		}
	}
}

// diffSchemaArrayConstraintsUnified compares array validation constraints
func (d *Differ) diffSchemaArrayConstraintsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxItems
	if source.MaxItems != nil && target.MaxItems != nil && *source.MaxItems != *target.MaxItems {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MaxItems < *source.MaxItems {
			severity = SeverityError
		}
		d.addChange(result, path+".maxItems", ChangeTypeModified, CategorySchema,
			severity, *source.MaxItems, *target.MaxItems, "maxItems constraint changed")
	} else if source.MaxItems == nil && target.MaxItems != nil {
		d.addChange(result, path+".maxItems", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MaxItems, "maxItems constraint added")
	}

	// MinItems
	if source.MinItems != nil && target.MinItems != nil && *source.MinItems != *target.MinItems {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MinItems > *source.MinItems {
			severity = SeverityError
		}
		d.addChange(result, path+".minItems", ChangeTypeModified, CategorySchema,
			severity, *source.MinItems, *target.MinItems, "minItems constraint changed")
	} else if source.MinItems == nil && target.MinItems != nil {
		d.addChange(result, path+".minItems", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MinItems, "minItems constraint added")
	}

	// UniqueItems
	if source.UniqueItems != target.UniqueItems {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && !source.UniqueItems && target.UniqueItems {
			severity = SeverityError
		}
		d.addChange(result, path+".uniqueItems", ChangeTypeModified, CategorySchema,
			severity, source.UniqueItems, target.UniqueItems, "uniqueItems constraint changed")
	}
}

// diffSchemaObjectConstraintsUnified compares object validation constraints
func (d *Differ) diffSchemaObjectConstraintsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// MaxProperties
	if source.MaxProperties != nil && target.MaxProperties != nil && *source.MaxProperties != *target.MaxProperties {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MaxProperties < *source.MaxProperties {
			severity = SeverityError
		}
		d.addChange(result, path+".maxProperties", ChangeTypeModified, CategorySchema,
			severity, *source.MaxProperties, *target.MaxProperties, "maxProperties constraint changed")
	} else if source.MaxProperties == nil && target.MaxProperties != nil {
		d.addChange(result, path+".maxProperties", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MaxProperties, "maxProperties constraint added")
	}

	// MinProperties
	if source.MinProperties != nil && target.MinProperties != nil && *source.MinProperties != *target.MinProperties {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && *target.MinProperties > *source.MinProperties {
			severity = SeverityError
		}
		d.addChange(result, path+".minProperties", ChangeTypeModified, CategorySchema,
			severity, *source.MinProperties, *target.MinProperties, "minProperties constraint changed")
	} else if source.MinProperties == nil && target.MinProperties != nil {
		d.addChange(result, path+".minProperties", ChangeTypeAdded, CategorySchema,
			SeverityError, nil, *target.MinProperties, "minProperties constraint added")
	}
}

// diffSchemaRequiredFieldsUnified compares required field lists
func (d *Differ) diffSchemaRequiredFieldsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	sourceRequired := make(map[string]bool)
	for _, req := range source.Required {
		sourceRequired[req] = true
	}
	targetRequired := make(map[string]bool)
	for _, req := range target.Required {
		targetRequired[req] = true
	}

	// Removed required fields - relaxing
	for req := range sourceRequired {
		if !targetRequired[req] {
			d.addChange(result, fmt.Sprintf("%s.required[%s]", path, req), ChangeTypeRemoved, CategorySchema,
				SeverityInfo, nil, nil, fmt.Sprintf("required field %q removed", req))
		}
	}

	// Added required fields - stricter
	for req := range targetRequired {
		if !sourceRequired[req] {
			d.addChange(result, fmt.Sprintf("%s.required[%s]", path, req), ChangeTypeAdded, CategorySchema,
				SeverityError, nil, nil, fmt.Sprintf("required field %q added", req))
		}
	}
}

// diffSchemaOASFieldsUnified compares OAS-specific schema fields
func (d *Differ) diffSchemaOASFieldsUnified(source, target *parser.Schema, path string, result *DiffResult) {
	// Nullable
	if source.Nullable != target.Nullable {
		// Removing nullable is breaking (was accepting null, now not)
		severity := SeverityWarning
		if d.Mode == ModeBreaking && source.Nullable && !target.Nullable {
			severity = SeverityError
		}
		d.addChange(result, path+".nullable", ChangeTypeModified, CategorySchema,
			severity, source.Nullable, target.Nullable, "nullable changed")
	}

	// ReadOnly
	if source.ReadOnly != target.ReadOnly {
		d.addChange(result, path+".readOnly", ChangeTypeModified, CategorySchema,
			SeverityWarning, source.ReadOnly, target.ReadOnly, "readOnly changed")
	}

	// WriteOnly
	if source.WriteOnly != target.WriteOnly {
		d.addChange(result, path+".writeOnly", ChangeTypeModified, CategorySchema,
			SeverityWarning, source.WriteOnly, target.WriteOnly, "writeOnly changed")
	}

	// Deprecated
	if source.Deprecated != target.Deprecated {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && !source.Deprecated && target.Deprecated {
			severity = SeverityWarning
		}
		d.addChange(result, path+".deprecated", ChangeTypeModified, CategorySchema,
			severity, source.Deprecated, target.Deprecated, "deprecated status changed")
	}
}

// diffEnumUnified compares enum values
func (d *Differ) diffEnumUnified(source, target []any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	sourceMap := make(map[string]struct{})
	for _, val := range source {
		sourceMap[anyToString(val)] = struct{}{}
	}

	targetMap := make(map[string]struct{})
	for _, val := range target {
		targetMap[anyToString(val)] = struct{}{}
	}

	// Removed enum values - restricts valid values
	for val := range sourceMap {
		if _, ok := targetMap[val]; !ok {
			d.addChange(result, path, ChangeTypeRemoved, CategoryParameter,
				SeverityError, nil, nil, fmt.Sprintf("enum value %q removed", val))
		}
	}

	// Added enum values - expands valid values
	for val := range targetMap {
		if _, ok := sourceMap[val]; !ok {
			d.addChange(result, path, ChangeTypeAdded, CategoryParameter,
				SeverityInfo, nil, nil, fmt.Sprintf("enum value %q added", val))
		}
	}
}

// diffSchemaPropertiesUnified compares schema properties maps
func (d *Differ) diffSchemaPropertiesUnified(source, target map[string]*parser.Schema, sourceRequired, targetRequired []string, path string, visited *schemaVisited, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed properties
	for name, sourceSchema := range source {
		propPath := fmt.Sprintf("%s.properties.%s", path, name)
		if targetSchema, exists := target[name]; !exists {
			// Severity depends on whether it was required
			severity := SeverityWarning
			if d.Mode == ModeBreaking && isPropertyRequired(name, sourceRequired) {
				severity = SeverityError
			}
			d.addChange(result, propPath, ChangeTypeRemoved, CategorySchema,
				severity, sourceSchema, nil, fmt.Sprintf("property %q removed", name))
		} else {
			// Property exists in both - recursive comparison
			d.diffSchemaRecursiveUnified(sourceSchema, targetSchema, propPath, visited, result)
		}
	}

	// Find added properties
	for name, targetSchema := range target {
		if _, exists := source[name]; !exists {
			propPath := fmt.Sprintf("%s.properties.%s", path, name)
			// Severity depends on whether it's required
			severity := SeverityInfo
			if d.Mode == ModeBreaking && isPropertyRequired(name, targetRequired) {
				severity = SeverityWarning
			}
			d.addChange(result, propPath, ChangeTypeAdded, CategorySchema,
				severity, nil, targetSchema, fmt.Sprintf("property %q added", name))
		}
	}
}

// diffSchemaItemsUnified compares schema Items field
func (d *Differ) diffSchemaItemsUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaItemsType(source)
	targetType := getSchemaItemsType(target)
	itemsPath := path + ".items"

	// Handle unknown types
	if sourceType == schemaItemsTypeUnknown && targetType == schemaItemsTypeUnknown {
		return
	}
	if sourceType == schemaItemsTypeUnknown {
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("items has unexpected type in source: %T", source))
		return
	}
	if targetType == schemaItemsTypeUnknown {
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("items has unexpected type in target: %T", target))
		return
	}

	// Both nil
	if sourceType == schemaItemsTypeNil && targetType == schemaItemsTypeNil {
		return
	}

	// Items added
	if sourceType == schemaItemsTypeNil && targetType != schemaItemsTypeNil {
		d.addChange(result, itemsPath, ChangeTypeAdded, CategorySchema,
			SeverityWarning, nil, target, "items schema added")
		return
	}

	// Items removed
	if sourceType != schemaItemsTypeNil && targetType == schemaItemsTypeNil {
		d.addChange(result, itemsPath, ChangeTypeRemoved, CategorySchema,
			SeverityError, source, nil, "items schema removed")
		return
	}

	// Type changed
	if sourceType != targetType {
		severity := SeverityError
		if sourceType == schemaItemsTypeBool && targetType == schemaItemsTypeSchema {
			severity = SeverityWarning
		}
		d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
			severity, source, target, "items type changed")
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaItemsTypeSchema:
		sourceSchema := source.(*parser.Schema)
		targetSchema := target.(*parser.Schema)
		d.diffSchemaRecursiveUnified(sourceSchema, targetSchema, itemsPath, visited, result)
	case schemaItemsTypeBool:
		sourceBool := source.(bool)
		targetBool := target.(bool)
		if sourceBool != targetBool {
			severity := SeverityWarning
			if d.Mode == ModeBreaking && sourceBool && !targetBool {
				severity = SeverityError
			}
			d.addChange(result, itemsPath, ChangeTypeModified, CategorySchema,
				severity, sourceBool, targetBool, fmt.Sprintf("items changed from %v to %v", sourceBool, targetBool))
		}
	case schemaItemsTypeNil, schemaItemsTypeUnknown:
		// Already handled above before the switch
	}
}

// diffSchemaAdditionalPropertiesUnified compares additionalProperties field
func (d *Differ) diffSchemaAdditionalPropertiesUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaAdditionalPropsType(source)
	targetType := getSchemaAdditionalPropsType(target)
	addPropsPath := path + ".additionalProperties"

	// Handle unknown types
	if sourceType == schemaAdditionalPropsTypeUnknown && targetType == schemaAdditionalPropsTypeUnknown {
		return
	}
	if sourceType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("additionalProperties has unexpected type in source: %T", source))
		return
	}
	if targetType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("additionalProperties has unexpected type in target: %T", target))
		return
	}

	// Both nil
	if sourceType == schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		return
	}

	// additionalProperties added
	if sourceType == schemaAdditionalPropsTypeNil && targetType != schemaAdditionalPropsTypeNil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && targetType == schemaAdditionalPropsTypeBool && !target.(bool) {
			severity = SeverityError
		}
		d.addChange(result, addPropsPath, ChangeTypeAdded, CategorySchema,
			severity, nil, target, "additionalProperties constraint added")
		return
	}

	// additionalProperties removed
	if sourceType != schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && sourceType == schemaAdditionalPropsTypeBool && !source.(bool) {
			severity = SeverityInfo
		}
		d.addChange(result, addPropsPath, ChangeTypeRemoved, CategorySchema,
			severity, source, nil, "additionalProperties constraint removed")
		return
	}

	// Type changed
	if sourceType != targetType {
		d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, target, "additionalProperties type changed")
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaAdditionalPropsTypeSchema:
		sourceSchema := source.(*parser.Schema)
		targetSchema := target.(*parser.Schema)
		d.diffSchemaRecursiveUnified(sourceSchema, targetSchema, addPropsPath, visited, result)
	case schemaAdditionalPropsTypeBool:
		sourceBool := source.(bool)
		targetBool := target.(bool)
		if sourceBool != targetBool {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && sourceBool && !targetBool {
				severity = SeverityError
			}
			d.addChange(result, addPropsPath, ChangeTypeModified, CategorySchema,
				severity, sourceBool, targetBool, fmt.Sprintf("additionalProperties changed from %v to %v", sourceBool, targetBool))
		}
	case schemaAdditionalPropsTypeNil, schemaAdditionalPropsTypeUnknown:
		// Already handled above before the switch
	}
}

// diffSchemaAllOfUnified compares allOf composition schemas
func (d *Differ) diffSchemaAllOfUnified(source, target []*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	allOfPath := path + ".allOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Compare by index
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", allOfPath, i)
		if i < len(target) {
			d.diffSchemaRecursiveUnified(sourceSchema, target[i], schemaPath, visited, result)
		} else {
			// Schema removed - relaxes validation
			d.addChange(result, schemaPath, ChangeTypeRemoved, CategorySchema,
				SeverityInfo, sourceSchema, nil, fmt.Sprintf("allOf schema at index %d removed", i))
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", allOfPath, i)
		// Adding makes validation stricter
		d.addChange(result, schemaPath, ChangeTypeAdded, CategorySchema,
			SeverityError, nil, target[i], fmt.Sprintf("allOf schema at index %d added", i))
	}
}

// diffSchemaAnyOfUnified compares anyOf composition schemas
func (d *Differ) diffSchemaAnyOfUnified(source, target []*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	anyOfPath := path + ".anyOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Compare by index
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", anyOfPath, i)
		if i < len(target) {
			d.diffSchemaRecursiveUnified(sourceSchema, target[i], schemaPath, visited, result)
		} else {
			// Removing reduces choices
			d.addChange(result, schemaPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceSchema, nil, fmt.Sprintf("anyOf schema at index %d removed", i))
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", anyOfPath, i)
		// Adding provides more choices
		d.addChange(result, schemaPath, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, target[i], fmt.Sprintf("anyOf schema at index %d added", i))
	}
}

// diffSchemaOneOfUnified compares oneOf composition schemas
func (d *Differ) diffSchemaOneOfUnified(source, target []*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	oneOfPath := path + ".oneOf"

	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Compare by index
	for i, sourceSchema := range source {
		schemaPath := fmt.Sprintf("%s[%d]", oneOfPath, i)
		if i < len(target) {
			d.diffSchemaRecursiveUnified(sourceSchema, target[i], schemaPath, visited, result)
		} else {
			// Changes exclusive validation
			d.addChange(result, schemaPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceSchema, nil, fmt.Sprintf("oneOf schema at index %d removed", i))
		}
	}

	// Find added schemas
	for i := len(source); i < len(target); i++ {
		schemaPath := fmt.Sprintf("%s[%d]", oneOfPath, i)
		// Changes exclusive validation
		d.addChange(result, schemaPath, ChangeTypeAdded, CategorySchema,
			SeverityWarning, nil, target[i], fmt.Sprintf("oneOf schema at index %d added", i))
	}
}

// diffSchemaNotUnified compares not schemas
func (d *Differ) diffSchemaNotUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	notPath := path + ".not"

	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, notPath, ChangeTypeAdded, CategorySchema,
			SeverityWarning, nil, target, "not schema added")
		return
	}

	if target == nil {
		d.addChange(result, notPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "not schema removed")
		return
	}

	d.diffSchemaRecursiveUnified(source, target, notPath, visited, result)
}

// diffSchemaConditionalUnified compares conditional schemas (if/then/else)
func (d *Differ) diffSchemaConditionalUnified(sourceIf, sourceThen, sourceElse, targetIf, targetThen, targetElse *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	// Compare if condition
	if sourceIf != nil || targetIf != nil {
		ifPath := path + ".if"
		if sourceIf == nil {
			d.addChange(result, ifPath, ChangeTypeAdded, CategorySchema,
				SeverityWarning, nil, targetIf, "conditional if schema added")
		} else if targetIf == nil {
			d.addChange(result, ifPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceIf, nil, "conditional if schema removed")
		} else {
			d.diffSchemaRecursiveUnified(sourceIf, targetIf, ifPath, visited, result)
		}
	}

	// Compare then branch
	if sourceThen != nil || targetThen != nil {
		thenPath := path + ".then"
		if sourceThen == nil {
			d.addChange(result, thenPath, ChangeTypeAdded, CategorySchema,
				SeverityWarning, nil, targetThen, "conditional then schema added")
		} else if targetThen == nil {
			d.addChange(result, thenPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceThen, nil, "conditional then schema removed")
		} else {
			d.diffSchemaRecursiveUnified(sourceThen, targetThen, thenPath, visited, result)
		}
	}

	// Compare else branch
	if sourceElse != nil || targetElse != nil {
		elsePath := path + ".else"
		if sourceElse == nil {
			d.addChange(result, elsePath, ChangeTypeAdded, CategorySchema,
				SeverityWarning, nil, targetElse, "conditional else schema added")
		} else if targetElse == nil {
			d.addChange(result, elsePath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, sourceElse, nil, "conditional else schema removed")
		} else {
			d.diffSchemaRecursiveUnified(sourceElse, targetElse, elsePath, visited, result)
		}
	}
}

// diffSchemaUnevaluatedPropertiesUnified compares unevaluatedProperties (JSON Schema 2020-12)
func (d *Differ) diffSchemaUnevaluatedPropertiesUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaAdditionalPropsType(source)
	targetType := getSchemaAdditionalPropsType(target)
	fieldPath := path + ".unevaluatedProperties"

	// Handle unknown types
	if sourceType == schemaAdditionalPropsTypeUnknown && targetType == schemaAdditionalPropsTypeUnknown {
		return
	}
	if sourceType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("unevaluatedProperties has unexpected type in source: %T", source))
		return
	}
	if targetType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("unevaluatedProperties has unexpected type in target: %T", target))
		return
	}

	// Both nil - no change
	if sourceType == schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		return
	}

	// Added
	if sourceType == schemaAdditionalPropsTypeNil && targetType != schemaAdditionalPropsTypeNil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && targetType == schemaAdditionalPropsTypeBool && !target.(bool) {
			severity = SeverityError
		}
		d.addChange(result, fieldPath, ChangeTypeAdded, CategorySchema,
			severity, nil, target, "unevaluatedProperties constraint added")
		return
	}

	// Removed
	if sourceType != schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		d.addChange(result, fieldPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "unevaluatedProperties constraint removed")
		return
	}

	// Type changed
	if sourceType != targetType {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, target, "unevaluatedProperties type changed")
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaAdditionalPropsTypeSchema:
		d.diffSchemaRecursiveUnified(source.(*parser.Schema), target.(*parser.Schema), fieldPath, visited, result)
	case schemaAdditionalPropsTypeBool:
		if source.(bool) != target.(bool) {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && source.(bool) && !target.(bool) {
				severity = SeverityError
			}
			d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
				severity, source, target, fmt.Sprintf("unevaluatedProperties changed from %v to %v", source, target))
		}
	}
}

// diffSchemaUnevaluatedItemsUnified compares unevaluatedItems (JSON Schema 2020-12)
func (d *Differ) diffSchemaUnevaluatedItemsUnified(source, target any, path string, visited *schemaVisited, result *DiffResult) {
	sourceType := getSchemaAdditionalPropsType(source)
	targetType := getSchemaAdditionalPropsType(target)
	fieldPath := path + ".unevaluatedItems"

	// Handle unknown types
	if sourceType == schemaAdditionalPropsTypeUnknown && targetType == schemaAdditionalPropsTypeUnknown {
		return
	}
	if sourceType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, nil, fmt.Sprintf("unevaluatedItems has unexpected type in source: %T", source))
		return
	}
	if targetType == schemaAdditionalPropsTypeUnknown {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, nil, target, fmt.Sprintf("unevaluatedItems has unexpected type in target: %T", target))
		return
	}

	// Both nil - no change
	if sourceType == schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		return
	}

	// Added
	if sourceType == schemaAdditionalPropsTypeNil && targetType != schemaAdditionalPropsTypeNil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && targetType == schemaAdditionalPropsTypeBool && !target.(bool) {
			severity = SeverityError
		}
		d.addChange(result, fieldPath, ChangeTypeAdded, CategorySchema,
			severity, nil, target, "unevaluatedItems constraint added")
		return
	}

	// Removed
	if sourceType != schemaAdditionalPropsTypeNil && targetType == schemaAdditionalPropsTypeNil {
		d.addChange(result, fieldPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "unevaluatedItems constraint removed")
		return
	}

	// Type changed
	if sourceType != targetType {
		d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
			SeverityWarning, source, target, "unevaluatedItems type changed")
		return
	}

	// Both same type - compare
	switch sourceType {
	case schemaAdditionalPropsTypeSchema:
		d.diffSchemaRecursiveUnified(source.(*parser.Schema), target.(*parser.Schema), fieldPath, visited, result)
	case schemaAdditionalPropsTypeBool:
		if source.(bool) != target.(bool) {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && source.(bool) && !target.(bool) {
				severity = SeverityError
			}
			d.addChange(result, fieldPath, ChangeTypeModified, CategorySchema,
				severity, source, target, fmt.Sprintf("unevaluatedItems changed from %v to %v", source, target))
		}
	}
}

// diffSchemaContentFieldsUnified compares content keywords (JSON Schema 2020-12)
func (d *Differ) diffSchemaContentFieldsUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	// ContentEncoding
	if source.ContentEncoding != target.ContentEncoding {
		if source.ContentEncoding == "" {
			d.addChange(result, path+".contentEncoding", ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target.ContentEncoding, "contentEncoding added")
		} else if target.ContentEncoding == "" {
			d.addChange(result, path+".contentEncoding", ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source.ContentEncoding, nil, "contentEncoding removed")
		} else {
			d.addChange(result, path+".contentEncoding", ChangeTypeModified, CategorySchema,
				SeverityWarning, source.ContentEncoding, target.ContentEncoding,
				fmt.Sprintf("contentEncoding changed from %q to %q", source.ContentEncoding, target.ContentEncoding))
		}
	}

	// ContentMediaType
	if source.ContentMediaType != target.ContentMediaType {
		if source.ContentMediaType == "" {
			d.addChange(result, path+".contentMediaType", ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target.ContentMediaType, "contentMediaType added")
		} else if target.ContentMediaType == "" {
			d.addChange(result, path+".contentMediaType", ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source.ContentMediaType, nil, "contentMediaType removed")
		} else {
			d.addChange(result, path+".contentMediaType", ChangeTypeModified, CategorySchema,
				SeverityWarning, source.ContentMediaType, target.ContentMediaType,
				fmt.Sprintf("contentMediaType changed from %q to %q", source.ContentMediaType, target.ContentMediaType))
		}
	}

	// ContentSchema
	if source.ContentSchema != nil || target.ContentSchema != nil {
		contentSchemaPath := path + ".contentSchema"
		if source.ContentSchema == nil {
			d.addChange(result, contentSchemaPath, ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target.ContentSchema, "contentSchema added")
		} else if target.ContentSchema == nil {
			d.addChange(result, contentSchemaPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source.ContentSchema, nil, "contentSchema removed")
		} else {
			d.diffSchemaRecursiveUnified(source.ContentSchema, target.ContentSchema, contentSchemaPath, visited, result)
		}
	}
}

// diffSchemaPrefixItemsUnified compares prefixItems arrays (JSON Schema 2020-12)
func (d *Differ) diffSchemaPrefixItemsUnified(source, target []*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	prefixPath := path + ".prefixItems"

	// Both nil/empty
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Added
	if len(source) == 0 && len(target) > 0 {
		d.addChange(result, prefixPath, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, target, "prefixItems added")
		return
	}

	// Removed
	if len(source) > 0 && len(target) == 0 {
		d.addChange(result, prefixPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "prefixItems removed")
		return
	}

	// Compare each item
	maxLen := len(source)
	if len(target) > maxLen {
		maxLen = len(target)
	}

	for i := 0; i < maxLen; i++ {
		itemPath := fmt.Sprintf("%s[%d]", prefixPath, i)
		if i >= len(source) {
			d.addChange(result, itemPath, ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target[i], "prefixItem added")
		} else if i >= len(target) {
			d.addChange(result, itemPath, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source[i], nil, "prefixItem removed")
		} else {
			d.diffSchemaRecursiveUnified(source[i], target[i], itemPath, visited, result)
		}
	}
}

// diffSchemaContainsUnified compares contains schema (JSON Schema 2020-12)
func (d *Differ) diffSchemaContainsUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	containsPath := path + ".contains"

	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, containsPath, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, target, "contains constraint added")
		return
	}

	if target == nil {
		d.addChange(result, containsPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "contains constraint removed")
		return
	}

	d.diffSchemaRecursiveUnified(source, target, containsPath, visited, result)
}

// diffSchemaPropertyNamesUnified compares propertyNames schema (JSON Schema 2020-12)
func (d *Differ) diffSchemaPropertyNamesUnified(source, target *parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	propNamesPath := path + ".propertyNames"

	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, propNamesPath, ChangeTypeAdded, CategorySchema,
			SeverityError, nil, target, "propertyNames constraint added")
		return
	}

	if target == nil {
		d.addChange(result, propNamesPath, ChangeTypeRemoved, CategorySchema,
			SeverityWarning, source, nil, "propertyNames constraint removed")
		return
	}

	d.diffSchemaRecursiveUnified(source, target, propNamesPath, visited, result)
}

// diffSchemaDependentSchemasUnified compares dependentSchemas (JSON Schema 2020-12)
func (d *Differ) diffSchemaDependentSchemasUnified(source, target map[string]*parser.Schema, path string, visited *schemaVisited, result *DiffResult) {
	depPath := path + ".dependentSchemas"

	// Build sets for comparison
	sourceKeys := make(map[string]bool)
	targetKeys := make(map[string]bool)
	for k := range source {
		sourceKeys[k] = true
	}
	for k := range target {
		targetKeys[k] = true
	}

	// Check for added/removed keys
	for key := range sourceKeys {
		if !targetKeys[key] {
			d.addChange(result, depPath+"."+key, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source[key], nil, fmt.Sprintf("dependentSchema %q removed", key))
		}
	}
	for key := range targetKeys {
		if !sourceKeys[key] {
			d.addChange(result, depPath+"."+key, ChangeTypeAdded, CategorySchema,
				SeverityError, nil, target[key], fmt.Sprintf("dependentSchema %q added", key))
		}
	}

	// Compare existing keys
	for key := range sourceKeys {
		if targetKeys[key] {
			d.diffSchemaRecursiveUnified(source[key], target[key], depPath+"."+key, visited, result)
		}
	}
}

// ============================================================================
// Phase 5: Operation functions (unified implementations)
// ============================================================================

// diffParameterUnified compares individual Parameter objects
func (d *Differ) diffParameterUnified(source, target *parser.Parameter, path string, result *DiffResult) {
	// Required changed
	if source.Required != target.Required {
		// Making optional parameter required is error, making required optional is warning
		d.addChangeConditional(result, path+".required", ChangeTypeModified, CategoryParameter,
			!source.Required && target.Required, SeverityError, SeverityWarning,
			source.Required, target.Required, fmt.Sprintf("required changed from %v to %v", source.Required, target.Required))
	}

	// Type changed
	if source.Type != target.Type {
		// Check for compatible type changes
		severity := SeverityWarning
		if d.Mode == ModeBreaking && !isCompatibleTypeChange(source.Type, target.Type) {
			severity = SeverityError
		}
		d.addChange(result, path+".type", ChangeTypeModified, CategoryParameter,
			severity, source.Type, target.Type, fmt.Sprintf("type changed from %q to %q", source.Type, target.Type))
	}

	// Format changed
	if source.Format != target.Format {
		d.addChange(result, path+".format", ChangeTypeModified, CategoryParameter,
			SeverityWarning, source.Format, target.Format, fmt.Sprintf("format changed from %q to %q", source.Format, target.Format))
	}

	// Schema comparison (OAS 3.x)
	if source.Schema != nil || target.Schema != nil {
		if source.Schema != nil && target.Schema != nil {
			d.diffSchemaUnified(source.Schema, target.Schema, path+".schema", result)
		} else if source.Schema == nil {
			d.addChange(result, path+".schema", ChangeTypeAdded, CategoryParameter,
				SeverityInfo, nil, target.Schema, "parameter schema added")
		} else {
			d.addChange(result, path+".schema", ChangeTypeRemoved, CategoryParameter,
				SeverityWarning, source.Schema, nil, "parameter schema removed")
		}
	}

	// Compare Parameter extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffParametersUnified compares Parameter slices
func (d *Differ) diffParametersUnified(source, target []*parser.Parameter, path string, result *DiffResult) {
	// Build maps by name+in for easier comparison
	sourceMap := make(map[string]*parser.Parameter)
	for _, param := range source {
		key := param.Name + ":" + param.In
		sourceMap[key] = param
	}

	targetMap := make(map[string]*parser.Parameter)
	for _, param := range target {
		key := param.Name + ":" + param.In
		targetMap[key] = param
	}

	// Find removed parameters
	for key, sourceParam := range sourceMap {
		if _, exists := targetMap[key]; !exists {
			severity := SeverityWarning
			if d.Mode == ModeBreaking && sourceParam.Required {
				severity = SeverityError
			}
			d.addChange(result, fmt.Sprintf("%s[%s]", path, key), ChangeTypeRemoved, CategoryParameter,
				severity, sourceParam, nil, fmt.Sprintf("parameter %q in %s removed", sourceParam.Name, sourceParam.In))
		}
	}

	// Find added or modified parameters
	for key, targetParam := range targetMap {
		sourceParam, exists := sourceMap[key]
		if !exists {
			severity := SeverityInfo
			if d.Mode == ModeBreaking && targetParam.Required {
				severity = SeverityWarning
			}
			d.addChange(result, fmt.Sprintf("%s[%s]", path, key), ChangeTypeAdded, CategoryParameter,
				severity, nil, targetParam, fmt.Sprintf("parameter %q in %s added", targetParam.Name, targetParam.In))
			continue
		}

		// Compare parameter details
		d.diffParameterUnified(sourceParam, targetParam, fmt.Sprintf("%s[%s]", path, key), result)
	}
}

// diffRequestBodyUnified compares RequestBody objects (OAS 3.x)
func (d *Differ) diffRequestBodyUnified(source, target *parser.RequestBody, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && target.Required {
			severity = SeverityWarning
		}
		d.addChange(result, path, ChangeTypeAdded, CategoryRequestBody,
			severity, nil, target, "request body added")
		return
	}

	if target == nil {
		severity := SeverityWarning
		if d.Mode == ModeBreaking && source.Required {
			severity = SeverityError
		}
		d.addChange(result, path, ChangeTypeRemoved, CategoryRequestBody,
			severity, source, nil, "request body removed")
		return
	}

	// Required changed
	if source.Required != target.Required {
		// Making optional required is error, making required optional is info
		d.addChangeConditional(result, path+".required", ChangeTypeModified, CategoryRequestBody,
			!source.Required && target.Required, SeverityError, SeverityInfo,
			source.Required, target.Required, fmt.Sprintf("required changed from %v to %v", source.Required, target.Required))
	}

	// Compare content media types
	d.diffRequestBodyContentUnified(source.Content, target.Content, path, result)

	// Compare RequestBody extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffRequestBodyContentUnified compares request body content maps
func (d *Differ) diffRequestBodyContentUnified(source, target map[string]*parser.MediaType, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed media types
	for mediaType := range source {
		if _, exists := target[mediaType]; !exists {
			d.addChange(result, fmt.Sprintf("%s.content.%s", path, mediaType), ChangeTypeRemoved, CategoryRequestBody,
				SeverityError, nil, nil, fmt.Sprintf("request body media type %q removed", mediaType))
		}
	}

	// Find added or modified media types
	for mediaType, targetMedia := range target {
		sourceMedia, exists := source[mediaType]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.content.%s", path, mediaType), ChangeTypeAdded, CategoryRequestBody,
				SeverityInfo, nil, nil, fmt.Sprintf("request body media type %q added", mediaType))
			continue
		}

		// Compare media type details
		d.diffRequestBodyMediaTypeUnified(sourceMedia, targetMedia, fmt.Sprintf("%s.content.%s", path, mediaType), result)
	}
}

// diffRequestBodyMediaTypeUnified compares request body MediaType objects
func (d *Differ) diffRequestBodyMediaTypeUnified(source, target *parser.MediaType, path string, result *DiffResult) {
	// Compare schemas if present
	if source.Schema != nil && target.Schema != nil {
		d.diffSchemaUnified(source.Schema, target.Schema, path+".schema", result)
	} else if source.Schema != nil && target.Schema == nil {
		d.addChange(result, path+".schema", ChangeTypeRemoved, CategoryRequestBody,
			SeverityError, nil, nil, "request body schema removed")
	} else if source.Schema == nil && target.Schema != nil {
		d.addChange(result, path+".schema", ChangeTypeAdded, CategoryRequestBody,
			SeverityWarning, nil, nil, "request body schema added")
	}

	// Compare MediaType extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffOperationUnified compares Operation objects
func (d *Differ) diffOperationUnified(source, target *parser.Operation, path string, result *DiffResult) {
	// Compare deprecated flag
	if source.Deprecated != target.Deprecated {
		severity := SeverityInfo
		if d.Mode == ModeBreaking && !source.Deprecated && target.Deprecated {
			severity = SeverityWarning
		}
		d.addChange(result, path+".deprecated", ChangeTypeModified, CategoryOperation,
			severity, source.Deprecated, target.Deprecated, fmt.Sprintf("deprecated changed from %v to %v", source.Deprecated, target.Deprecated))
	}

	// Compare parameters
	d.diffParametersUnified(source.Parameters, target.Parameters, path+".parameters", result)

	// Compare responses
	d.diffResponsesUnified(source.Responses, target.Responses, path+".responses", result)

	// Compare request body (OAS 3.x)
	if source.RequestBody != nil || target.RequestBody != nil {
		d.diffRequestBodyUnified(source.RequestBody, target.RequestBody, path+".requestBody", result)
	}

	// Compare Operation extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffPathItemUnified compares PathItem objects
func (d *Differ) diffPathItemUnified(source, target *parser.PathItem, path string, result *DiffResult) {
	operations := map[string]struct {
		source *parser.Operation
		target *parser.Operation
	}{
		"get":     {source.Get, target.Get},
		"put":     {source.Put, target.Put},
		"post":    {source.Post, target.Post},
		"delete":  {source.Delete, target.Delete},
		"options": {source.Options, target.Options},
		"head":    {source.Head, target.Head},
		"patch":   {source.Patch, target.Patch},
		"trace":   {source.Trace, target.Trace},
	}

	for method, ops := range operations {
		opPath := fmt.Sprintf("%s.%s", path, method)

		if ops.source == nil && ops.target == nil {
			continue
		}

		if ops.source == nil && ops.target != nil {
			d.addChange(result, opPath, ChangeTypeAdded, CategoryOperation,
				SeverityInfo, nil, ops.target, fmt.Sprintf("operation %s added", method))
			continue
		}

		if ops.source != nil && ops.target == nil {
			d.addChange(result, opPath, ChangeTypeRemoved, CategoryOperation,
				SeverityError, ops.source, nil, fmt.Sprintf("operation %s removed", method))
			continue
		}

		// Compare operations
		d.diffOperationUnified(ops.source, ops.target, opPath, result)
	}

	// Compare PathItem extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffPathsUnified compares Paths objects
func (d *Differ) diffPathsUnified(source, target parser.Paths, path string, result *DiffResult) {
	// Find removed paths
	for pathName, sourceItem := range source {
		targetItem, exists := target[pathName]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, pathName), ChangeTypeRemoved, CategoryEndpoint,
				SeverityCritical, sourceItem, nil, fmt.Sprintf("endpoint %q removed", pathName))
			continue
		}

		// Compare path items
		d.diffPathItemUnified(sourceItem, targetItem, fmt.Sprintf("%s.%s", path, pathName), result)
	}

	// Find added paths
	for pathName, targetItem := range target {
		if _, exists := source[pathName]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, pathName), ChangeTypeAdded, CategoryEndpoint,
				SeverityInfo, nil, targetItem, fmt.Sprintf("endpoint %q added", pathName))
		}
	}
}

// diffComponentsUnified compares Components objects (OAS 3.x)
func (d *Differ) diffComponentsUnified(source, target *parser.Components, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, path, ChangeTypeAdded, CategorySchema,
			SeverityInfo, nil, nil, "components added")
		return
	}

	if target == nil {
		d.addChange(result, path, ChangeTypeRemoved, CategorySchema,
			SeverityError, nil, nil, "components removed")
		return
	}

	// Compare schemas
	d.diffSchemasUnified(source.Schemas, target.Schemas, path+".schemas", result)

	// Compare security schemes
	d.diffSecuritySchemesUnified(source.SecuritySchemes, target.SecuritySchemes, path+".securitySchemes", result)

	// Compare mediaTypes (OAS 3.2+)
	d.diffMediaTypesUnified(source.MediaTypes, target.MediaTypes, path+".mediaTypes", result)

	// Compare Components extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffMediaTypesUnified compares reusable MediaType definitions (OAS 3.2+)
func (d *Differ) diffMediaTypesUnified(source, target map[string]*parser.MediaType, path string, result *DiffResult) {
	// Build key sets
	sourceKeys := make(map[string]bool)
	targetKeys := make(map[string]bool)
	for k := range source {
		sourceKeys[k] = true
	}
	for k := range target {
		targetKeys[k] = true
	}

	// Check for removed media types
	for key := range sourceKeys {
		if !targetKeys[key] {
			d.addChange(result, path+"."+key, ChangeTypeRemoved, CategorySchema,
				SeverityWarning, source[key], nil, fmt.Sprintf("mediaType %q removed", key))
		}
	}

	// Check for added media types
	for key := range targetKeys {
		if !sourceKeys[key] {
			d.addChange(result, path+"."+key, ChangeTypeAdded, CategorySchema,
				SeverityInfo, nil, target[key], fmt.Sprintf("mediaType %q added", key))
		}
	}

	// Compare existing media types
	for key := range sourceKeys {
		if targetKeys[key] {
			d.diffMediaTypeUnified(source[key], target[key], path+"."+key, result)
		}
	}
}

// ============================================================================
// Phase 6: Top-level functions (unified implementations)
// ============================================================================

// diffOAS2Unified compares two OAS 2.0 documents
func (d *Differ) diffOAS2Unified(source, target *parser.OAS2Document, result *DiffResult) {
	basePath := "document"

	// Compare Info
	d.diffInfoUnified(source.Info, target.Info, basePath+".info", result)

	// Compare Host, BasePath, Schemes
	if source.Host != target.Host {
		d.addChange(result, basePath+".host", ChangeTypeModified, CategoryServer,
			SeverityWarning, source.Host, target.Host, fmt.Sprintf("host changed from %q to %q", source.Host, target.Host))
	}

	if source.BasePath != target.BasePath {
		d.addChange(result, basePath+".basePath", ChangeTypeModified, CategoryServer,
			SeverityWarning, source.BasePath, target.BasePath, fmt.Sprintf("basePath changed from %q to %q", source.BasePath, target.BasePath))
	}

	d.diffStringSlicesUnified(source.Schemes, target.Schemes, basePath+".schemes", CategoryServer, "scheme", result)

	// Compare Consumes/Produces
	d.diffStringSlicesUnified(source.Consumes, target.Consumes, basePath+".consumes", CategoryOperation, "consumes media type", result)
	d.diffStringSlicesUnified(source.Produces, target.Produces, basePath+".produces", CategoryOperation, "produces media type", result)

	// Compare Paths
	d.diffPathsUnified(source.Paths, target.Paths, basePath+".paths", result)

	// Compare Definitions
	d.diffSchemasUnified(source.Definitions, target.Definitions, basePath+".definitions", result)

	// Compare Security Definitions
	d.diffSecuritySchemesUnified(source.SecurityDefinitions, target.SecurityDefinitions, basePath+".securityDefinitions", result)

	// Compare Tags
	d.diffTagsUnified(source.Tags, target.Tags, basePath+".tags", result)

	// Compare Extensions
	d.diffExtrasUnified(source.Extra, target.Extra, basePath, result)
}

// diffOAS3Unified compares two OAS 3.x documents
func (d *Differ) diffOAS3Unified(source, target *parser.OAS3Document, result *DiffResult) {
	basePath := "document"

	// Compare Info
	d.diffInfoUnified(source.Info, target.Info, basePath+".info", result)

	// Compare Servers
	d.diffServersUnified(source.Servers, target.Servers, basePath+".servers", result)

	// Compare Paths
	d.diffPathsUnified(source.Paths, target.Paths, basePath+".paths", result)

	// Compare Webhooks (OAS 3.1+)
	if source.Webhooks != nil || target.Webhooks != nil {
		d.diffWebhooksUnified(source.Webhooks, target.Webhooks, basePath+".webhooks", result)
	}

	// Compare Components
	if source.Components != nil || target.Components != nil {
		d.diffComponentsUnified(source.Components, target.Components, basePath+".components", result)
	}

	// Compare Tags
	d.diffTagsUnified(source.Tags, target.Tags, basePath+".tags", result)

	// Compare Extensions
	d.diffExtrasUnified(source.Extra, target.Extra, basePath, result)
}

// diffCrossVersionUnified compares documents of different OAS versions
func (d *Differ) diffCrossVersionUnified(source, target parser.ParseResult, result *DiffResult) {
	// Report version change
	d.addChange(result, "document", ChangeTypeModified, CategoryInfo,
		SeverityWarning, source.Version, target.Version, fmt.Sprintf("OAS version changed from %s to %s (cross-version diff has limitations)", source.Version, target.Version))

	// Compare what we can (Info and Paths exist in both)
	var sourceInfo, targetInfo *parser.Info
	var sourcePaths, targetPaths parser.Paths

	if doc, ok := source.OAS2Document(); ok {
		sourceInfo = doc.Info
		sourcePaths = doc.Paths
	} else if doc, ok := source.OAS3Document(); ok {
		sourceInfo = doc.Info
		sourcePaths = doc.Paths
	}

	if doc, ok := target.OAS2Document(); ok {
		targetInfo = doc.Info
		targetPaths = doc.Paths
	} else if doc, ok := target.OAS3Document(); ok {
		targetInfo = doc.Info
		targetPaths = doc.Paths
	}

	d.diffInfoUnified(sourceInfo, targetInfo, "document.info", result)
	d.diffPathsUnified(sourcePaths, targetPaths, "document.paths", result)
}

// diffUnified performs the unified diff that handles both ModeSimple and ModeBreaking
func (d *Differ) diffUnified(source, target parser.ParseResult, result *DiffResult) {
	// Compare based on OAS version
	sourceOAS2, sourceIsOAS2 := source.OAS2Document()
	targetOAS2, targetIsOAS2 := target.OAS2Document()
	sourceOAS3, sourceIsOAS3 := source.OAS3Document()
	targetOAS3, targetIsOAS3 := target.OAS3Document()

	switch {
	case sourceIsOAS2 && targetIsOAS2:
		d.diffOAS2Unified(sourceOAS2, targetOAS2, result)
	case sourceIsOAS3 && targetIsOAS3:
		d.diffOAS3Unified(sourceOAS3, targetOAS3, result)
	default:
		// Cross-version comparison
		d.diffCrossVersionUnified(source, target, result)
	}
}
