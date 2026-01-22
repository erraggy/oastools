package issues

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperationContextString(t *testing.T) {
	tests := []struct {
		name     string
		ctx      OperationContext
		expected string
	}{
		{
			name: "operation with operationId",
			ctx: OperationContext{
				Method:      "GET",
				Path:        "/users/{id}",
				OperationID: "getUser",
			},
			expected: "(operationId: getUser)",
		},
		{
			name: "operation without operationId",
			ctx: OperationContext{
				Method: "GET",
				Path:   "/users/{id}",
			},
			expected: "(GET /users/{id})",
		},
		{
			name: "path-level (no method)",
			ctx: OperationContext{
				Path: "/users/{id}",
			},
			expected: "(path: /users/{id})",
		},
		{
			name: "reusable component with operationId",
			ctx: OperationContext{
				Method:              "GET",
				Path:                "/users",
				OperationID:         "listUsers",
				IsReusableComponent: true,
				AdditionalRefs:      3,
			},
			expected: "(operationId: listUsers, +3 operations)",
		},
		{
			name: "reusable component without operationId",
			ctx: OperationContext{
				Method:              "POST",
				Path:                "/orders",
				IsReusableComponent: true,
				AdditionalRefs:      5,
			},
			expected: "(POST /orders, +5 operations)",
		},
		{
			name: "reusable component single ref",
			ctx: OperationContext{
				Method:              "GET",
				Path:                "/users",
				OperationID:         "listUsers",
				IsReusableComponent: true,
				AdditionalRefs:      0,
			},
			expected: "(operationId: listUsers)",
		},
		{
			name: "unused component",
			ctx: OperationContext{
				IsReusableComponent: true,
				AdditionalRefs:      -1, // sentinel for unused
			},
			expected: "(unused component)",
		},
		{
			name: "webhook context",
			ctx: OperationContext{
				Method:    "POST",
				Path:      "orderCreated",
				IsWebhook: true,
			},
			expected: "(webhook: orderCreated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOperationContextIsEmpty(t *testing.T) {
	assert.True(t, OperationContext{}.IsEmpty())
	assert.False(t, OperationContext{Path: "/users"}.IsEmpty())
	assert.False(t, OperationContext{Method: "GET"}.IsEmpty())
	assert.False(t, OperationContext{OperationID: "test"}.IsEmpty())
	assert.False(t, OperationContext{IsReusableComponent: true, AdditionalRefs: -1}.IsEmpty())
}
