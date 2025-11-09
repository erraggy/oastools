package parser

// Schema represents a JSON Schema
// Supports OAS 2.0, OAS 3.0, OAS 3.1+ (JSON Schema Draft 2020-12)
type Schema struct {
	// JSON Schema Core
	Ref    string `yaml:"$ref,omitempty"`
	Schema string `yaml:"$schema,omitempty"` // JSON Schema Draft version

	// Metadata
	Title       string        `yaml:"title,omitempty"`
	Description string        `yaml:"description,omitempty"`
	Default     interface{}   `yaml:"default,omitempty"`
	Examples    []interface{} `yaml:"examples,omitempty"` // OAS 3.0+, JSON Schema Draft 2020-12

	// Type validation
	Type  interface{}   `yaml:"type,omitempty"` // string or []string (OAS 3.1+)
	Enum  []interface{} `yaml:"enum,omitempty"`
	Const interface{}   `yaml:"const,omitempty"` // JSON Schema Draft 2020-12

	// Numeric validation
	MultipleOf       *float64    `yaml:"multipleOf,omitempty"`
	Maximum          *float64    `yaml:"maximum,omitempty"`
	ExclusiveMaximum interface{} `yaml:"exclusiveMaximum,omitempty"` // bool in OAS 2.0/3.0, number in 3.1+
	Minimum          *float64    `yaml:"minimum,omitempty"`
	ExclusiveMinimum interface{} `yaml:"exclusiveMinimum,omitempty"` // bool in OAS 2.0/3.0, number in 3.1+

	// String validation
	MaxLength *int   `yaml:"maxLength,omitempty"`
	MinLength *int   `yaml:"minLength,omitempty"`
	Pattern   string `yaml:"pattern,omitempty"`

	// Array validation
	Items           interface{} `yaml:"items,omitempty"`           // *Schema or bool (OAS 3.1+)
	PrefixItems     []*Schema   `yaml:"prefixItems,omitempty"`     // JSON Schema Draft 2020-12
	AdditionalItems interface{} `yaml:"additionalItems,omitempty"` // *Schema or bool
	MaxItems        *int        `yaml:"maxItems,omitempty"`
	MinItems        *int        `yaml:"minItems,omitempty"`
	UniqueItems     bool        `yaml:"uniqueItems,omitempty"`
	Contains        *Schema     `yaml:"contains,omitempty"`    // JSON Schema Draft 2020-12
	MaxContains     *int        `yaml:"maxContains,omitempty"` // JSON Schema Draft 2020-12
	MinContains     *int        `yaml:"minContains,omitempty"` // JSON Schema Draft 2020-12

	// Object validation
	Properties           map[string]*Schema  `yaml:"properties,omitempty"`
	PatternProperties    map[string]*Schema  `yaml:"patternProperties,omitempty"`
	AdditionalProperties interface{}         `yaml:"additionalProperties,omitempty"` // *Schema or bool
	Required             []string            `yaml:"required,omitempty"`
	PropertyNames        *Schema             `yaml:"propertyNames,omitempty"` // JSON Schema Draft 2020-12
	MaxProperties        *int                `yaml:"maxProperties,omitempty"`
	MinProperties        *int                `yaml:"minProperties,omitempty"`
	DependentRequired    map[string][]string `yaml:"dependentRequired,omitempty"` // JSON Schema Draft 2020-12
	DependentSchemas     map[string]*Schema  `yaml:"dependentSchemas,omitempty"`  // JSON Schema Draft 2020-12

	// Conditional schemas
	If   *Schema `yaml:"if,omitempty"`   // JSON Schema Draft 2020-12, OAS 3.1+
	Then *Schema `yaml:"then,omitempty"` // JSON Schema Draft 2020-12, OAS 3.1+
	Else *Schema `yaml:"else,omitempty"` // JSON Schema Draft 2020-12, OAS 3.1+

	// Schema composition
	AllOf []*Schema `yaml:"allOf,omitempty"`
	AnyOf []*Schema `yaml:"anyOf,omitempty"`
	OneOf []*Schema `yaml:"oneOf,omitempty"`
	Not   *Schema   `yaml:"not,omitempty"`

	// OAS specific extensions
	Nullable      bool           `yaml:"nullable,omitempty"`      // OAS 3.0 only (replaced by type: [T, "null"] in 3.1+)
	Discriminator *Discriminator `yaml:"discriminator,omitempty"` // OAS 3.0+
	ReadOnly      bool           `yaml:"readOnly,omitempty"`      // OAS 2.0+
	WriteOnly     bool           `yaml:"writeOnly,omitempty"`     // OAS 3.0+
	XML           *XML           `yaml:"xml,omitempty"`           // OAS 2.0+
	ExternalDocs  *ExternalDocs  `yaml:"externalDocs,omitempty"`  // OAS 2.0+
	Example       interface{}    `yaml:"example,omitempty"`       // OAS 2.0, 3.0 (deprecated in 3.1+)
	Deprecated    bool           `yaml:"deprecated,omitempty"`    // OAS 3.0+

	// Format
	Format string `yaml:"format,omitempty"` // e.g., "date-time", "email", "uri", etc.

	// OAS 2.0 specific
	CollectionFormat string `yaml:"collectionFormat,omitempty"` // OAS 2.0

	// JSON Schema Draft 2020-12 additional fields
	ID            string             `yaml:"$id,omitempty"`
	Anchor        string             `yaml:"$anchor,omitempty"`
	DynamicRef    string             `yaml:"$dynamicRef,omitempty"`
	DynamicAnchor string             `yaml:"$dynamicAnchor,omitempty"`
	Vocabulary    map[string]bool    `yaml:"$vocabulary,omitempty"`
	Comment       string             `yaml:"$comment,omitempty"`
	Defs          map[string]*Schema `yaml:"$defs,omitempty"`

	// Extension fields
	Extra map[string]interface{} `yaml:",inline"`
}

// Discriminator represents a discriminator for polymorphism (OAS 3.0+)
type Discriminator struct {
	PropertyName string                 `yaml:"propertyName"`
	Mapping      map[string]string      `yaml:"mapping,omitempty"`
	Extra        map[string]interface{} `yaml:",inline"`
}

// XML represents metadata for XML encoding (OAS 2.0+)
type XML struct {
	Name      string                 `yaml:"name,omitempty"`
	Namespace string                 `yaml:"namespace,omitempty"`
	Prefix    string                 `yaml:"prefix,omitempty"`
	Attribute bool                   `yaml:"attribute,omitempty"`
	Wrapped   bool                   `yaml:"wrapped,omitempty"`
	Extra     map[string]interface{} `yaml:",inline"`
}
