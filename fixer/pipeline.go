package fixer

import "github.com/erraggy/oastools/parser"

// fixPipeline defines version-specific callbacks for the fix pipeline.
type fixPipeline struct {
	fixMissingPathParams func(*Fixer, any, *FixResult)
	fixInvalidSchemas    func(*Fixer, any, *FixResult)
	fixCSVEnums          func(*Fixer, any, *FixResult)
	pruneUnusedSchemas   func(*Fixer, any, *FixResult)
	getPaths             func(any) parser.Paths
	getVersion           func(any) parser.OASVersion
}

// applyFixPipeline runs the shared fix pipeline on a document.
func (f *Fixer) applyFixPipeline(doc any, result *FixResult, pipeline fixPipeline) {
	// Apply enabled fixes in order:
	// 1. Missing path parameters
	if f.isFixEnabled(FixTypeMissingPathParameter) {
		pipeline.fixMissingPathParams(f, doc, result)
	}

	// 2. Rename invalid schema names (must happen BEFORE pruning)
	if f.isFixEnabled(FixTypeRenamedGenericSchema) {
		pipeline.fixInvalidSchemas(f, doc, result)
	}

	// 3. Expand CSV enums
	if f.isFixEnabled(FixTypeEnumCSVExpanded) {
		pipeline.fixCSVEnums(f, doc, result)
	}

	// 4. Prune unused schemas
	if f.isFixEnabled(FixTypePrunedUnusedSchema) {
		pipeline.pruneUnusedSchemas(f, doc, result)
	}

	// 5. Prune empty paths
	if f.isFixEnabled(FixTypePrunedEmptyPath) {
		f.pruneEmptyPaths(pipeline.getPaths(doc), result, pipeline.getVersion(doc))
	}

	// Update result
	result.Document = doc
	result.FixCount = len(result.Fixes)
}

// mustOAS2 asserts that doc is an OAS 2.0 document, panicking with a clear message if not.
func mustOAS2(doc any) *parser.OAS2Document {
	if d, ok := doc.(*parser.OAS2Document); ok {
		return d
	}
	panic("fixer: expected *parser.OAS2Document, got different type (wrong pipeline?)")
}

// oas2Pipeline is the fix pipeline for OAS 2.0 documents.
var oas2Pipeline = fixPipeline{
	fixMissingPathParams: func(f *Fixer, doc any, result *FixResult) {
		f.fixMissingPathParametersOAS2(mustOAS2(doc), result)
	},
	fixInvalidSchemas: func(f *Fixer, doc any, result *FixResult) {
		f.fixInvalidSchemaNamesOAS2(mustOAS2(doc), result)
	},
	fixCSVEnums: func(f *Fixer, doc any, result *FixResult) {
		f.fixCSVEnumsOAS2(mustOAS2(doc), result)
	},
	pruneUnusedSchemas: func(f *Fixer, doc any, result *FixResult) {
		f.pruneUnusedSchemasOAS2(mustOAS2(doc), result)
	},
	getPaths: func(doc any) parser.Paths {
		return mustOAS2(doc).Paths
	},
	getVersion: func(doc any) parser.OASVersion {
		return parser.OASVersion20
	},
}

// mustOAS3 asserts that doc is an OAS 3.x document, panicking with a clear message if not.
func mustOAS3(doc any) *parser.OAS3Document {
	if d, ok := doc.(*parser.OAS3Document); ok {
		return d
	}
	panic("fixer: expected *parser.OAS3Document, got different type (wrong pipeline?)")
}

// oas3Pipeline is the fix pipeline for OAS 3.x documents.
var oas3Pipeline = fixPipeline{
	fixMissingPathParams: func(f *Fixer, doc any, result *FixResult) {
		f.fixMissingPathParametersOAS3(mustOAS3(doc), result)
	},
	fixInvalidSchemas: func(f *Fixer, doc any, result *FixResult) {
		f.fixInvalidSchemaNamesOAS3(mustOAS3(doc), result)
	},
	fixCSVEnums: func(f *Fixer, doc any, result *FixResult) {
		f.fixCSVEnumsOAS3(mustOAS3(doc), result)
	},
	pruneUnusedSchemas: func(f *Fixer, doc any, result *FixResult) {
		f.pruneUnusedSchemasOAS3(mustOAS3(doc), result)
	},
	getPaths: func(doc any) parser.Paths {
		return mustOAS3(doc).Paths
	},
	getVersion: func(doc any) parser.OASVersion {
		return mustOAS3(doc).OASVersion
	},
}
