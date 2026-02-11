package builder

import "github.com/erraggy/oastools/parser"

// processParameters processes a slice of parameter builders and returns the resulting parameters.
// This is the shared implementation used by both operation and webhook builders.
func (b *Builder) processParameters(paramBuilders []*parameterBuilder) []*parser.Parameter {
	parameters := make([]*parser.Parameter, 0, len(paramBuilders))
	for _, paramBuilder := range paramBuilders {
		param := paramBuilder.param

		// Generate schema if type is provided
		if paramBuilder.pType != nil {
			param.Schema = b.generateSchema(paramBuilder.pType)
		}

		// Apply constraints from config if present
		if paramBuilder.config != nil {
			pCfg := paramBuilder.config
			// Validate constraints
			if err := validateParamConstraints(pCfg); err != nil {
				b.errors = append(b.errors, err)
				continue
			}

			// Apply type/format overrides (schemaOverride takes precedence)
			param.Schema = applyTypeFormatOverrides(param.Schema, pCfg)

			if b.version != parser.OASVersion20 {
				// OAS 3.x: Apply constraints to schema
				param.Schema = applyParamConstraintsToSchema(param.Schema, pCfg)
				// Extensions are still applied to the parameter (not schema)
				if len(pCfg.extensions) > 0 {
					param.Extra = pCfg.extensions
				}
			} else {
				// OAS 2.0: Apply type/format to parameter-level fields
				applyTypeFormatOverridesToOAS2Param(param, param.Schema, pCfg)
				// Apply constraints directly to parameter
				applyParamConstraintsToParam(param, pCfg)
			}
		}

		parameters = append(parameters, param)
	}
	return parameters
}
