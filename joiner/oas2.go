package joiner

import (
	"fmt"
	"math"

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

	result := &JoinResult{
		Version:       docs[0].Version,
		OASVersion:    docs[0].OASVersion,
		SourceFormat:  docs[0].SourceFormat,
		Warnings:      make([]string, 0),
		firstFilePath: docs[0].SourcePath,
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

	// Merge all documents
	for i, doc := range docs {
		oas2Doc, ok := doc.OAS2Document()
		if !ok || oas2Doc == nil {
			return nil, fmt.Errorf("joiner: document at index %d (path: %s) is not a valid OAS 2.0 document", i, doc.SourcePath)
		}
		ctx := documentContext{
			filePath: doc.SourcePath,
			docIndex: i,
			result:   &doc,
		}

		if err := j.mergeOAS2Document(joined, oas2Doc, ctx, result); err != nil {
			return nil, err
		}
	}

	result.Document = joined

	// Apply semantic deduplication if enabled
	if j.config.SemanticDeduplication && len(joined.Definitions) > 1 {
		compare := func(left, right *parser.Schema) bool {
			res := CompareSchemas(left, right, EquivalenceModeDeep)
			return res.Equivalent
		}
		config := schemautil.DefaultDeduplicationConfig()
		deduper := schemautil.NewSchemaDeduplicator(config, compare)
		dedupeResult, err := deduper.Deduplicate(joined.Definitions)
		if err != nil {
			return nil, fmt.Errorf("joiner: semantic deduplication failed: %w", err)
		}

		// Apply results: replace definitions map with canonical schemas only
		joined.Definitions = dedupeResult.CanonicalSchemas

		// Register aliases for reference rewriting
		if len(dedupeResult.Aliases) > 0 {
			if result.rewriter == nil {
				result.rewriter = NewSchemaRewriter()
			}
			for alias, canonical := range dedupeResult.Aliases {
				result.rewriter.RegisterRename(alias, canonical, joined.OASVersion)
			}
			result.AddWarning(NewSemanticDedupSummaryWarning(dedupeResult.RemovedCount, "definition"))
		}
	}

	result.Stats = parser.GetDocumentStats(joined)

	// Apply reference rewriting if definitions were renamed
	if result.rewriter != nil {
		if err := result.rewriter.RewriteDocument(joined); err != nil {
			return nil, fmt.Errorf("joiner: failed to rewrite references after definition renames: %w", err)
		}
	}

	return result, nil
}

// mergeOAS2Document merges a single OAS2 document into the joined document
func (j *Joiner) mergeOAS2Document(joined *parser.OAS2Document, oas2Doc *parser.OAS2Document, ctx documentContext, result *JoinResult) error {
	// Merge paths
	if err := j.mergeOAS2Paths(joined, oas2Doc, ctx, result); err != nil {
		return err
	}

	// Build reference graph if operation context is enabled
	var sourceGraph *RefGraph
	if j.config.OperationContext {
		sourceGraph = buildRefGraphOAS2(oas2Doc)
	}

	// Merge definitions (schemas)
	if err := j.mergeOAS2Definitions(joined, oas2Doc, ctx, result, sourceGraph); err != nil {
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
func (j *Joiner) mergeOAS2Definitions(joined, source *parser.OAS2Document, ctx documentContext, result *JoinResult, sourceGraph *RefGraph) error {
	schemaStrategy := j.getEffectiveStrategy(j.config.SchemaStrategy)

	// Get namespace prefix for this source (if configured)
	sourcePrefix := j.getNamespacePrefix(ctx.filePath)

	for name, schema := range source.Definitions {
		// Determine the effective name for this definition
		effectiveName := name

		// If AlwaysApplyPrefix is true and source has a prefix, apply it to all definitions
		if j.config.AlwaysApplyPrefix && sourcePrefix != "" {
			effectiveName = j.generatePrefixedSchemaName(name, sourcePrefix)

			// Register rename for reference rewriting (original name -> prefixed name)
			if result.rewriter == nil {
				result.rewriter = NewSchemaRewriter()
			}
			result.rewriter.RegisterRename(name, effectiveName, result.OASVersion)

			line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", name))
			result.AddWarning(NewNamespacePrefixWarning(name, effectiveName, "definition", ctx.filePath, line, col))
		}

		if _, exists := joined.Definitions[effectiveName]; exists {
			// Handle collision based on strategy
			result.CollisionCount++

			// Invoke collision handler if configured for schemas
			if j.shouldInvokeHandler(CollisionTypeSchema) {
				collision := CollisionContext{
					Type:               CollisionTypeSchema,
					Name:               effectiveName,
					JSONPath:           fmt.Sprintf("$.definitions.%s", effectiveName),
					LeftSource:         result.firstFilePath,
					LeftLocation:       j.getLocationPtr(result.firstFilePath, fmt.Sprintf("$.definitions.%s", effectiveName)),
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
				switch j.config.EquivalenceMode {
				case "shallow":
					mode = EquivalenceModeShallow
				case "deep":
					mode = EquivalenceModeDeep
				}

				if mode != EquivalenceModeNone {
					eqResult := CompareSchemas(joined.Definitions[effectiveName], schema, mode)
					if eqResult.Equivalent {
						// Schemas are equivalent, keep existing and skip
						line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", effectiveName))
						result.AddWarning(NewSchemaDedupWarning(effectiveName, "definition", ctx.filePath, line, col))
						j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, schemaStrategy, "deduplicated", "")
						continue
					}
					// Not equivalent, fall back to fail
					return fmt.Errorf("definition '%s' collision: not equivalent, deduplicate strategy requires identical schemas (found %d differences)", effectiveName, len(eqResult.Differences))
				}
				return fmt.Errorf("definition '%s' collision: deduplicate strategy requires equivalence mode to be 'shallow' or 'deep'", effectiveName)

			case StrategyRenameLeft:
				// Rename the existing (left) definition and keep the new (right) definition under original name
				// Use namespace prefix if available for the left source, otherwise use template
				leftPrefix := j.getNamespacePrefix(result.firstFilePath)
				var newName string
				if leftPrefix != "" {
					newName = j.generatePrefixedSchemaName(effectiveName, leftPrefix)
				} else {
					newName = j.generateRenamedSchemaName(effectiveName, result.firstFilePath, 0, nil)
				}

				// Move existing definition to new name
				joined.Definitions[newName] = joined.Definitions[effectiveName]

				// Add new definition under original name
				joined.Definitions[effectiveName] = schema

				// Register rename for reference rewriting
				if result.rewriter == nil {
					result.rewriter = NewSchemaRewriter()
				}
				result.rewriter.RegisterRename(effectiveName, newName, result.OASVersion)

				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", effectiveName))
				result.AddWarning(NewSchemaRenamedWarning(effectiveName, newName, "definition", ctx.filePath, line, col, true))
				j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, schemaStrategy, "renamed", newName)

			case StrategyRenameRight:
				// Rename the new (right) definition and keep existing (left) definition under original name
				// Use namespace prefix if available, otherwise use template
				var newName string
				if sourcePrefix != "" && !j.config.AlwaysApplyPrefix {
					// Source has prefix but AlwaysApplyPrefix is false - apply prefix now on collision
					newName = j.generatePrefixedSchemaName(name, sourcePrefix)
				} else {
					newName = j.generateRenamedSchemaName(effectiveName, ctx.filePath, ctx.docIndex, sourceGraph)
				}

				// Add new definition under renamed name
				joined.Definitions[newName] = schema

				// Keep existing definition under original name (no change needed)

				// Register rename for reference rewriting
				if result.rewriter == nil {
					result.rewriter = NewSchemaRewriter()
				}
				result.rewriter.RegisterRename(effectiveName, newName, result.OASVersion)

				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", effectiveName))
				result.AddWarning(NewSchemaRenamedWarning(effectiveName, newName, "definition", ctx.filePath, line, col, false))
				j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, schemaStrategy, "renamed", newName)

			default:
				// Handle existing strategies
				if err := j.handleCollision(effectiveName, "definitions", schemaStrategy, result.firstFilePath, ctx.filePath); err != nil {
					return err
				}
				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.definitions.%s", effectiveName))
				if j.shouldOverwrite(schemaStrategy) {
					joined.Definitions[effectiveName] = schema
					result.AddWarning(NewSchemaCollisionWarning(effectiveName, "overwritten", "definitions", result.firstFilePath, ctx.filePath, line, col))
					j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, schemaStrategy, "kept-right", "")
				} else {
					result.AddWarning(NewSchemaCollisionWarning(effectiveName, "kept from first document", "definitions", result.firstFilePath, ctx.filePath, line, col))
					j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, schemaStrategy, "kept-left", "")
				}
			}
		} else {
			joined.Definitions[effectiveName] = schema
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
