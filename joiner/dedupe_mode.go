package joiner

import (
	"fmt"
	"math"

	"github.com/erraggy/oastools/internal/schemarefs"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// DeduplicationMode selects how semantic deduplication resolves an equivalence
// group once the surviving name is chosen. It does not change which schemas
// compare equal, nor which names may be consolidated: an equivalence group is
// built the same way under every mode, and DeduplicationScope still decides
// what may be consolidated out of one.
type DeduplicationMode string

const (
	// DeduplicationModeRemove deletes every consolidated name and repoints the
	// references to it at the survivor. This is the default.
	//
	//	{pets.Label, store.Tag}   definitions: orders.Marker
	//	                          the Pets response now says orders.Marker
	DeduplicationModeRemove DeduplicationMode = "remove"

	// DeduplicationModePointer stores the shape once under the surviving name
	// and leaves every other name of the group in place, as a schema that is a
	// bare $ref to the survivor. References are not rewritten, so a reference
	// to any name in the group still resolves to the shape it always did,
	// wherever that reference sits:
	//
	//	{pets.Label, store.Tag}   definitions: pets.Label
	//	                                       store.Tag -> $ref pets.Label
	//	                          the Pets response still says pets.Label
	//
	// A bare $ref is legal as a definitions entry and as a
	// components.schemas entry, in OAS 2.0 and OAS 3.x alike. OAS 2.0 ignores
	// a $ref's siblings, so a pointer cannot carry a description of its own:
	// where two documents described the same shape differently, the survivor's
	// description is the one that remains.
	DeduplicationModePointer DeduplicationMode = "pointer"
)

// ValidDeduplicationModes returns all valid deduplication mode strings.
func ValidDeduplicationModes() []string {
	return []string{
		string(DeduplicationModeRemove),
		string(DeduplicationModePointer),
	}
}

// IsValidDeduplicationMode checks if a deduplication mode string is valid.
// The empty string is valid and means DeduplicationModeRemove, so a config that
// never set one behaves as it did before the mode existed.
func IsValidDeduplicationMode(mode string) bool {
	switch DeduplicationMode(mode) {
	case "", DeduplicationModeRemove, DeduplicationModePointer:
		return true
	default:
		return false
	}
}

// resolveDeduplicationMode returns the mode a config asks for. JoinerConfig is a
// struct a caller can fill in directly, so an unrecognized value reaches here
// and reads as the default.
func resolveDeduplicationMode(mode DeduplicationMode) DeduplicationMode {
	if !IsValidDeduplicationMode(string(mode)) {
		return DeduplicationModeRemove
	}
	if mode == "" {
		return DeduplicationModeRemove
	}
	return mode
}

// needsDeclarationOrder reports whether the join has to record which document
// declared each schema, which only DeduplicationModePointer reads. Building the
// ownership map costs a pass over every source document's components, so a join
// that will not rank by it does not pay for it.
func (j *Joiner) needsDeclarationOrder(schemaCount int) bool {
	return j.config.SemanticDeduplication && schemaCount > 1 &&
		resolveDeduplicationMode(j.config.DeduplicationMode) == DeduplicationModePointer
}

// dedupeTarget is the one document semantic deduplication runs over, gathered
// so the OAS 2.0 and OAS 3.x paths can share the pass.
type dedupeTarget struct {
	// document is the joined document, for reference rewriting and for reading
	// how the schemas reference one another.
	document any
	// schemas is the joined document's schema map: Definitions at OAS 2.0,
	// Components.Schemas at OAS 3.x.
	schemas map[string]*parser.Schema
	// version selects the $ref spelling a pointer is written with.
	version parser.OASVersion
	// owner maps a schema to the source document that first declared it, or is
	// nil when nothing needed it. See declarationOrder.
	owner map[any]int
	// copied is the copy-on-write set shared with the rename pass, so an entry
	// is copied once per join rather than once per pass.
	copied map[any]bool
	// section names the schema container in warnings: "definition" or "schema".
	section string
	// apply stores a schema map back on the document. The pass calls it before
	// rewriting references, because the rewriter is copy-on-write: it replaces
	// an entry with a deep copy in the map it is walking, so a map installed
	// afterwards would carry the originals and lose every rewrite.
	apply func(map[string]*parser.Schema)
}

// deduplicateSchemas runs semantic deduplication over one document's schemas and
// applies the result in the configured mode, storing the schema map the document
// should carry through dedupeTarget.apply.
func (j *Joiner) deduplicateSchemas(t dedupeTarget, result *JoinResult) error {
	mode := resolveDeduplicationMode(j.config.DeduplicationMode)

	compareOpts := j.buildCompareOptions(EquivalenceModeDeep)
	compare := func(left, right *parser.Schema) bool {
		return CompareSchemasWithOptions(left, right, compareOpts).Equivalent
	}

	config := schemautil.DefaultDeduplicationConfig()
	config.Outranks = j.outranksFor(mode, t, result)
	distinct, err := schemarefs.Collect(t.document)
	if err != nil {
		return fmt.Errorf("joiner: failed to record schema references before semantic deduplication: %w", err)
	}
	config.Split = distinct.Split
	config.Foldable = foldableForScope(j.config.DeduplicationScope, result.generated)

	deduper := schemautil.NewSchemaDeduplicator(config, compare)
	dedupeResult, err := deduper.Deduplicate(t.schemas)
	if err != nil {
		return fmt.Errorf("joiner: semantic deduplication failed: %w", err)
	}

	j.recordConsolidations(result, dedupeResult, mode)

	schemas := dedupeResult.CanonicalSchemas
	if mode == DeduplicationModePointer {
		// Each consolidated name stays in place, as a schema that is a bare
		// $ref to the survivor, so every name in the group still resolves.
		for alias, canonical := range dedupeResult.Aliases {
			schemas[alias] = &parser.Schema{Ref: schemaRefPath(canonical, t.version)}
		}
	}
	t.apply(schemas)

	if len(dedupeResult.Aliases) == 0 {
		return nil
	}
	if mode == DeduplicationModePointer {
		// No reference is rewritten: a reference to any name in the group still
		// resolves to the shape it always did.
		result.AddWarning(NewSemanticDedupPointerSummaryWarning(len(dedupeResult.Aliases), t.section))
		return nil
	}

	if err := rewriteDedupeAliases(t.document, dedupeResult.Aliases, t.version, t.copied); err != nil {
		return fmt.Errorf("joiner: failed to rewrite references after semantic deduplication: %w", err)
	}
	result.AddWarning(NewSemanticDedupSummaryWarning(dedupeResult.RemovedCount, t.section))
	return nil
}

// outranksFor returns the ranking that picks each equivalence group's surviving
// name.
//
// DeduplicationModeRemove keeps the collapse pass's ranking, which is what makes
// the two passes agree on which of several equivalent names survives (#498).
// DeduplicationModePointer keeps every name, so the survivor is only the name
// the shape is stored under, and ranking it by the document that declared it
// first stops that from being decided by sort order (#553).
func (j *Joiner) outranksFor(mode DeduplicationMode, t dedupeTarget, result *JoinResult) schemautil.OutranksFunc {
	if mode != DeduplicationModePointer {
		return outranksGenerated(result.generated)
	}
	return outranksDeclaration(result.generated, declarationOrder(t.schemas, t.owner))
}

// declarationOrder maps each schema name to the source document that declared
// it, read from the ownership map the rename pass builds. A name whose schema
// no document claimed is left out, which declarationIndex reads as last.
//
// It is taken after renames are applied, so a name reads under the spelling the
// joined document stores it as, and the document it maps to is the one that
// declared the schema rather than the one whose collision forced the rename.
func declarationOrder(schemas map[string]*parser.Schema, owner map[any]int) map[string]int {
	if len(owner) == 0 {
		return nil
	}
	order := make(map[string]int, len(schemas))
	for name, schema := range schemas {
		if schema == nil {
			continue
		}
		if docIndex, claimed := owner[schema]; claimed {
			order[name] = docIndex
		}
	}
	return order
}

// declarationIndex returns the source document that declared name, or MaxInt
// when none is recorded, so an unattributed name never outranks an attributed
// one.
func declarationIndex(declaredIn map[string]int, name string) int {
	if docIndex, known := declaredIn[name]; known {
		return docIndex
	}
	return math.MaxInt
}

// outranksDeclaration ranks equivalent names by provenance, then by the document
// that declared them, then alphabetically. It returns nil when it has nothing to
// go on, leaving the deduplicator's own alphabetical tiebreak in place.
func outranksDeclaration(generated map[string]bool, declaredIn map[string]int) schemautil.OutranksFunc {
	if len(generated) == 0 && len(declaredIn) == 0 {
		return nil
	}
	return func(name, candidate string) bool {
		// A name no rename generated wins, the same first question the collapse
		// pass asks, so a group mixing the two still keeps the declared name.
		if generated[name] != generated[candidate] {
			return !generated[name]
		}
		first := declarationIndex(declaredIn, name)
		against := declarationIndex(declaredIn, candidate)
		if first != against {
			return first < against
		}
		return name < candidate
	}
}
