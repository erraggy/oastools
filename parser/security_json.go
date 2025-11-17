package parser

import (
	"encoding/json"
)

// MarshalJSON implements custom JSON marshaling for SecurityScheme.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (ss *SecurityScheme) MarshalJSON() ([]byte, error) {
	type Alias SecurityScheme
	aux, err := json.Marshal((*Alias)(ss))
	if err != nil {
		return nil, err
	}

	if len(ss.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range ss.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for SecurityScheme.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (ss *SecurityScheme) UnmarshalJSON(data []byte) error {
	type Alias SecurityScheme
	aux := (*Alias)(ss)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"$ref":             true,
		"type":             true,
		"description":      true,
		"name":             true,
		"in":               true,
		"scheme":           true,
		"bearerFormat":     true,
		"flows":            true,
		"flow":             true,
		"authorizationUrl": true,
		"tokenUrl":         true,
		"scopes":           true,
		"openIdConnectUrl": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		ss.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for OAuthFlows.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (of *OAuthFlows) MarshalJSON() ([]byte, error) {
	type Alias OAuthFlows
	aux, err := json.Marshal((*Alias)(of))
	if err != nil {
		return nil, err
	}

	if len(of.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range of.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for OAuthFlows.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (of *OAuthFlows) UnmarshalJSON(data []byte) error {
	type Alias OAuthFlows
	aux := (*Alias)(of)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"implicit":          true,
		"password":          true,
		"clientCredentials": true,
		"authorizationCode": true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		of.Extra = extra
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for OAuthFlow.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (of *OAuthFlow) MarshalJSON() ([]byte, error) {
	type Alias OAuthFlow
	aux, err := json.Marshal((*Alias)(of))
	if err != nil {
		return nil, err
	}

	if len(of.Extra) == 0 {
		return aux, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(aux, &m); err != nil {
		return nil, err
	}

	for k, v := range of.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for OAuthFlow.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (of *OAuthFlow) UnmarshalJSON(data []byte) error {
	type Alias OAuthFlow
	aux := (*Alias)(of)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	knownFields := map[string]bool{
		"authorizationUrl": true,
		"tokenUrl":         true,
		"refreshUrl":       true,
		"scopes":           true,
	}

	extra := make(map[string]interface{})
	for k, v := range m {
		if !knownFields[k] {
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		of.Extra = extra
	}

	return nil
}
