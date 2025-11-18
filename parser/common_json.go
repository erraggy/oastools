package parser

import (
	"encoding/json"
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

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 7+len(i.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	m["title"] = i.Title
	m["version"] = i.Version

	if i.Description != "" {
		m["description"] = i.Description
	}
	if i.TermsOfService != "" {
		m["termsOfService"] = i.TermsOfService
	}
	if i.Contact != nil {
		m["contact"] = i.Contact
	}
	if i.License != nil {
		m["license"] = i.License
	}
	if i.Summary != "" {
		m["summary"] = i.Summary
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range i.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Info.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (i *Info) UnmarshalJSON(data []byte) error {
	type Alias Info
	aux := (*Alias)(i)

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

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 3+len(c.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if c.Name != "" {
		m["name"] = c.Name
	}
	if c.URL != "" {
		m["url"] = c.URL
	}
	if c.Email != "" {
		m["email"] = c.Email
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range c.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Contact.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (c *Contact) UnmarshalJSON(data []byte) error {
	type Alias Contact
	aux := (*Alias)(c)

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

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 3+len(l.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if l.Name != "" {
		m["name"] = l.Name
	}
	if l.URL != "" {
		m["url"] = l.URL
	}
	if l.Identifier != "" {
		m["identifier"] = l.Identifier
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range l.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for License.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (l *License) UnmarshalJSON(data []byte) error {
	type Alias License
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

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 2+len(e.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if e.Description != "" {
		m["description"] = e.Description
	}
	m["url"] = e.URL

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range e.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for ExternalDocs.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (e *ExternalDocs) UnmarshalJSON(data []byte) error {
	type Alias ExternalDocs
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

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 3+len(t.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	m["name"] = t.Name

	if t.Description != "" {
		m["description"] = t.Description
	}
	if t.ExternalDocs != nil {
		m["externalDocs"] = t.ExternalDocs
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range t.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Tag.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (t *Tag) UnmarshalJSON(data []byte) error {
	type Alias Tag
	aux := (*Alias)(t)

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

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 3+len(s.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	m["url"] = s.URL

	if s.Description != "" {
		m["description"] = s.Description
	}
	if len(s.Variables) > 0 {
		m["variables"] = s.Variables
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range s.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Server.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (s *Server) UnmarshalJSON(data []byte) error {
	type Alias Server
	aux := (*Alias)(s)

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

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 3+len(sv.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if len(sv.Enum) > 0 {
		m["enum"] = sv.Enum
	}
	m["default"] = sv.Default

	if sv.Description != "" {
		m["description"] = sv.Description
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range sv.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for ServerVariable.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (sv *ServerVariable) UnmarshalJSON(data []byte) error {
	type Alias ServerVariable
	aux := (*Alias)(sv)

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

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]interface{}, 3+len(r.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	m["$ref"] = r.Ref

	if r.Summary != "" {
		m["summary"] = r.Summary
	}
	if r.Description != "" {
		m["description"] = r.Description
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range r.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Reference.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (r *Reference) UnmarshalJSON(data []byte) error {
	type Alias Reference
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
