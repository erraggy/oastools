package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for Info.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (i *Info) MarshalJSON() ([]byte, error) {
	type Alias Info
	aux, err := json.Marshal((*Alias)(i))
	if err != nil {
		return nil, err
	}

	if len(i.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"title":          true,
		"description":    true,
		"termsOfService": true,
		"contact":        true,
		"license":        true,
		"version":        true,
		"summary":        true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
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
	type Alias Contact
	aux, err := json.Marshal((*Alias)(c))
	if err != nil {
		return nil, err
	}

	if len(c.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"name":  true,
		"url":   true,
		"email": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
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
	type Alias License
	aux, err := json.Marshal((*Alias)(l))
	if err != nil {
		return nil, err
	}

	if len(l.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"name":       true,
		"url":        true,
		"identifier": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
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
	type Alias ExternalDocs
	aux, err := json.Marshal((*Alias)(e))
	if err != nil {
		return nil, err
	}

	if len(e.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"description": true,
		"url":         true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
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
	type Alias Tag
	aux, err := json.Marshal((*Alias)(t))
	if err != nil {
		return nil, err
	}

	if len(t.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"name":         true,
		"description":  true,
		"externalDocs": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
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
	type Alias Server
	aux, err := json.Marshal((*Alias)(s))
	if err != nil {
		return nil, err
	}

	if len(s.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"url":         true,
		"description": true,
		"variables":   true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
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
	type Alias ServerVariable
	aux, err := json.Marshal((*Alias)(sv))
	if err != nil {
		return nil, err
	}

	if len(sv.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"enum":        true,
		"default":     true,
		"description": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
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
	type Alias Reference
	aux, err := json.Marshal((*Alias)(r))
	if err != nil {
		return nil, err
	}

	if len(r.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

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

	knownFields := map[string]bool{
		"$ref":        true,
		"summary":     true,
		"description": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		r.Extra = extra
	}

	return nil
}
