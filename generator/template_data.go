package generator

// HeaderData contains data for file header templates
type HeaderData struct {
	PackageName string
	Imports     []string
}

// FieldData contains data for a struct field
type FieldData struct {
	Comment string
	Name    string
	Type    string
	Tags    string
}

// StructData contains data for a struct type
type StructData struct {
	Comment             string
	TypeName            string
	OriginalName        string
	Fields              []FieldData
	HasAdditionalProps  bool
	AdditionalPropsType string
}

// EnumValueData contains data for a single enum value
type EnumValueData struct {
	ConstName string
	Type      string
	Value     string
}

// EnumData contains data for an enum type
type EnumData struct {
	Comment  string
	TypeName string
	BaseType string
	Values   []EnumValueData
}

// AliasData contains data for a type alias
type AliasData struct {
	Comment    string
	TypeName   string
	TargetType string
	IsAlias    bool // true for type alias (=), false for defined type
}

// AllOfData contains data for AllOf composition
type AllOfData struct {
	Comment       string
	TypeName      string
	EmbeddedTypes []string
	Fields        []FieldData
}

// OneOfData contains data for OneOf union type
type OneOfData struct {
	Comment               string
	TypeName              string
	Discriminator         string
	DiscriminatorField    string
	DiscriminatorJSONName string
	Variants              []OneOfVariant
	HasUnmarshal          bool
	UnmarshalCases        []UnmarshalCase
}

// OneOfVariant contains data for a OneOf variant
type OneOfVariant struct {
	Name string
	Type string
}

// UnmarshalCase contains data for an unmarshal case
type UnmarshalCase struct {
	Value    string
	TypeName string
}

// TypesFileData contains all data for a types.go file
type TypesFileData struct {
	Header HeaderData
	Types  []TypeDefinition
}

// TypeDefinition is a union type for different kind of type definitions
type TypeDefinition struct {
	Kind string // "struct", "enum", "alias", "allof", "oneof"

	Struct *StructData
	Enum   *EnumData
	Alias  *AliasData
	AllOf  *AllOfData
	OneOf  *OneOfData
}

// ClientFileData contains all data for a client.go file
type ClientFileData struct {
	Header           HeaderData
	DefaultUserAgent string
	Methods          []ClientMethodData
	ParamsStructs    []ParamsStructData
}

// ClientMethodData contains data for a client method
type ClientMethodData struct {
	Comment         string
	MethodName      string
	Params          string
	ResponseType    string
	PathTemplate    string
	PathArgs        []string
	HasQueryParams  bool
	QueryParamsType string
	HasBody         bool
	BodyType        string
	ContentType     string
	MethodBody      string // Complex body handled in Go
}

// ParamsStructData contains data for a query params struct
type ParamsStructData struct {
	MethodName string
	Fields     []FieldData
}

// ServerFileData contains all data for a server.go file
type ServerFileData struct {
	Header       HeaderData
	Methods      []ServerMethodData
	RequestTypes []RequestTypeData
}

// ServerMethodData contains data for a server interface method
type ServerMethodData struct {
	Comment      string
	MethodName   string
	ResponseType string
}

// RequestTypeData contains data for a request struct
type RequestTypeData struct {
	MethodName string
	Fields     []FieldData
}
