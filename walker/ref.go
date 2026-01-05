package walker

// RefInfo contains information about a $ref encountered during traversal.
type RefInfo struct {
	// Ref is the $ref value (e.g., "#/components/schemas/User")
	Ref string

	// SourcePath is the JSON path where the ref was encountered
	SourcePath string

	// NodeType is the type of node containing the ref (e.g., "schema", "parameter")
	NodeType string
}

// RefHandler is called when a $ref is encountered during traversal.
// Return Stop to halt traversal, Continue to proceed.
type RefHandler func(wc *WalkContext, ref *RefInfo) Action
