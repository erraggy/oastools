package joiner

// CollisionType identifies what kind of component collided.
type CollisionType string

const (
	// CollisionTypeSchema indicates a schema collision in components.schemas or definitions.
	CollisionTypeSchema CollisionType = "schema"
	// CollisionTypePath indicates a path collision in paths.
	CollisionTypePath CollisionType = "path"
	// CollisionTypeWebhook indicates a webhook collision.
	CollisionTypeWebhook CollisionType = "webhook"
	// CollisionTypeResponse indicates a response collision in components.responses.
	CollisionTypeResponse CollisionType = "response"
	// CollisionTypeParameter indicates a parameter collision in components.parameters.
	CollisionTypeParameter CollisionType = "parameter"
	// CollisionTypeExample indicates an example collision in components.examples.
	CollisionTypeExample CollisionType = "example"
	// CollisionTypeRequestBody indicates a request body collision in components.requestBodies.
	CollisionTypeRequestBody CollisionType = "requestBody"
	// CollisionTypeHeader indicates a header collision in components.headers.
	CollisionTypeHeader CollisionType = "header"
	// CollisionTypeSecurityScheme indicates a security scheme collision.
	CollisionTypeSecurityScheme CollisionType = "securityScheme"
	// CollisionTypeLink indicates a link collision in components.links.
	CollisionTypeLink CollisionType = "link"
	// CollisionTypeCallback indicates a callback collision in components.callbacks.
	CollisionTypeCallback CollisionType = "callback"
)

// ResolutionAction specifies what the joiner should do after a collision.
type ResolutionAction int

// String returns a human-readable name for the resolution action.
func (r ResolutionAction) String() string {
	switch r {
	case ResolutionContinue:
		return "continue"
	case ResolutionAcceptLeft:
		return "accept-left"
	case ResolutionAcceptRight:
		return "accept-right"
	case ResolutionRename:
		return "rename"
	case ResolutionDeduplicate:
		return "deduplicate"
	case ResolutionFail:
		return "fail"
	case ResolutionCustom:
		return "custom"
	default:
		return "unknown"
	}
}

const (
	// ResolutionContinue delegates to the configured strategy (observe-only).
	ResolutionContinue ResolutionAction = iota
	// ResolutionAcceptLeft keeps the left (base) value.
	ResolutionAcceptLeft
	// ResolutionAcceptRight keeps the right (incoming) value.
	ResolutionAcceptRight
	// ResolutionRename renames the right value to avoid collision.
	ResolutionRename
	// ResolutionDeduplicate treats colliding values as equivalent.
	ResolutionDeduplicate
	// ResolutionFail aborts the join with an error.
	ResolutionFail
	// ResolutionCustom uses the CustomValue provided in CollisionResolution.
	ResolutionCustom
)

// CollisionContext provides full details about a detected collision.
type CollisionContext struct {
	// Type identifies what kind of component collided.
	Type CollisionType
	// Name is the colliding name (e.g., "User", "/pets").
	Name string
	// JSONPath is the full path (e.g., "$.components.schemas.User").
	JSONPath string

	// LeftSource is the source file/identifier for left document.
	LeftSource string
	// LeftLocation is the line/column in left document (nil if unknown).
	LeftLocation *SourceLocation
	// LeftValue is the left component (*parser.Schema, *parser.PathItem, etc.).
	LeftValue any

	// RightSource is the source file/identifier for right document.
	RightSource string
	// RightLocation is the line/column in right document (nil if unknown).
	RightLocation *SourceLocation
	// RightValue is the right component.
	RightValue any

	// RenameInfo provides operation context if available (nil otherwise).
	RenameInfo *RenameContext

	// ConfiguredStrategy is the strategy that would apply without handler.
	ConfiguredStrategy CollisionStrategy
}

// SourceLocation represents a position in a source file.
// This is a local type to avoid circular imports with the parser package.
type SourceLocation struct {
	// Line is the 1-based line number (0 if unknown).
	Line int
	// Column is the 1-based column number (0 if unknown).
	Column int
}

// CollisionResolution is returned by the handler to indicate desired action.
type CollisionResolution struct {
	// Action specifies what the joiner should do.
	Action ResolutionAction
	// CustomValue is used when Action is ResolutionCustom.
	CustomValue any
	// Message is an optional message for logging/warnings.
	Message string
}

// CollisionHandler is called when a collision is detected.
// Return an error to log a warning and fall back to configured strategy.
type CollisionHandler func(collision CollisionContext) (CollisionResolution, error)
