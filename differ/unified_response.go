package differ

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

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
