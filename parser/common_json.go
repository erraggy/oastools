package parser

import (
	"encoding/json"

	"github.com/erraggy/oastools/parser/internal/jsonhelpers"
)

// MarshalJSON implements custom JSON marshaling for Info.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (i *Info) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(i.Extra) == 0 {
		type Alias Info
		return json.Marshal((*Alias)(i))
	}

	// Build map with known fields
	m := map[string]any{
		"title":   i.Title,   // Required field, always include
		"version": i.Version, // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "description", i.Description)
	jsonhelpers.SetIfNotEmpty(m, "termsOfService", i.TermsOfService)
	jsonhelpers.SetIfNotNil(m, "contact", i.Contact)
	jsonhelpers.SetIfNotNil(m, "license", i.License)
	jsonhelpers.SetIfNotEmpty(m, "summary", i.Summary)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, i.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Info.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (i *Info) UnmarshalJSON(data []byte) error {
	type Alias Info
	aux := (*Alias)(i)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		i.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Contact.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (c *Contact) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(c.Extra) == 0 {
		type Alias Contact
		return json.Marshal((*Alias)(c))
	}

	// Build map with known fields
	m := make(map[string]any, 3+len(c.Extra))
	jsonhelpers.SetIfNotEmpty(m, "name", c.Name)
	jsonhelpers.SetIfNotEmpty(m, "url", c.URL)
	jsonhelpers.SetIfNotEmpty(m, "email", c.Email)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, c.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Contact.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (c *Contact) UnmarshalJSON(data []byte) error {
	type Alias Contact
	aux := (*Alias)(c)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		c.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for License.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (l *License) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(l.Extra) == 0 {
		type Alias License
		return json.Marshal((*Alias)(l))
	}

	// Build map with known fields
	m := make(map[string]any, 3+len(l.Extra))
	jsonhelpers.SetIfNotEmpty(m, "name", l.Name)
	jsonhelpers.SetIfNotEmpty(m, "url", l.URL)
	jsonhelpers.SetIfNotEmpty(m, "identifier", l.Identifier)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, l.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for License.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (l *License) UnmarshalJSON(data []byte) error {
	type Alias License
	aux := (*Alias)(l)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
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

// MarshalJSON implements custom JSON marshaling for ExternalDocs.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (e *ExternalDocs) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(e.Extra) == 0 {
		type Alias ExternalDocs
		return json.Marshal((*Alias)(e))
	}

	// Build map with known fields
	m := map[string]any{
		"url": e.URL, // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "description", e.Description)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, e.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for ExternalDocs.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (e *ExternalDocs) UnmarshalJSON(data []byte) error {
	type Alias ExternalDocs
	aux := (*Alias)(e)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
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

// MarshalJSON implements custom JSON marshaling for Tag.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (t *Tag) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(t.Extra) == 0 {
		type Alias Tag
		return json.Marshal((*Alias)(t))
	}

	// Build map with known fields
	m := map[string]any{
		"name": t.Name, // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "description", t.Description)
	jsonhelpers.SetIfNotNil(m, "externalDocs", t.ExternalDocs)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, t.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Tag.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (t *Tag) UnmarshalJSON(data []byte) error {
	type Alias Tag
	aux := (*Alias)(t)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		t.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Server.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (s *Server) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(s.Extra) == 0 {
		type Alias Server
		return json.Marshal((*Alias)(s))
	}

	// Build map with known fields
	m := map[string]any{
		"url": s.URL, // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "description", s.Description)
	jsonhelpers.SetIfMapNotEmpty(m, "variables", s.Variables)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, s.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Server.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (s *Server) UnmarshalJSON(data []byte) error {
	type Alias Server
	aux := (*Alias)(s)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		s.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for ServerVariable.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (sv *ServerVariable) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(sv.Extra) == 0 {
		type Alias ServerVariable
		return json.Marshal((*Alias)(sv))
	}

	// Build map with known fields
	m := map[string]any{
		"default": sv.Default, // Required field, always include
	}
	jsonhelpers.SetIfNotNil(m, "enum", sv.Enum)
	jsonhelpers.SetIfNotEmpty(m, "description", sv.Description)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, sv.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for ServerVariable.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (sv *ServerVariable) UnmarshalJSON(data []byte) error {
	type Alias ServerVariable
	aux := (*Alias)(sv)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
	for k, v := range m {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		sv.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Reference.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (r *Reference) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(r.Extra) == 0 {
		type Alias Reference
		return json.Marshal((*Alias)(r))
	}

	// Build map with known fields
	m := map[string]any{
		"$ref": r.Ref, // Required field, always include
	}
	jsonhelpers.SetIfNotEmpty(m, "summary", r.Summary)
	jsonhelpers.SetIfNotEmpty(m, "description", r.Description)

	// Merge in Extra fields and marshal
	return jsonhelpers.MarshalWithExtras(m, r.Extra)
}

// UnmarshalJSON implements custom JSON unmarshaling for Reference.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (r *Reference) UnmarshalJSON(data []byte) error {
	type Alias Reference
	aux := (*Alias)(r)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract specification extensions (fields starting with "x-")
	extra := make(map[string]any)
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
