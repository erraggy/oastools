package walker

// RefNodeType identifies the type of node containing a $ref.
type RefNodeType string

// RefNodeType constants for all supported reference types.
const (
	RefNodeSchema         RefNodeType = "schema"
	RefNodeParameter      RefNodeType = "parameter"
	RefNodeResponse       RefNodeType = "response"
	RefNodeRequestBody    RefNodeType = "requestBody"
	RefNodeHeader         RefNodeType = "header"
	RefNodeLink           RefNodeType = "link"
	RefNodeExample        RefNodeType = "example"
	RefNodePathItem       RefNodeType = "pathItem"
	RefNodeSecurityScheme RefNodeType = "securityScheme"
)

// RefInfo contains information about a $ref encountered during traversal.
type RefInfo struct {
	// Ref is the $ref value (e.g., "#/components/schemas/User")
	Ref string

	// SourcePath is the JSON path where the ref was encountered
	SourcePath string

	// NodeType is the type of node containing the ref.
	// Use the RefNode* constants for comparison (e.g., RefNodeSchema, RefNodeParameter).
	NodeType RefNodeType
}

// RefHandler is called when a $ref is encountered during traversal.
// Return Stop to halt traversal, Continue to proceed.
type RefHandler func(wc *WalkContext, ref *RefInfo) Action
