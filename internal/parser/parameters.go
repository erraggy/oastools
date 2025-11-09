package parser

// Parameter describes a single operation parameter
type Parameter struct {
	Ref         string `yaml:"$ref,omitempty"`
	Name        string `yaml:"name"`
	In          string `yaml:"in"` // "query", "header", "path", "cookie" (OAS 3.0+), "formData", "body" (OAS 2.0)
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
	Deprecated  bool   `yaml:"deprecated,omitempty"` // OAS 3.0+

	// OAS 3.0+ fields
	Style         string                `yaml:"style,omitempty"`
	Explode       *bool                 `yaml:"explode,omitempty"`
	AllowReserved bool                  `yaml:"allowReserved,omitempty"`
	Schema        *Schema               `yaml:"schema,omitempty"`
	Example       interface{}           `yaml:"example,omitempty"`
	Examples      map[string]*Example   `yaml:"examples,omitempty"`
	Content       map[string]*MediaType `yaml:"content,omitempty"`

	// OAS 2.0 fields
	Type             string        `yaml:"type,omitempty"`             // OAS 2.0
	Format           string        `yaml:"format,omitempty"`           // OAS 2.0
	AllowEmptyValue  bool          `yaml:"allowEmptyValue,omitempty"`  // OAS 2.0
	Items            *Items        `yaml:"items,omitempty"`            // OAS 2.0
	CollectionFormat string        `yaml:"collectionFormat,omitempty"` // OAS 2.0
	Default          interface{}   `yaml:"default,omitempty"`          // OAS 2.0
	Maximum          *float64      `yaml:"maximum,omitempty"`          // OAS 2.0
	ExclusiveMaximum bool          `yaml:"exclusiveMaximum,omitempty"` // OAS 2.0
	Minimum          *float64      `yaml:"minimum,omitempty"`          // OAS 2.0
	ExclusiveMinimum bool          `yaml:"exclusiveMinimum,omitempty"` // OAS 2.0
	MaxLength        *int          `yaml:"maxLength,omitempty"`        // OAS 2.0
	MinLength        *int          `yaml:"minLength,omitempty"`        // OAS 2.0
	Pattern          string        `yaml:"pattern,omitempty"`          // OAS 2.0
	MaxItems         *int          `yaml:"maxItems,omitempty"`         // OAS 2.0
	MinItems         *int          `yaml:"minItems,omitempty"`         // OAS 2.0
	UniqueItems      bool          `yaml:"uniqueItems,omitempty"`      // OAS 2.0
	Enum             []interface{} `yaml:"enum,omitempty"`             // OAS 2.0
	MultipleOf       *float64      `yaml:"multipleOf,omitempty"`       // OAS 2.0

	Extra map[string]interface{} `yaml:",inline"`
}

// Items represents items object for array parameters (OAS 2.0)
type Items struct {
	Type             string                 `yaml:"type"`
	Format           string                 `yaml:"format,omitempty"`
	Items            *Items                 `yaml:"items,omitempty"`
	CollectionFormat string                 `yaml:"collectionFormat,omitempty"`
	Default          interface{}            `yaml:"default,omitempty"`
	Maximum          *float64               `yaml:"maximum,omitempty"`
	ExclusiveMaximum bool                   `yaml:"exclusiveMaximum,omitempty"`
	Minimum          *float64               `yaml:"minimum,omitempty"`
	ExclusiveMinimum bool                   `yaml:"exclusiveMinimum,omitempty"`
	MaxLength        *int                   `yaml:"maxLength,omitempty"`
	MinLength        *int                   `yaml:"minLength,omitempty"`
	Pattern          string                 `yaml:"pattern,omitempty"`
	MaxItems         *int                   `yaml:"maxItems,omitempty"`
	MinItems         *int                   `yaml:"minItems,omitempty"`
	UniqueItems      bool                   `yaml:"uniqueItems,omitempty"`
	Enum             []interface{}          `yaml:"enum,omitempty"`
	MultipleOf       *float64               `yaml:"multipleOf,omitempty"`
	Extra            map[string]interface{} `yaml:",inline"`
}

// RequestBody describes a single request body (OAS 3.0+)
type RequestBody struct {
	Ref         string                 `yaml:"$ref,omitempty"`
	Description string                 `yaml:"description,omitempty"`
	Content     map[string]*MediaType  `yaml:"content"`
	Required    bool                   `yaml:"required,omitempty"`
	Extra       map[string]interface{} `yaml:",inline"`
}

// Header represents a header object
type Header struct {
	Ref         string `yaml:"$ref,omitempty"`
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
	Deprecated  bool   `yaml:"deprecated,omitempty"` // OAS 3.0+

	// OAS 3.0+ fields
	Style    string                `yaml:"style,omitempty"`
	Explode  *bool                 `yaml:"explode,omitempty"`
	Schema   *Schema               `yaml:"schema,omitempty"`
	Example  interface{}           `yaml:"example,omitempty"`
	Examples map[string]*Example   `yaml:"examples,omitempty"`
	Content  map[string]*MediaType `yaml:"content,omitempty"`

	// OAS 2.0 fields
	Type             string        `yaml:"type,omitempty"`             // OAS 2.0
	Format           string        `yaml:"format,omitempty"`           // OAS 2.0
	Items            *Items        `yaml:"items,omitempty"`            // OAS 2.0
	CollectionFormat string        `yaml:"collectionFormat,omitempty"` // OAS 2.0
	Default          interface{}   `yaml:"default,omitempty"`          // OAS 2.0
	Maximum          *float64      `yaml:"maximum,omitempty"`          // OAS 2.0
	ExclusiveMaximum bool          `yaml:"exclusiveMaximum,omitempty"` // OAS 2.0
	Minimum          *float64      `yaml:"minimum,omitempty"`          // OAS 2.0
	ExclusiveMinimum bool          `yaml:"exclusiveMinimum,omitempty"` // OAS 2.0
	MaxLength        *int          `yaml:"maxLength,omitempty"`        // OAS 2.0
	MinLength        *int          `yaml:"minLength,omitempty"`        // OAS 2.0
	Pattern          string        `yaml:"pattern,omitempty"`          // OAS 2.0
	MaxItems         *int          `yaml:"maxItems,omitempty"`         // OAS 2.0
	MinItems         *int          `yaml:"minItems,omitempty"`         // OAS 2.0
	UniqueItems      bool          `yaml:"uniqueItems,omitempty"`      // OAS 2.0
	Enum             []interface{} `yaml:"enum,omitempty"`             // OAS 2.0
	MultipleOf       *float64      `yaml:"multipleOf,omitempty"`       // OAS 2.0

	Extra map[string]interface{} `yaml:",inline"`
}
