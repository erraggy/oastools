package parser

// Schema represents a JSON Schema
// Supports OAS 2.0, OAS 3.0, OAS 3.1+ (JSON Schema Draft 2020-12)
type Schema struct {
	// JSON Schema Core
	Ref    string `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Schema string `yaml:"$schema,omitempty" json:"$schema,omitempty"` // JSON Schema Draft version

	// Metadata
	Title       string        `yaml:"title,omitempty" json:"title,omitempty"`
	Description string        `yaml:"description,omitempty" json:"description,omitempty"`
	Default     interface{}   `yaml:"default,omitempty" json:"default,omitempty"`
	Examples    []interface{} `yaml:"examples,omitempty" json:"examples,omitempty"` // OAS 3.0+, JSON Schema Draft 2020-12

	// Type validation
	Type  interface{}   `yaml:"type,omitempty" json:"type,omitempty"` // string or []string (OAS 3.1+)
	Enum  []interface{} `yaml:"enum,omitempty" json:"enum,omitempty"`
	Const interface{}   `yaml:"const,omitempty" json:"const,omitempty"` // JSON Schema Draft 2020-12

	// Numeric validation
	MultipleOf       *float64    `yaml:"multipleOf,omitempty" json:"multipleOf,omitempty"`
	Maximum          *float64    `yaml:"maximum,omitempty" json:"maximum,omitempty"`
	ExclusiveMaximum interface{} `yaml:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty"` // bool in OAS 2.0/3.0, number in 3.1+
	Minimum          *float64    `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	ExclusiveMinimum interface{} `yaml:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty"` // bool in OAS 2.0/3.0, number in 3.1+

	// String validation
	MaxLength *int   `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	MinLength *int   `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	Pattern   string `yaml:"pattern,omitempty" json:"pattern,omitempty"`

	// Array validation
	Items           interface{} `yaml:"items,omitempty" json:"items,omitempty"`                     // *Schema or bool (OAS 3.1+)
	PrefixItems     []*Schema   `yaml:"prefixItems,omitempty" json:"prefixItems,omitempty"`         // JSON Schema Draft 2020-12
	AdditionalItems interface{} `yaml:"additionalItems,omitempty" json:"additionalItems,omitempty"` // *Schema or bool
	MaxItems        *int        `yaml:"maxItems,omitempty" json:"maxItems,omitempty"`
	MinItems        *int        `yaml:"minItems,omitempty" json:"minItems,omitempty"`
	UniqueItems     bool        `yaml:"uniqueItems,omitempty" json:"uniqueItems,omitempty"`
	Contains        *Schema     `yaml:"contains,omitempty" json:"contains,omitempty"`       // JSON Schema Draft 2020-12
	MaxContains     *int        `yaml:"maxContains,omitempty" json:"maxContains,omitempty"` // JSON Schema Draft 2020-12
	MinContains     *int        `yaml:"minContains,omitempty" json:"minContains,omitempty"` // JSON Schema Draft 2020-12

	// Object validation
	Properties           map[string]*Schema  `yaml:"properties,omitempty" json:"properties,omitempty"`
	PatternProperties    map[string]*Schema  `yaml:"patternProperties,omitempty" json:"patternProperties,omitempty"`
	AdditionalProperties interface{}         `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"` // *Schema or bool
	Required             []string            `yaml:"required,omitempty" json:"required,omitempty"`
	PropertyNames        *Schema             `yaml:"propertyNames,omitempty" json:"propertyNames,omitempty"` // JSON Schema Draft 2020-12
	MaxProperties        *int                `yaml:"maxProperties,omitempty" json:"maxProperties,omitempty"`
	MinProperties        *int                `yaml:"minProperties,omitempty" json:"minProperties,omitempty"`
	DependentRequired    map[string][]string `yaml:"dependentRequired,omitempty" json:"dependentRequired,omitempty"` // JSON Schema Draft 2020-12
	DependentSchemas     map[string]*Schema  `yaml:"dependentSchemas,omitempty" json:"dependentSchemas,omitempty"`   // JSON Schema Draft 2020-12

	// Conditional schemas
	If   *Schema `yaml:"if,omitempty" json:"if,omitempty"`     // JSON Schema Draft 2020-12, OAS 3.1+
	Then *Schema `yaml:"then,omitempty" json:"then,omitempty"` // JSON Schema Draft 2020-12, OAS 3.1+
	Else *Schema `yaml:"else,omitempty" json:"else,omitempty"` // JSON Schema Draft 2020-12, OAS 3.1+

	// Schema composition
	AllOf []*Schema `yaml:"allOf,omitempty" json:"allOf,omitempty"`
	AnyOf []*Schema `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	OneOf []*Schema `yaml:"oneOf,omitempty" json:"oneOf,omitempty"`
	Not   *Schema   `yaml:"not,omitempty" json:"not,omitempty"`

	// OAS specific extensions
	Nullable      bool           `yaml:"nullable,omitempty" json:"nullable,omitempty"`           // OAS 3.0 only (replaced by type: [T, "null"] in 3.1+)
	Discriminator *Discriminator `yaml:"discriminator,omitempty" json:"discriminator,omitempty"` // OAS 3.0+
	ReadOnly      bool           `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`           // OAS 2.0+
	WriteOnly     bool           `yaml:"writeOnly,omitempty" json:"writeOnly,omitempty"`         // OAS 3.0+
	XML           *XML           `yaml:"xml,omitempty" json:"xml,omitempty"`                     // OAS 2.0+
	ExternalDocs  *ExternalDocs  `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`   // OAS 2.0+
	Example       interface{}    `yaml:"example,omitempty" json:"example,omitempty"`             // OAS 2.0, 3.0 (deprecated in 3.1+)
	Deprecated    bool           `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`       // OAS 3.0+

	// Format
	Format string `yaml:"format,omitempty" json:"format,omitempty"` // e.g., "date-time", "email", "uri", etc.

	// OAS 2.0 specific
	CollectionFormat string `yaml:"collectionFormat,omitempty" json:"collectionFormat,omitempty"` // OAS 2.0

	// JSON Schema Draft 2020-12 additional fields
	ID            string             `yaml:"$id,omitempty" json:"$id,omitempty"`
	Anchor        string             `yaml:"$anchor,omitempty" json:"$anchor,omitempty"`
	DynamicRef    string             `yaml:"$dynamicRef,omitempty" json:"$dynamicRef,omitempty"`
	DynamicAnchor string             `yaml:"$dynamicAnchor,omitempty" json:"$dynamicAnchor,omitempty"`
	Vocabulary    map[string]bool    `yaml:"$vocabulary,omitempty" json:"$vocabulary,omitempty"`
	Comment       string             `yaml:"$comment,omitempty" json:"$comment,omitempty"`
	Defs          map[string]*Schema `yaml:"$defs,omitempty" json:"$defs,omitempty"`

	// Extension fields
	// Extra captures specification extensions (fields starting with "x-")
	Extra map[string]interface{} `yaml:",inline" json:"-"`
}

// Discriminator represents a discriminator for polymorphism (OAS 3.0+)
type Discriminator struct {
	PropertyName string                 `yaml:"propertyName" json:"propertyName"`
	Mapping      map[string]string      `yaml:"mapping,omitempty" json:"mapping,omitempty"`
	Extra        map[string]interface{} `yaml:",inline" json:"-"`
}

// XML represents metadata for XML encoding (OAS 2.0+)
type XML struct {
	Name      string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace string                 `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Prefix    string                 `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	Attribute bool                   `yaml:"attribute,omitempty" json:"attribute,omitempty"`
	Wrapped   bool                   `yaml:"wrapped,omitempty" json:"wrapped,omitempty"`
	Extra     map[string]interface{} `yaml:",inline" json:"-"`
}
