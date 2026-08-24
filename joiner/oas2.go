package joiner

import (
	"fmt"
	"math"
	"slices"

	"github.com/erraggy/oastools/internal/schemarefs"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// joinOAS2Documents joins multiple OAS 2.0 (Swagger) documents
func (j *Joiner) joinOAS2Documents(docs []parser.ParseResult) (*JoinResult, error) {
	// Start with a copy of the first document
	baseDoc, ok := docs[0].OAS2Document()
	if !ok || baseDoc == nil {
		return nil, fmt.Errorf("joiner: first document is not a valid OAS 2.0 document")
	}

	// The rewriting after the merge needs to know which document contributed what.
	sources := make([]*parser.OAS2Document, len(docs))
	for i, doc := range docs {
		oas2Doc, ok := doc.OAS2Document()
		if !ok || oas2Doc == nil {
			return nil, fmt.Errorf("joiner: document at index %d (path: %s) is not a valid OAS 2.0 document", i, doc.SourcePath)
		}
		// If already present, store a copy: the two positions would otherwise share
		// every definition, and keeping both sides would store one schema under two
		// names (#481).
		if slices.Contains(sources[:i], oas2Doc) {
			oas2Doc = oas2Doc.DeepCopy()
		}
		sources[i] = oas2Doc
	}

	result := &JoinResult{
		Version:       docs[0].Version,
		OASVersion:    docs[0].OASVersion,
		SourceFormat:  docs[0].SourceFormat,
		Warnings:      make([]string, 0),
		firstFilePath: docs[0].SourcePath,
		scope:         newRenameScope(len(docs), docs[0].OASVersion),
	}

	// Initialize collision report if enabled
	if j.config.CollisionReport {
		result.CollisionDetails = NewCollisionReport()
	}

	// Create the joined document starting with the base
	joined := &parser.OAS2Document{
		Swagger:             baseDoc.Swagger,
		Info:                copyInfo(baseDoc.Info),
		Host:                baseDoc.Host,
		BasePath:            baseDoc.BasePath,
		Schemes:             copyStringSlice(baseDoc.Schemes),
		Consumes:            copyStringSlice(baseDoc.Consumes),
		Produces:            copyStringSlice(baseDoc.Produces),
		Paths:               make(parser.Paths),
		Definitions:         make(map[string]*parser.Schema),
		Parameters:          make(map[string]*parser.Parameter),
		Responses:           make(map[string]*parser.Response),
		SecurityDefinitions: make(map[string]*parser.SecurityScheme),
		Security:            copySecurityRequirements(baseDoc.Security),
		Tags:                copyTags(baseDoc.Tags),
		ExternalDocs:        copyExternalDocs(baseDoc.ExternalDocs),
		OASVersion:          baseDoc.OASVersion,
	}

	graphs := newRefGraphs(j.config.OperationContext, func(docIndex int) *RefGraph {
		return buildRefGraphOAS2(sources[docIndex])
	})

	// Merge all documents
	for i, doc := range docs {
		ctx := documentContext{
			filePath: doc.SourcePath,
			docIndex: i,
			result:   &doc,
		}

		if err := j.mergeOAS2Document(joined, sources[i], ctx, result, graphs); err != nil {
			return nil, err
		}
	}

	result.Document = joined

	// Taken before the collapse, which redirects a rename onto the name it kept:
	// a set taken afterwards would report that name as generated (#498).
	result.generated = result.scope.generatedNames()

	// The renames are all known and none is applied yet, which is the one point
	// at which a definition headed for a collapse can be dropped rather than
	// rewritten and copied (#487).
	var owner map[any]int
	if !result.scope.empty() {
		owner = ownersOAS2(sources)
		j.collapseDeferredRenames(joined.Definitions, owner, result)
	}

	// Before deduplication, so comparison sees references in their final form.
	copied, err := result.scope.applyOAS2(joined, owner)
	if err != nil {
		return nil, fmt.Errorf("joiner: failed to rewrite references after definition renames: %w", err)
	}

	// Apply semantic deduplication if enabled
	if j.config.SemanticDeduplication && len(joined.Definitions) > 1 {
		compareOpts := j.buildCompareOptions(EquivalenceModeDeep)
		compare := func(left, right *parser.Schema) bool {
			res := CompareSchemasWithOptions(left, right, compareOpts)
			return res.Equivalent
		}
		config := schemautil.DefaultDeduplicationConfig()
		config.Outranks = outranksGenerated(result.generated)
		distinct, err := schemarefs.Collect(joined)
		if err != nil {
			return nil, fmt.Errorf("joiner: failed to record schema references before semantic deduplication: %w", err)
		}
		config.Split = distinct.Split
		deduper := schemautil.NewSchemaDeduplicator(config, compare)
		dedupeResult, err := deduper.Deduplicate(joined.Definitions)
		if err != nil {
			return nil, fmt.Errorf("joiner: semantic deduplication failed: %w", err)
		}

		// Apply results: replace definitions map with canonical schemas only
		joined.Definitions = dedupeResult.CanonicalSchemas

		if len(dedupeResult.Aliases) > 0 {
			if err := rewriteDedupeAliases(joined, dedupeResult.Aliases, joined.OASVersion, copied); err != nil {
				return nil, fmt.Errorf("joiner: failed to rewrite references after semantic deduplication: %w", err)
			}
			result.AddWarning(NewSemanticDedupSummaryWarning(dedupeResult.RemovedCount, "definition"))
		}
	}

	result.Stats = parser.GetDocumentStats(joined)

	return result, nil
}

// mergeOAS2Document merges a single OAS2 document into the joined document
func (j *Joiner) mergeOAS2Document(joined *parser.OAS2Document, oas2Doc *parser.OAS2Document, ctx documentContext, result *JoinResult, graphs *refGraphs) error {
	// Merge paths
	if err := j.mergeOAS2Paths(joined, oas2Doc, ctx, result); err != nil {
		return err
	}

	// Merge definitions (schemas)
	if err := j.mergeOAS2Definitions(joined, oas2Doc, ctx, result, graphs); err != nil {
		return err
	}

	// Merge components (parameters, responses, security definitions)
	if err := j.mergeOAS2Components(joined, oas2Doc, ctx, result); err != nil {
		return err
	}

	// Merge arrays and metadata
	j.mergeOAS2Arrays(joined, oas2Doc, ctx, result)

	return nil
}

// mergeOAS2Paths merges paths from source document
func (j *Joiner) mergeOAS2Paths(joined, source *parser.OAS2Document, ctx documentContext, result *JoinResult) error {
	pathStrategy := j.getEffectiveStrategy(j.config.PathStrategy)
	return j.mergePathsMap(joined.Paths, source.Paths, pathStrategy, ctx, result)
}

// mergeOAS2Definitions merges definitions (schemas) from source document
func (j *Joiner) mergeOAS2Definitions(joined, source *parser.OAS2Document, ctx documentContext, result *JoinResult, graphs *refGraphs) error {
	schemaStrategy := j.getEffectiveStrategy(j.config.SchemaStrategy)
	sourceGraph := graphs.forDoc(ctx.docIndex)

	// Get namespace prefix for this source (if configured)
	sourcePrefix := j.getNamespacePrefix(ctx.filePath)

	for name, schema := range source.Definitions {
		// Determine the effective name for this definition
		effectiveName := name

		// If AlwaysApplyPrefix is true and source has a prefix, apply it to all definitions
		if j.config.AlwaysApplyPrefix && sourcePrefix != "" {
			effectiveName = j.generatePrefixedSchemaName(name, sourcePrefix)

			// Register rename for reference rewriting (original name -> prefixed name)
			result.scope.registerRight(ctx.docIndex, name, effectiveName)

			line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", name))
			result.AddWarning(NewNamespacePrefixWarning(name, effectiveName, "definition", ctx.filePath, line, col))
		}

		if _, exists := joined.Definitions[effectiveName]; exists {
			// Handle collision based on strategy
			result.CollisionCount++

			// The left side is whichever document contributed the definition now under
			// this name, which is the first only until something replaces it.
			leftSource := result.originOf(sectionDefinitions, effectiveName).filePath

			// Invoke collision handler if configured for schemas
			if j.shouldInvokeHandler(CollisionTypeSchema) {
				collision := CollisionContext{
					Type:               CollisionTypeSchema,
					Name:               effectiveName,
					JSONPath:           fmt.Sprintf("$.definitions.%s", effectiveName),
					LeftSource:         leftSource,
					LeftLocation:       j.getLocationPtr(leftSource, fmt.Sprintf("$.definitions.%s", effectiveName)),
					LeftValue:          joined.Definitions[effectiveName],
					RightSource:        ctx.filePath,
					RightLocation:      j.getLocationPtr(ctx.filePath, fmt.Sprintf("$.definitions.%s", name)),
					RightValue:         schema,
					RenameInfo:         buildRenameContextPtr(effectiveName, ctx.filePath, ctx.docIndex, sourceGraph, j.config.PrimaryOperationPolicy),
					ConfiguredStrategy: schemaStrategy,
				}

				resolution, handlerErr := j.collisionHandler(collision)
				if handlerErr != nil {
					// Log warning and fall back to configured strategy
					line, col := j.getLocation(ctx.filePath, collision.JSONPath)
					result.AddWarning(NewHandlerErrorWarning(
						collision.JSONPath,
						fmt.Sprintf("collision handler error: %v; using %s strategy", handlerErr, schemaStrategy),
						ctx.filePath, line, col,
					))
					// Fall through to strategy switch below
				} else {
					// Apply the resolution
					applied, err := j.applySchemaResolution(schemaResolutionParams{
						collision:   collision,
						resolution:  resolution,
						target:      joined.Definitions,
						result:      result,
						ctx:         ctx,
						sourceName:  name,
						section:     sectionDefinitions,
						sourceGraph: sourceGraph,
						label:       "definition",
					})
					if err != nil {
						return err
					}
					if applied {
						continue // Resolution handled, skip strategy switch
					}
					// ResolutionContinue falls through to strategy switch
				}
			}

			switch schemaStrategy {
			case StrategyDeduplicateEquivalent:
				// Use semantic equivalence to determine if schemas are identical
				mode := EquivalenceModeNone
				switch EquivalenceMode(j.config.EquivalenceMode) {
				case EquivalenceModeShallow:
					mode = EquivalenceModeShallow
				case EquivalenceModeDeep:
					mode = EquivalenceModeDeep
				}

				if mode != EquivalenceModeNone {
					eqResult := CompareSchemasWithOptions(joined.Definitions[effectiveName], schema, j.buildCompareOptions(mode))
					if eqResult.Equivalent {
						// Schemas are equivalent, keep existing and skip
						line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", name))
						result.AddWarning(NewSchemaDedupWarning(effectiveName, "definition", ctx.filePath, line, col))
						j.recordCollisionEvent(result, effectiveName, leftSource, ctx.filePath, schemaStrategy, resolutionDeduplicated, "")
						continue
					}
					// Not equivalent, fall back to fail
					return fmt.Errorf("definition '%s' collision: not equivalent, deduplicate strategy requires identical schemas (found %d differences)", effectiveName, len(eqResult.Differences))
				}
				return fmt.Errorf("definition '%s' collision: deduplicate strategy requires equivalence mode to be 'shallow' or 'deep'", effectiveName)

			case StrategyRenameLeft:
				// Rename the existing (left) definition and keep the new (right) definition under original name
				// Name it after the contributing document, not always the first (#479).
				leftOrigin := result.originOf(sectionDefinitions, effectiveName)
				leftPrefix := j.getNamespacePrefix(leftOrigin.filePath)
				var newName string
				if leftPrefix != "" {
					newName = j.generatePrefixedSchemaName(effectiveName, leftPrefix)
				} else {
					newName = j.generateRenamedSchemaName(effectiveName, leftOrigin.filePath, leftOrigin.docIndex, graphs.forDoc(leftOrigin.docIndex))
				}
				newName = uniqueSchemaName(joined.Definitions, newName)

				// Move existing definition to new name
				joined.Definitions[newName] = joined.Definitions[effectiveName]
				result.moveOrigin(sectionDefinitions, effectiveName, newName)

				// Add new definition under original name
				joined.Definitions[effectiveName] = schema
				result.recordOrigin(sectionDefinitions, effectiveName, ctx)

				// Only documents merged before this one referenced the moved definition.
				result.scope.registerLeft(ctx.docIndex, effectiveName, newName)

				line, col := j.getLocation(leftOrigin.filePath, fmt.Sprintf("$.definitions.%s", effectiveName))
				result.AddWarning(NewSchemaRenamedWarning(effectiveName, newName, "definition", leftOrigin.filePath, line, col, true))
				j.recordCollisionEvent(result, effectiveName, leftOrigin.filePath, ctx.filePath, schemaStrategy, resolutionRenamed, newName)

			case StrategyDeduplicateOrRename:
				// Rename now, decide later. Whether these two definitions are
				// interchangeable depends on where their references resolve, and
				// documents still to be merged can move those targets, so the
				// verdict cannot be reached here (#487).
				newName := uniqueSchemaName(joined.Definitions,
					j.renamedRightName(name, effectiveName, sourcePrefix, ctx, sourceGraph))

				joined.Definitions[newName] = schema
				result.recordOrigin(sectionDefinitions, newName, ctx)
				result.scope.registerRight(ctx.docIndex, name, newName)

				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", name))
				// No warning and no collision event here. Either would be
				// wrong for every rename collapseDeferredRenames withdraws, so
				// both wait until it has decided.
				result.deferred = append(result.deferred, deferredRename{
					name:        effectiveName,
					newName:     newName,
					label:       "definition",
					leftSource:  leftSource,
					rightSource: ctx.filePath,
					line:        line,
					column:      col,
				})

			case StrategyRenameRight:
				// Rename the new (right) definition and keep existing (left) definition under original name
				newName := uniqueSchemaName(joined.Definitions,
					j.renamedRightName(name, effectiveName, sourcePrefix, ctx, sourceGraph))

				// Add new definition under renamed name
				joined.Definitions[newName] = schema
				result.recordOrigin(sectionDefinitions, newName, ctx)

				// Keep existing definition under original name (no change needed)

				// Only this document referenced the renamed definition.
				result.scope.registerRight(ctx.docIndex, name, newName)

				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", name))
				result.AddWarning(NewSchemaRenamedWarning(effectiveName, newName, "definition", ctx.filePath, line, col, false))
				j.recordCollisionEvent(result, effectiveName, leftSource, ctx.filePath, schemaStrategy, resolutionRenamed, newName)

			default:
				// Handle existing strategies
				if err := j.handleCollision(effectiveName, "definitions", schemaStrategy, leftSource, ctx.filePath); err != nil {
					return err
				}
				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", name))
				if j.shouldOverwrite(schemaStrategy) {
					joined.Definitions[effectiveName] = schema
					result.recordOrigin(sectionDefinitions, effectiveName, ctx)
					result.AddWarning(NewSchemaCollisionWarning(effectiveName, "overwritten", "definitions", leftSource, ctx.filePath, line, col))
					j.recordCollisionEvent(result, effectiveName, leftSource, ctx.filePath, schemaStrategy, resolutionKeptRight, "")
				} else {
					result.AddWarning(NewSchemaCollisionWarning(effectiveName, "kept existing value", "definitions", leftSource, ctx.filePath, line, col))
					j.recordCollisionEvent(result, effectiveName, leftSource, ctx.filePath, schemaStrategy, resolutionKeptLeft, "")
				}
			}
		} else {
			joined.Definitions[effectiveName] = schema
			result.recordOrigin(sectionDefinitions, effectiveName, ctx)
		}
	}
	return nil
}

// mergeOAS2Components merges parameters, responses, and security definitions
func (j *Joiner) mergeOAS2Components(joined, source *parser.OAS2Document, ctx documentContext, result *JoinResult) error {
	componentStrategy := j.getEffectiveStrategy(j.config.ComponentStrategy)

	if err := mergeMap(j, joined.Parameters, source.Parameters, "parameters", CollisionTypeParameter, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := mergeMap(j, joined.Responses, source.Responses, "responses", CollisionTypeResponse, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := mergeMap(j, joined.SecurityDefinitions, source.SecurityDefinitions, "securityDefinitions", CollisionTypeSecurityScheme, componentStrategy, ctx, result); err != nil {
		return err
	}
	return nil
}

// mergeOAS2Arrays merges array fields and handles metadata
func (j *Joiner) mergeOAS2Arrays(joined, source *parser.OAS2Document, ctx documentContext, result *JoinResult) {
	if j.config.MergeArrays && ctx.docIndex > 0 {
		joined.Schemes = j.mergeUniqueStrings(joined.Schemes, source.Schemes)
		joined.Consumes = j.mergeUniqueStrings(joined.Consumes, source.Consumes)
		joined.Produces = j.mergeUniqueStrings(joined.Produces, source.Produces)
		joined.Security = append(joined.Security, copySecurityRequirements(source.Security)...)
	}

	if ctx.docIndex > 0 {
		joined.Tags = j.mergeTags(joined.Tags, source.Tags)

		if source.Host != "" && source.Host != joined.Host {
			result.AddWarning(NewMetadataOverrideWarning("host", joined.Host, source.Host, ctx.filePath))
		}
		if source.BasePath != "" && source.BasePath != joined.BasePath {
			result.AddWarning(NewMetadataOverrideWarning("basePath", joined.BasePath, source.BasePath, ctx.filePath))
		}

		// Info object is always taken from the first document
		// Additional info sections from subsequent documents are ignored
	}
}

// mergeUniqueStrings merges two string slices, removing duplicates
func (j *Joiner) mergeUniqueStrings(a, b []string) []string {
	seen := make(map[string]bool)
	// Guard against overflow when computing capacity (CWE-190)
	// Use uint64 to safely compute the sum, then check if it fits in int for the current platform
	capacity := 0
	sum := uint64(len(a)) + uint64(len(b))
	if sum <= uint64(math.MaxInt) {
		capacity = int(sum)
	}
	result := make([]string, 0, capacity)

	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}
