package parser

// Schema represents a JSON Schema
// Supports OAS 2.0, OAS 3.0, OAS 3.1+ (JSON Schema Draft 2020-12)
type Schema struct {
	// JSON Schema Core
	Ref    string `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Schema string `yaml:"$schema,omitempty" json:"$schema,omitempty"` // JSON Schema Draft version

	// Metadata
	Title       string `yaml:"title,omitempty" json:"title,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Default     any    `yaml:"default,omitempty" json:"default,omitempty"`
	Examples    []any  `yaml:"examples,omitempty" json:"examples,omitempty"` // OAS 3.0+, JSON Schema Draft 2020-12

	// Type validation
	Type  any   `yaml:"type,omitempty" json:"type,omitempty"` // string or []string (OAS 3.1+)
	Enum  []any `yaml:"enum,omitempty" json:"enum,omitempty"`
	Const any   `yaml:"const,omitempty" json:"const,omitempty"` // JSON Schema Draft 2020-12

	// Numeric validation
	MultipleOf       *float64 `yaml:"multipleOf,omitempty" json:"multipleOf,omitempty"`
	Maximum          *float64 `yaml:"maximum,omitempty" json:"maximum,omitempty"`
	ExclusiveMaximum any      `yaml:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty"` // bool in OAS 2.0/3.0, number in 3.1+
	Minimum          *float64 `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	ExclusiveMinimum any      `yaml:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty"` // bool in OAS 2.0/3.0, number in 3.1+

	// String validation
	MaxLength *int   `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	MinLength *int   `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	Pattern   string `yaml:"pattern,omitempty" json:"pattern,omitempty"`

	// Array validation
	Items            any       `yaml:"items,omitempty" json:"items,omitempty"`                       // *Schema or bool (OAS 3.1+)
	PrefixItems      []*Schema `yaml:"prefixItems,omitempty" json:"prefixItems,omitempty"`           // JSON Schema Draft 2020-12
	AdditionalItems  any       `yaml:"additionalItems,omitempty" json:"additionalItems,omitempty"`   // *Schema or bool
	UnevaluatedItems any       `yaml:"unevaluatedItems,omitempty" json:"unevaluatedItems,omitempty"` // JSON Schema Draft 2020-12: *Schema or bool
	MaxItems         *int      `yaml:"maxItems,omitempty" json:"maxItems,omitempty"`
	MinItems         *int      `yaml:"minItems,omitempty" json:"minItems,omitempty"`
	UniqueItems      bool      `yaml:"uniqueItems,omitempty" json:"uniqueItems,omitempty"`
	Contains         *Schema   `yaml:"contains,omitempty" json:"contains,omitempty"`       // JSON Schema Draft 2020-12
	MaxContains      *int      `yaml:"maxContains,omitempty" json:"maxContains,omitempty"` // JSON Schema Draft 2020-12
	MinContains      *int      `yaml:"minContains,omitempty" json:"minContains,omitempty"` // JSON Schema Draft 2020-12

	// Object validation
	Properties            map[string]*Schema  `yaml:"properties,omitempty" json:"properties,omitempty"`
	PatternProperties     map[string]*Schema  `yaml:"patternProperties,omitempty" json:"patternProperties,omitempty"`
	AdditionalProperties  any                 `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`   // *Schema or bool
	UnevaluatedProperties any                 `yaml:"unevaluatedProperties,omitempty" json:"unevaluatedProperties,omitempty"` // JSON Schema Draft 2020-12: *Schema or bool
	Required              []string            `yaml:"required,omitempty" json:"required,omitempty"`
	PropertyNames         *Schema             `yaml:"propertyNames,omitempty" json:"propertyNames,omitempty"` // JSON Schema Draft 2020-12
	MaxProperties         *int                `yaml:"maxProperties,omitempty" json:"maxProperties,omitempty"`
	MinProperties         *int                `yaml:"minProperties,omitempty" json:"minProperties,omitempty"`
	DependentRequired     map[string][]string `yaml:"dependentRequired,omitempty" json:"dependentRequired,omitempty"` // JSON Schema Draft 2020-12
	DependentSchemas      map[string]*Schema  `yaml:"dependentSchemas,omitempty" json:"dependentSchemas,omitempty"`   // JSON Schema Draft 2020-12

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
	Discriminator *Discriminator `yaml:"discriminator,omitempty" json:"discriminator,omitempty"` // OAS 2.0+ (bare string in 2.0, object in 3.0+)
	ReadOnly      bool           `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`           // OAS 2.0+
	WriteOnly     bool           `yaml:"writeOnly,omitempty" json:"writeOnly,omitempty"`         // OAS 3.0+
	XML           *XML           `yaml:"xml,omitempty" json:"xml,omitempty"`                     // OAS 2.0+
	ExternalDocs  *ExternalDocs  `yaml:"externalDocs,omitempty" json:"externalDocs,omitempty"`   // OAS 2.0+
	Example       any            `yaml:"example,omitempty" json:"example,omitempty"`             // OAS 2.0, 3.0 (deprecated in 3.1+)
	Deprecated    bool           `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`       // OAS 3.0+

	// Format
	Format string `yaml:"format,omitempty" json:"format,omitempty"` // e.g., "date-time", "email", "uri", etc.

	// Content keywords (JSON Schema Draft 2020-12)
	ContentEncoding  string  `yaml:"contentEncoding,omitempty" json:"contentEncoding,omitempty"`   // e.g., "base64", "base32"
	ContentMediaType string  `yaml:"contentMediaType,omitempty" json:"contentMediaType,omitempty"` // e.g., "application/json"
	ContentSchema    *Schema `yaml:"contentSchema,omitempty" json:"contentSchema,omitempty"`       // Schema for decoded content

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
	Extra map[string]any `yaml:",inline" json:"-"`
}

// Discriminator represents a discriminator for polymorphism.
//
// The two OAS dialects spell this differently. In OAS 2.0 the Schema Object's
// discriminator is a bare string naming the property; in OAS 3.0+ it is an
// object with propertyName and an optional mapping. A single Go type serves
// both: the string form decodes into PropertyName with StringForm set, so the
// dialect a document was written in survives a parse/serialize round trip.
type Discriminator struct {
	PropertyName string            `yaml:"propertyName" json:"propertyName"`
	Mapping      map[string]string `yaml:"mapping,omitempty" json:"mapping,omitempty"` // OAS 3.0+ only

	// DefaultMapping names the schema to use when the discriminating property is
	// absent, or holds a value no Mapping key covers (OAS 3.2+).
	// https://spec.openapis.org/oas/v3.2.0.html#discriminator-default-mapping
	//
	// Reference-bearing exactly as a Mapping value is: a schema name or a URI
	// reference. Every pass that rewrites Mapping (joining, renaming, converting)
	// must rewrite this too, or leave a dangling reference while reporting success.
	//
	// Its conditional requirement is a validator rule, not a parser one: whether the
	// discriminating property is optional is only knowable from the enclosing
	// schema.
	DefaultMapping string         `yaml:"defaultMapping,omitempty" json:"defaultMapping,omitempty"` // OAS 3.2+
	Extra          map[string]any `yaml:",inline" json:"-"`

	// StringForm reports that this discriminator came from — and should be
	// written back as — the OAS 2.0 bare-string form (`discriminator: petType`)
	// rather than the OAS 3.0+ object form. It is not itself a specification
	// field, so it is excluded from both JSON and YAML.
	//
	// Mapping and Extra have no representation in the string form and are
	// dropped when serializing with StringForm set. Converting between
	// dialects must set or clear this flag; see the converter package.
	StringForm bool `yaml:"-" json:"-"`
}

// XML represents metadata for XML encoding (OAS 2.0+)
// https://spec.openapis.org/oas/v3.2.0.html#xml-object
//
// NodeType supersedes Attribute and Wrapped in OAS 3.2 and is modeled beside them
// rather than derived from either. See the field's own comment for why.
type XML struct {
	Name      string `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Prefix    string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	Attribute bool   `yaml:"attribute,omitempty" json:"attribute,omitempty"`
	Wrapped   bool   `yaml:"wrapped,omitempty" json:"wrapped,omitempty"`

	// NodeType selects the XML node a schema maps to, superseding Attribute and
	// Wrapped (OAS 3.2+).
	// https://spec.openapis.org/oas/v3.2.0.html#xml-node-type
	//
	// Deliberately independent of those two bools rather than reconciled with them.
	// Its default depends on the enclosing Schema Object, which an XML Object cannot
	// see, so deriving a value here would invent one the document never stated.
	// Keeping them independent lets each survive a round trip as written; the spec
	// forbids setting both, and the validator enforces that.
	NodeType string `yaml:"nodeType,omitempty" json:"nodeType,omitempty"` // OAS 3.2+

	Extra map[string]any `yaml:",inline" json:"-"`
}
