package parser

import (
	"encoding/json"
	"fmt"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser/internal/jsonhelpers"
)

// MarshalJSON implements custom JSON marshaling for PathItem.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (p *PathItem) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(p.Extra) == 0 {
		type Alias PathItem
		return marshalToJSON((*Alias)(p))
	}

	// Build map with known fields
	m := make(map[string]any, 12+len(p.Extra))
	jsonhelpers.SetIfNotEmpty(m, "$ref", p.Ref)
	jsonhelpers.SetIfNotEmpty(m, "summary", p.Summary)
	jsonhelpers.SetIfNotEmpty(m, "description", p.Description)
	jsonhelpers.SetIfNotNil(m, "get", p.Get)
	jsonhelpers.SetIfNotNil(m, "put", p.Put)
	jsonhelpers.SetIfNotNil(m, "post", p.Post)
	jsonhelpers.SetIfNotNil(m, "delete", p.Delete)
	jsonhelpers.SetIfNotNil(m, "options", p.Options)
	jsonhelpers.SetIfNotNil(m, "head", p.Head)
	jsonhelpers.SetIfNotNil(m, "patch", p.Patch)
	jsonhelpers.SetIfNotNil(m, "trace", p.Trace)
	jsonhelpers.SetIfSliceNotEmpty(m, "servers", p.Servers)
	jsonhelpers.SetIfSliceNotEmpty(m, "parameters", p.Parameters)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, p.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for PathItem.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (p *PathItem) UnmarshalJSON(data []byte) error {
	type Alias PathItem
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return err
	}
	p.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for Operation.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (o *Operation) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(o.Extra) == 0 {
		type Alias Operation
		return marshalToJSON((*Alias)(o))
	}

	// Build map with known fields
	m := map[string]any{
		"responses": o.Responses, // Required field, always include
	}
	jsonhelpers.SetIfSliceNotEmpty(m, "tags", o.Tags)
	jsonhelpers.SetIfNotEmpty(m, "summary", o.Summary)
	jsonhelpers.SetIfNotEmpty(m, "description", o.Description)
	jsonhelpers.SetIfNotNil(m, "externalDocs", o.ExternalDocs)
	jsonhelpers.SetIfNotEmpty(m, "operationId", o.OperationID)
	jsonhelpers.SetIfSliceNotEmpty(m, "parameters", o.Parameters)
	jsonhelpers.SetIfNotNil(m, "requestBody", o.RequestBody)
	jsonhelpers.SetIfMapNotEmpty(m, "callbacks", o.Callbacks)
	jsonhelpers.SetIfTrue(m, "deprecated", o.Deprecated)
	jsonhelpers.SetIfSliceNotEmpty(m, "security", o.Security)
	jsonhelpers.SetIfSliceNotEmpty(m, "servers", o.Servers)
	jsonhelpers.SetIfSliceNotEmpty(m, "consumes", o.Consumes)
	jsonhelpers.SetIfSliceNotEmpty(m, "produces", o.Produces)
	jsonhelpers.SetIfSliceNotEmpty(m, "schemes", o.Schemes)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, o.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Operation.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (o *Operation) UnmarshalJSON(data []byte) error {
	type Alias Operation
	if err := json.Unmarshal(data, (*Alias)(o)); err != nil {
		return err
	}
	o.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for Response.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (r *Response) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(r.Extra) == 0 {
		type Alias Response
		return marshalToJSON((*Alias)(r))
	}

	// Build map with known fields
	m := map[string]any{
		"description": r.Description, // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "$ref", r.Ref)
	jsonhelpers.SetIfMapNotEmpty(m, "headers", r.Headers)
	jsonhelpers.SetIfMapNotEmpty(m, "content", r.Content)
	jsonhelpers.SetIfMapNotEmpty(m, "links", r.Links)
	jsonhelpers.SetIfNotNil(m, "schema", r.Schema)
	jsonhelpers.SetIfMapNotEmpty(m, "examples", r.Examples)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, r.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Response.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (r *Response) UnmarshalJSON(data []byte) error {
	type Alias Response
	if err := json.Unmarshal(data, (*Alias)(r)); err != nil {
		return err
	}
	r.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for Link.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (l *Link) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(l.Extra) == 0 {
		type Alias Link
		return marshalToJSON((*Alias)(l))
	}

	// Build map with known fields
	m := make(map[string]any, 7+len(l.Extra))
	jsonhelpers.SetIfNotEmpty(m, "$ref", l.Ref)
	jsonhelpers.SetIfNotEmpty(m, "operationRef", l.OperationRef)
	jsonhelpers.SetIfNotEmpty(m, "operationId", l.OperationID)
	jsonhelpers.SetIfMapNotEmpty(m, "parameters", l.Parameters)
	jsonhelpers.SetIfNotNil(m, "requestBody", l.RequestBody)
	jsonhelpers.SetIfNotEmpty(m, "description", l.Description)
	jsonhelpers.SetIfNotNil(m, "server", l.Server)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, l.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Link.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (l *Link) UnmarshalJSON(data []byte) error {
	type Alias Link
	if err := json.Unmarshal(data, (*Alias)(l)); err != nil {
		return err
	}
	l.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for MediaType.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (mt *MediaType) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(mt.Extra) == 0 {
		type Alias MediaType
		return marshalToJSON((*Alias)(mt))
	}

	// Build map with known fields
	m := make(map[string]any, 4+len(mt.Extra))
	jsonhelpers.SetIfNotNil(m, "schema", mt.Schema)
	jsonhelpers.SetIfNotNil(m, "example", mt.Example)
	jsonhelpers.SetIfMapNotEmpty(m, "examples", mt.Examples)
	jsonhelpers.SetIfMapNotEmpty(m, "encoding", mt.Encoding)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, mt.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for MediaType.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (mt *MediaType) UnmarshalJSON(data []byte) error {
	type Alias MediaType
	if err := json.Unmarshal(data, (*Alias)(mt)); err != nil {
		return err
	}
	mt.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for Example.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (e *Example) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(e.Extra) == 0 {
		type Alias Example
		return marshalToJSON((*Alias)(e))
	}

	// Build map with known fields
	m := make(map[string]any, 5+len(e.Extra))
	jsonhelpers.SetIfNotEmpty(m, "$ref", e.Ref)
	jsonhelpers.SetIfNotEmpty(m, "summary", e.Summary)
	jsonhelpers.SetIfNotEmpty(m, "description", e.Description)
	jsonhelpers.SetIfNotNil(m, "value", e.Value)
	jsonhelpers.SetIfNotEmpty(m, "externalValue", e.ExternalValue)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, e.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Example.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (e *Example) UnmarshalJSON(data []byte) error {
	type Alias Example
	if err := json.Unmarshal(data, (*Alias)(e)); err != nil {
		return err
	}
	e.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for Encoding.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (e *Encoding) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(e.Extra) == 0 {
		type Alias Encoding
		return marshalToJSON((*Alias)(e))
	}

	// Build map with known fields
	m := make(map[string]any, 5+len(e.Extra))
	jsonhelpers.SetIfNotEmpty(m, "contentType", e.ContentType)
	jsonhelpers.SetIfMapNotEmpty(m, "headers", e.Headers)
	jsonhelpers.SetIfNotEmpty(m, "style", e.Style)
	jsonhelpers.SetIfNotNil(m, "explode", e.Explode)
	jsonhelpers.SetIfTrue(m, "allowReserved", e.AllowReserved)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, e.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Encoding.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (e *Encoding) UnmarshalJSON(data []byte) error {
	type Alias Encoding
	if err := json.Unmarshal(data, (*Alias)(e)); err != nil {
		return err
	}
	e.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for Responses.
// This flattens the Codes map into the top-level JSON object, where each
// HTTP status code (e.g., "200", "404") or wildcard pattern (e.g., "2XX")
// becomes a direct field in the JSON output. The "default" response is also
// included at the top level if present.
func (r *Responses) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)

	// Add default if present
	if r.Default != nil {
		m["default"] = r.Default
	}

	// Add status code responses
	for code, response := range r.Codes {
		m[code] = response
	}

	return marshalToJSON(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Responses.
// This captures status code fields in the Codes map and validates that each
// status code is either a valid HTTP status code (e.g., "200", "404"), a
// wildcard pattern (e.g., "2XX"), or a specification extension (e.g., "x-custom").
// Returns an error if an invalid status code is encountered.
func (r *Responses) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	r.Codes = make(map[string]*Response)

	for key, value := range m {
		if key == "default" {
			var defaultResp Response
			if err := json.Unmarshal(value, &defaultResp); err != nil {
				return err
			}
			r.Default = &defaultResp
		} else {
			// Validate status code - must be valid HTTP status code or extension field
			if !httputil.ValidateStatusCode(key) {
				return fmt.Errorf("invalid status code '%s' in responses: must be a valid HTTP status code (e.g., \"200\", \"404\"), wildcard pattern (e.g., \"2XX\"), or extension field (e.g., \"x-custom\")", key)
			}
			var resp Response
			if err := json.Unmarshal(value, &resp); err != nil {
				return err
			}
			r.Codes[key] = &resp
		}
	}

	return nil
}
