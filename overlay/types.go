package overlay

import (
	"github.com/erraggy/oastools/parser"
)

// Overlay represents an OpenAPI Overlay document (v1.0.0).
//
// The Overlay specification provides a standardized mechanism for augmenting
// OpenAPI documents through targeted transformations using JSONPath expressions.
type Overlay struct {
	// Version is the overlay specification version (e.g., "1.0.0").
	// This field is required.
	Version string `yaml:"overlay" json:"overlay"`

	// Info contains metadata about the overlay.
	// This field is required.
	Info Info `yaml:"info" json:"info"`

	// Extends is an optional URI reference to the target OpenAPI document.
	// When specified, it indicates which document this overlay is designed for.
	Extends string `yaml:"extends,omitempty" json:"extends,omitempty"`

	// Actions is the ordered list of transformation actions.
	// At least one action is required.
	Actions []Action `yaml:"actions" json:"actions"`
}

// Info contains metadata about an overlay document.
type Info struct {
	// Title is the human-readable name of the overlay.
	// This field is required.
	Title string `yaml:"title" json:"title"`

	// Version is the version of the overlay document.
	// This field is required.
	Version string `yaml:"version" json:"version"`
}

// Action represents a single transformation action in an overlay.
//
// Each action targets specific locations in the OpenAPI document using
// JSONPath expressions and either updates or removes the matched nodes.
type Action struct {
	// Target is a JSONPath expression selecting nodes to operate on.
	// This field is required.
	Target string `yaml:"target" json:"target"`

	// Description is an optional human-readable explanation of the action.
	// CommonMark syntax may be used for rich text representation.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Update specifies content to merge with selected nodes.
	// For objects, properties are recursively merged.
	// For arrays, the update value is appended.
	Update any `yaml:"update,omitempty" json:"update,omitempty"`

	// Remove, when true, removes the target from its parent.
	// Remove takes precedence over Update when both are specified.
	Remove bool `yaml:"remove,omitempty" json:"remove,omitempty"`
}

// ApplyResult contains the result of applying an overlay to a document.
type ApplyResult struct {
	// Document is the transformed OpenAPI document.
	Document any

	// SourceFormat is the original document format (YAML or JSON).
	SourceFormat parser.SourceFormat

	// ActionsApplied is the number of actions that were successfully applied.
	ActionsApplied int

	// ActionsSkipped is the number of actions that were skipped (e.g., no matches).
	ActionsSkipped int

	// Changes records details of each applied change.
	Changes []ChangeRecord

	// Warnings contains non-fatal issues encountered during application.
	Warnings []string
}

// ChangeRecord describes a single change made during overlay application.
type ChangeRecord struct {
	// ActionIndex is the zero-based index of the action in the overlay.
	ActionIndex int

	// Target is the JSONPath expression that was evaluated.
	Target string

	// Operation describes what was done: "update", "remove", or "append".
	Operation string

	// MatchCount is the number of nodes matched by the target.
	MatchCount int
}

// HasChanges returns true if any actions were applied.
func (r *ApplyResult) HasChanges() bool {
	return r.ActionsApplied > 0
}

// HasWarnings returns true if any warnings were generated.
func (r *ApplyResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// DryRunResult contains the result of a dry-run overlay preview.
//
// A dry-run evaluates the overlay without modifying the document,
// allowing users to see what changes would be made.
type DryRunResult struct {
	// WouldApply is the number of actions that would be successfully applied.
	WouldApply int

	// WouldSkip is the number of actions that would be skipped (e.g., no matches).
	WouldSkip int

	// Changes lists the proposed changes that would be made.
	Changes []ProposedChange

	// Warnings contains non-fatal issues that would occur during application.
	Warnings []string
}

// ProposedChange describes a change that would be made during overlay application.
type ProposedChange struct {
	// ActionIndex is the zero-based index of the action in the overlay.
	ActionIndex int

	// Target is the JSONPath expression that was evaluated.
	Target string

	// Description is the action's description, if provided.
	Description string

	// Operation describes what would be done: "update", "remove", "replace", or "append".
	Operation string

	// MatchCount is the number of nodes that would be affected.
	MatchCount int

	// MatchedPaths lists the JSONPath locations of matched nodes (up to 10).
	MatchedPaths []string
}

// HasChanges returns true if any changes would be made.
func (r *DryRunResult) HasChanges() bool {
	return r.WouldApply > 0
}

// HasWarnings returns true if any warnings would occur.
func (r *DryRunResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}
