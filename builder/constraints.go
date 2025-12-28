package builder

import "github.com/erraggy/oastools/parser"

// constraintTarget is implemented by types that can receive validation constraints.
// Both parser.Schema and parser.Parameter have these fields.
type constraintTarget interface {
	setMinimum(*float64)
	setMaximum(*float64)
	setExclusiveMinimum(any)
	setExclusiveMaximum(any)
	setMultipleOf(*float64)
	setMinLength(*int)
	setMaxLength(*int)
	setPattern(string)
	setMinItems(*int)
	setMaxItems(*int)
	setUniqueItems(bool)
	setEnum([]any)
	setDefault(any)
}

// schemaConstraintAdapter wraps *parser.Schema to implement constraintTarget.
type schemaConstraintAdapter struct {
	s *parser.Schema
}

func (a *schemaConstraintAdapter) setMinimum(v *float64)     { a.s.Minimum = v }
func (a *schemaConstraintAdapter) setMaximum(v *float64)     { a.s.Maximum = v }
func (a *schemaConstraintAdapter) setExclusiveMinimum(v any) { a.s.ExclusiveMinimum = v }
func (a *schemaConstraintAdapter) setExclusiveMaximum(v any) { a.s.ExclusiveMaximum = v }
func (a *schemaConstraintAdapter) setMultipleOf(v *float64)  { a.s.MultipleOf = v }
func (a *schemaConstraintAdapter) setMinLength(v *int)       { a.s.MinLength = v }
func (a *schemaConstraintAdapter) setMaxLength(v *int)       { a.s.MaxLength = v }
func (a *schemaConstraintAdapter) setPattern(v string)       { a.s.Pattern = v }
func (a *schemaConstraintAdapter) setMinItems(v *int)        { a.s.MinItems = v }
func (a *schemaConstraintAdapter) setMaxItems(v *int)        { a.s.MaxItems = v }
func (a *schemaConstraintAdapter) setUniqueItems(v bool)     { a.s.UniqueItems = v }
func (a *schemaConstraintAdapter) setEnum(v []any)           { a.s.Enum = v }
func (a *schemaConstraintAdapter) setDefault(v any)          { a.s.Default = v }

// paramConstraintAdapter wraps *parser.Parameter to implement constraintTarget.
// Note: Parameter.ExclusiveMinimum/Maximum are bool (OAS 2.0), while
// Schema.ExclusiveMinimum/Maximum are any (OAS 3.0/3.1 support bool or number).
type paramConstraintAdapter struct {
	p *parser.Parameter
}

func (a *paramConstraintAdapter) setMinimum(v *float64) { a.p.Minimum = v }
func (a *paramConstraintAdapter) setMaximum(v *float64) { a.p.Maximum = v }
func (a *paramConstraintAdapter) setExclusiveMinimum(v any) {
	if b, ok := v.(bool); ok {
		a.p.ExclusiveMinimum = b
	}
}
func (a *paramConstraintAdapter) setExclusiveMaximum(v any) {
	if b, ok := v.(bool); ok {
		a.p.ExclusiveMaximum = b
	}
}
func (a *paramConstraintAdapter) setMultipleOf(v *float64) { a.p.MultipleOf = v }
func (a *paramConstraintAdapter) setMinLength(v *int)      { a.p.MinLength = v }
func (a *paramConstraintAdapter) setMaxLength(v *int)      { a.p.MaxLength = v }
func (a *paramConstraintAdapter) setPattern(v string)      { a.p.Pattern = v }
func (a *paramConstraintAdapter) setMinItems(v *int)       { a.p.MinItems = v }
func (a *paramConstraintAdapter) setMaxItems(v *int)       { a.p.MaxItems = v }
func (a *paramConstraintAdapter) setUniqueItems(v bool)    { a.p.UniqueItems = v }
func (a *paramConstraintAdapter) setEnum(v []any)          { a.p.Enum = v }
func (a *paramConstraintAdapter) setDefault(v any)         { a.p.Default = v }

// applyConstraintsToTarget applies parameter constraints to any constraint target.
// This is the shared implementation used by both applyParamConstraintsToSchema
// and applyParamConstraintsToParam.
func applyConstraintsToTarget(target constraintTarget, cfg *paramConfig) {
	if cfg.minimum != nil {
		target.setMinimum(cfg.minimum)
	}
	if cfg.maximum != nil {
		target.setMaximum(cfg.maximum)
	}
	if cfg.exclusiveMinimum {
		target.setExclusiveMinimum(cfg.exclusiveMinimum)
	}
	if cfg.exclusiveMaximum {
		target.setExclusiveMaximum(cfg.exclusiveMaximum)
	}
	if cfg.multipleOf != nil {
		target.setMultipleOf(cfg.multipleOf)
	}
	if cfg.minLength != nil {
		target.setMinLength(cfg.minLength)
	}
	if cfg.maxLength != nil {
		target.setMaxLength(cfg.maxLength)
	}
	if cfg.pattern != "" {
		target.setPattern(cfg.pattern)
	}
	if cfg.minItems != nil {
		target.setMinItems(cfg.minItems)
	}
	if cfg.maxItems != nil {
		target.setMaxItems(cfg.maxItems)
	}
	if cfg.uniqueItems {
		target.setUniqueItems(cfg.uniqueItems)
	}
	if len(cfg.enum) > 0 {
		target.setEnum(cfg.enum)
	}
	if cfg.defaultValue != nil {
		target.setDefault(cfg.defaultValue)
	}
}
