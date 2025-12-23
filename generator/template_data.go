package generator

// HeaderData contains data for file header templates
type HeaderData struct {
	PackageName string
	Imports     []string
	Comment     string // Optional file-level comment
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

// AliasData contains data for a type alias or defined type
type AliasData struct {
	Comment    string
	TypeName   string
	TargetType string
	IsDefined  bool // true for defined type, false for type alias (=)
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

// ServerResponsesFileData contains data for server_responses.go
type ServerResponsesFileData struct {
	Header     HeaderData
	Operations []ResponseOperationData
}

// ResponseOperationData contains response data for a single operation
type ResponseOperationData struct {
	MethodName   string // e.g., "ListPets"
	ResponseType string // e.g., "ListPetsResponse"
	StatusCodes  []StatusCodeData
}

// StatusCodeData contains data for a single status code response
type StatusCodeData struct {
	Code          string // e.g., "200", "4XX", "default"
	MethodName    string // e.g., "Status200"
	BodyType      string // e.g., "[]Pet", "*Error"
	HasBody       bool
	IsSuccess     bool   // true for 2XX codes
	Description   string // From OpenAPI description
	ContentType   string // e.g., "application/json"
	IsDefault     bool   // true for "default" response
	IsWildcard    bool   // true for "2XX", "4XX", etc.
	StatusCodeInt int    // numeric value for non-wildcard codes (0 for wildcard/default)
}

// ServerBinderFileData contains data for server_binder.go
type ServerBinderFileData struct {
	Header     HeaderData
	Operations []BinderOperationData
}

// BinderOperationData contains binding data for a single operation
type BinderOperationData struct {
	MethodName   string
	RequestType  string // e.g., "ListPetsRequest"
	PathParams   []ParamBindData
	QueryParams  []ParamBindData
	HeaderParams []ParamBindData
	CookieParams []ParamBindData
	HasBody      bool
	BodyType     string
}

// ParamBindData contains data for binding a single parameter
type ParamBindData struct {
	Name       string // original name from spec
	FieldName  string // Go field name
	GoType     string // Go type
	Required   bool
	IsPointer  bool
	SchemaType string // "integer", "string", "array", etc.
}

// ServerMiddlewareFileData contains data for server_middleware.go
type ServerMiddlewareFileData struct {
	Header HeaderData
}

// ServerRouterFileData contains data for server_router.go
type ServerRouterFileData struct {
	Header     HeaderData
	Framework  string // "stdlib" or "chi"
	Routes     []RouteData
	Operations []RouterOperationData
}

// RouteData contains data for a single route
type RouteData struct {
	PathTemplate string   // OpenAPI path template e.g., "/pets/{petId}"
	GoPath       string   // Go path pattern e.g., "/pets/{petId}"
	Methods      []string // HTTP methods for this path
	ParamNames   []string // Path parameter names
}

// RouterOperationData contains operation data for routing
type RouterOperationData struct {
	MethodName   string          // e.g., "ListPets"
	HTTPMethod   string          // e.g., "GET"
	Method       string          // e.g., "GET" (alias for template compatibility)
	PathTemplate string          // e.g., "/pets/{petId}"
	Path         string          // e.g., "/pets/{petId}" (alias for template compatibility)
	RequestType  string          // e.g., "ListPetsRequest"
	OperationID  string          // original operationId from spec
	PathParams   []ParamBindData // path parameters for this operation
}

// ServerStubsFileData contains data for server_stubs.go
type ServerStubsFileData struct {
	Header     HeaderData
	Operations []StubOperationData
}

// StubOperationData contains data for a single stub method
type StubOperationData struct {
	MethodName   string
	RequestType  string
	ResponseType string
	ZeroValue    string // zero value for ResponseType
}
