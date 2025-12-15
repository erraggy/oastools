package joiner

import (
	"testing"

	"github.com/erraggy/oastools/internal/severity"
	"github.com/stretchr/testify/assert"
)

func TestNewCollisionReport(t *testing.T) {
	report := NewCollisionReport()

	assert.NotNil(t, report)
	assert.Equal(t, 0, report.TotalCollisions)
	assert.Equal(t, 0, report.ResolvedByRename)
	assert.Equal(t, 0, report.ResolvedByDedup)
	assert.Equal(t, 0, report.ResolvedByAccept)
	assert.Equal(t, 0, report.FailedCollisions)
	assert.NotNil(t, report.Events)
	assert.Equal(t, 0, len(report.Events))
}

func TestCollisionReport_AddEvent(t *testing.T) {
	tests := []struct {
		name           string
		event          CollisionEvent
		expectedTotal  int
		expectedRename int
		expectedDedup  int
		expectedAccept int
		expectedFailed int
	}{
		{
			name: "renamed event",
			event: CollisionEvent{
				SchemaName: "User",
				Resolution: "renamed",
				NewName:    "User_left",
			},
			expectedTotal:  1,
			expectedRename: 1,
		},
		{
			name: "deduplicated event",
			event: CollisionEvent{
				SchemaName: "Product",
				Resolution: "deduplicated",
			},
			expectedTotal: 1,
			expectedDedup: 1,
		},
		{
			name: "kept-left event",
			event: CollisionEvent{
				SchemaName: "Order",
				Resolution: "kept-left",
			},
			expectedTotal:  1,
			expectedAccept: 1,
		},
		{
			name: "kept-right event",
			event: CollisionEvent{
				SchemaName: "Order",
				Resolution: "kept-right",
			},
			expectedTotal:  1,
			expectedAccept: 1,
		},
		{
			name: "failed event",
			event: CollisionEvent{
				SchemaName: "Error",
				Resolution: "failed",
			},
			expectedTotal:  1,
			expectedFailed: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := NewCollisionReport()
			report.AddEvent(tt.event)

			assert.Equal(t, tt.expectedTotal, report.TotalCollisions)
			assert.Equal(t, tt.expectedRename, report.ResolvedByRename)
			assert.Equal(t, tt.expectedDedup, report.ResolvedByDedup)
			assert.Equal(t, tt.expectedAccept, report.ResolvedByAccept)
			assert.Equal(t, tt.expectedFailed, report.FailedCollisions)
			assert.Equal(t, 1, len(report.Events))
		})
	}
}

func TestCollisionReport_AddMultipleEvents(t *testing.T) {
	report := NewCollisionReport()

	report.AddEvent(CollisionEvent{
		SchemaName: "User",
		Resolution: "renamed",
		NewName:    "User_left",
	})
	report.AddEvent(CollisionEvent{
		SchemaName: "Product",
		Resolution: "deduplicated",
	})
	report.AddEvent(CollisionEvent{
		SchemaName: "Order",
		Resolution: "kept-left",
	})

	assert.Equal(t, 3, report.TotalCollisions)
	assert.Equal(t, 1, report.ResolvedByRename)
	assert.Equal(t, 1, report.ResolvedByDedup)
	assert.Equal(t, 1, report.ResolvedByAccept)
	assert.Equal(t, 0, report.FailedCollisions)
	assert.Equal(t, 3, len(report.Events))
}

func TestCollisionReport_HasFailures(t *testing.T) {
	tests := []struct {
		name            string
		events          []CollisionEvent
		expectedFailure bool
	}{
		{
			name: "no failures",
			events: []CollisionEvent{
				{Resolution: "renamed"},
				{Resolution: "deduplicated"},
			},
			expectedFailure: false,
		},
		{
			name: "has failures",
			events: []CollisionEvent{
				{Resolution: "renamed"},
				{Resolution: "failed"},
			},
			expectedFailure: true,
		},
		{
			name:            "empty report",
			events:          []CollisionEvent{},
			expectedFailure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := NewCollisionReport()
			for _, event := range tt.events {
				report.AddEvent(event)
			}

			assert.Equal(t, tt.expectedFailure, report.HasFailures())
		})
	}
}

func TestCollisionReport_GetCriticalEvents(t *testing.T) {
	report := NewCollisionReport()

	report.AddEvent(CollisionEvent{
		SchemaName: "User",
		Resolution: "renamed",
		Severity:   severity.SeverityInfo,
	})
	report.AddEvent(CollisionEvent{
		SchemaName: "Product",
		Resolution: "failed",
		Severity:   severity.SeverityCritical,
	})
	report.AddEvent(CollisionEvent{
		SchemaName: "Order",
		Resolution: "kept-left",
		Severity:   severity.SeverityWarning,
	})
	report.AddEvent(CollisionEvent{
		SchemaName: "Payment",
		Resolution: "failed",
		Severity:   severity.SeverityCritical,
	})

	critical := report.GetCriticalEvents()

	assert.Equal(t, 2, len(critical))
	assert.Equal(t, "Product", critical[0].SchemaName)
	assert.Equal(t, "Payment", critical[1].SchemaName)
}

func TestCollisionReport_GetByResolution(t *testing.T) {
	report := NewCollisionReport()

	report.AddEvent(CollisionEvent{
		SchemaName: "User",
		Resolution: "renamed",
		NewName:    "User_left",
	})
	report.AddEvent(CollisionEvent{
		SchemaName: "Product",
		Resolution: "deduplicated",
	})
	report.AddEvent(CollisionEvent{
		SchemaName: "Order",
		Resolution: "renamed",
		NewName:    "Order_right",
	})
	report.AddEvent(CollisionEvent{
		SchemaName: "Payment",
		Resolution: "kept-left",
	})

	renamed := report.GetByResolution("renamed")
	assert.Equal(t, 2, len(renamed))
	assert.Equal(t, "User", renamed[0].SchemaName)
	assert.Equal(t, "Order", renamed[1].SchemaName)

	dedup := report.GetByResolution("deduplicated")
	assert.Equal(t, 1, len(dedup))
	assert.Equal(t, "Product", dedup[0].SchemaName)

	kept := report.GetByResolution("kept-left")
	assert.Equal(t, 1, len(kept))
	assert.Equal(t, "Payment", kept[0].SchemaName)

	failed := report.GetByResolution("failed")
	assert.Equal(t, 0, len(failed))
}

func TestSchemaDifference(t *testing.T) {
	diff := SchemaDifference{
		Path:        "properties.name.type",
		LeftValue:   "string",
		RightValue:  "integer",
		Description: "type mismatch",
	}

	assert.Equal(t, "properties.name.type", diff.Path)
	assert.Equal(t, "string", diff.LeftValue)
	assert.Equal(t, "integer", diff.RightValue)
	assert.Equal(t, "type mismatch", diff.Description)
}

func TestCollisionEvent(t *testing.T) {
	event := CollisionEvent{
		SchemaName:  "User",
		LeftSource:  "api-v1.yaml",
		RightSource: "api-v2.yaml",
		Strategy:    StrategyAcceptLeft,
		Resolution:  "renamed",
		NewName:     "User_v1",
		Differences: []SchemaDifference{
			{
				Path:        "properties.email",
				LeftValue:   "present",
				RightValue:  "absent",
				Description: "property difference",
			},
		},
		Severity: severity.SeverityInfo,
	}

	assert.Equal(t, "User", event.SchemaName)
	assert.Equal(t, "api-v1.yaml", event.LeftSource)
	assert.Equal(t, "api-v2.yaml", event.RightSource)
	assert.Equal(t, StrategyAcceptLeft, event.Strategy)
	assert.Equal(t, "renamed", event.Resolution)
	assert.Equal(t, "User_v1", event.NewName)
	assert.Equal(t, 1, len(event.Differences))
	assert.Equal(t, severity.SeverityInfo, event.Severity)
}

func TestCollisionEvent_WithLineNumbers(t *testing.T) {
	event := CollisionEvent{
		SchemaName:  "Pet",
		LeftSource:  "pets.yaml",
		LeftLine:    42,
		LeftColumn:  5,
		RightSource: "animals.yaml",
		RightLine:   108,
		RightColumn: 3,
		Strategy:    StrategyFailOnCollision,
		Resolution:  "failed",
	}

	assert.Equal(t, "Pet", event.SchemaName)
	assert.Equal(t, "pets.yaml", event.LeftSource)
	assert.Equal(t, 42, event.LeftLine)
	assert.Equal(t, 5, event.LeftColumn)
	assert.Equal(t, "animals.yaml", event.RightSource)
	assert.Equal(t, 108, event.RightLine)
	assert.Equal(t, 3, event.RightColumn)
}

func TestCollisionEvent_ZeroLineNumbers(t *testing.T) {
	// Line numbers of 0 indicate unknown location
	event := CollisionEvent{
		SchemaName:  "User",
		LeftSource:  "api.yaml",
		LeftLine:    0,
		LeftColumn:  0,
		RightSource: "other.yaml",
		RightLine:   0,
		RightColumn: 0,
	}

	assert.Equal(t, 0, event.LeftLine)
	assert.Equal(t, 0, event.LeftColumn)
	assert.Equal(t, 0, event.RightLine)
	assert.Equal(t, 0, event.RightColumn)
}

func TestSchemaDifference_WithLineNumbers(t *testing.T) {
	diff := SchemaDifference{
		Path:        "properties.email.format",
		LeftValue:   "email",
		LeftLine:    25,
		LeftColumn:  12,
		RightValue:  "string",
		RightLine:   30,
		RightColumn: 14,
		Description: "format mismatch",
	}

	assert.Equal(t, "properties.email.format", diff.Path)
	assert.Equal(t, "email", diff.LeftValue)
	assert.Equal(t, 25, diff.LeftLine)
	assert.Equal(t, 12, diff.LeftColumn)
	assert.Equal(t, "string", diff.RightValue)
	assert.Equal(t, 30, diff.RightLine)
	assert.Equal(t, 14, diff.RightColumn)
}
