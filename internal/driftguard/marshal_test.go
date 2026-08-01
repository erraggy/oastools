package driftguard

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// Every parser type marshals through two paths: a fast one that hands the struct
// to encoding/json and lets the tags do the work, and a slow one that hand-builds
// a map[string]any, taken whenever Extra is non-empty because Go cannot inline a
// map the way yaml:",inline" does.
//
// Adding a field to the struct wires up the fast path only. The slow path is a
// separate list edited by hand, so a field can be added, tested and shipped while
// silently vanishing from every document that also carries an x- extension.
//
// These two guards run the same field set through both paths. A field present in
// the tags but missing from the map builder passes the fast guard and fails the
// slow one, which is exactly the shape of #397 and of the nine fields #414 found.

// marshalExclusions lists fields the marshalers deliberately do not emit, with
// the reason. Everything else must survive both paths.
var marshalExclusions = map[string]map[string]string{
	"Discriminator": {
		// Spelling rather than meaning: it records which dialect the document used.
		// Setting it does not drop a key, it re-spells the Discriminator as the OAS
		// 2.0 bare string, which is why this case is checked before decoding.
		"StringForm": "selects the OAS 2.0 bare-string form",
	},
	"Schema": {
		// Same shape as StringForm, one level up: setting it re-spells the whole
		// Schema as a bare boolean, so the output is the scalar `true` or `false`
		// and there is no object left to decode. The value it carries is not lost —
		// it *is* the output — so this exclusion covers the key, not the meaning.
		"BoolForm": "re-spells the schema as a bare boolean",
	},
}

// marshalSubjects pairs each type carrying a hand-built MarshalJSON with a fresh
// value and its field list. A guard cannot reflect over a package, so this list
// is the one thing a new type has to be added to.
func marshalSubjects() map[string]func() (any, []field) {
	return map[string]func() (any, []field){
		"OAS3Document":   func() (any, []field) { return &parser.OAS3Document{}, fieldsOf[parser.OAS3Document]() },
		"OAS2Document":   func() (any, []field) { return &parser.OAS2Document{}, fieldsOf[parser.OAS2Document]() },
		"Components":     func() (any, []field) { return &parser.Components{}, fieldsOf[parser.Components]() },
		"PathItem":       func() (any, []field) { return &parser.PathItem{}, fieldsOf[parser.PathItem]() },
		"Operation":      func() (any, []field) { return &parser.Operation{}, fieldsOf[parser.Operation]() },
		"Response":       func() (any, []field) { return &parser.Response{}, fieldsOf[parser.Response]() },
		"MediaType":      func() (any, []field) { return &parser.MediaType{}, fieldsOf[parser.MediaType]() },
		"Example":        func() (any, []field) { return &parser.Example{}, fieldsOf[parser.Example]() },
		"Encoding":       func() (any, []field) { return &parser.Encoding{}, fieldsOf[parser.Encoding]() },
		"Link":           func() (any, []field) { return &parser.Link{}, fieldsOf[parser.Link]() },
		"Parameter":      func() (any, []field) { return &parser.Parameter{}, fieldsOf[parser.Parameter]() },
		"Header":         func() (any, []field) { return &parser.Header{}, fieldsOf[parser.Header]() },
		"Items":          func() (any, []field) { return &parser.Items{}, fieldsOf[parser.Items]() },
		"RequestBody":    func() (any, []field) { return &parser.RequestBody{}, fieldsOf[parser.RequestBody]() },
		"Schema":         func() (any, []field) { return &parser.Schema{}, fieldsOf[parser.Schema]() },
		"Discriminator":  func() (any, []field) { return &parser.Discriminator{}, fieldsOf[parser.Discriminator]() },
		"XML":            func() (any, []field) { return &parser.XML{}, fieldsOf[parser.XML]() },
		"Tag":            func() (any, []field) { return &parser.Tag{}, fieldsOf[parser.Tag]() },
		"Server":         func() (any, []field) { return &parser.Server{}, fieldsOf[parser.Server]() },
		"ServerVariable": func() (any, []field) { return &parser.ServerVariable{}, fieldsOf[parser.ServerVariable]() },
		"Info":           func() (any, []field) { return &parser.Info{}, fieldsOf[parser.Info]() },
		"Contact":        func() (any, []field) { return &parser.Contact{}, fieldsOf[parser.Contact]() },
		"License":        func() (any, []field) { return &parser.License{}, fieldsOf[parser.License]() },
		"ExternalDocs":   func() (any, []field) { return &parser.ExternalDocs{}, fieldsOf[parser.ExternalDocs]() },
		"SecurityScheme": func() (any, []field) { return &parser.SecurityScheme{}, fieldsOf[parser.SecurityScheme]() },
		"OAuthFlows":     func() (any, []field) { return &parser.OAuthFlows{}, fieldsOf[parser.OAuthFlows]() },
		"OAuthFlow":      func() (any, []field) { return &parser.OAuthFlow{}, fieldsOf[parser.OAuthFlow]() },
	}
}

func TestMarshalSlowPathEmitsEveryField(t *testing.T) {
	runMarshalGuard(t, true,
		"%s.%s is set but the slow MarshalJSON path did not emit it; add it to that path's map builder")
}

// TestMarshalFastPathEmitsEveryField is the control: the same fields with no
// extension present go through the struct tags, which is the path that already
// worked. A failure here means the struct tag itself is wrong.
func TestMarshalFastPathEmitsEveryField(t *testing.T) {
	runMarshalGuard(t, false,
		"%s.%s is set but the struct tags did not emit it")
}

func runMarshalGuard(t *testing.T, withExtension bool, message string) {
	t.Helper()

	for typeName, subject := range marshalSubjects() {
		_, fields := subject()
		for _, f := range fields {
			t.Run(typeName+"/"+f.name, func(t *testing.T) {
				value, _ := subject()
				require.True(t, populate(value, f),
					"populate cannot produce a value for %s.%s; extend it rather than "+
						"leaving the field unchecked", typeName, f.name)
				if withExtension {
					require.True(t, setExtension(value),
						"%s has no Extra field, so the slow path cannot be reached", typeName)
				}

				encoded, err := json.Marshal(value)
				require.NoError(t, err)

				// An excluded field is asserted to stay excluded rather than skipped: a
				// skip checks nothing, so emitting one by accident would go unnoticed.
				//
				// Checked on the raw bytes because an excluded field can change the shape
				// of the output rather than merely drop a key. Discriminator.StringForm
				// re-spells the whole object as an OAS 2.0 bare string, so there is no
				// object left to decode.
				if reason, excluded := marshalExclusions[typeName][f.name]; excluded {
					assert.NotContains(t, string(encoded), `"`+f.name+`"`,
						"%s.%s is excluded from JSON (%s) but was emitted", typeName, f.name, reason)
					return
				}

				// Decoded rather than substring-matched: the key name can occur inside a
				// nested value too, and a guard that passes falsely is worse than none.
				var decoded map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(encoded, &decoded))

				if f.jsonKey == "" {
					// Tagged json:"-", so no spelling of the name should appear as a key.
					assert.NotContains(t, decoded, f.name,
						`%s.%s is tagged json:"-" but was emitted`, typeName, f.name)
					assert.NotContains(t, decoded, strings.ToLower(f.name[:1])+f.name[1:],
						`%s.%s is tagged json:"-" but was emitted`, typeName, f.name)
					return
				}

				assert.Contains(t, decoded, f.jsonKey, message, typeName, f.name)
			})
		}
	}
}

// setExtension puts an extension on the value so MarshalJSON takes the slow path,
// and reports whether it found somewhere to put it. Done by reflection rather
// than a type switch so a new type in marshalSubjects needs no second edit here.
func setExtension(value any) bool {
	extra := reflectExtraField(value)
	if !extra.IsValid() || !extra.CanSet() {
		return false
	}
	extra.Set(newExtension(extra.Type()))
	return true
}
