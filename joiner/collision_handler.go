package joiner

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

/*
Collision Handler Support

The joiner package supports collision handlers for custom collision resolution.
A collision handler is called when two documents being joined have conflicting
components (schemas, paths, webhooks, etc.).

Basic usage:

	result, err := joiner.JoinWithOptions(
	    joiner.WithFilePaths("base.yaml", "overlay.yaml"),
	    joiner.WithCollisionHandler(func(collision joiner.CollisionContext) (joiner.CollisionResolution, error) {
	        // Log all collisions
	        log.Printf("Collision: %s %s", collision.Type, collision.Name)
	        // Defer to configured strategy
	        return joiner.ContinueWithStrategy(), nil
	    }),
	)

Handler capabilities:

 1. Observe-only: Return ContinueWithStrategy() to log/observe and defer to strategy
 2. Decision-only: Return AcceptLeft(), AcceptRight(), Rename(), etc. to override strategy
 3. Custom resolution: Return UseCustomValue(mergedSchema) to provide a custom merged value

Error handling:

If the handler returns an error, the joiner logs a warning and falls back to the
configured strategy. This ensures handlers cannot break the join operation.

Type filtering:

Use WithCollisionHandlerFor to handle only specific collision types:

	joiner.WithCollisionHandlerFor(handler, joiner.CollisionTypeSchema, joiner.CollisionTypePath)

See the CollisionContext, CollisionResolution, and helper function documentation
for complete details.
*/

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

// schemaResolutionParams contains parameters for applySchemaResolution.
// This allows sharing the resolution logic between OAS2 definitions and OAS3 schemas.
type schemaResolutionParams struct {
	collision   CollisionContext
	resolution  CollisionResolution
	target      map[string]*parser.Schema
	result      *JoinResult
	ctx         documentContext
	sourceGraph *RefGraph
	label       string // "schema" for OAS3, "definition" for OAS2
}

// applySchemaResolution applies a CollisionResolution to a schema/definition collision.
// Returns true if the resolution was fully handled, false if strategy should still be applied.
// This is shared by both OAS2 (definitions) and OAS3 (schemas) collision handling.
func (j *Joiner) applySchemaResolution(p schemaResolutionParams) (bool, error) {
	// Record message as warning if provided
	if p.resolution.Message != "" {
		line, col := j.getLocation(p.ctx.filePath, p.collision.JSONPath)
		p.result.AddWarning(NewHandlerResolutionWarning(p.collision.JSONPath, p.resolution.Message, p.ctx.filePath, line, col))
	}

	schema, ok := p.collision.RightValue.(*parser.Schema)
	if !ok {
		return false, fmt.Errorf("collision handler: RightValue is %T, expected *parser.Schema", p.collision.RightValue)
	}

	switch p.resolution.Action {
	case ResolutionContinue:
		// Delegate to configured strategy
		return false, nil

	case ResolutionAcceptLeft:
		// Keep existing (left), discard incoming (right)
		j.recordCollisionEvent(p.result, p.collision.Name, p.collision.LeftSource, p.collision.RightSource, p.collision.ConfiguredStrategy, "kept-left", "")
		return true, nil

	case ResolutionAcceptRight:
		// Replace with incoming (right)
		p.target[p.collision.Name] = schema
		j.recordCollisionEvent(p.result, p.collision.Name, p.collision.LeftSource, p.collision.RightSource, p.collision.ConfiguredStrategy, "kept-right", "")
		return true, nil

	case ResolutionRename:
		// Rename right schema/definition
		newName := j.generateRenamedSchemaName(p.collision.Name, p.ctx.filePath, p.ctx.docIndex, p.sourceGraph)
		p.target[newName] = schema
		if p.result.rewriter == nil {
			p.result.rewriter = NewSchemaRewriter()
		}
		p.result.rewriter.RegisterRename(p.collision.Name, newName, p.result.OASVersion)
		line, col := j.getLocation(p.ctx.filePath, p.collision.JSONPath)
		p.result.AddWarning(NewSchemaRenamedWarning(p.collision.Name, newName, p.label, p.ctx.filePath, line, col, false))
		j.recordCollisionEvent(p.result, p.collision.Name, p.collision.LeftSource, p.collision.RightSource, p.collision.ConfiguredStrategy, "renamed", newName)
		return true, nil

	case ResolutionDeduplicate:
		// Keep left, discard right (treat as equivalent)
		line, col := j.getLocation(p.ctx.filePath, p.collision.JSONPath)
		p.result.AddWarning(NewSchemaDedupWarning(p.collision.Name, p.label, p.ctx.filePath, line, col))
		j.recordCollisionEvent(p.result, p.collision.Name, p.collision.LeftSource, p.collision.RightSource, p.collision.ConfiguredStrategy, "deduplicated", "")
		return true, nil

	case ResolutionFail:
		// Return error with handler's message
		msg := p.resolution.Message
		if msg == "" {
			msg = fmt.Sprintf("%s collision on %q rejected by handler", p.label, p.collision.Name)
		}
		return true, fmt.Errorf("collision handler: %s", msg)

	case ResolutionCustom:
		if p.resolution.CustomValue == nil {
			return true, fmt.Errorf("collision handler: ResolutionCustom requires CustomValue")
		}
		customSchema, ok := p.resolution.CustomValue.(*parser.Schema)
		if !ok {
			return true, fmt.Errorf("collision handler: CustomValue is %T, expected *parser.Schema for %s collisions", p.resolution.CustomValue, p.label)
		}
		p.target[p.collision.Name] = customSchema
		j.recordCollisionEvent(p.result, p.collision.Name, p.collision.LeftSource, p.collision.RightSource, p.collision.ConfiguredStrategy, "custom", "")
		return true, nil

	default:
		return false, fmt.Errorf("unknown resolution action: %d", p.resolution.Action)
	}
}

// componentResolutionParams contains parameters for applyComponentResolution.
// This is used for non-schema components (responses, parameters, etc.) that
// don't support Rename or Custom resolutions.
type componentResolutionParams struct {
	collision  CollisionContext
	resolution CollisionResolution
	result     *JoinResult
	ctx        documentContext
}

// applyComponentResolution applies a CollisionResolution to a generic component collision.
// Returns (handled, shouldOverwrite, error).
// - handled=true means the resolution was fully handled, no further action needed
// - shouldOverwrite=true means the target should be overwritten with source value
// This is for components that don't support Rename or Custom resolutions.
func (j *Joiner) applyComponentResolution(p componentResolutionParams) (handled bool, shouldOverwrite bool, err error) {
	// Record message as warning if provided
	if p.resolution.Message != "" {
		line, col := j.getLocation(p.ctx.filePath, p.collision.JSONPath)
		p.result.AddWarning(NewHandlerResolutionWarning(p.collision.JSONPath, p.resolution.Message, p.ctx.filePath, line, col))
	}

	switch p.resolution.Action {
	case ResolutionContinue:
		// Delegate to configured strategy
		return false, false, nil

	case ResolutionAcceptLeft:
		// Keep existing (left), discard incoming (right)
		j.recordCollisionEvent(p.result, p.collision.Name, p.collision.LeftSource, p.collision.RightSource, p.collision.ConfiguredStrategy, "kept-left", "")
		return true, false, nil

	case ResolutionAcceptRight:
		// Replace with incoming (right)
		j.recordCollisionEvent(p.result, p.collision.Name, p.collision.LeftSource, p.collision.RightSource, p.collision.ConfiguredStrategy, "kept-right", "")
		return true, true, nil

	case ResolutionDeduplicate:
		// Keep left, discard right (treat as equivalent)
		line, col := j.getLocation(p.ctx.filePath, p.collision.JSONPath)
		p.result.AddWarning(NewSchemaDedupWarning(p.collision.Name, string(p.collision.Type), p.ctx.filePath, line, col))
		j.recordCollisionEvent(p.result, p.collision.Name, p.collision.LeftSource, p.collision.RightSource, p.collision.ConfiguredStrategy, "deduplicated", "")
		return true, false, nil

	case ResolutionFail:
		// Return error with handler's message
		msg := p.resolution.Message
		if msg == "" {
			msg = fmt.Sprintf("%s collision on %q rejected by handler", p.collision.Type, p.collision.Name)
		}
		return true, false, fmt.Errorf("collision handler: %s", msg)

	case ResolutionRename:
		return true, false, fmt.Errorf("collision handler: ResolutionRename is not supported for %s collisions", p.collision.Type)

	case ResolutionCustom:
		return true, false, fmt.Errorf("collision handler: ResolutionCustom is not supported for %s collisions", p.collision.Type)

	default:
		return false, false, fmt.Errorf("unknown resolution action: %d", p.resolution.Action)
	}
}
