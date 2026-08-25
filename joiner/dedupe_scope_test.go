package joiner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// scopeDocs builds the shape #543 is written around: a and b declare the same
// name with divergent shapes, so a collision rename generates one, and a third
// document declares two equivalent names of its own that nothing holds apart.
func scopeDocs() []parser.ParseResult {
	return []parser.ParseResult{
		oas2With("a", map[string]*parser.Schema{"Common": object("fromA")}),
		oas2With("b", map[string]*parser.Schema{"Common": object("fromB")}),
		oas2With("c", map[string]*parser.Schema{
			"Inventory": object("sku"),
			"Stock":     object("sku"),
		}),
	}
}

func scopeConfig(scope DeduplicationScope) JoinerConfig {
	config := DefaultConfig()
	config.SchemaStrategy = StrategyDeduplicateOrRename
	config.EquivalenceMode = string(EquivalenceModeDeep)
	config.RenameTemplate = sourceTemplate
	config.SemanticDeduplication = true
	config.DeduplicationScope = scope
	return config
}

func joinWithScope(t *testing.T, scope DeduplicationScope, docs []parser.ParseResult) *JoinResult {
	t.Helper()
	result, err := New(scopeConfig(scope)).JoinParsed(docs)
	require.NoError(t, err)
	return result
}

// TestDeduplicationScopeDefaultUnchanged covers R5: the zero value and an
// explicit "all" both keep the behavior that predates the scope.
func TestDeduplicationScopeDefaultUnchanged(t *testing.T) {
	for _, scope := range []DeduplicationScope{"", DeduplicationScopeAll} {
		t.Run("scope="+string(scope), func(t *testing.T) {
			definitions := definitionsOf(t, joinWithScope(t, scope, scopeDocs()))
			assert.Contains(t, definitions, "Inventory")
			assert.NotContains(t, definitions, "Stock",
				"two equivalent declared names still consolidate under the default scope")
		})
	}
}

// TestDeduplicationScopeGeneratedOnlyKeepsDeclaredNames covers R1: a declared
// name is left alone even when an equivalent declared name survives beside it.
func TestDeduplicationScopeGeneratedOnlyKeepsDeclaredNames(t *testing.T) {
	definitions := definitionsOf(t, joinWithScope(t, DeduplicationScopeGeneratedOnly, scopeDocs()))

	assert.Contains(t, definitions, "Inventory")
	assert.Contains(t, definitions, "Stock",
		"a name a document declared is not folded away under generated-only")
}

// TestDeduplicationScopeGeneratedOnlyStillFoldsGenerated covers the half of R1
// that has to keep working: a generated alias folds into the declared name that
// outranks it, which is the whole reason to enable the pass.
func TestDeduplicationScopeGeneratedOnlyStillFoldsGenerated(t *testing.T) {
	// b's Common collides with a's and is renamed to Common_b, which is
	// equivalent to the Zebra that c declares.
	docs := []parser.ParseResult{
		oas2With("a", map[string]*parser.Schema{"Common": object("fromA")}),
		oas2With("b", map[string]*parser.Schema{"Common": object("shared")}),
		oas2With("c", map[string]*parser.Schema{"Zebra": object("shared")}),
	}

	definitions := definitionsOf(t, joinWithScope(t, DeduplicationScopeGeneratedOnly, docs))

	assert.Contains(t, definitions, "Zebra", "the declared name survives")
	assert.NotContains(t, definitions, "Common_b", "the generated alias still folds into it")
}

// TestDeduplicationScopeKeepsDeclaredSurvivor covers R2 under both scopes, with
// a template that puts the generated name first alphabetically.
func TestDeduplicationScopeKeepsDeclaredSurvivor(t *testing.T) {
	docs := []parser.ParseResult{
		oas2With("a", map[string]*parser.Schema{"Common": object("fromA")}),
		oas2With("b", map[string]*parser.Schema{"Common": object("shared")}),
		oas2With("c", map[string]*parser.Schema{"Zebra": object("shared")}),
	}

	for _, scope := range []DeduplicationScope{DeduplicationScopeAll, DeduplicationScopeGeneratedOnly} {
		t.Run("scope="+string(scope), func(t *testing.T) {
			config := scopeConfig(scope)
			config.RenameTemplate = aliasFirstTemplate // Api_{{.Name}} sorts before Zebra
			result, err := New(config).JoinParsed(docs)
			require.NoError(t, err)

			definitions := definitionsOf(t, result)
			assert.Contains(t, definitions, "Zebra",
				"the declared name survives even though the generated one sorts first")
			assert.NotContains(t, definitions, "Api_Common")
		})
	}
}

// TestDeduplicationReport covers R3: each consolidation names its survivor, the
// names folded into it, and where each name came from.
func TestDeduplicationReport(t *testing.T) {
	config := scopeConfig(DeduplicationScopeAll)
	config.DeduplicationReport = true

	result, err := New(config).JoinParsed(scopeDocs())
	require.NoError(t, err)

	require.Len(t, result.Consolidations, 1)
	consolidation := result.Consolidations[0]
	assert.Equal(t, "Inventory", consolidation.Survivor)
	assert.False(t, consolidation.SurvivorGenerated)
	assert.Equal(t, []FoldedName{{Name: "Stock", Generated: false}}, consolidation.Folded,
		"a declared name folded under the default scope is reported as declared")
}

// TestDeduplicationReportRecordsGeneratedProvenance pins the other provenance
// value, which is what tells a caller the consolidation was invisible to a
// consumer of the joined document.
func TestDeduplicationReportRecordsGeneratedProvenance(t *testing.T) {
	docs := []parser.ParseResult{
		oas2With("a", map[string]*parser.Schema{"Common": object("fromA")}),
		oas2With("b", map[string]*parser.Schema{"Common": object("shared")}),
		oas2With("c", map[string]*parser.Schema{"Zebra": object("shared")}),
	}

	config := scopeConfig(DeduplicationScopeGeneratedOnly)
	config.DeduplicationReport = true

	result, err := New(config).JoinParsed(docs)
	require.NoError(t, err)

	require.Len(t, result.Consolidations, 1)
	assert.Equal(t, "Zebra", result.Consolidations[0].Survivor)
	assert.Equal(t, []FoldedName{{Name: "Common_b", Generated: true}}, result.Consolidations[0].Folded)
}

// TestDeduplicationReportOffByDefault covers the other half of R5: the report
// costs nothing until it is asked for.
func TestDeduplicationReportOffByDefault(t *testing.T) {
	result := joinWithScope(t, DeduplicationScopeAll, scopeDocs())
	assert.Empty(t, result.Consolidations)
}

// TestDeduplicationScopeRejectsUnknownValue keeps a typo from silently reading
// as the default.
func TestDeduplicationScopeRejectsUnknownValue(t *testing.T) {
	_, err := JoinWithOptions(
		WithParsed(scopeDocs()...),
		WithDeduplicationScope("generated"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generated-only", "the error lists the valid values")
}

// TestDeduplicationScopeHoldsReferencedNamesApart covers R4: the guarantee that
// names one schema tree references stay distinct does not depend on the scope.
func TestDeduplicationScopeHoldsReferencedNamesApart(t *testing.T) {
	for _, scope := range []DeduplicationScope{DeduplicationScopeAll, DeduplicationScopeGeneratedOnly} {
		t.Run("scope="+string(scope), func(t *testing.T) {
			config := scopeConfig(scope)
			result, err := New(config).JoinParsed([]parser.ParseResult{
				emptyCombined(parser.OASVersion20),
				shipmentOAS2(true),
			})
			require.NoError(t, err)

			definitions := definitionsOf(t, result)
			assert.Contains(t, definitions, "OriginAddress")
			assert.Contains(t, definitions, "DestinationAddress",
				"a shipment's origin is not its destination, under any scope")
		})
	}
}

// TestDeduplicationScopeComposesWithOperationContext covers R4 for the options
// the reporter uses together.
func TestDeduplicationScopeComposesWithOperationContext(t *testing.T) {
	config := scopeConfig(DeduplicationScopeGeneratedOnly)
	config.OperationContext = true
	config.RenameTemplate = "{{.Name}}_{{.Source}}"

	result, err := New(config).JoinParsed(scopeDocs())
	require.NoError(t, err)

	definitions := definitionsOf(t, result)
	assert.Contains(t, definitions, "Inventory")
	assert.Contains(t, definitions, "Stock")
}

// TestFoldableForScope pins the resolution of a scope, including the value a
// caller can only produce by filling JoinerConfig directly.
func TestFoldableForScope(t *testing.T) {
	generated := map[string]bool{"Api_Common": true}

	tests := []struct {
		name     string
		scope    DeduplicationScope
		foldable bool // whether the returned func is non-nil
	}{
		{"all folds every name", DeduplicationScopeAll, false},
		{"the zero value reads as all", "", false},
		{"an unrecognized value reads as all", "generated_only", false},
		{"generated-only refuses a declared name", DeduplicationScopeGeneratedOnly, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := foldableForScope(tt.scope, generated)
			if !tt.foldable {
				assert.Nil(t, got, "a scope that folds everything supplies no FoldableFunc")
				return
			}
			require.NotNil(t, got)
			assert.True(t, got("Api_Common"), "a generated name folds")
			assert.False(t, got("Common"), "a declared name does not")
		})
	}
}
