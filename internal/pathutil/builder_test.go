package pathutil

import "testing"

func TestPathBuilder_Basic(t *testing.T) {
	p := &PathBuilder{}
	p.Push("properties")
	p.Push("name")

	got := p.String()
	want := "properties.name"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPathBuilder_WithIndex(t *testing.T) {
	p := &PathBuilder{}
	p.Push("allOf")
	p.PushIndex(0)
	p.Push("properties")

	got := p.String()
	want := "allOf[0].properties"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPathBuilder_PushPop(t *testing.T) {
	p := &PathBuilder{}
	p.Push("a")
	p.Push("b")
	p.Pop()
	p.Push("c")

	got := p.String()
	want := "a.c"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPathBuilder_Empty(t *testing.T) {
	p := &PathBuilder{}
	got := p.String()
	if got != "" {
		t.Errorf("String() on empty = %q, want empty", got)
	}
}

func TestPathBuilder_PopEmpty(t *testing.T) {
	p := &PathBuilder{}
	p.Pop() // Should not panic
	got := p.String()
	if got != "" {
		t.Errorf("String() after Pop on empty = %q, want empty", got)
	}
}

func TestPathBuilder_Reset(t *testing.T) {
	p := &PathBuilder{}
	p.Push("a")
	p.Push("b")
	p.Reset()

	got := p.String()
	if got != "" {
		t.Errorf("String() after Reset = %q, want empty", got)
	}

	// Should be reusable after reset
	p.Push("c")
	got = p.String()
	if got != "c" {
		t.Errorf("String() after Reset+Push = %q, want %q", got, "c")
	}
}

func TestPool_GetPut(t *testing.T) {
	p := Get()
	if p == nil {
		t.Fatal("Get() returned nil")
	}

	p.Push("test")
	Put(p)

	// Get another - may or may not be same instance
	p2 := Get()
	if p2 == nil {
		t.Fatal("Get() returned nil after Put")
	}
	// After Get, should be reset
	if p2.String() != "" {
		t.Errorf("Get() returned non-empty PathBuilder: %q", p2.String())
	}
	Put(p2)
}

func TestSchemaRef(t *testing.T) {
	got := SchemaRef("Pet")
	want := "#/components/schemas/Pet"
	if got != want {
		t.Errorf("SchemaRef(Pet) = %q, want %q", got, want)
	}
}

func TestDefinitionRef(t *testing.T) {
	got := DefinitionRef("Pet")
	want := "#/definitions/Pet"
	if got != want {
		t.Errorf("DefinitionRef(Pet) = %q, want %q", got, want)
	}
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
		if got != tt.want {
			t.Errorf("ParameterRef(%q, oas2=%v) = %q, want %q", tt.name, tt.version == 2, got, tt.want)
		}
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
		if got != tt.want {
			t.Errorf("ResponseRef(%q, oas2=%v) = %q, want %q", tt.name, tt.version == 2, got, tt.want)
		}
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
		if got != tt.want {
			t.Errorf("SecuritySchemeRef(%q, oas2=%v) = %q, want %q", tt.name, tt.version == 2, got, tt.want)
		}
	}
}

func TestHeaderRef(t *testing.T) {
	got := HeaderRef("X-Rate-Limit")
	want := "#/components/headers/X-Rate-Limit"
	if got != want {
		t.Errorf("HeaderRef(X-Rate-Limit) = %q, want %q", got, want)
	}
}

func TestRequestBodyRef(t *testing.T) {
	got := RequestBodyRef("PetRequest")
	want := "#/components/requestBodies/PetRequest"
	if got != want {
		t.Errorf("RequestBodyRef(PetRequest) = %q, want %q", got, want)
	}
}

func TestExampleRef(t *testing.T) {
	got := ExampleRef("PetExample")
	want := "#/components/examples/PetExample"
	if got != want {
		t.Errorf("ExampleRef(PetExample) = %q, want %q", got, want)
	}
}

func TestLinkRef(t *testing.T) {
	got := LinkRef("GetPetById")
	want := "#/components/links/GetPetById"
	if got != want {
		t.Errorf("LinkRef(GetPetById) = %q, want %q", got, want)
	}
}

func TestCallbackRef(t *testing.T) {
	got := CallbackRef("onData")
	want := "#/components/callbacks/onData"
	if got != want {
		t.Errorf("CallbackRef(onData) = %q, want %q", got, want)
	}
}

func TestPathItemRef(t *testing.T) {
	got := PathItemRef("UserPath")
	want := "#/components/pathItems/UserPath"
	if got != want {
		t.Errorf("PathItemRef(UserPath) = %q, want %q", got, want)
	}
}
