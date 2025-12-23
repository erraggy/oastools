package differ

import (
	"github.com/erraggy/oastools/internal/severity"
)

// SubType constants for rule matching
const (
	subTypeDescription = "description"
)

// BreakingChangeRule configures how a specific change type is treated.
type BreakingChangeRule struct {
	// Severity overrides the default severity for this change type.
	// If nil, the default severity is used.
	Severity *Severity

	// Ignore completely ignores this change type (not included in results).
	Ignore bool
}

// BreakingRulesConfig configures which changes are considered breaking
// and their severity levels. Use this to customize breaking change detection
// based on your organization's API compatibility policies.
//
// Example:
//
//	rules := &differ.BreakingRulesConfig{
//	    Operation: &differ.OperationRules{
//	        OperationIDModified: &differ.BreakingChangeRule{
//	            Severity: differ.SeverityPtr(differ.SeverityInfo), // Not breaking for us
//	        },
//	    },
//	    Schema: &differ.SchemaRules{
//	        PropertyRemoved: &differ.BreakingChangeRule{Ignore: true}, // We handle this differently
//	    },
//	}
//	d := differ.New()
//	d.BreakingRules = rules
type BreakingRulesConfig struct {
	// Operation configures rules for operation-level changes
	Operation *OperationRules

	// Parameter configures rules for parameter changes
	Parameter *ParameterRules

	// RequestBody configures rules for request body changes
	RequestBody *RequestBodyRules

	// Response configures rules for response changes
	Response *ResponseRules

	// Schema configures rules for schema changes
	Schema *SchemaRules

	// Security configures rules for security scheme changes
	Security *SecurityRules

	// Server configures rules for server changes
	Server *ServerRules

	// Endpoint configures rules for endpoint (path) changes
	Endpoint *EndpointRules

	// Info configures rules for info object changes
	Info *InfoRules

	// Extension configures rules for extension (x-*) changes
	Extension *ExtensionRules
}

// OperationRules configures rules for HTTP operation changes.
type OperationRules struct {
	// Removed configures the rule for when an operation is removed.
	// Default: SeverityCritical
	Removed *BreakingChangeRule

	// OperationIDModified configures the rule for operationId changes.
	// Default: SeverityWarning
	OperationIDModified *BreakingChangeRule

	// SummaryModified configures the rule for summary changes.
	// Default: SeverityInfo
	SummaryModified *BreakingChangeRule

	// DescriptionModified configures the rule for description changes.
	// Default: SeverityInfo
	DescriptionModified *BreakingChangeRule

	// DeprecatedModified configures the rule for deprecated flag changes.
	// Default: SeverityInfo
	DeprecatedModified *BreakingChangeRule

	// TagsModified configures the rule for operation tags changes.
	// Default: SeverityInfo
	TagsModified *BreakingChangeRule

	// Added configures the rule for when an operation is added.
	// Default: SeverityInfo
	Added *BreakingChangeRule
}

// ParameterRules configures rules for parameter changes.
type ParameterRules struct {
	// Removed configures the rule for when a parameter is removed.
	// Default: SeverityError (required) / SeverityWarning (optional)
	Removed *BreakingChangeRule

	// Added configures the rule for when a parameter is added.
	// Default: SeverityError (required) / SeverityInfo (optional)
	Added *BreakingChangeRule

	// RequiredChanged configures the rule for required field changes.
	// Default: SeverityError (false→true) / SeverityInfo (true→false)
	RequiredChanged *BreakingChangeRule

	// TypeChanged configures the rule for type changes.
	// Default: SeverityError
	TypeChanged *BreakingChangeRule

	// FormatChanged configures the rule for format changes.
	// Default: SeverityWarning
	FormatChanged *BreakingChangeRule

	// StyleChanged configures the rule for style changes.
	// Default: SeverityWarning
	StyleChanged *BreakingChangeRule

	// SchemaChanged configures the rule for schema changes.
	// Default: varies by schema change
	SchemaChanged *BreakingChangeRule

	// DescriptionModified configures the rule for description changes.
	// Default: SeverityInfo
	DescriptionModified *BreakingChangeRule
}

// RequestBodyRules configures rules for request body changes.
type RequestBodyRules struct {
	// Removed configures the rule for when a request body is removed.
	// Default: SeverityError
	Removed *BreakingChangeRule

	// Added configures the rule for when a request body is added.
	// Default: SeverityError (required) / SeverityInfo (optional)
	Added *BreakingChangeRule

	// RequiredChanged configures the rule for required field changes.
	// Default: SeverityError (false→true) / SeverityInfo (true→false)
	RequiredChanged *BreakingChangeRule

	// MediaTypeRemoved configures the rule for when a media type is removed.
	// Default: SeverityWarning
	MediaTypeRemoved *BreakingChangeRule

	// MediaTypeAdded configures the rule for when a media type is added.
	// Default: SeverityInfo
	MediaTypeAdded *BreakingChangeRule

	// SchemaChanged configures the rule for schema changes.
	// Default: varies by schema change
	SchemaChanged *BreakingChangeRule
}

// ResponseRules configures rules for response changes.
type ResponseRules struct {
	// Removed configures the rule for when a response is removed.
	// Default: SeverityError (success codes) / SeverityWarning (error codes)
	Removed *BreakingChangeRule

	// Added configures the rule for when a response is added.
	// Default: SeverityInfo
	Added *BreakingChangeRule

	// DescriptionModified configures the rule for description changes.
	// Default: SeverityInfo
	DescriptionModified *BreakingChangeRule

	// MediaTypeRemoved configures the rule for when a media type is removed.
	// Default: SeverityWarning
	MediaTypeRemoved *BreakingChangeRule

	// MediaTypeAdded configures the rule for when a media type is added.
	// Default: SeverityInfo
	MediaTypeAdded *BreakingChangeRule

	// HeaderRemoved configures the rule for when a header is removed.
	// Default: SeverityWarning
	HeaderRemoved *BreakingChangeRule

	// HeaderAdded configures the rule for when a header is added.
	// Default: SeverityInfo
	HeaderAdded *BreakingChangeRule

	// SchemaChanged configures the rule for schema changes.
	// Default: varies by schema change
	SchemaChanged *BreakingChangeRule
}

// SchemaRules configures rules for schema changes.
type SchemaRules struct {
	// Removed configures the rule for when a schema is removed.
	// Default: SeverityError
	Removed *BreakingChangeRule

	// Added configures the rule for when a schema is added.
	// Default: SeverityInfo
	Added *BreakingChangeRule

	// TypeChanged configures the rule for type changes.
	// Default: SeverityError
	TypeChanged *BreakingChangeRule

	// FormatChanged configures the rule for format changes.
	// Default: SeverityWarning
	FormatChanged *BreakingChangeRule

	// RequiredAdded configures the rule for when a required field is added.
	// Default: SeverityError
	RequiredAdded *BreakingChangeRule

	// RequiredRemoved configures the rule for when a required field is removed.
	// Default: SeverityInfo
	RequiredRemoved *BreakingChangeRule

	// PropertyRemoved configures the rule for when a property is removed.
	// Default: SeverityWarning
	PropertyRemoved *BreakingChangeRule

	// PropertyAdded configures the rule for when a property is added.
	// Default: SeverityInfo
	PropertyAdded *BreakingChangeRule

	// EnumValueRemoved configures the rule for when an enum value is removed.
	// Default: SeverityError
	EnumValueRemoved *BreakingChangeRule

	// EnumValueAdded configures the rule for when an enum value is added.
	// Default: SeverityInfo
	EnumValueAdded *BreakingChangeRule

	// MaximumDecreased configures the rule for when maximum is decreased.
	// Default: SeverityError
	MaximumDecreased *BreakingChangeRule

	// MinimumIncreased configures the rule for when minimum is increased.
	// Default: SeverityError
	MinimumIncreased *BreakingChangeRule

	// MaxLengthDecreased configures the rule for when maxLength is decreased.
	// Default: SeverityError
	MaxLengthDecreased *BreakingChangeRule

	// MinLengthIncreased configures the rule for when minLength is increased.
	// Default: SeverityWarning
	MinLengthIncreased *BreakingChangeRule

	// PatternChanged configures the rule for when pattern is changed.
	// Default: SeverityWarning
	PatternChanged *BreakingChangeRule

	// NullableRemoved configures the rule for when nullable is removed.
	// Default: SeverityError
	NullableRemoved *BreakingChangeRule

	// NullableAdded configures the rule for when nullable is added.
	// Default: SeverityInfo
	NullableAdded *BreakingChangeRule

	// AdditionalPropertiesChanged configures the rule for additionalProperties changes.
	// Default: SeverityWarning
	AdditionalPropertiesChanged *BreakingChangeRule

	// DescriptionModified configures the rule for description changes.
	// Default: SeverityInfo
	DescriptionModified *BreakingChangeRule
}

// SecurityRules configures rules for security scheme changes.
type SecurityRules struct {
	// Removed configures the rule for when a security scheme is removed.
	// Default: SeverityError
	Removed *BreakingChangeRule

	// Added configures the rule for when a security scheme is added.
	// Default: SeverityWarning
	Added *BreakingChangeRule

	// TypeChanged configures the rule for type changes.
	// Default: SeverityError
	TypeChanged *BreakingChangeRule

	// ScopeRemoved configures the rule for when a scope is removed.
	// Default: SeverityWarning
	ScopeRemoved *BreakingChangeRule

	// ScopeAdded configures the rule for when a scope is added.
	// Default: SeverityInfo
	ScopeAdded *BreakingChangeRule
}

// ServerRules configures rules for server changes.
type ServerRules struct {
	// Removed configures the rule for when a server is removed.
	// Default: SeverityWarning
	Removed *BreakingChangeRule

	// Added configures the rule for when a server is added.
	// Default: SeverityInfo
	Added *BreakingChangeRule

	// DescriptionModified configures the rule for description changes.
	// Default: SeverityInfo
	DescriptionModified *BreakingChangeRule

	// VariableChanged configures the rule for server variable changes.
	// Default: SeverityWarning
	VariableChanged *BreakingChangeRule
}

// EndpointRules configures rules for endpoint (path) changes.
type EndpointRules struct {
	// Removed configures the rule for when an endpoint is removed.
	// Default: SeverityCritical
	Removed *BreakingChangeRule

	// Added configures the rule for when an endpoint is added.
	// Default: SeverityInfo
	Added *BreakingChangeRule

	// DescriptionModified configures the rule for description changes.
	// Default: SeverityInfo
	DescriptionModified *BreakingChangeRule
}

// InfoRules configures rules for info object changes.
type InfoRules struct {
	// TitleModified configures the rule for title changes.
	// Default: SeverityInfo
	TitleModified *BreakingChangeRule

	// VersionModified configures the rule for version changes.
	// Default: SeverityInfo
	VersionModified *BreakingChangeRule

	// DescriptionModified configures the rule for description changes.
	// Default: SeverityInfo
	DescriptionModified *BreakingChangeRule
}

// ExtensionRules configures rules for specification extension (x-*) changes.
type ExtensionRules struct {
	// Removed configures the rule for when an extension is removed.
	// Default: SeverityInfo
	Removed *BreakingChangeRule

	// Added configures the rule for when an extension is added.
	// Default: SeverityInfo
	Added *BreakingChangeRule

	// Modified configures the rule for when an extension is modified.
	// Default: SeverityInfo
	Modified *BreakingChangeRule
}

// SeverityPtr is a helper function to create a pointer to a Severity value.
// This is useful when configuring BreakingChangeRule.Severity.
func SeverityPtr(s Severity) *Severity {
	return &s
}

// RuleKey identifies a specific change type for rule lookup.
type RuleKey struct {
	Category   ChangeCategory
	ChangeType ChangeType
	SubType    string // Additional context (e.g., "operationId", "required")
}

// getRule looks up the rule for a specific change type.
// Returns nil if no rule is configured (use default behavior).
func (c *BreakingRulesConfig) getRule(key RuleKey) *BreakingChangeRule {
	if c == nil {
		return nil
	}

	switch key.Category {
	case CategoryOperation:
		return c.getOperationRule(key)
	case CategoryParameter:
		return c.getParameterRule(key)
	case CategoryRequestBody:
		return c.getRequestBodyRule(key)
	case CategoryResponse:
		return c.getResponseRule(key)
	case CategorySchema:
		return c.getSchemaRule(key)
	case CategorySecurity:
		return c.getSecurityRule(key)
	case CategoryServer:
		return c.getServerRule(key)
	case CategoryEndpoint:
		return c.getEndpointRule(key)
	case CategoryInfo:
		return c.getInfoRule(key)
	case CategoryExtension:
		return c.getExtensionRule(key)
	}
	return nil
}

func (c *BreakingRulesConfig) getOperationRule(key RuleKey) *BreakingChangeRule {
	if c.Operation == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		return c.Operation.Removed
	case ChangeTypeAdded:
		return c.Operation.Added
	case ChangeTypeModified:
		switch key.SubType {
		case "operationId":
			return c.Operation.OperationIDModified
		case "summary":
			return c.Operation.SummaryModified
		case subTypeDescription:
			return c.Operation.DescriptionModified
		case "deprecated":
			return c.Operation.DeprecatedModified
		case "tags":
			return c.Operation.TagsModified
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getParameterRule(key RuleKey) *BreakingChangeRule {
	if c.Parameter == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		return c.Parameter.Removed
	case ChangeTypeAdded:
		return c.Parameter.Added
	case ChangeTypeModified:
		switch key.SubType {
		case "required":
			return c.Parameter.RequiredChanged
		case "type":
			return c.Parameter.TypeChanged
		case "format":
			return c.Parameter.FormatChanged
		case "style":
			return c.Parameter.StyleChanged
		case "schema":
			return c.Parameter.SchemaChanged
		case subTypeDescription:
			return c.Parameter.DescriptionModified
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getRequestBodyRule(key RuleKey) *BreakingChangeRule {
	if c.RequestBody == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		if key.SubType == "mediaType" {
			return c.RequestBody.MediaTypeRemoved
		}
		return c.RequestBody.Removed
	case ChangeTypeAdded:
		if key.SubType == "mediaType" {
			return c.RequestBody.MediaTypeAdded
		}
		return c.RequestBody.Added
	case ChangeTypeModified:
		switch key.SubType {
		case "required":
			return c.RequestBody.RequiredChanged
		case "schema":
			return c.RequestBody.SchemaChanged
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getResponseRule(key RuleKey) *BreakingChangeRule {
	if c.Response == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		switch key.SubType {
		case "mediaType":
			return c.Response.MediaTypeRemoved
		case "header":
			return c.Response.HeaderRemoved
		default:
			return c.Response.Removed
		}
	case ChangeTypeAdded:
		switch key.SubType {
		case "mediaType":
			return c.Response.MediaTypeAdded
		case "header":
			return c.Response.HeaderAdded
		default:
			return c.Response.Added
		}
	case ChangeTypeModified:
		switch key.SubType {
		case subTypeDescription:
			return c.Response.DescriptionModified
		case "schema":
			return c.Response.SchemaChanged
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getSchemaRule(key RuleKey) *BreakingChangeRule {
	if c.Schema == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		switch key.SubType {
		case "property":
			return c.Schema.PropertyRemoved
		case "required":
			return c.Schema.RequiredRemoved
		case "enum":
			return c.Schema.EnumValueRemoved
		case "nullable":
			return c.Schema.NullableRemoved
		default:
			return c.Schema.Removed
		}
	case ChangeTypeAdded:
		switch key.SubType {
		case "property":
			return c.Schema.PropertyAdded
		case "required":
			return c.Schema.RequiredAdded
		case "enum":
			return c.Schema.EnumValueAdded
		case "nullable":
			return c.Schema.NullableAdded
		default:
			return c.Schema.Added
		}
	case ChangeTypeModified:
		switch key.SubType {
		case "type":
			return c.Schema.TypeChanged
		case "format":
			return c.Schema.FormatChanged
		case "pattern":
			return c.Schema.PatternChanged
		case "maximum":
			return c.Schema.MaximumDecreased
		case "minimum":
			return c.Schema.MinimumIncreased
		case "maxLength":
			return c.Schema.MaxLengthDecreased
		case "minLength":
			return c.Schema.MinLengthIncreased
		case "additionalProperties":
			return c.Schema.AdditionalPropertiesChanged
		case subTypeDescription:
			return c.Schema.DescriptionModified
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getSecurityRule(key RuleKey) *BreakingChangeRule {
	if c.Security == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		if key.SubType == "scope" {
			return c.Security.ScopeRemoved
		}
		return c.Security.Removed
	case ChangeTypeAdded:
		if key.SubType == "scope" {
			return c.Security.ScopeAdded
		}
		return c.Security.Added
	case ChangeTypeModified:
		if key.SubType == "type" {
			return c.Security.TypeChanged
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getServerRule(key RuleKey) *BreakingChangeRule {
	if c.Server == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		return c.Server.Removed
	case ChangeTypeAdded:
		return c.Server.Added
	case ChangeTypeModified:
		switch key.SubType {
		case subTypeDescription:
			return c.Server.DescriptionModified
		case "variable":
			return c.Server.VariableChanged
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getEndpointRule(key RuleKey) *BreakingChangeRule {
	if c.Endpoint == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		return c.Endpoint.Removed
	case ChangeTypeAdded:
		return c.Endpoint.Added
	case ChangeTypeModified:
		if key.SubType == subTypeDescription {
			return c.Endpoint.DescriptionModified
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getInfoRule(key RuleKey) *BreakingChangeRule {
	if c.Info == nil {
		return nil
	}
	if key.ChangeType == ChangeTypeModified {
		switch key.SubType {
		case "title":
			return c.Info.TitleModified
		case "version":
			return c.Info.VersionModified
		case subTypeDescription:
			return c.Info.DescriptionModified
		}
	}
	return nil
}

func (c *BreakingRulesConfig) getExtensionRule(key RuleKey) *BreakingChangeRule {
	if c.Extension == nil {
		return nil
	}
	switch key.ChangeType {
	case ChangeTypeRemoved:
		return c.Extension.Removed
	case ChangeTypeAdded:
		return c.Extension.Added
	case ChangeTypeModified:
		return c.Extension.Modified
	}
	return nil
}

// ApplyRule applies a rule to the given default severity.
// Returns the (possibly overridden) severity and whether to ignore the change.
func (r *BreakingChangeRule) ApplyRule(defaultSeverity Severity) (Severity, bool) {
	if r == nil {
		return defaultSeverity, false
	}
	if r.Ignore {
		return 0, true
	}
	if r.Severity != nil {
		return *r.Severity, false
	}
	return defaultSeverity, false
}

// DefaultRules returns a BreakingRulesConfig with all default behaviors.
// This is equivalent to not setting any rules.
func DefaultRules() *BreakingRulesConfig {
	return &BreakingRulesConfig{}
}

// StrictRules returns a BreakingRulesConfig that treats more changes as breaking.
// This elevates many warnings to errors.
func StrictRules() *BreakingRulesConfig {
	return &BreakingRulesConfig{
		Operation: &OperationRules{
			OperationIDModified: &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
		},
		Parameter: &ParameterRules{
			FormatChanged: &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
			StyleChanged:  &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
		},
		Schema: &SchemaRules{
			FormatChanged:      &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
			PatternChanged:     &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
			PropertyRemoved:    &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
			MinLengthIncreased: &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
		},
		Security: &SecurityRules{
			Added:        &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
			ScopeRemoved: &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
		},
		Server: &ServerRules{
			Removed:         &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
			VariableChanged: &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityError)},
		},
	}
}

// LenientRules returns a BreakingRulesConfig that treats fewer changes as breaking.
// This downgrades many errors to warnings.
func LenientRules() *BreakingRulesConfig {
	return &BreakingRulesConfig{
		Schema: &SchemaRules{
			EnumValueRemoved: &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityWarning)},
			RequiredAdded:    &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityWarning)},
		},
		Security: &SecurityRules{
			Removed: &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityWarning)},
		},
		Parameter: &ParameterRules{
			RequiredChanged: &BreakingChangeRule{Severity: SeverityPtr(severity.SeverityWarning)},
		},
	}
}
