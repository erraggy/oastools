package joiner

import (
	"fmt"
	"slices"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// joinOAS3Documents joins multiple OAS 3.x documents
func (j *Joiner) joinOAS3Documents(docs []parser.ParseResult) (*JoinResult, error) {
	// Start with a copy of the first document
	baseDoc, ok := docs[0].OAS3Document()
	if !ok || baseDoc == nil {
		return nil, fmt.Errorf("joiner: first document is not a valid OAS 3.x document")
	}

	// Collect the typed documents up front: the reference rewriting after the
	// merge needs to know which of them contributed each part of the join.
	sources := make([]*parser.OAS3Document, len(docs))
	for i, doc := range docs {
		oas3Doc, ok := doc.OAS3Document()
		if !ok || oas3Doc == nil {
			return nil, fmt.Errorf("joiner: document at index %d (path: %s) is not a valid OAS 3.x document", i, doc.SourcePath)
		}
		// A document handed in twice would share every schema with its other
		// position, so a strategy that keeps both sides would store one schema
		// under two names. Copy the repeat so the two positions are independent
		// and each can be told apart by the rewriting below (#481).
		if slices.Contains(sources[:i], oas3Doc) {
			oas3Doc = oas3Doc.DeepCopy()
		}
		sources[i] = oas3Doc
	}

	result := &JoinResult{
		Version:       docs[0].Version,
		OASVersion:    docs[0].OASVersion,
		SourceFormat:  docs[0].SourceFormat,
		Warnings:      make([]string, 0),
		firstFilePath: docs[0].SourcePath,
		scope:         newRenameScope(len(docs), docs[0].OASVersion),
	}
	if j.tracksSchemaOrigins() {
		result.origins = make(map[string]schemaOrigin)
	}

	// Initialize collision report if enabled
	if j.config.CollisionReport {
		result.CollisionDetails = NewCollisionReport()
	}

	// Create the joined document starting with the base
	joined := &parser.OAS3Document{
		OpenAPI:           baseDoc.OpenAPI,
		Info:              copyInfo(baseDoc.Info),
		JSONSchemaDialect: baseDoc.JSONSchemaDialect,
		Servers:           copyServers(baseDoc.Servers),
		Paths:             make(parser.Paths),
		Webhooks:          make(map[string]*parser.PathItem),
		Components:        &parser.Components{},
		Security:          copySecurityRequirements(baseDoc.Security),
		Tags:              copyTags(baseDoc.Tags),
		ExternalDocs:      copyExternalDocs(baseDoc.ExternalDocs),
		OASVersion:        baseDoc.OASVersion,
	}

	// Initialize component maps
	if joined.Components == nil {
		joined.Components = &parser.Components{}
	}
	joined.Components.Schemas = make(map[string]*parser.Schema)
	joined.Components.Responses = make(map[string]*parser.Response)
	joined.Components.Parameters = make(map[string]*parser.Parameter)
	joined.Components.Examples = make(map[string]*parser.Example)
	joined.Components.RequestBodies = make(map[string]*parser.RequestBody)
	joined.Components.Headers = make(map[string]*parser.Header)
	joined.Components.SecuritySchemes = make(map[string]*parser.SecurityScheme)
	joined.Components.Links = make(map[string]*parser.Link)
	joined.Components.Callbacks = make(map[string]*parser.Callback)
	joined.Components.CallbackRefs = make(map[string]*parser.Reference)
	joined.Components.PathItems = make(map[string]*parser.PathItem)

	// Merge all documents
	for i, doc := range docs {
		oas3Doc := sources[i]
		ctx := documentContext{
			filePath: doc.SourcePath,
			docIndex: i,
			result:   &doc,
		}

		// Merge paths
		pathStrategy := j.getEffectiveStrategy(j.config.PathStrategy)
		if err := j.mergePathsMap(joined.Paths, oas3Doc.Paths, pathStrategy, ctx, result); err != nil {
			return nil, err
		}

		// Merge webhooks (OAS 3.1+)
		for name, webhook := range oas3Doc.Webhooks {
			existingWebhook, exists := joined.Webhooks[name]
			if exists {
				jsonPath := fmt.Sprintf("$.webhooks.%s", name)
				result.CollisionCount++

				// Invoke collision handler if registered and applicable
				if j.collisionHandler != nil && j.shouldInvokeHandler(CollisionTypeWebhook) {
					collision := CollisionContext{
						Type:               CollisionTypeWebhook,
						Name:               name,
						JSONPath:           jsonPath,
						LeftSource:         result.firstFilePath,
						LeftLocation:       j.getLocationPtr(result.firstFilePath, jsonPath),
						LeftValue:          existingWebhook,
						RightSource:        ctx.filePath,
						RightLocation:      j.getLocationPtr(ctx.filePath, jsonPath),
						RightValue:         webhook,
						ConfiguredStrategy: pathStrategy,
					}

					resolution, handlerErr := j.collisionHandler(collision)
					if handlerErr != nil {
						// Log warning and fall back to configured strategy
						line, col := j.getLocation(ctx.filePath, jsonPath)
						result.AddWarning(NewHandlerErrorWarning(
							jsonPath,
							fmt.Sprintf("collision handler error: %v; using %s strategy", handlerErr, pathStrategy),
							ctx.filePath, line, col,
						))
						// Fall through to strategy handling below
					} else {
						// Apply the resolution
						handled, shouldOverwrite, err := j.applyComponentResolution(componentResolutionParams{
							collision:  collision,
							resolution: resolution,
							result:     result,
							ctx:        ctx,
						})
						if err != nil {
							return nil, err
						}
						if handled {
							if shouldOverwrite {
								joined.Webhooks[name] = webhook
							}
							continue // Resolution handled, skip strategy handling
						}
						// ResolutionContinue falls through to strategy handling
					}
				}

				// Default strategy handling (or fallback from handler)
				if err := j.handleCollision(name, "webhooks", pathStrategy, result.firstFilePath, ctx.filePath); err != nil {
					return nil, err
				}
				if j.shouldOverwrite(pathStrategy) {
					joined.Webhooks[name] = webhook
					line, col := j.getLocation(ctx.filePath, jsonPath)
					result.AddWarning(NewWebhookCollisionWarning(name, "overwritten", result.firstFilePath, ctx.filePath, line, col))
				} else {
					line, col := j.getLocation(ctx.filePath, jsonPath)
					result.AddWarning(NewWebhookCollisionWarning(name, "kept from first document", result.firstFilePath, ctx.filePath, line, col))
				}
			} else {
				joined.Webhooks[name] = webhook
			}
		}

		// Merge components
		if oas3Doc.Components != nil {
			// Build reference graph if operation context is enabled
			var sourceGraph *RefGraph
			if j.config.OperationContext {
				sourceGraph = buildRefGraphOAS3(oas3Doc, oas3Doc.OASVersion)
			}

			if err := j.mergeOAS3Components(joined.Components, oas3Doc.Components, ctx, result, sourceGraph); err != nil {
				return nil, err
			}
		}

		// Merge servers (if configured)
		if j.config.MergeArrays && i > 0 {
			joined.Servers = append(joined.Servers, copyServers(oas3Doc.Servers)...)
		}

		// Merge security requirements (if configured)
		if j.config.MergeArrays && i > 0 {
			joined.Security = append(joined.Security, copySecurityRequirements(oas3Doc.Security)...)
		}

		// Merge tags
		if i > 0 {
			joined.Tags = j.mergeTags(joined.Tags, oas3Doc.Tags)
		}

		// Info object is always taken from the first document
		// Additional info sections from subsequent documents are ignored
	}

	result.Document = joined

	// Rewrite each document's references using the renames recorded against that
	// document. This runs before deduplication so that comparison sees references
	// in their final form.
	if err := result.scope.applyOAS3(joined, sources); err != nil {
		return nil, fmt.Errorf("joiner: failed to rewrite references after schema renames: %w", err)
	}

	// Apply semantic deduplication if enabled
	if j.config.SemanticDeduplication && len(joined.Components.Schemas) > 1 {
		compareOpts := j.buildCompareOptions(EquivalenceModeDeep)
		compare := func(left, right *parser.Schema) bool {
			res := CompareSchemasWithOptions(left, right, compareOpts)
			return res.Equivalent
		}
		config := schemautil.DefaultDeduplicationConfig()
		deduper := schemautil.NewSchemaDeduplicator(config, compare)
		dedupeResult, err := deduper.Deduplicate(joined.Components.Schemas)
		if err != nil {
			return nil, fmt.Errorf("joiner: semantic deduplication failed: %w", err)
		}

		// Apply results: replace schemas map with canonical schemas only
		joined.Components.Schemas = dedupeResult.CanonicalSchemas

		if len(dedupeResult.Aliases) > 0 {
			if err := rewriteDedupeAliases(joined, dedupeResult.Aliases, joined.OASVersion); err != nil {
				return nil, fmt.Errorf("joiner: failed to rewrite references after semantic deduplication: %w", err)
			}
			result.AddWarning(NewSemanticDedupSummaryWarning(dedupeResult.RemovedCount, "schema"))
		}
	}

	result.Stats = parser.GetDocumentStats(joined)

	return result, nil
}

// mergeOAS3Components merges components from source into target
func (j *Joiner) mergeOAS3Components(target, source *parser.Components, ctx documentContext, result *JoinResult, sourceGraph *RefGraph) error {
	schemaStrategy := j.getEffectiveStrategy(j.config.SchemaStrategy)
	componentStrategy := j.getEffectiveStrategy(j.config.ComponentStrategy)

	// Merge schemas with detailed warnings
	if err := j.mergeSchemas(target.Schemas, source.Schemas, schemaStrategy, ctx, result, sourceGraph); err != nil {
		return err
	}

	// Merge other components
	if err := j.mergeResponses(target.Responses, source.Responses, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := j.mergeParameters(target.Parameters, source.Parameters, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := j.mergeExamples(target.Examples, source.Examples, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := j.mergeRequestBodies(target.RequestBodies, source.RequestBodies, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := j.mergeHeaders(target.Headers, source.Headers, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := j.mergeSecuritySchemes(target.SecuritySchemes, source.SecuritySchemes, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := j.mergeLinks(target.Links, source.Links, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := j.mergeAllCallbacks(target, source, componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := j.mergePathItems(target.PathItems, source.PathItems, componentStrategy, ctx, result); err != nil {
		return err
	}

	return nil
}

// mergeSchemas is a specialized merger for schemas with detailed warnings
func (j *Joiner) mergeSchemas(target, source map[string]*parser.Schema, strategy CollisionStrategy, ctx documentContext, result *JoinResult, sourceGraph *RefGraph) error {
	// Get namespace prefix for this source (if configured)
	sourcePrefix := j.getNamespacePrefix(ctx.filePath)

	for name, schema := range source {
		// Determine the effective name for this schema
		effectiveName := name

		// If AlwaysApplyPrefix is true and source has a prefix, apply it to all schemas
		if j.config.AlwaysApplyPrefix && sourcePrefix != "" {
			effectiveName = j.generatePrefixedSchemaName(name, sourcePrefix)

			// Register rename for reference rewriting (original name -> prefixed name)
			result.scope.registerRight(ctx.docIndex, name, effectiveName)

			line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", name))
			result.AddWarning(NewNamespacePrefixWarning(name, effectiveName, "schema", ctx.filePath, line, col))
		}

		if _, exists := target[effectiveName]; exists {
			// Handle collision based on strategy
			result.CollisionCount++

			// Invoke collision handler if configured
			if j.shouldInvokeHandler(CollisionTypeSchema) {
				// The left side belongs to whichever document contributed the schema
				// now under this name, not necessarily the first (#479).
				leftSource := result.originOf(effectiveName).filePath
				collision := CollisionContext{
					Type:               CollisionTypeSchema,
					Name:               effectiveName,
					JSONPath:           fmt.Sprintf("$.components.schemas.%s", effectiveName),
					LeftSource:         leftSource,
					LeftLocation:       j.getLocationPtr(leftSource, fmt.Sprintf("$.components.schemas.%s", effectiveName)),
					LeftValue:          target[effectiveName],
					RightSource:        ctx.filePath,
					RightLocation:      j.getLocationPtr(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", name)),
					RightValue:         schema,
					RenameInfo:         buildRenameContextPtr(effectiveName, ctx.filePath, ctx.docIndex, sourceGraph, j.config.PrimaryOperationPolicy),
					ConfiguredStrategy: strategy,
				}

				resolution, handlerErr := j.collisionHandler(collision)
				if handlerErr != nil {
					// Log warning and fall back to configured strategy
					line, col := j.getLocation(ctx.filePath, collision.JSONPath)
					result.AddWarning(NewHandlerErrorWarning(
						collision.JSONPath,
						fmt.Sprintf("collision handler error: %v; using %s strategy", handlerErr, strategy),
						ctx.filePath, line, col,
					))
					// Fall through to strategy switch below
				} else {
					// Apply the resolution
					applied, err := j.applySchemaResolution(schemaResolutionParams{
						collision:   collision,
						resolution:  resolution,
						target:      target,
						result:      result,
						ctx:         ctx,
						sourceName:  name,
						sourceGraph: sourceGraph,
						label:       "schema",
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

			switch strategy {
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
					eqResult := CompareSchemasWithOptions(target[effectiveName], schema, j.buildCompareOptions(mode))
					if eqResult.Equivalent {
						// Schemas are equivalent, keep existing and skip
						line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
						result.AddWarning(NewSchemaDedupWarning(effectiveName, "schema", ctx.filePath, line, col))
						j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, resolutionDeduplicated, "")
						continue
					}
					// Not equivalent, fall back to default strategy or fail
					return fmt.Errorf("schema '%s' collision: not equivalent, deduplicate strategy requires identical schemas (found %d differences)", effectiveName, len(eqResult.Differences))
				}
				return fmt.Errorf("schema '%s' collision: deduplicate strategy requires equivalence mode to be 'shallow' or 'deep'", effectiveName)

			case StrategyRenameLeft:
				// Rename the existing (left) schema and keep the new (right) schema under original name
				// Name it after the document that contributed it, which is the first
				// document only in a two document join (#479).
				leftOrigin := result.originOf(effectiveName)
				leftPrefix := j.getNamespacePrefix(leftOrigin.filePath)
				var newName string
				if leftPrefix != "" {
					newName = j.generatePrefixedSchemaName(effectiveName, leftPrefix)
				} else {
					// No graph: operation-aware rename templates are not wired up for
					// the left side, which would need that document's graph built here.
					// See #482.
					newName = j.generateRenamedSchemaName(effectiveName, leftOrigin.filePath, leftOrigin.docIndex, nil)
				}
				newName = uniqueSchemaName(target, newName)

				// Move existing schema to new name
				target[newName] = target[effectiveName]
				result.moveOrigin(effectiveName, newName)

				// Add new schema under original name
				target[effectiveName] = schema
				result.recordOrigin(effectiveName, ctx)

				// Register rename for reference rewriting. Only the documents merged
				// before this one referenced the schema being moved.
				result.scope.registerLeft(ctx.docIndex, effectiveName, newName)

				line, col := j.getLocation(leftOrigin.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
				result.AddWarning(NewSchemaRenamedWarning(effectiveName, newName, "schema", leftOrigin.filePath, line, col, true))
				j.recordCollisionEvent(result, effectiveName, leftOrigin.filePath, ctx.filePath, strategy, resolutionRenamed, newName)

			case StrategyRenameRight:
				// Rename the new (right) schema and keep existing (left) schema under original name
				// Use namespace prefix if available, otherwise use template
				var newName string
				if sourcePrefix != "" && !j.config.AlwaysApplyPrefix {
					// Source has prefix but AlwaysApplyPrefix is false - apply prefix now on collision
					newName = j.generatePrefixedSchemaName(name, sourcePrefix)
				} else {
					// Pass sourceGraph for operation-aware renaming of the right/new schema
					newName = j.generateRenamedSchemaName(effectiveName, ctx.filePath, ctx.docIndex, sourceGraph)
				}
				newName = uniqueSchemaName(target, newName)

				// Add new schema under renamed name
				target[newName] = schema
				result.recordOrigin(newName, ctx)

				// Keep existing schema under original name (no change needed)

				// Register rename for reference rewriting. Only this document
				// referenced the schema being renamed.
				result.scope.registerRight(ctx.docIndex, name, newName)

				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
				result.AddWarning(NewSchemaRenamedWarning(effectiveName, newName, "schema", ctx.filePath, line, col, false))
				j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, resolutionRenamed, newName)

			default:
				// Handle existing strategies (accept-left, accept-right, fail, fail-on-paths)
				if err := j.handleCollision(effectiveName, "components.schemas", strategy, result.firstFilePath, ctx.filePath); err != nil {
					return err
				}
				if j.shouldOverwrite(strategy) {
					target[effectiveName] = schema
					result.recordOrigin(effectiveName, ctx)
					line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
					result.AddWarning(NewSchemaCollisionWarning(effectiveName, "overwritten", "components.schemas", result.firstFilePath, ctx.filePath, line, col))
					j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, resolutionKeptRight, "")
				} else {
					line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
					result.AddWarning(NewSchemaCollisionWarning(effectiveName, "kept from first document", "components.schemas", result.firstFilePath, ctx.filePath, line, col))
					j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, resolutionKeptLeft, "")
				}
			}
		} else {
			target[effectiveName] = schema
			result.recordOrigin(effectiveName, ctx)
		}
	}
	return nil
}

// Helper functions for merging specific component types
func (j *Joiner) mergeResponses(target, source map[string]*parser.Response, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.responses", CollisionTypeResponse, strategy, ctx, result)
}

func (j *Joiner) mergeParameters(target, source map[string]*parser.Parameter, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.parameters", CollisionTypeParameter, strategy, ctx, result)
}

func (j *Joiner) mergeExamples(target, source map[string]*parser.Example, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.examples", CollisionTypeExample, strategy, ctx, result)
}

func (j *Joiner) mergeRequestBodies(target, source map[string]*parser.RequestBody, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.requestBodies", CollisionTypeRequestBody, strategy, ctx, result)
}

func (j *Joiner) mergeHeaders(target, source map[string]*parser.Header, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.headers", CollisionTypeHeader, strategy, ctx, result)
}

func (j *Joiner) mergeSecuritySchemes(target, source map[string]*parser.SecurityScheme, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.securitySchemes", CollisionTypeSecurityScheme, strategy, ctx, result)
}

func (j *Joiner) mergeLinks(target, source map[string]*parser.Link, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.links", CollisionTypeLink, strategy, ctx, result)
}

// mergeAllCallbacks merges the components.callbacks section, which two Go maps
// carry between them: see parser.Callback.
//
// They share one namespace, so the same name in different forms is a collision.
// mergeMap only compares like with like, so the check below covers the pairing
// it misses. Left through, the joined document would hold that name in both
// maps, which cannot be written.
//
// That check fails the join outright, without consulting the collision handler
// or the configured CollisionStrategy, unlike every same-form collision. There is
// no resolution to offer: keeping either side, renaming, or deduplicating all
// leave a name whose form depends on which document won, and the result is a
// document rather than a component the caller chose. Same-form collisions still
// route through the handler in mergeCallbacks and mergeCallbackRefs below.
func (j *Joiner) mergeAllCallbacks(target, source *parser.Components, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	// The clash may be between the two documents or inside the incoming one:
	// decoding cannot produce a name in both maps, but a document assembled in
	// Go can, and merging it would carry the clash into the result.
	for name := range source.CallbackRefs {
		_, inSource := source.Callbacks[name]
		_, inTarget := target.Callbacks[name]
		if inSource || inTarget {
			return errCallbackFormCollision(name)
		}
	}
	for name := range source.Callbacks {
		if _, ok := target.CallbackRefs[name]; ok {
			return errCallbackFormCollision(name)
		}
	}
	if err := j.mergeCallbacks(target.Callbacks, source.Callbacks, strategy, ctx, result); err != nil {
		return err
	}
	return j.mergeCallbackRefs(target.CallbackRefs, source.CallbackRefs, strategy, ctx, result)
}

func errCallbackFormCollision(name string) error {
	return fmt.Errorf("joiner: components.callbacks.%s is a Callback Object in one document "+
		"and a Reference Object in another: the two forms cannot be merged", name)
}

func (j *Joiner) mergeCallbacks(target, source map[string]*parser.Callback, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.callbacks", CollisionTypeCallback, strategy, ctx, result)
}

func (j *Joiner) mergeCallbackRefs(target, source map[string]*parser.Reference, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.callbacks", CollisionTypeCallback, strategy, ctx, result)
}

func (j *Joiner) mergePathItems(target, source map[string]*parser.PathItem, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	// Note: pathItems in components is distinct from paths at the document root
	// We don't have a specific collision type for pathItems, so we treat them like paths
	return mergeMap(j, target, source, "components.pathItems", CollisionTypePath, strategy, ctx, result)
}

// mergeMap is a generic helper function to merge component maps with collision handler support.
func mergeMap[T any](j *Joiner, target, source map[string]T, section string, collisionType CollisionType, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	for name, item := range source {
		existing, exists := target[name]
		if exists {
			jsonPath := fmt.Sprintf("$.%s.%s", section, name)
			result.CollisionCount++

			// Invoke collision handler if registered and applicable
			if j.collisionHandler != nil && j.shouldInvokeHandler(collisionType) {
				collision := CollisionContext{
					Type:               collisionType,
					Name:               name,
					JSONPath:           jsonPath,
					LeftSource:         result.firstFilePath,
					LeftLocation:       j.getLocationPtr(result.firstFilePath, jsonPath),
					LeftValue:          existing,
					RightSource:        ctx.filePath,
					RightLocation:      j.getLocationPtr(ctx.filePath, jsonPath),
					RightValue:         item,
					ConfiguredStrategy: strategy,
				}

				resolution, handlerErr := j.collisionHandler(collision)
				if handlerErr != nil {
					// Log warning and fall back to configured strategy
					line, col := j.getLocation(ctx.filePath, jsonPath)
					result.AddWarning(NewHandlerErrorWarning(
						jsonPath,
						fmt.Sprintf("collision handler error: %v; using %s strategy", handlerErr, strategy),
						ctx.filePath, line, col,
					))
					// Fall through to strategy handling below
				} else {
					// Apply the resolution
					handled, shouldOverwrite, err := j.applyComponentResolution(componentResolutionParams{
						collision:  collision,
						resolution: resolution,
						result:     result,
						ctx:        ctx,
					})
					if err != nil {
						return err
					}
					if handled {
						if shouldOverwrite {
							target[name] = item
						}
						continue // Resolution handled, skip strategy handling
					}
					// ResolutionContinue falls through to strategy handling
				}
			}

			// Default strategy handling (or fallback from handler)
			if err := j.handleCollision(name, section, strategy, result.firstFilePath, ctx.filePath); err != nil {
				return err
			}
			if j.shouldOverwrite(strategy) {
				target[name] = item
			}
		} else {
			target[name] = item
		}
	}
	return nil
}
