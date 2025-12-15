package joiner

import "github.com/erraggy/oastools/internal/severity"

// CollisionReport provides detailed analysis of collisions encountered during join operations
type CollisionReport struct {
	TotalCollisions  int
	ResolvedByRename int
	ResolvedByDedup  int
	ResolvedByAccept int
	FailedCollisions int
	Events           []CollisionEvent
}

// CollisionEvent represents a single collision occurrence with resolution details
type CollisionEvent struct {
	SchemaName  string
	LeftSource  string
	LeftLine    int // 1-based line number in left source (0 if unknown)
	LeftColumn  int // 1-based column number in left source (0 if unknown)
	RightSource string
	RightLine   int // 1-based line number in right source (0 if unknown)
	RightColumn int // 1-based column number in right source (0 if unknown)
	Strategy    CollisionStrategy
	Resolution  string // "renamed", "deduplicated", "kept-left", "kept-right", "failed"
	NewName     string // For rename resolutions
	Differences []SchemaDifference
	Severity    severity.Severity
}

// SchemaDifference describes a structural difference between two schemas
type SchemaDifference struct {
	Path        string // JSON path to differing element (e.g., "properties.name.type")
	LeftValue   any
	LeftLine    int // 1-based line number for left value (0 if unknown)
	LeftColumn  int // 1-based column number for left value (0 if unknown)
	RightValue  any
	RightLine   int // 1-based line number for right value (0 if unknown)
	RightColumn int // 1-based column number for right value (0 if unknown)
	Description string
}

// NewCollisionReport creates an empty collision report
func NewCollisionReport() *CollisionReport {
	return &CollisionReport{
		Events: make([]CollisionEvent, 0),
	}
}

// AddEvent adds a collision event to the report and updates counters
func (r *CollisionReport) AddEvent(event CollisionEvent) {
	r.Events = append(r.Events, event)
	r.TotalCollisions++

	switch event.Resolution {
	case "renamed":
		r.ResolvedByRename++
	case "deduplicated":
		r.ResolvedByDedup++
	case "kept-left", "kept-right":
		r.ResolvedByAccept++
	case "failed":
		r.FailedCollisions++
	}
}

// HasFailures returns true if any collisions failed to resolve
func (r *CollisionReport) HasFailures() bool {
	return r.FailedCollisions > 0
}

// GetCriticalEvents returns events with Critical severity
func (r *CollisionReport) GetCriticalEvents() []CollisionEvent {
	var critical []CollisionEvent
	for _, event := range r.Events {
		if event.Severity == severity.SeverityCritical {
			critical = append(critical, event)
		}
	}
	return critical
}

// GetByResolution returns events with a specific resolution type
func (r *CollisionReport) GetByResolution(resolution string) []CollisionEvent {
	var filtered []CollisionEvent
	for _, event := range r.Events {
		if event.Resolution == resolution {
			filtered = append(filtered, event)
		}
	}
	return filtered
}
