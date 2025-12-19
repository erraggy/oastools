package parser

import (
	"encoding/json"

	"github.com/erraggy/oastools/parser/internal/jsonhelpers"
)

// MarshalJSON implements custom JSON marshaling for SecurityScheme.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (ss *SecurityScheme) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(ss.Extra) == 0 {
		type Alias SecurityScheme
		return json.Marshal((*Alias)(ss))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]any, 11+len(ss.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if ss.Ref != "" {
		m["$ref"] = ss.Ref
	}
	// Type is required, always include
	m["type"] = ss.Type
	if ss.Description != "" {
		m["description"] = ss.Description
	}
	if ss.Name != "" {
		m["name"] = ss.Name
	}
	if ss.In != "" {
		m["in"] = ss.In
	}
	if ss.Scheme != "" {
		m["scheme"] = ss.Scheme
	}
	if ss.BearerFormat != "" {
		m["bearerFormat"] = ss.BearerFormat
	}
	if ss.Flows != nil {
		m["flows"] = ss.Flows
	}
	if ss.Flow != "" {
		m["flow"] = ss.Flow
	}
	if ss.AuthorizationURL != "" {
		m["authorizationUrl"] = ss.AuthorizationURL
	}
	if ss.TokenURL != "" {
		m["tokenUrl"] = ss.TokenURL
	}
	if len(ss.Scopes) > 0 {
		m["scopes"] = ss.Scopes
	}
	if ss.OpenIDConnectURL != "" {
		m["openIdConnectUrl"] = ss.OpenIDConnectURL
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range ss.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for SecurityScheme.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (ss *SecurityScheme) UnmarshalJSON(data []byte) error {
	type Alias SecurityScheme
	if err := json.Unmarshal(data, (*Alias)(ss)); err != nil {
		return err
	}
	ss.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for OAuthFlows.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (of *OAuthFlows) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(of.Extra) == 0 {
		type Alias OAuthFlows
		return json.Marshal((*Alias)(of))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]any, 4+len(of.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if of.Implicit != nil {
		m["implicit"] = of.Implicit
	}
	if of.Password != nil {
		m["password"] = of.Password
	}
	if of.ClientCredentials != nil {
		m["clientCredentials"] = of.ClientCredentials
	}
	if of.AuthorizationCode != nil {
		m["authorizationCode"] = of.AuthorizationCode
	}

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range of.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for OAuthFlows.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (of *OAuthFlows) UnmarshalJSON(data []byte) error {
	type Alias OAuthFlows
	if err := json.Unmarshal(data, (*Alias)(of)); err != nil {
		return err
	}
	of.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}

// MarshalJSON implements custom JSON marshaling for OAuthFlow.
// This is required to flatten Extra fields (specification extensions like x-*)
// into the top-level JSON object, as Go's encoding/json doesn't support
// inline maps like yaml:",inline".
func (of *OAuthFlow) MarshalJSON() ([]byte, error) {
	// Fast path: no Extra fields, use standard marshaling
	if len(of.Extra) == 0 {
		type Alias OAuthFlow
		return json.Marshal((*Alias)(of))
	}

	// Build map directly to avoid double-marshal pattern
	m := make(map[string]any, 4+len(of.Extra))

	// Add known fields (omit zero values to match json:",omitempty" behavior)
	if of.AuthorizationURL != "" {
		m["authorizationUrl"] = of.AuthorizationURL
	}
	if of.TokenURL != "" {
		m["tokenUrl"] = of.TokenURL
	}
	if of.RefreshURL != "" {
		m["refreshUrl"] = of.RefreshURL
	}
	// Scopes is required, always include
	m["scopes"] = of.Scopes

	// Add Extra fields (spec extensions must start with "x-")
	for k, v := range of.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for OAuthFlow.
// This captures unknown fields (specification extensions like x-*) in the Extra map.
func (of *OAuthFlow) UnmarshalJSON(data []byte) error {
	type Alias OAuthFlow
	if err := json.Unmarshal(data, (*Alias)(of)); err != nil {
		return err
	}
	of.Extra = jsonhelpers.ExtractExtensions(data)
	return nil
}
