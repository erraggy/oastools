package parser

// Parameter describes a single operation parameter.
//
// Name and In are required by the specification but are still tagged omitempty:
// a $ref parameter arrives from the parser with every sibling field empty, and
// emitting name: "" and in: "" alongside the $ref corrupts the object on every
// round trip. An inline parameter that genuinely lacks them is invalid either
// way, and the validator reports it — the serializer is not the right place to
// surface that defect.
type Parameter struct {
	Ref         string `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`
	In          string `yaml:"in,omitempty" json:"in,omitempty"` // "query", "header", "path", "cookie" (OAS 3.0+), "formData", "body" (OAS 2.0)
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
	Deprecated  bool   `yaml:"deprecated,omitempty" json:"deprecated,omitempty"` // OAS 3.0+

	// OAS 3.0+ fields
	Style   string `yaml:"style,omitempty" json:"style,omitempty"`
	Explode *bool  `yaml:"explode,omitempty" json:"explode,omitempty"`

	// AllowReserved lets RFC 3986 reserved characters pass through a parameter
	// value unencoded. Where it may legally appear depends on `in` and `style`;
	// the validator's allowReservedPermitted holds the per-version table.
	//
	// Note: the specification constrains the field's presence rather than its
	// value, which a bool cannot express. An illegal `allowReserved: false`
	// decodes the same as an absent field, so oastools catches the violation
	// only when the value is true. Changing the type would break the v1 API.
	AllowReserved bool                  `yaml:"allowReserved,omitempty" json:"allowReserved,omitempty"`
	Schema        *Schema               `yaml:"schema,omitempty" json:"schema,omitempty"`
	Example       any                   `yaml:"example,omitempty" json:"example,omitempty"`
	Examples      map[string]*Example   `yaml:"examples,omitempty" json:"examples,omitempty"`
	Content       map[string]*MediaType `yaml:"content,omitempty" json:"content,omitempty"`

	// OAS 2.0 fields
	Type             string   `yaml:"type,omitempty" json:"type,omitempty"`                         // OAS 2.0
	Format           string   `yaml:"format,omitempty" json:"format,omitempty"`                     // OAS 2.0
	AllowEmptyValue  bool     `yaml:"allowEmptyValue,omitempty" json:"allowEmptyValue,omitempty"`   // OAS 2.0
	Items            *Items   `yaml:"items,omitempty" json:"items,omitempty"`                       // OAS 2.0
	CollectionFormat string   `yaml:"collectionFormat,omitempty" json:"collectionFormat,omitempty"` // OAS 2.0
	Default          any      `yaml:"default,omitempty" json:"default,omitempty"`                   // OAS 2.0
	Maximum          *float64 `yaml:"maximum,omitempty" json:"maximum,omitempty"`                   // OAS 2.0
	ExclusiveMaximum bool     `yaml:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty"` // OAS 2.0
	Minimum          *float64 `yaml:"minimum,omitempty" json:"minimum,omitempty"`                   // OAS 2.0
	ExclusiveMinimum bool     `yaml:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty"` // OAS 2.0
	MaxLength        *int     `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`               // OAS 2.0
	MinLength        *int     `yaml:"minLength,omitempty" json:"minLength,omitempty"`               // OAS 2.0
	Pattern          string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`                   // OAS 2.0
	MaxItems         *int     `yaml:"maxItems,omitempty" json:"maxItems,omitempty"`                 // OAS 2.0
	MinItems         *int     `yaml:"minItems,omitempty" json:"minItems,omitempty"`                 // OAS 2.0
	UniqueItems      bool     `yaml:"uniqueItems,omitempty" json:"uniqueItems,omitempty"`           // OAS 2.0
	Enum             []any    `yaml:"enum,omitempty" json:"enum,omitempty"`                         // OAS 2.0
	MultipleOf       *float64 `yaml:"multipleOf,omitempty" json:"multipleOf,omitempty"`             // OAS 2.0

	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Items represents items object for array parameters (OAS 2.0)
type Items struct {
	Type             string         `yaml:"type" json:"type"`
	Format           string         `yaml:"format,omitempty" json:"format,omitempty"`
	Items            *Items         `yaml:"items,omitempty" json:"items,omitempty"`
	CollectionFormat string         `yaml:"collectionFormat,omitempty" json:"collectionFormat,omitempty"`
	Default          any            `yaml:"default,omitempty" json:"default,omitempty"`
	Maximum          *float64       `yaml:"maximum,omitempty" json:"maximum,omitempty"`
	ExclusiveMaximum bool           `yaml:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty"`
	Minimum          *float64       `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	ExclusiveMinimum bool           `yaml:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty"`
	MaxLength        *int           `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	MinLength        *int           `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	Pattern          string         `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	MaxItems         *int           `yaml:"maxItems,omitempty" json:"maxItems,omitempty"`
	MinItems         *int           `yaml:"minItems,omitempty" json:"minItems,omitempty"`
	UniqueItems      bool           `yaml:"uniqueItems,omitempty" json:"uniqueItems,omitempty"`
	Enum             []any          `yaml:"enum,omitempty" json:"enum,omitempty"`
	MultipleOf       *float64       `yaml:"multipleOf,omitempty" json:"multipleOf,omitempty"`
	Extra            map[string]any `yaml:",inline" json:"-"`
}

// RequestBody describes a single request body (OAS 3.0+).
//
// Content is tagged omitempty for the same reason as [Parameter.Name]: a $ref
// request body would otherwise serialize with a null content sibling.
type RequestBody struct {
	Ref         string                `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Description string                `yaml:"description,omitempty" json:"description,omitempty"`
	Content     map[string]*MediaType `yaml:"content,omitempty" json:"content,omitempty"`
	Required    bool                  `yaml:"required,omitempty" json:"required,omitempty"`
	Extra       map[string]any        `yaml:",inline" json:"-"`
}

// Header represents a header object
type Header struct {
	Ref         string `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
	Deprecated  bool   `yaml:"deprecated,omitempty" json:"deprecated,omitempty"` // OAS 3.0+

	// OAS 3.0+ fields
	Style    string                `yaml:"style,omitempty" json:"style,omitempty"`
	Explode  *bool                 `yaml:"explode,omitempty" json:"explode,omitempty"`
	Schema   *Schema               `yaml:"schema,omitempty" json:"schema,omitempty"`
	Example  any                   `yaml:"example,omitempty" json:"example,omitempty"`
	Examples map[string]*Example   `yaml:"examples,omitempty" json:"examples,omitempty"`
	Content  map[string]*MediaType `yaml:"content,omitempty" json:"content,omitempty"`

	// OAS 2.0 fields
	Type             string   `yaml:"type,omitempty" json:"type,omitempty"`                         // OAS 2.0
	Format           string   `yaml:"format,omitempty" json:"format,omitempty"`                     // OAS 2.0
	Items            *Items   `yaml:"items,omitempty" json:"items,omitempty"`                       // OAS 2.0
	CollectionFormat string   `yaml:"collectionFormat,omitempty" json:"collectionFormat,omitempty"` // OAS 2.0
	Default          any      `yaml:"default,omitempty" json:"default,omitempty"`                   // OAS 2.0
	Maximum          *float64 `yaml:"maximum,omitempty" json:"maximum,omitempty"`                   // OAS 2.0
	ExclusiveMaximum bool     `yaml:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty"` // OAS 2.0
	Minimum          *float64 `yaml:"minimum,omitempty" json:"minimum,omitempty"`                   // OAS 2.0
	ExclusiveMinimum bool     `yaml:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty"` // OAS 2.0
	MaxLength        *int     `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`               // OAS 2.0
	MinLength        *int     `yaml:"minLength,omitempty" json:"minLength,omitempty"`               // OAS 2.0
	Pattern          string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`                   // OAS 2.0
	MaxItems         *int     `yaml:"maxItems,omitempty" json:"maxItems,omitempty"`                 // OAS 2.0
	MinItems         *int     `yaml:"minItems,omitempty" json:"minItems,omitempty"`                 // OAS 2.0
	UniqueItems      bool     `yaml:"uniqueItems,omitempty" json:"uniqueItems,omitempty"`           // OAS 2.0
	Enum             []any    `yaml:"enum,omitempty" json:"enum,omitempty"`                         // OAS 2.0
	MultipleOf       *float64 `yaml:"multipleOf,omitempty" json:"multipleOf,omitempty"`             // OAS 2.0

	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]any `yaml:",inline" json:"-"`
}
