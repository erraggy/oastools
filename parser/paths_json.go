package parser

import (
	"encoding/json"
	"fmt"

	"github.com/erraggy/oastools/internal/httputil"
)

// MarshalJSON implements custom JSON marshaling for PathItem.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (p *PathItem) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(p.Extra) == 0 {
		type Alias PathItem
		return json.Marshal((*Alias)(p))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 12+len(p.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if p.Ref != "" {
		m["$ref"] = p.Ref
	}
	if p.Summary != "" {
		m["summary"] = p.Summary
	}
	if p.Description != "" {
		m["description"] = p.Description
	}
	if p.Get != nil {
		m["get"] = p.Get
	}
	if p.Put != nil {
		m["put"] = p.Put
	}
	if p.Post != nil {
		m["post"] = p.Post
	}
	if p.Delete != nil {
		m["delete"] = p.Delete
	}
	if p.Options != nil {
		m["options"] = p.Options
	}
	if p.Head != nil {
		m["head"] = p.Head
	}
	if p.Patch != nil {
		m["patch"] = p.Patch
	}
	if p.Trace != nil {
		m["trace"] = p.Trace
	}
	if len(p.Servers) > 0 {
		m["servers"] = p.Servers
	}
	if len(p.Parameters) > 0 {
		m["parameters"] = p.Parameters
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range p.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for PathItem.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (p *PathItem) UnmarshalJSON(data []byte) error {
	type Alias PathItem
	aux := (*Alias)(p)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		p.Extra = extra
	}

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
		return json.Marshal((*Alias)(o))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 14+len(o.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if len(o.Tags) > 0 {
		m["tags"] = o.Tags
	}
	if o.Summary != "" {
		m["summary"] = o.Summary
	}
	if o.Description != "" {
		m["description"] = o.Description
	}
	if o.ExternalDocs != nil {
		m["externalDocs"] = o.ExternalDocs
	}
	if o.OperationID != "" {
		m["operationId"] = o.OperationID
	}
	if len(o.Parameters) > 0 {
		m["parameters"] = o.Parameters
	}
	if o.RequestBody != nil {
		m["requestBody"] = o.RequestBody
	}
	// Responses is required, always include
	m["responses"] = o.Responses
	if len(o.Callbacks) > 0 {
		m["callbacks"] = o.Callbacks
	}
	if o.Deprecated {
		m["deprecated"] = o.Deprecated
	}
	if len(o.Security) > 0 {
		m["security"] = o.Security
	}
	if len(o.Servers) > 0 {
		m["servers"] = o.Servers
	}
	if len(o.Consumes) > 0 {
		m["consumes"] = o.Consumes
	}
	if len(o.Produces) > 0 {
		m["produces"] = o.Produces
	}
	if len(o.Schemes) > 0 {
		m["schemes"] = o.Schemes
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range o.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Operation.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (o *Operation) UnmarshalJSON(data []byte) error {
	type Alias Operation
	aux := (*Alias)(o)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		o.Extra = extra
	}

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
		return json.Marshal((*Alias)(r))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 7+len(r.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if r.Ref != "" {
		m["$ref"] = r.Ref
	}
	// Description is required, always include
	m["description"] = r.Description
	if len(r.Headers) > 0 {
		m["headers"] = r.Headers
	}
	if len(r.Content) > 0 {
		m["content"] = r.Content
	}
	if len(r.Links) > 0 {
		m["links"] = r.Links
	}
	if r.Schema != nil {
		m["schema"] = r.Schema
	}
	if len(r.Examples) > 0 {
		m["examples"] = r.Examples
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range r.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Response.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (r *Response) UnmarshalJSON(data []byte) error {
	type Alias Response
	aux := (*Alias)(r)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		r.Extra = extra
	}

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
		return json.Marshal((*Alias)(l))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 7+len(l.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if l.Ref != "" {
		m["$ref"] = l.Ref
	}
	if l.OperationRef != "" {
		m["operationRef"] = l.OperationRef
	}
	if l.OperationID != "" {
		m["operationId"] = l.OperationID
	}
	if len(l.Parameters) > 0 {
		m["parameters"] = l.Parameters
	}
	if l.RequestBody != nil {
		m["requestBody"] = l.RequestBody
	}
	if l.Description != "" {
		m["description"] = l.Description
	}
	if l.Server != nil {
		m["server"] = l.Server
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range l.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Link.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (l *Link) UnmarshalJSON(data []byte) error {
	type Alias Link
	aux := (*Alias)(l)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		l.Extra = extra
	}

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
		return json.Marshal((*Alias)(mt))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 4+len(mt.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if mt.Schema != nil {
		m["schema"] = mt.Schema
	}
	if mt.Example != nil {
		m["example"] = mt.Example
	}
	if len(mt.Examples) > 0 {
		m["examples"] = mt.Examples
	}
	if len(mt.Encoding) > 0 {
		m["encoding"] = mt.Encoding
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range mt.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for MediaType.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (mt *MediaType) UnmarshalJSON(data []byte) error {
	type Alias MediaType
	aux := (*Alias)(mt)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		mt.Extra = extra
	}

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
		return json.Marshal((*Alias)(e))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 5+len(e.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if e.Ref != "" {
		m["$ref"] = e.Ref
	}
	if e.Summary != "" {
		m["summary"] = e.Summary
	}
	if e.Description != "" {
		m["description"] = e.Description
	}
	if e.Value != nil {
		m["value"] = e.Value
	}
	if e.ExternalValue != "" {
		m["externalValue"] = e.ExternalValue
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range e.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Example.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (e *Example) UnmarshalJSON(data []byte) error {
	type Alias Example
	aux := (*Alias)(e)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		e.Extra = extra
	}

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
		return json.Marshal((*Alias)(e))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 5+len(e.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if e.ContentType != "" {
		m["contentType"] = e.ContentType
	}
	if len(e.Headers) > 0 {
		m["headers"] = e.Headers
	}
	if e.Style != "" {
		m["style"] = e.Style
	}
	if e.Explode != nil {
		m["explode"] = e.Explode
	}
	if e.AllowReserved {
		m["allowReserved"] = e.AllowReserved
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range e.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Encoding.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (e *Encoding) UnmarshalJSON(data []byte) error {
	type Alias Encoding
	aux := (*Alias)(e)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]interface{})
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		e.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Responses.
// This flattens the Codes map into the top-level JSON object, where each
// HTTP status code (e.g., "200", "404") or wildcard pattern (e.g., "2XX")
// becomes a direct field in the JSON output. The "default" response is also
// included at the top level if present.
func (r *Responses) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	// Add default if present
	if r.Default != nil {
		m["default"] = r.Default
	}

	// Add status code responses
	for code, response := range r.Codes {
		m[code] = response
	}

	return json.Marshal(m)
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
