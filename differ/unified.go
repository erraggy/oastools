package differ

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// diffOAS2Unified compares two OAS 2.0 documents
func (d *Differ) diffOAS2Unified(source, target *parser.OAS2Document, result *DiffResult) {
	basePath := "document"

	// Compare Info
	d.diffInfoUnified(source.Info, target.Info, basePath+".info", result)

	// Compare Host, BasePath, Schemes
	if source.Host != target.Host {
		d.addChange(result, basePath+".host", ChangeTypeModified, CategoryServer,
			SeverityWarning, source.Host, target.Host, fmt.Sprintf("host changed from %q to %q", source.Host, target.Host))
	}

	if source.BasePath != target.BasePath {
		d.addChange(result, basePath+".basePath", ChangeTypeModified, CategoryServer,
			SeverityWarning, source.BasePath, target.BasePath, fmt.Sprintf("basePath changed from %q to %q", source.BasePath, target.BasePath))
	}

	d.diffStringSlicesUnified(source.Schemes, target.Schemes, basePath+".schemes", CategoryServer, "scheme", result)

	// Compare Consumes/Produces
	d.diffStringSlicesUnified(source.Consumes, target.Consumes, basePath+".consumes", CategoryOperation, "consumes media type", result)
	d.diffStringSlicesUnified(source.Produces, target.Produces, basePath+".produces", CategoryOperation, "produces media type", result)

	// Compare Paths
	d.diffPathsUnified(source.Paths, target.Paths, basePath+".paths", result)

	// Compare Definitions
	d.diffSchemasUnified(source.Definitions, target.Definitions, basePath+".definitions", result)

	// Compare Security Definitions
	d.diffSecuritySchemesUnified(source.SecurityDefinitions, target.SecurityDefinitions, basePath+".securityDefinitions", result)

	// Compare Tags
	d.diffTagsUnified(source.Tags, target.Tags, basePath+".tags", result)

	// Compare Extensions
	d.diffExtrasUnified(source.Extra, target.Extra, basePath, result)
}

// diffOAS3Unified compares two OAS 3.x documents
func (d *Differ) diffOAS3Unified(source, target *parser.OAS3Document, result *DiffResult) {
	basePath := "document"

	// Compare Info
	d.diffInfoUnified(source.Info, target.Info, basePath+".info", result)

	// Compare Servers
	d.diffServersUnified(source.Servers, target.Servers, basePath+".servers", result)

	// Compare Paths
	d.diffPathsUnified(source.Paths, target.Paths, basePath+".paths", result)

	// Compare Webhooks (OAS 3.1+)
	if source.Webhooks != nil || target.Webhooks != nil {
		d.diffWebhooksUnified(source.Webhooks, target.Webhooks, basePath+".webhooks", result)
	}

	// Compare Components
	if source.Components != nil || target.Components != nil {
		d.diffComponentsUnified(source.Components, target.Components, basePath+".components", result)
	}

	// Compare Tags
	d.diffTagsUnified(source.Tags, target.Tags, basePath+".tags", result)

	// Compare Extensions
	d.diffExtrasUnified(source.Extra, target.Extra, basePath, result)
}

// diffCrossVersionUnified compares documents of different OAS versions
func (d *Differ) diffCrossVersionUnified(source, target parser.ParseResult, result *DiffResult) {
	// Report version change
	d.addChange(result, "document", ChangeTypeModified, CategoryInfo,
		SeverityWarning, source.Version, target.Version, fmt.Sprintf("OAS version changed from %s to %s (cross-version diff has limitations)", source.Version, target.Version))

	// Compare what we can (Info and Paths exist in both)
	var sourceInfo, targetInfo *parser.Info
	var sourcePaths, targetPaths parser.Paths

	if doc, ok := source.OAS2Document(); ok {
		sourceInfo = doc.Info
		sourcePaths = doc.Paths
	} else if doc, ok := source.OAS3Document(); ok {
		sourceInfo = doc.Info
		sourcePaths = doc.Paths
	}

	if doc, ok := target.OAS2Document(); ok {
		targetInfo = doc.Info
		targetPaths = doc.Paths
	} else if doc, ok := target.OAS3Document(); ok {
		targetInfo = doc.Info
		targetPaths = doc.Paths
	}

	d.diffInfoUnified(sourceInfo, targetInfo, "document.info", result)
	d.diffPathsUnified(sourcePaths, targetPaths, "document.paths", result)
}

// diffUnified performs the unified diff that handles both ModeSimple and ModeBreaking
func (d *Differ) diffUnified(source, target parser.ParseResult, result *DiffResult) {
	// Compare based on OAS version
	sourceOAS2, sourceIsOAS2 := source.OAS2Document()
	targetOAS2, targetIsOAS2 := target.OAS2Document()
	sourceOAS3, sourceIsOAS3 := source.OAS3Document()
	targetOAS3, targetIsOAS3 := target.OAS3Document()

	switch {
	case sourceIsOAS2 && targetIsOAS2:
		d.diffOAS2Unified(sourceOAS2, targetOAS2, result)
	case sourceIsOAS3 && targetIsOAS3:
		d.diffOAS3Unified(sourceOAS3, targetOAS3, result)
	default:
		// Cross-version comparison
		d.diffCrossVersionUnified(source, target, result)
	}
}
