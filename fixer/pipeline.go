package fixer

import "github.com/erraggy/oastools/parser"

// fixPipeline defines version-specific callbacks for the fix pipeline.
type fixPipeline struct {
	fixMissingPathParams func(*Fixer, any, *FixResult)
	fixInvalidSchemas    func(*Fixer, any, *FixResult)
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

	// 3. Prune unused schemas
	if f.isFixEnabled(FixTypePrunedUnusedSchema) {
		pipeline.pruneUnusedSchemas(f, doc, result)
	}

	// 4. Prune empty paths
	if f.isFixEnabled(FixTypePrunedEmptyPath) {
		f.pruneEmptyPaths(pipeline.getPaths(doc), result, pipeline.getVersion(doc))
	}

	// Update result
	result.Document = doc
	result.FixCount = len(result.Fixes)
}

// oas2Pipeline is the fix pipeline for OAS 2.0 documents.
var oas2Pipeline = fixPipeline{
	fixMissingPathParams: func(f *Fixer, doc any, result *FixResult) {
		f.fixMissingPathParametersOAS2(doc.(*parser.OAS2Document), result)
	},
	fixInvalidSchemas: func(f *Fixer, doc any, result *FixResult) {
		f.fixInvalidSchemaNamesOAS2(doc.(*parser.OAS2Document), result)
	},
	pruneUnusedSchemas: func(f *Fixer, doc any, result *FixResult) {
		f.pruneUnusedSchemasOAS2(doc.(*parser.OAS2Document), result)
	},
	getPaths: func(doc any) parser.Paths {
		return doc.(*parser.OAS2Document).Paths
	},
	getVersion: func(doc any) parser.OASVersion {
		return parser.OASVersion20
	},
}

// oas3Pipeline is the fix pipeline for OAS 3.x documents.
var oas3Pipeline = fixPipeline{
	fixMissingPathParams: func(f *Fixer, doc any, result *FixResult) {
		f.fixMissingPathParametersOAS3(doc.(*parser.OAS3Document), result)
	},
	fixInvalidSchemas: func(f *Fixer, doc any, result *FixResult) {
		f.fixInvalidSchemaNamesOAS3(doc.(*parser.OAS3Document), result)
	},
	pruneUnusedSchemas: func(f *Fixer, doc any, result *FixResult) {
		f.pruneUnusedSchemasOAS3(doc.(*parser.OAS3Document), result)
	},
	getPaths: func(doc any) parser.Paths {
		return doc.(*parser.OAS3Document).Paths
	},
	getVersion: func(doc any) parser.OASVersion {
		return doc.(*parser.OAS3Document).OASVersion
	},
}
