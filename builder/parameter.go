package builder

import (
	"github.com/erraggy/oastools/parser"
)

// AddParameter adds a reusable parameter to components.parameters.
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

// ParameterRef returns a reference to a named parameter in components.
func ParameterRef(name string) string {
	return "#/components/parameters/" + name
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
