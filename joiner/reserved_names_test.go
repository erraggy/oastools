package joiner

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// sourceTemplate names a renamed schema after its document, so a rename in
// these tests produces a name a later document can be written to declare.
const sourceTemplate = `{{.Name}}_{{.Source}}`

// oas2With builds an OAS 2.0 document declaring the given definitions, each
// reachable from its own operation so no pass prunes it.
func oas2With(name string, definitions map[string]*parser.Schema) parser.ParseResult {
	paths := parser.Paths{}
	for definition := range definitions {
		paths["/"+name+"/"+definition] = &parser.PathItem{Get: &parser.Operation{
			OperationID: "get" + name + definition,
			Responses: &parser.Responses{Codes: map[string]*parser.Response{
				"200": {Description: "ok", Schema: &parser.Schema{Ref: "#/definitions/" + definition}},
			}},
		}}
	}
	return parser.ParseResult{
		Document: &parser.OAS2Document{
			Swagger:     "2.0",
			Info:        &parser.Info{Title: name, Version: "1.0.0"},
			Paths:       paths,
			Definitions: definitions,
			OASVersion:  parser.OASVersion20,
		},
		Version:      "2.0",
		OASVersion:   parser.OASVersion20,
		SourcePath:   name,
		SourceFormat: parser.SourceFormatJSON,
	}
}

func definitionsOf(t *testing.T, result *JoinResult) map[string]*parser.Schema {
	t.Helper()
	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok, "expected an OAS 2.0 document")
	return doc.Definitions
}

// TestRenameKeepsNameALaterDocumentDeclares covers #547: a rename picked its
// target against the documents merged so far, so a document merged afterwards
// was renamed out of a name it declared itself.
func TestRenameKeepsNameALaterDocumentDeclares(t *testing.T) {
	// a and b collide on Common, which mints Common_b for b's copy. c declares
	// Common_b itself, and is merged after the rename has already happened.
	docs := []parser.ParseResult{
		oas2With("a", map[string]*parser.Schema{"Common": object("fromA")}),
		oas2With("b", map[string]*parser.Schema{"Common": object("fromB")}),
		oas2With("c", map[string]*parser.Schema{"Common_b": object("fromC")}),
	}

	config := DefaultConfig()
	config.SchemaStrategy = StrategyDeduplicateOrRename
	config.EquivalenceMode = string(EquivalenceModeDeep)
	config.RenameTemplate = sourceTemplate

	result, err := New(config).JoinParsed(docs)
	require.NoError(t, err)
	definitions := definitionsOf(t, result)

	// c declared Common_b, so c's schema is what sits under it.
	require.Contains(t, definitions, "Common_b")
	assert.Contains(t, definitions["Common_b"].Properties, "fromC",
		"the declared name holds the schema its document declared")

	// b's renamed schema goes to the next free name rather than taking c's.
	require.Contains(t, definitions, "Common_b_2")
	assert.Contains(t, definitions["Common_b_2"].Properties, "fromB")

	assert.ElementsMatch(t, []string{"Common", "Common_b", "Common_b_2"},
		slices.Sorted(maps.Keys(definitions)))
}

// TestReservedNamesKeepProvenanceExact covers the second half of #547: a name
// that is both minted and declared was recorded as minted only, so semantic
// deduplication folded away a name a document had declared.
func TestReservedNamesKeepProvenanceExact(t *testing.T) {
	// c's Common_b is equal to b's Common, so without the reservation the two
	// merge and the surviving name carries both origins. Zebra is equal to
	// both, which is what gives semantic deduplication a group to rank.
	docs := []parser.ParseResult{
		oas2With("a", map[string]*parser.Schema{"Common": object("fromA")}),
		oas2With("b", map[string]*parser.Schema{"Common": object("shared")}),
		oas2With("c", map[string]*parser.Schema{
			"Common_b": object("shared"),
			"Zebra":    object("shared"),
		}),
	}

	config := DefaultConfig()
	config.SchemaStrategy = StrategyDeduplicateOrRename
	config.EquivalenceMode = string(EquivalenceModeDeep)
	config.RenameTemplate = sourceTemplate
	config.SemanticDeduplication = true

	result, err := New(config).JoinParsed(docs)
	require.NoError(t, err)

	assert.False(t, result.generated["Common_b"],
		"a name a document declares is not recorded as minted")

	definitions := definitionsOf(t, result)
	assert.Contains(t, definitions, "Common_b",
		"semantic deduplication keeps the declared name")
}

// TestRenameKeepsNameALaterDocumentDeclaresOAS3 is the OAS 3.x dialect of
// TestRenameKeepsNameALaterDocumentDeclares, which reaches a different merger.
func TestRenameKeepsNameALaterDocumentDeclaresOAS3(t *testing.T) {
	docs := []parser.ParseResult{
		parsedAs(flatDoc("/a", map[string]*parser.Schema{"Common": object("fromA")}), "a"),
		parsedAs(flatDoc("/b", map[string]*parser.Schema{"Common": object("fromB")}), "b"),
		parsedAs(flatDoc("/c", map[string]*parser.Schema{"Common_b": object("fromC")}), "c"),
	}

	config := DefaultConfig()
	config.SchemaStrategy = StrategyDeduplicateOrRename
	config.EquivalenceMode = string(EquivalenceModeDeep)
	config.RenameTemplate = sourceTemplate

	result, err := New(config).JoinParsed(docs)
	require.NoError(t, err)
	schemas := joinedSchemas(t, result)

	require.Contains(t, schemas, "Common_b")
	assert.Contains(t, schemas["Common_b"].Properties, "fromC")
	require.Contains(t, schemas, "Common_b_2")
	assert.Contains(t, schemas["Common_b_2"].Properties, "fromB")
}

// TestReservedNamesUseTheMergedSpelling covers the prefix case: a prefixed
// document is stored under the prefixed name, so that is the name a rename
// must avoid, and the name it declared is free for a rename to take.
func TestReservedNamesUseTheMergedSpelling(t *testing.T) {
	docs := []parser.ParseResult{
		oas2With("a", map[string]*parser.Schema{"Common": object("fromA")}),
		oas2With("b", map[string]*parser.Schema{"Common": object("fromB")}),
		oas2With("c", map[string]*parser.Schema{"Common_b": object("fromC")}),
	}

	config := DefaultConfig()
	config.SchemaStrategy = StrategyDeduplicateOrRename
	config.EquivalenceMode = string(EquivalenceModeDeep)
	config.RenameTemplate = sourceTemplate
	config.AlwaysApplyPrefix = true
	config.NamespacePrefix = map[string]string{"c": "pfx"}

	result, err := New(config).JoinParsed(docs)
	require.NoError(t, err)
	definitions := definitionsOf(t, result)

	// c's declaration is merged as pfx_Common_b, so Common_b is nobody's
	// declared name and b's rename is free to use it.
	require.Contains(t, definitions, "pfx_Common_b")
	assert.Contains(t, definitions["pfx_Common_b"].Properties, "fromC")
	require.Contains(t, definitions, "Common_b")
	assert.Contains(t, definitions["Common_b"].Properties, "fromB",
		"a name no document is merged under stays available to a rename")
}
