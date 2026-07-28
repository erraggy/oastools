package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// The Schema Object's discriminator is a bare string in OAS 2.0 and an object
// in OAS 3.0+. One Go type serves both, so every decode path has to recognize
// both spellings and every encode path has to reproduce the one it was given.
// The parser has three independent decode paths: encoding/json, YAML, and the
// generated decodeFromMap used when ResolveRefs is enabled.

func TestDiscriminatorUnmarshalJSONStringForm(t *testing.T) {
	var d Discriminator
	require.NoError(t, json.Unmarshal([]byte(`"petType"`), &d))

	assert.Equal(t, "petType", d.PropertyName)
	assert.True(t, d.StringForm, "string form must be recorded so it round-trips")
	assert.Nil(t, d.Mapping)
}

func TestDiscriminatorUnmarshalJSONObjectForm(t *testing.T) {
	var d Discriminator
	require.NoError(t, json.Unmarshal(
		[]byte(`{"propertyName":"petType","mapping":{"dog":"#/components/schemas/Dog"},"x-vendor":true}`), &d))

	assert.Equal(t, "petType", d.PropertyName)
	assert.False(t, d.StringForm)
	assert.Equal(t, map[string]string{"dog": "#/components/schemas/Dog"}, d.Mapping)
	assert.Equal(t, true, d.Extra["x-vendor"])
}

func TestDiscriminatorUnmarshalJSONInvalidForm(t *testing.T) {
	var d Discriminator
	assert.Error(t, json.Unmarshal([]byte(`["petType"]`), &d))
}

func TestDiscriminatorMarshalJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"OAS 2.0 string form", `"petType"`},
		{"OAS 3.x object form", `{"propertyName":"petType"}`},
		{"OAS 3.x with mapping", `{"propertyName":"petType","mapping":{"dog":"Dog"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Discriminator
			require.NoError(t, json.Unmarshal([]byte(tt.input), &d))

			out, err := json.Marshal(&d)
			require.NoError(t, err)
			assert.JSONEq(t, tt.input, string(out))
		})
	}
}

func TestDiscriminatorUnmarshalYAMLStringForm(t *testing.T) {
	var s Schema
	require.NoError(t, yaml.Unmarshal([]byte("type: object\ndiscriminator: petType\n"), &s))

	require.NotNil(t, s.Discriminator)
	assert.Equal(t, "petType", s.Discriminator.PropertyName)
	assert.True(t, s.Discriminator.StringForm)
}

func TestDiscriminatorUnmarshalYAMLObjectForm(t *testing.T) {
	var s Schema
	require.NoError(t, yaml.Unmarshal(
		[]byte("type: object\ndiscriminator:\n  propertyName: petType\n  mapping:\n    dog: Dog\n"), &s))

	require.NotNil(t, s.Discriminator)
	assert.Equal(t, "petType", s.Discriminator.PropertyName)
	assert.False(t, s.Discriminator.StringForm)
	assert.Equal(t, map[string]string{"dog": "Dog"}, s.Discriminator.Mapping)
}

func TestDiscriminatorUnmarshalYAMLAnchor(t *testing.T) {
	// An anchored scalar must still be classified as the string form rather
	// than falling through to the mapping branch.
	src := "x-name: &petType petType\ntype: object\ndiscriminator: *petType\n"

	var s Schema
	require.NoError(t, yaml.Unmarshal([]byte(src), &s))

	require.NotNil(t, s.Discriminator)
	assert.Equal(t, "petType", s.Discriminator.PropertyName)
	assert.True(t, s.Discriminator.StringForm)
}

func TestDiscriminatorUnmarshalYAMLInvalidForm(t *testing.T) {
	var s Schema
	err := yaml.Unmarshal([]byte("type: object\ndiscriminator:\n  - petType\n"), &s)

	require.Error(t, err)
	assert.Contains(t, err.Error(),
		"'discriminator' must be a string (OAS 2.0) or a mapping (OAS 3.0+), but got: SequenceNode")
	// yaml supplies its own "line N:" prefix; ours must not duplicate it.
	assert.NotContains(t, err.Error(), "line 3: line 3:")
}

func TestDiscriminatorMarshalYAMLRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"OAS 2.0 string form", "discriminator: petType\n", "discriminator: petType\n"},
		{"OAS 3.x object form", "discriminator:\n  propertyName: petType\n", "propertyName: petType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Schema
			require.NoError(t, yaml.Unmarshal([]byte(tt.src), &s))

			out, err := yaml.Marshal(&s)
			require.NoError(t, err)
			assert.Contains(t, string(out), tt.want)
		})
	}
}

func TestDiscriminatorDecodeFromMap(t *testing.T) {
	// decodeFromMap is the path taken when ResolveRefs is enabled. Before the
	// string form was handled here it was silently dropped rather than erroring.
	tests := []struct {
		name           string
		value          any
		wantProperty   string
		wantStringForm bool
		wantNil        bool
	}{
		{name: "OAS 2.0 string form", value: "petType", wantProperty: "petType", wantStringForm: true},
		{name: "OAS 3.x object form", value: map[string]any{"propertyName": "petType"}, wantProperty: "petType"},
		{name: "unsupported form", value: []any{"petType"}, wantNil: true},
		{name: "absent", value: nil, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]any{"type": "object"}
			if tt.value != nil {
				m["discriminator"] = tt.value
			}

			var s Schema
			s.decodeFromMap(m)

			if tt.wantNil {
				assert.Nil(t, s.Discriminator)
				return
			}
			require.NotNil(t, s.Discriminator)
			assert.Equal(t, tt.wantProperty, s.Discriminator.PropertyName)
			assert.Equal(t, tt.wantStringForm, s.Discriminator.StringForm)
		})
	}
}

func TestParseOAS2StringDiscriminator(t *testing.T) {
	// Regression for erraggy/oastools#394: a spec-conformant OAS 2.0 document
	// with a string discriminator was rejected outright at parse time.
	for _, path := range []string{
		"../testdata/discriminator-2.0.yaml",
		"../testdata/discriminator-2.0.json",
	} {
		t.Run(path, func(t *testing.T) {
			result, err := New().Parse(path)
			require.NoError(t, err)
			require.Empty(t, result.Errors)

			doc, ok := result.Document.(*OAS2Document)
			require.True(t, ok)

			pet := doc.Definitions["Pet"]
			require.NotNil(t, pet)
			require.NotNil(t, pet.Discriminator)
			assert.Equal(t, "petType", pet.Discriminator.PropertyName)
			assert.True(t, pet.Discriminator.StringForm)
		})
	}
}

func TestParseOAS2StringDiscriminatorWithResolvedRefs(t *testing.T) {
	// ResolveRefs swaps in decodeFromMap, a decode path the plain Parse tests
	// never reach.
	p := New()
	p.ResolveRefs = true

	result, err := p.Parse("../testdata/discriminator-2.0.yaml")
	require.NoError(t, err)

	doc, ok := result.Document.(*OAS2Document)
	require.True(t, ok)

	pet := doc.Definitions["Pet"]
	require.NotNil(t, pet)
	require.NotNil(t, pet.Discriminator, "discriminator must survive $ref resolution")
	assert.Equal(t, "petType", pet.Discriminator.PropertyName)
	assert.True(t, pet.Discriminator.StringForm)
}

func TestOAS2StringDiscriminatorSurvivesSerialization(t *testing.T) {
	// The CLI writes fixed/joined documents by marshaling the typed document,
	// so a 2.0 document must not come back out in the OAS 3.x object form.
	for _, path := range []string{
		"../testdata/discriminator-2.0.yaml",
		"../testdata/discriminator-2.0.json",
	} {
		t.Run(path, func(t *testing.T) {
			result, err := New().Parse(path)
			require.NoError(t, err)

			jsonOut, err := json.Marshal(result.Document)
			require.NoError(t, err)
			assert.Contains(t, string(jsonOut), `"discriminator":"petType"`)
			assert.NotContains(t, string(jsonOut), `"propertyName"`)

			yamlOut, err := yaml.Marshal(result.Document)
			require.NoError(t, err)
			assert.Contains(t, string(yamlOut), "discriminator: petType")
			assert.NotContains(t, string(yamlOut), "propertyName")
		})
	}
}

func TestDiscriminatorDeepCopyPreservesStringForm(t *testing.T) {
	d := &Discriminator{PropertyName: "petType", StringForm: true}

	got := d.DeepCopy()
	require.NotNil(t, got)
	assert.True(t, got.StringForm)
	assert.Equal(t, "petType", got.PropertyName)
}

func TestDiscriminatorEqualsIgnoresStringForm(t *testing.T) {
	// StringForm records the dialect's spelling, not the meaning: both forms
	// select the same property, so a cross-version diff must not report one.
	a := &Discriminator{PropertyName: "petType", StringForm: true}
	b := &Discriminator{PropertyName: "petType"}

	assert.True(t, equalDiscriminator(a, b))
}
