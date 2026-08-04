package parser

import (
	"encoding/json"

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
	jsonhelpers.SetIfNotEmpty(m, jsonKeyRef, p.Ref)
	jsonhelpers.SetIfNotEmpty(m, "summary", p.Summary)
	jsonhelpers.SetIfNotEmpty(m, jsonKeyDescription, p.Description)
	jsonhelpers.SetIfNotNil(m, "get", p.Get)
	jsonhelpers.SetIfNotNil(m, "put", p.Put)
	jsonhelpers.SetIfNotNil(m, "post", p.Post)
	jsonhelpers.SetIfNotNil(m, "delete", p.Delete)
	jsonhelpers.SetIfNotNil(m, "options", p.Options)
	jsonhelpers.SetIfNotNil(m, "head", p.Head)
	jsonhelpers.SetIfNotNil(m, "patch", p.Patch)
	jsonhelpers.SetIfNotNil(m, "trace", p.Trace)
	jsonhelpers.SetIfNotNil(m, "query", p.Query)                                    // OAS 3.2+
	jsonhelpers.SetIfMapNotEmpty(m, "additionalOperations", p.AdditionalOperations) // OAS 3.2+
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
	// Fast path: no Extra fields and nil security, use standard marshaling.
	// Security is excluded from the fast path because a non-nil empty slice
	// must serialize as [] (disable inherited security), not be omitted.
	// CallbackRefs is excluded because the struct tags cannot merge it back
	// into the callbacks object; see [Callback].
	if len(o.Extra) == 0 && o.Security == nil && len(o.CallbackRefs) == 0 {
		type Alias Operation
		return marshalToJSON((*Alias)(o))
	}

	callbacks, err := mergedCallbacks(o.Callbacks, o.CallbackRefs)
	if err != nil {
		return nil, err
	}

	// Build map with known fields
	m := map[string]any{
		"responses": o.Responses, // Required field, always include
	}
	jsonhelpers.SetIfSliceNotEmpty(m, "tags", o.Tags)
	jsonhelpers.SetIfNotEmpty(m, "summary", o.Summary)
	jsonhelpers.SetIfNotEmpty(m, jsonKeyDescription, o.Description)
	jsonhelpers.SetIfNotNil(m, "externalDocs", o.ExternalDocs)
	jsonhelpers.SetIfNotEmpty(m, "operationId", o.OperationID)
	jsonhelpers.SetIfSliceNotEmpty(m, "parameters", o.Parameters)
	jsonhelpers.SetIfNotNil(m, "requestBody", o.RequestBody)
	jsonhelpers.SetIfMapNotEmpty(m, "callbacks", callbacks)
	jsonhelpers.SetIfTrue(m, "deprecated", o.Deprecated)
	jsonhelpers.SetIfSliceNotNil(m, "security", o.Security)
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
	refs, extra, err := unmarshalJSONWithCallbackRefs(data, (*Alias)(o))
	if err != nil {
		return err
	}
	o.CallbackRefs, o.Extra = refs, extra
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

	// Build map with known fields conditionally so this path agrees with the
	// struct tags used by the fast path above. See [Response] for why
	// description is omitempty.
	m := make(map[string]any, 7+len(r.Extra))
	jsonhelpers.SetIfNotEmpty(m, jsonKeyDescription, r.Description)
	jsonhelpers.SetIfNotEmpty(m, jsonKeyRef, r.Ref)
	jsonhelpers.SetIfMapNotEmpty(m, "headers", r.Headers)
	jsonhelpers.SetIfMapNotEmpty(m, "content", r.Content)
	jsonhelpers.SetIfMapNotEmpty(m, "links", r.Links)
	jsonhelpers.SetIfNotNil(m, "schema", r.Schema)
	jsonhelpers.SetIfMapNotEmpty(m, "examples", r.Examples)
	jsonhelpers.SetIfNotEmpty(m, "summary", r.Summary)

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
	jsonhelpers.SetIfNotEmpty(m, jsonKeyRef, l.Ref)
	jsonhelpers.SetIfNotEmpty(m, "operationRef", l.OperationRef)
	jsonhelpers.SetIfNotEmpty(m, "operationId", l.OperationID)
	jsonhelpers.SetIfMapNotEmpty(m, "parameters", l.Parameters)
	jsonhelpers.SetIfNotNil(m, "requestBody", l.RequestBody)
	jsonhelpers.SetIfNotEmpty(m, jsonKeyDescription, l.Description)
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
	jsonhelpers.SetIfNotNil(m, "itemSchema", mt.ItemSchema)
	jsonhelpers.SetIfNotNil(m, "itemEncoding", mt.ItemEncoding)
	jsonhelpers.SetIfSliceNotEmpty(m, "prefixEncoding", mt.PrefixEncoding)

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
	jsonhelpers.SetIfNotEmpty(m, jsonKeyRef, e.Ref)
	jsonhelpers.SetIfNotEmpty(m, "summary", e.Summary)
	jsonhelpers.SetIfNotEmpty(m, jsonKeyDescription, e.Description)
	jsonhelpers.SetIfNotNil(m, "value", e.Value)
	jsonhelpers.SetIfNotEmpty(m, "externalValue", e.ExternalValue)
	jsonhelpers.SetIfNotNil(m, "dataValue", e.DataValue)
	jsonhelpers.SetIfNotEmpty(m, "serializedValue", e.SerializedValue)

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
	jsonhelpers.SetIfMapNotEmpty(m, "encoding", e.Encoding)
	jsonhelpers.SetIfNotNil(m, "itemEncoding", e.ItemEncoding)
	jsonhelpers.SetIfSliceNotEmpty(m, "prefixEncoding", e.PrefixEncoding)

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
		m[jsonKeyDefault] = r.Default
	}

	// Add status code responses
	for code, response := range r.Codes {
		m[code] = response
	}

	// Specification extensions belong in the same object. Codes wins a clash,
	// which only a caller-assembled document can produce.
	for key, value := range r.Extra {
		if _, dup := m[key]; !dup {
			m[key] = value
		}
	}

	return marshalToJSON(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Responses. It sorts the
// object's keys into Default, Extra and Codes: `default` to its own field, a
// specification extension (e.g. "x-custom") to Extra, and everything else to
// Codes.
//
// A key that is no legal status code is not rejected here. It lands in Codes and
// validateStructure reports it, so that a document's verdict does not depend on
// which decoder read it. It returns an error only when a value cannot be decoded
// into the type its key calls for.
func (r *Responses) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Reset every field, so decoding into a reused value reflects this document
	// alone rather than merging with whatever it held before.
	r.Codes = make(map[string]*Response)
	r.Extra = nil
	r.Default = nil

	for key, value := range m {
		if key == jsonKeyDefault {
			var defaultResp Response
			if err := json.Unmarshal(value, &defaultResp); err != nil {
				return err
			}
			r.Default = &defaultResp
		} else if httputil.IsExtensionKey(key) {
			// A specification extension, which the Responses Object admits.
			// It is not a status code, so it does not belong in Codes.
			var ext any
			if err := json.Unmarshal(value, &ext); err != nil {
				return err
			}
			if r.Extra == nil {
				r.Extra = make(map[string]any)
			}
			r.Extra[key] = ext
		} else {
			// Everything else is a status code, and a key that is not one is
			// kept rather than rejected here: validateStructure reports it, the
			// way it does for a document decoded from a map. Rejecting at this
			// depth is what made the same document pass or fail depending on
			// which decoder ran (#449).
			var resp Response
			if err := json.Unmarshal(value, &resp); err != nil {
				return err
			}
			r.Codes[key] = &resp
		}
	}

	return nil
}
