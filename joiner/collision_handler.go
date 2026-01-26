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

// ContinueWithStrategy returns a resolution that defers to the configured strategy.
// Use this for observe-only handlers that just want to log collisions.
func ContinueWithStrategy() CollisionResolution {
	return CollisionResolution{Action: ResolutionContinue}
}

// ContinueWithStrategyWithMessage returns a resolution that defers to strategy with a log message.
func ContinueWithStrategyWithMessage(message string) CollisionResolution {
	return CollisionResolution{Action: ResolutionContinue, Message: message}
}

// AcceptLeft returns a resolution that keeps the left (base) value.
func AcceptLeft() CollisionResolution {
	return CollisionResolution{Action: ResolutionAcceptLeft}
}

// AcceptLeftWithMessage returns a resolution that keeps the left value with a log message.
func AcceptLeftWithMessage(message string) CollisionResolution {
	return CollisionResolution{Action: ResolutionAcceptLeft, Message: message}
}

// AcceptRight returns a resolution that keeps the right (incoming) value.
func AcceptRight() CollisionResolution {
	return CollisionResolution{Action: ResolutionAcceptRight}
}

// AcceptRightWithMessage returns a resolution that keeps the right value with a log message.
func AcceptRightWithMessage(message string) CollisionResolution {
	return CollisionResolution{Action: ResolutionAcceptRight, Message: message}
}

// Rename returns a resolution that renames the right value to avoid collision.
func Rename() CollisionResolution {
	return CollisionResolution{Action: ResolutionRename}
}

// RenameWithMessage returns a resolution that renames with a log message.
func RenameWithMessage(message string) CollisionResolution {
	return CollisionResolution{Action: ResolutionRename, Message: message}
}

// Deduplicate returns a resolution that treats colliding values as equivalent.
func Deduplicate() CollisionResolution {
	return CollisionResolution{Action: ResolutionDeduplicate}
}

// DeduplicateWithMessage returns a resolution that deduplicates with a log message.
func DeduplicateWithMessage(message string) CollisionResolution {
	return CollisionResolution{Action: ResolutionDeduplicate, Message: message}
}

// Fail returns a resolution that aborts the join with an error.
func Fail() CollisionResolution {
	return CollisionResolution{Action: ResolutionFail}
}

// FailWithMessage returns a resolution that aborts with a custom error message.
func FailWithMessage(message string) CollisionResolution {
	return CollisionResolution{Action: ResolutionFail, Message: message}
}

// UseCustomValue returns a resolution that uses a caller-provided merged value.
func UseCustomValue(value any) CollisionResolution {
	return CollisionResolution{Action: ResolutionCustom, CustomValue: value}
}

// UseCustomValueWithMessage returns a resolution with custom value and log message.
func UseCustomValueWithMessage(value any, message string) CollisionResolution {
	return CollisionResolution{Action: ResolutionCustom, CustomValue: value, Message: message}
}
