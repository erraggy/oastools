package builder

import (
	"github.com/erraggy/oastools/parser"
)

// paramConfig holds configuration for parameter building.
type paramConfig struct {
	description string
	required    bool
	deprecated  bool
	example     any

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

// AddParameter adds a reusable parameter to components.parameters (OAS 3.x)
// or parameters (OAS 2.0).
func (b *Builder) AddParameter(name string, in string, paramName string, paramType any, opts ...ParamOption) *Builder {
	pCfg := &paramConfig{}
	for _, opt := range opts {
		opt(pCfg)
	}

	// Path parameters are always required
	required := pCfg.required
	if in == parser.ParamInPath {
		required = true
	}

	schema := b.generateSchema(paramType)

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
		applyParamConstraintsToParam(param, pCfg)
	}

	b.parameters[name] = param
	return b
}

// applyParamConstraintsToSchema applies parameter constraints to a schema.
// This is used for OAS 3.x where constraints are applied to the schema field.
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

// applyParamConstraintsToParam applies parameter constraints directly to a parameter.
// This is used for OAS 2.0 where constraints are set on the parameter itself.
func applyParamConstraintsToParam(param *parser.Parameter, cfg *paramConfig) {
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
}

// hasParamConstraints returns true if any constraint is set.
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

// parameterRefPrefix returns the appropriate $ref prefix for parameters.
// OAS 2.0 uses "#/parameters/" while OAS 3.x uses "#/components/parameters/".
func (b *Builder) parameterRefPrefix() string {
	if b.version == parser.OASVersion20 {
		return "#/parameters/"
	}
	return "#/components/parameters/"
}

// ParameterRef returns a reference to a named parameter.
// This method returns the version-appropriate ref path.
func (b *Builder) ParameterRef(name string) string {
	return b.parameterRefPrefix() + name
}

// WithParameterRef adds a parameter reference to the operation.
func WithParameterRef(ref string) OperationOption {
	return func(cfg *operationConfig) {
		param := &parser.Parameter{
			Ref: ref,
		}
		cfg.parameters = append(cfg.parameters, param)
	}
}
