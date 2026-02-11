package builder

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/oaserrors"
	"github.com/erraggy/oastools/parser"
)

// ConstraintError represents an invalid constraint configuration.
// It provides context about which field failed validation and why.
type ConstraintError struct {
	// Field is the constraint field that failed (e.g., "minimum", "pattern").
	Field string
	// Message describes the constraint violation.
	Message string
	// ParamName is the parameter name where this constraint was applied (optional).
	ParamName string
	// OperationContext describes the operation context (e.g., "POST /users").
	OperationContext string
}

// Error implements the error interface.
func (e *ConstraintError) Error() string {
	if e.ParamName != "" {
		return fmt.Sprintf("constraint error on parameter %q field %s: %s", e.ParamName, e.Field, e.Message)
	}
	return fmt.Sprintf("constraint error on %s: %s", e.Field, e.Message)
}

// HasLocation returns true if this error has location context.
func (e *ConstraintError) HasLocation() bool {
	return e.ParamName != "" || e.OperationContext != ""
}

// Location returns a descriptive location string.
func (e *ConstraintError) Location() string {
	if e.OperationContext != "" && e.ParamName != "" {
		return fmt.Sprintf("%s parameter %q", e.OperationContext, e.ParamName)
	}
	if e.ParamName != "" {
		return fmt.Sprintf("parameter %q", e.ParamName)
	}
	if e.OperationContext != "" {
		return e.OperationContext
	}
	return e.Field
}

// Unwrap returns nil as ConstraintError has no underlying cause.
// This method exists for interface consistency with BuilderError.
func (e *ConstraintError) Unwrap() error {
	return nil
}

// Is reports whether target matches this error type.
// ConstraintError matches oaserrors.ErrConfig for programmatic error handling.
func (e *ConstraintError) Is(target error) bool {
	return target == oaserrors.ErrConfig
}

// paramConfig holds configuration for parameter building.
type paramConfig struct {
	description string
	required    bool
	deprecated  bool
	example     any

	// Type/Format override fields
	typeOverride   string         // Explicit type override (e.g., "string", "integer")
	formatOverride string         // Explicit format override (e.g., "uuid", "email", "date")
	schemaOverride *parser.Schema // Complete schema override (takes precedence)

	// Constraint fields
	minimum          *float64
	maximum          *float64
	exclusiveMinimum bool
	exclusiveMaximum bool
	multipleOf       *float64
	minLength        *int
	maxLength        *int
	pattern          string
	minItems         *int
	maxItems         *int
	uniqueItems      bool
	enum             []any
	defaultValue     any

	// OAS 2.0 specific fields
	allowEmptyValue  bool
	collectionFormat string

	// Extension fields
	extensions map[string]any
}

// ParamOption configures a parameter.
type ParamOption func(*paramConfig)

// WithParamDescription sets the parameter description.
func WithParamDescription(desc string) ParamOption {
	return func(cfg *paramConfig) {
		cfg.description = desc
	}
}

// WithParamRequired sets whether the parameter is required.
func WithParamRequired(required bool) ParamOption {
	return func(cfg *paramConfig) {
		cfg.required = required
	}
}

// WithParamExample sets the parameter example.
func WithParamExample(example any) ParamOption {
	return func(cfg *paramConfig) {
		cfg.example = example
	}
}

// WithParamDeprecated marks the parameter as deprecated.
func WithParamDeprecated(deprecated bool) ParamOption {
	return func(cfg *paramConfig) {
		cfg.deprecated = deprecated
	}
}

// WithParamType sets an explicit OpenAPI type for the parameter.
// This overrides the type that would be inferred from the Go type.
//
// Valid types per OpenAPI specification: "string", "integer", "number",
// "boolean", "array", "object".
//
// Example:
//
//	builder.WithQueryParam("data", []byte{},
//	    builder.WithParamType("string"),
//	    builder.WithParamFormat("byte"),
//	)
func WithParamType(typeName string) ParamOption {
	return func(cfg *paramConfig) {
		cfg.typeOverride = typeName
	}
}

// WithParamFormat sets an explicit OpenAPI format for the parameter.
// This overrides the format that would be inferred from the Go type.
//
// Common formats include: "int32", "int64", "float", "double", "byte",
// "binary", "date", "date-time", "password", "email", "uri", "uuid",
// "hostname", "ipv4", "ipv6".
//
// Example:
//
//	builder.WithQueryParam("user_id", "",
//	    builder.WithParamFormat("uuid"),
//	)
func WithParamFormat(format string) ParamOption {
	return func(cfg *paramConfig) {
		cfg.formatOverride = format
	}
}

// WithParamSchema sets a complete schema for the parameter.
// This takes precedence over type/format inference and the
// WithParamType/WithParamFormat options.
//
// Use this for complex schemas that cannot be easily represented
// with Go types (e.g., oneOf, arrays with specific item constraints).
//
// Example:
//
//	builder.WithQueryParam("ids", nil,
//	    builder.WithParamSchema(&parser.Schema{
//	        Type:  "array",
//	        Items: &parser.Schema{Type: "string", Format: "uuid"},
//	    }),
//	)
func WithParamSchema(schema *parser.Schema) ParamOption {
	return func(cfg *paramConfig) {
		cfg.schemaOverride = schema
	}
}

// WithParamMinimum sets the minimum value for numeric parameters.
func WithParamMinimum(min float64) ParamOption {
	return func(cfg *paramConfig) {
		cfg.minimum = &min
	}
}

// WithParamMaximum sets the maximum value for numeric parameters.
func WithParamMaximum(max float64) ParamOption {
	return func(cfg *paramConfig) {
		cfg.maximum = &max
	}
}

// WithParamExclusiveMinimum sets whether the minimum is exclusive.
func WithParamExclusiveMinimum(exclusive bool) ParamOption {
	return func(cfg *paramConfig) {
		cfg.exclusiveMinimum = exclusive
	}
}

// WithParamExclusiveMaximum sets whether the maximum is exclusive.
func WithParamExclusiveMaximum(exclusive bool) ParamOption {
	return func(cfg *paramConfig) {
		cfg.exclusiveMaximum = exclusive
	}
}

// WithParamMultipleOf sets the multipleOf constraint for numeric parameters.
func WithParamMultipleOf(value float64) ParamOption {
	return func(cfg *paramConfig) {
		cfg.multipleOf = &value
	}
}

// WithParamMinLength sets the minimum length for string parameters.
func WithParamMinLength(min int) ParamOption {
	return func(cfg *paramConfig) {
		cfg.minLength = &min
	}
}

// WithParamMaxLength sets the maximum length for string parameters.
func WithParamMaxLength(max int) ParamOption {
	return func(cfg *paramConfig) {
		cfg.maxLength = &max
	}
}

// WithParamPattern sets the pattern (regex) for string parameters.
func WithParamPattern(pattern string) ParamOption {
	return func(cfg *paramConfig) {
		cfg.pattern = pattern
	}
}

// WithParamMinItems sets the minimum number of items for array parameters.
func WithParamMinItems(min int) ParamOption {
	return func(cfg *paramConfig) {
		cfg.minItems = &min
	}
}

// WithParamMaxItems sets the maximum number of items for array parameters.
func WithParamMaxItems(max int) ParamOption {
	return func(cfg *paramConfig) {
		cfg.maxItems = &max
	}
}

// WithParamUniqueItems sets whether array items must be unique.
func WithParamUniqueItems(unique bool) ParamOption {
	return func(cfg *paramConfig) {
		cfg.uniqueItems = unique
	}
}

// WithParamEnum sets the allowed values for the parameter.
func WithParamEnum(values ...any) ParamOption {
	return func(cfg *paramConfig) {
		cfg.enum = values
	}
}

// WithParamDefault sets the default value for the parameter.
func WithParamDefault(value any) ParamOption {
	return func(cfg *paramConfig) {
		cfg.defaultValue = value
	}
}

// WithParamAllowEmptyValue sets whether the parameter allows empty values.
// This is only applicable to OAS 2.0 specifications for query and formData parameters.
// Setting this to true allows sending a parameter with an empty value.
//
// Example:
//
//	builder.WithQueryParam("filter", "",
//	    builder.WithParamAllowEmptyValue(true),
//	)
func WithParamAllowEmptyValue(allow bool) ParamOption {
	return func(cfg *paramConfig) {
		cfg.allowEmptyValue = allow
	}
}

// WithParamCollectionFormat sets the collection format for array parameters.
// This is only applicable to OAS 2.0 specifications.
//
// Valid values are:
//   - "csv": comma separated values (default) - foo,bar
//   - "ssv": space separated values - foo bar
//   - "tsv": tab separated values - foo\tbar
//   - "pipes": pipe separated values - foo|bar
//   - "multi": corresponds to multiple parameter instances - foo=bar&foo=baz
//
// Example:
//
//	builder.WithQueryParam("tags", []string{},
//	    builder.WithParamCollectionFormat("csv"),
//	)
func WithParamCollectionFormat(format string) ParamOption {
	return func(cfg *paramConfig) {
		cfg.collectionFormat = format
	}
}

// WithParamExtension adds a vendor extension (x-* field) to the parameter.
// The key must start with "x-" as per the OpenAPI specification.
// Extensions are preserved in both OAS 2.0 and OAS 3.x output.
//
// Example:
//
//	builder.WithQueryParam("limit", 0,
//	    builder.WithParamExtension("x-example-values", []int{10, 25, 50}),
//	)
func WithParamExtension(key string, value any) ParamOption {
	return func(cfg *paramConfig) {
		if cfg.extensions == nil {
			cfg.extensions = make(map[string]any)
		}
		cfg.extensions[key] = value
	}
}

// AddParameter adds a reusable parameter to components.parameters (OAS 3.x)
// or parameters (OAS 2.0).
//
// Constraint validation is performed and any errors are accumulated in the builder.
// Use BuildOAS2() or BuildOAS3() to check for accumulated errors.
func (b *Builder) AddParameter(name string, in string, paramName string, paramType any, opts ...ParamOption) *Builder {
	pCfg := &paramConfig{}
	for _, opt := range opts {
		opt(pCfg)
	}

	// Validate constraints
	if err := validateParamConstraints(pCfg); err != nil {
		b.errors = append(b.errors, err)
		return b
	}

	// Path parameters are always required
	required := pCfg.required
	if in == parser.ParamInPath {
		required = true
	}

	// Generate schema from Go type
	schema := b.generateSchema(paramType)

	// Apply type/format overrides (schemaOverride takes precedence)
	schema = applyTypeFormatOverrides(schema, pCfg)

	// Apply constraints to schema for OAS 3.x
	if b.version != parser.OASVersion20 {
		schema = applyParamConstraintsToSchema(schema, pCfg)
	}

	param := &parser.Parameter{
		Name:        paramName,
		In:          in,
		Description: pCfg.description,
		Required:    required,
		Deprecated:  pCfg.deprecated,
		Schema:      schema,
		Example:     pCfg.example,
	}

	// Apply constraints directly to parameter for OAS 2.0
	if b.version == parser.OASVersion20 {
		// Apply type/format to parameter-level fields for OAS 2.0
		applyTypeFormatOverridesToOAS2Param(param, schema, pCfg)
		applyParamConstraintsToParam(param, pCfg)
	} else {
		// OAS 3.x: Extensions are still applied to the parameter (not schema)
		if len(pCfg.extensions) > 0 {
			param.Extra = pCfg.extensions
		}
	}

	b.parameters[name] = param
	return b
}

// validateParamConstraints validates that parameter constraints are logically consistent.
// It checks for:
//   - minimum <= maximum (if both set)
//   - minLength <= maxLength (if both set)
//   - minItems <= maxItems (if both set)
//   - non-negative values for length and items constraints
//   - valid regex pattern syntax
//   - positive multipleOf value
//
// All validation errors are collected and returned as a joined error using errors.Join.
func validateParamConstraints(cfg *paramConfig) error {
	var errs []error

	// Validate min/max numeric bounds
	if cfg.minimum != nil && cfg.maximum != nil && *cfg.minimum > *cfg.maximum {
		errs = append(errs, &ConstraintError{
			Field:   "minimum/maximum",
			Message: fmt.Sprintf("minimum (%v) cannot be greater than maximum (%v)", *cfg.minimum, *cfg.maximum),
		})
	}

	// Validate minLength/maxLength bounds
	if cfg.minLength != nil && cfg.maxLength != nil && *cfg.minLength > *cfg.maxLength {
		errs = append(errs, &ConstraintError{
			Field:   "minLength/maxLength",
			Message: fmt.Sprintf("minLength (%d) cannot be greater than maxLength (%d)", *cfg.minLength, *cfg.maxLength),
		})
	}

	// Validate non-negative minLength
	if cfg.minLength != nil && *cfg.minLength < 0 {
		errs = append(errs, &ConstraintError{
			Field:   "minLength",
			Message: fmt.Sprintf("minLength (%d) cannot be negative", *cfg.minLength),
		})
	}

	// Validate non-negative maxLength
	if cfg.maxLength != nil && *cfg.maxLength < 0 {
		errs = append(errs, &ConstraintError{
			Field:   "maxLength",
			Message: fmt.Sprintf("maxLength (%d) cannot be negative", *cfg.maxLength),
		})
	}

	// Validate minItems/maxItems bounds
	if cfg.minItems != nil && cfg.maxItems != nil && *cfg.minItems > *cfg.maxItems {
		errs = append(errs, &ConstraintError{
			Field:   "minItems/maxItems",
			Message: fmt.Sprintf("minItems (%d) cannot be greater than maxItems (%d)", *cfg.minItems, *cfg.maxItems),
		})
	}

	// Validate non-negative minItems
	if cfg.minItems != nil && *cfg.minItems < 0 {
		errs = append(errs, &ConstraintError{
			Field:   "minItems",
			Message: fmt.Sprintf("minItems (%d) cannot be negative", *cfg.minItems),
		})
	}

	// Validate non-negative maxItems
	if cfg.maxItems != nil && *cfg.maxItems < 0 {
		errs = append(errs, &ConstraintError{
			Field:   "maxItems",
			Message: fmt.Sprintf("maxItems (%d) cannot be negative", *cfg.maxItems),
		})
	}

	// Validate positive multipleOf
	if cfg.multipleOf != nil && *cfg.multipleOf <= 0 {
		errs = append(errs, &ConstraintError{
			Field:   "multipleOf",
			Message: fmt.Sprintf("multipleOf (%v) must be greater than 0", *cfg.multipleOf),
		})
	}

	// Validate regex pattern syntax
	if cfg.pattern != "" {
		if _, err := regexp.Compile(cfg.pattern); err != nil {
			errs = append(errs, &ConstraintError{
				Field:   "pattern",
				Message: fmt.Sprintf("invalid regex pattern: %v", err),
			})
		}
	}

	return errors.Join(errs...)
}

// applyParamConstraintsToSchema applies parameter constraints to a schema.
// This is used for OAS 3.x where constraints are applied to the schema field.
//
// The function creates a copy of the schema to avoid mutating shared schema references.
// If no constraints are set, the original schema is returned unchanged.
//
// Parameters:
//   - schema: The schema to apply constraints to (may be nil)
//   - cfg: The parameter configuration containing constraint values
//
// Returns the schema with constraints applied, or nil if the input schema is nil.
func applyParamConstraintsToSchema(schema *parser.Schema, cfg *paramConfig) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Check if any constraints are set
	if !hasParamConstraints(cfg) {
		return schema
	}

	// Create a copy if we're modifying a referenced schema
	result := copySchema(schema)

	// Apply constraints directly to schema fields
	if cfg.minimum != nil {
		result.Minimum = cfg.minimum
	}
	if cfg.maximum != nil {
		result.Maximum = cfg.maximum
	}
	if cfg.exclusiveMinimum {
		result.ExclusiveMinimum = cfg.exclusiveMinimum
	}
	if cfg.exclusiveMaximum {
		result.ExclusiveMaximum = cfg.exclusiveMaximum
	}
	if cfg.multipleOf != nil {
		result.MultipleOf = cfg.multipleOf
	}
	if cfg.minLength != nil {
		result.MinLength = cfg.minLength
	}
	if cfg.maxLength != nil {
		result.MaxLength = cfg.maxLength
	}
	if cfg.pattern != "" {
		result.Pattern = cfg.pattern
	}
	if cfg.minItems != nil {
		result.MinItems = cfg.minItems
	}
	if cfg.maxItems != nil {
		result.MaxItems = cfg.maxItems
	}
	if cfg.uniqueItems {
		result.UniqueItems = cfg.uniqueItems
	}
	if len(cfg.enum) > 0 {
		result.Enum = cfg.enum
	}
	if cfg.defaultValue != nil {
		result.Default = cfg.defaultValue
	}

	return result
}

// applyTypeFormatOverrides applies explicit type and format overrides to a schema.
// This function handles the precedence rules:
//  1. schemaOverride takes highest precedence (returns as-is)
//  2. typeOverride replaces the inferred type
//  3. formatOverride replaces the inferred format
//
// The function creates a copy of the schema to avoid mutating shared references.
// If no overrides are set, the original schema is returned unchanged.
//
// Parameters:
//   - schema: The schema to apply overrides to (may be nil)
//   - cfg: The parameter configuration containing override values
//
// Returns the schema with overrides applied, or the schemaOverride if set.
func applyTypeFormatOverrides(schema *parser.Schema, cfg *paramConfig) *parser.Schema {
	// Schema override takes highest precedence
	if cfg.schemaOverride != nil {
		return cfg.schemaOverride
	}

	// If no type/format overrides, return original
	if cfg.typeOverride == "" && cfg.formatOverride == "" {
		return schema
	}

	// Handle nil schema (for cases like WithQueryParam("x", nil, WithParamSchema(...)))
	if schema == nil {
		schema = &parser.Schema{}
	}

	// Create a copy to avoid mutating shared references
	result := copySchema(schema)

	// Apply type override
	if cfg.typeOverride != "" {
		result.Type = cfg.typeOverride
	}

	// Apply format override
	if cfg.formatOverride != "" {
		result.Format = cfg.formatOverride
	}

	return result
}

// applyTypeFormatOverridesToOAS2Param applies type/format to an OAS 2.0 parameter.
// In OAS 2.0, type and format are top-level parameter fields (not nested in schema).
// This function copies type/format from the schema and applies any overrides.
//
// Precedence:
//  1. schemaOverride.Type/Format takes highest precedence
//  2. typeOverride/formatOverride replace the inferred values
//  3. schema.Type/Format (inferred from Go type) is the default
//
// The schema parameter provides the inferred type/format from Go type reflection.
// For Schema.Type, we perform type assertion since it may be any in OAS 3.1+.
func applyTypeFormatOverridesToOAS2Param(param *parser.Parameter, schema *parser.Schema, cfg *paramConfig) {
	// Start with inferred type/format from schema
	if schema != nil {
		if typeStr := schemautil.GetPrimaryType(schema); typeStr != "" {
			param.Type = typeStr
		}
		param.Format = schema.Format
	}

	// Apply overrides (schemaOverride takes precedence over individual overrides)
	if cfg.schemaOverride != nil {
		// Schema.Type is any to support OAS 3.1 array types
		// For OAS 2.0, extract the primary (non-null) type
		if typeStr := schemautil.GetPrimaryType(cfg.schemaOverride); typeStr != "" {
			param.Type = typeStr
		}
		param.Format = cfg.schemaOverride.Format
	} else {
		if cfg.typeOverride != "" {
			param.Type = cfg.typeOverride
		}
		if cfg.formatOverride != "" {
			param.Format = cfg.formatOverride
		}
	}
}

// applyParamConstraintsToParam applies parameter constraints directly to a parameter.
// This is used for OAS 2.0 where constraints are set on the parameter itself,
// rather than on a nested schema field.
//
// In OAS 2.0, parameters have their own constraint fields (minimum, maximum,
// minLength, etc.) that are applied directly, whereas in OAS 3.x these
// constraints are placed on the parameter's Schema object.
//
// Parameters:
//   - param: The parameter to apply constraints to
//   - cfg: The parameter configuration containing constraint values
func applyParamConstraintsToParam(param *parser.Parameter, cfg *paramConfig) {
	// Apply constraints directly to parameter fields
	if cfg.minimum != nil {
		param.Minimum = cfg.minimum
	}
	if cfg.maximum != nil {
		param.Maximum = cfg.maximum
	}
	if cfg.exclusiveMinimum {
		param.ExclusiveMinimum = cfg.exclusiveMinimum
	}
	if cfg.exclusiveMaximum {
		param.ExclusiveMaximum = cfg.exclusiveMaximum
	}
	if cfg.multipleOf != nil {
		param.MultipleOf = cfg.multipleOf
	}
	if cfg.minLength != nil {
		param.MinLength = cfg.minLength
	}
	if cfg.maxLength != nil {
		param.MaxLength = cfg.maxLength
	}
	if cfg.pattern != "" {
		param.Pattern = cfg.pattern
	}
	if cfg.minItems != nil {
		param.MinItems = cfg.minItems
	}
	if cfg.maxItems != nil {
		param.MaxItems = cfg.maxItems
	}
	if cfg.uniqueItems {
		param.UniqueItems = cfg.uniqueItems
	}
	if len(cfg.enum) > 0 {
		param.Enum = cfg.enum
	}
	if cfg.defaultValue != nil {
		param.Default = cfg.defaultValue
	}

	// OAS 2.0 specific fields
	if cfg.allowEmptyValue {
		param.AllowEmptyValue = cfg.allowEmptyValue
	}
	if cfg.collectionFormat != "" {
		param.CollectionFormat = cfg.collectionFormat
	}

	// Extensions (applicable to all OAS versions)
	if len(cfg.extensions) > 0 {
		param.Extra = cfg.extensions
	}
}

// hasParamConstraints checks if any constraint field is set in the parameter configuration.
// This is used to determine whether schema copying is necessary before applying constraints,
// avoiding unnecessary allocations when no constraints are specified.
//
// Returns true if any of the following are set:
//   - Numeric constraints: minimum, maximum, exclusiveMinimum, exclusiveMaximum, multipleOf
//   - String constraints: minLength, maxLength, pattern
//   - Array constraints: minItems, maxItems, uniqueItems
//   - Value constraints: enum, defaultValue
func hasParamConstraints(cfg *paramConfig) bool {
	return cfg.minimum != nil || cfg.maximum != nil ||
		cfg.exclusiveMinimum || cfg.exclusiveMaximum ||
		cfg.multipleOf != nil ||
		cfg.minLength != nil || cfg.maxLength != nil ||
		cfg.pattern != "" ||
		cfg.minItems != nil || cfg.maxItems != nil ||
		cfg.uniqueItems ||
		len(cfg.enum) > 0 ||
		cfg.defaultValue != nil
}

// ParameterRef returns a reference to a named parameter.
// This method returns the version-appropriate ref path.
func (b *Builder) ParameterRef(name string) string {
	return pathutil.ParameterRef(name, b.version == parser.OASVersion20)
}

// WithParameterRef adds a parameter reference to the operation.
func WithParameterRef(ref string) OperationOption {
	return func(cfg *operationConfig) {
		cfg.parameters = append(cfg.parameters, &parameterBuilder{
			param: &parser.Parameter{
				Ref: ref,
			},
		})
	}
}
