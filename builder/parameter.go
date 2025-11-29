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

	param := &parser.Parameter{
		Name:        paramName,
		In:          in,
		Description: pCfg.description,
		Required:    required,
		Deprecated:  pCfg.deprecated,
		Schema:      schema,
		Example:     pCfg.example,
	}

	b.parameters[name] = param
	return b
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
