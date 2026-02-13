package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathBuilder_Basic(t *testing.T) {
	p := &PathBuilder{}
	p.Push("properties")
	p.Push("name")

	got := p.String()
	want := "properties.name"
	assert.Equal(t, want, got)
}

func TestPathBuilder_WithIndex(t *testing.T) {
	p := &PathBuilder{}
	p.Push("allOf")
	p.PushIndex(0)
	p.Push("properties")

	got := p.String()
	want := "allOf[0].properties"
	assert.Equal(t, want, got)
}

func TestPathBuilder_PushPop(t *testing.T) {
	p := &PathBuilder{}
	p.Push("a")
	p.Push("b")
	p.Pop()
	p.Push("c")

	got := p.String()
	want := "a.c"
	assert.Equal(t, want, got)
}

func TestPathBuilder_Empty(t *testing.T) {
	p := &PathBuilder{}
	got := p.String()
	assert.Equal(t, "", got)
}

func TestPathBuilder_PopEmpty(t *testing.T) {
	p := &PathBuilder{}
	p.Pop() // Should not panic
	got := p.String()
	assert.Equal(t, "", got)
}

func TestPathBuilder_Reset(t *testing.T) {
	p := &PathBuilder{}
	p.Push("a")
	p.Push("b")
	p.Reset()

	got := p.String()
	assert.Equal(t, "", got)

	// Should be reusable after reset
	p.Push("c")
	got = p.String()
	assert.Equal(t, "c", got)
}

func TestPool_GetPut(t *testing.T) {
	p := Get()
	require.NotNil(t, p)

	p.Push("test")
	Put(p)

	// Get another - may or may not be same instance
	p2 := Get()
	require.NotNil(t, p2)
	// After Get, should be reset
	assert.Equal(t, "", p2.String())
	Put(p2)
}

func TestSchemaRef(t *testing.T) {
	got := SchemaRef("Pet")
	want := "#/components/schemas/Pet"
	assert.Equal(t, want, got)
}

func TestDefinitionRef(t *testing.T) {
	got := DefinitionRef("Pet")
	want := "#/definitions/Pet"
	assert.Equal(t, want, got)
}

func TestParameterRef(t *testing.T) {
	tests := []struct {
		name    string
		version int // 2 for OAS2, 3 for OAS3
		want    string
	}{
		{"limitParam", 2, "#/parameters/limitParam"},
		{"limitParam", 3, "#/components/parameters/limitParam"},
	}
	for _, tt := range tests {
		got := ParameterRef(tt.name, tt.version == 2)
		assert.Equal(t, tt.want, got, "ParameterRef(%q, oas2=%v)", tt.name, tt.version == 2)
	}
}

func TestResponseRef(t *testing.T) {
	tests := []struct {
		name    string
		version int
		want    string
	}{
		{"NotFound", 2, "#/responses/NotFound"},
		{"NotFound", 3, "#/components/responses/NotFound"},
	}
	for _, tt := range tests {
		got := ResponseRef(tt.name, tt.version == 2)
		assert.Equal(t, tt.want, got, "ResponseRef(%q, oas2=%v)", tt.name, tt.version == 2)
	}
}

func TestSecuritySchemeRef(t *testing.T) {
	tests := []struct {
		name    string
		version int
		want    string
	}{
		{"api_key", 2, "#/securityDefinitions/api_key"},
		{"api_key", 3, "#/components/securitySchemes/api_key"},
	}
	for _, tt := range tests {
		got := SecuritySchemeRef(tt.name, tt.version == 2)
		assert.Equal(t, tt.want, got, "SecuritySchemeRef(%q, oas2=%v)", tt.name, tt.version == 2)
	}
}

func TestHeaderRef(t *testing.T) {
	got := HeaderRef("X-Rate-Limit")
	want := "#/components/headers/X-Rate-Limit"
	assert.Equal(t, want, got)
}

func TestRequestBodyRef(t *testing.T) {
	got := RequestBodyRef("PetRequest")
	want := "#/components/requestBodies/PetRequest"
	assert.Equal(t, want, got)
}

func TestExampleRef(t *testing.T) {
	got := ExampleRef("PetExample")
	want := "#/components/examples/PetExample"
	assert.Equal(t, want, got)
}

func TestLinkRef(t *testing.T) {
	got := LinkRef("GetPetById")
	want := "#/components/links/GetPetById"
	assert.Equal(t, want, got)
}

func TestCallbackRef(t *testing.T) {
	got := CallbackRef("onData")
	want := "#/components/callbacks/onData"
	assert.Equal(t, want, got)
}

func TestPathItemRef(t *testing.T) {
	got := PathItemRef("UserPath")
	want := "#/components/pathItems/UserPath"
	assert.Equal(t, want, got)
}
