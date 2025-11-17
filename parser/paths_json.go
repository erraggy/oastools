package parser

import (
	"encoding/json"

	"github.com/erraggy/oastools/internal/httputil"
)

// MarshalJSON implements custom JSON marshaling for PathItem
func (p *PathItem) MarshalJSON() ([]byte, error) {
	type Alias PathItem
	aux, err := json.Marshal((*Alias)(p))
	if err != nil {
		return nil, err
	}

	if len(p.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range p.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for PathItem
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

	knownFields := map[string]bool{
		"$ref": true, "summary": true, "description": true,
		"get": true, "put": true, "post": true, "delete": true, "options": true, "head": true, "patch": true, "trace": true,
		"servers": true, "parameters": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		p.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Operation
func (o *Operation) MarshalJSON() ([]byte, error) {
	type Alias Operation
	aux, err := json.Marshal((*Alias)(o))
	if err != nil {
		return nil, err
	}

	if len(o.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range o.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for Operation
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

	knownFields := map[string]bool{
		"tags": true, "summary": true, "description": true, "externalDocs": true, "operationId": true,
		"parameters": true, "requestBody": true, "responses": true, "callbacks": true, "deprecated": true,
		"security": true, "servers": true, "consumes": true, "produces": true, "schemes": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		o.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Response
func (r *Response) MarshalJSON() ([]byte, error) {
	type Alias Response
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

// UnmarshalJSON implements custom JSON unmarshaling for Response
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

	knownFields := map[string]bool{
		"$ref":        true,
		"description": true,
		"headers":     true,
		"content":     true,
		"links":       true,
		"schema":      true,
		"examples":    true,
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

// MarshalJSON implements custom JSON marshaling for Link
func (l *Link) MarshalJSON() ([]byte, error) {
	type Alias Link
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

// UnmarshalJSON implements custom JSON unmarshaling for Link
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

	knownFields := map[string]bool{
		"$ref":         true,
		"operationRef": true,
		"operationId":  true,
		"parameters":   true,
		"requestBody":  true,
		"description":  true,
		"server":       true,
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

// MarshalJSON implements custom JSON marshaling for MediaType
func (mt *MediaType) MarshalJSON() ([]byte, error) {
	type Alias MediaType
	aux, err := json.Marshal((*Alias)(mt))
	if err != nil {
		return nil, err
	}

	if len(mt.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range mt.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for MediaType
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

	knownFields := map[string]bool{
		"schema":   true,
		"example":  true,
		"examples": true,
		"encoding": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		mt.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Example
func (e *Example) MarshalJSON() ([]byte, error) {
	type Alias Example
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

// UnmarshalJSON implements custom JSON unmarshaling for Example
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

	knownFields := map[string]bool{
		"$ref":          true,
		"summary":       true,
		"description":   true,
		"value":         true,
		"externalValue": true,
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

// MarshalJSON implements custom JSON marshaling for Encoding
func (e *Encoding) MarshalJSON() ([]byte, error) {
	type Alias Encoding
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

// UnmarshalJSON implements custom JSON unmarshaling for Encoding
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

	knownFields := map[string]bool{
		"contentType":   true,
		"headers":       true,
		"style":         true,
		"explode":       true,
		"allowReserved": true,
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

// MarshalJSON implements custom JSON marshaling for Responses
// This flattens the Codes map into the top-level JSON object
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

// UnmarshalJSON implements custom JSON unmarshaling for Responses
// This captures status code fields in the Codes map
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
			// Validate status code
			if !httputil.ValidateStatusCode(key) {
				// Ignore invalid status codes (they'll be lost, but this maintains compatibility)
				continue
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
