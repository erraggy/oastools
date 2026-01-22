package validator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/issues"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationContextIntegration(t *testing.T) {
	// Test with testdata/invalid-oas3.yaml
	v := New()
	result, err := v.Validate("../testdata/invalid-oas3.yaml")
	require.NoError(t, err)

	t.Logf("Found %d errors, %d warnings", result.ErrorCount, result.WarningCount)

	for i, e := range result.Errors {
		str := e.String()
		t.Logf("Error %d: %s", i+1, str)

		// Verify format based on path type
		if strings.HasPrefix(e.Path, "paths.") {
			// Should have some form of operation context
			if e.OperationContext != nil && !e.OperationContext.IsEmpty() {
				assert.Contains(t, str, "(", "paths error should have context in output")
			}
		}
	}

	for i, w := range result.Warnings {
		str := w.String()
		t.Logf("Warning %d: %s", i+1, str)
	}
}

func TestOperationContextFormats(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		opCtx        *issues.OperationContext
		wantContains string
	}{
		{
			name: "operation with operationId",
			path: "paths./users.get.responses",
			opCtx: &issues.OperationContext{
				Method:      "GET",
				Path:        "/users",
				OperationID: "listUsers",
			},
			wantContains: "(operationId: listUsers)",
		},
		{
			name: "operation without operationId",
			path: "paths./orders.post.requestBody",
			opCtx: &issues.OperationContext{
				Method: "POST",
				Path:   "/orders",
			},
			wantContains: "(POST /orders)",
		},
		{
			name: "path-level parameter",
			path: "paths./users/{id}.parameters[0]",
			opCtx: &issues.OperationContext{
				Path: "/users/{id}",
			},
			wantContains: "(path: /users/{id})",
		},
		{
			name: "shared schema with multiple refs",
			path: "components.schemas.User.properties.email",
			opCtx: &issues.OperationContext{
				Method:              "GET",
				Path:                "/users",
				OperationID:         "listUsers",
				IsReusableComponent: true,
				AdditionalRefs:      3,
			},
			wantContains: "(operationId: listUsers, +3 operations)",
		},
		{
			name: "unused component",
			path: "components.schemas.Orphan.type",
			opCtx: &issues.OperationContext{
				IsReusableComponent: true,
				AdditionalRefs:      -1,
			},
			wantContains: "(unused component)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := ValidationError{
				Path:             tt.path,
				Message:          "test error",
				Severity:         SeverityError,
				OperationContext: tt.opCtx,
			}
			str := issue.String()
			assert.Contains(t, str, tt.wantContains)
		})
	}
}
