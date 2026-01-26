package joiner

import (
	"fmt"

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
	joined.Components.PathItems = make(map[string]*parser.PathItem)

	// Merge all documents
	for i, doc := range docs {
		oas3Doc, ok := doc.OAS3Document()
		if !ok || oas3Doc == nil {
			return nil, fmt.Errorf("joiner: document at index %d (path: %s) is not a valid OAS 3.x document", i, doc.SourcePath)
		}
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
			if _, exists := joined.Webhooks[name]; exists {
				if err := j.handleCollision(name, "webhooks", pathStrategy, result.firstFilePath, ctx.filePath); err != nil {
					return nil, err
				}
				result.CollisionCount++
				if j.shouldOverwrite(pathStrategy) {
					joined.Webhooks[name] = webhook
					line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.webhooks.%s", name))
					result.AddWarning(NewWebhookCollisionWarning(name, "overwritten", result.firstFilePath, ctx.filePath, line, col))
				} else {
					line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.webhooks.%s", name))
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

	// Apply semantic deduplication if enabled
	if j.config.SemanticDeduplication && len(joined.Components.Schemas) > 1 {
		compare := func(left, right *parser.Schema) bool {
			res := CompareSchemas(left, right, EquivalenceModeDeep)
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

		// Register aliases for reference rewriting
		if len(dedupeResult.Aliases) > 0 {
			if result.rewriter == nil {
				result.rewriter = NewSchemaRewriter()
			}
			for alias, canonical := range dedupeResult.Aliases {
				result.rewriter.RegisterRename(alias, canonical, joined.OASVersion)
			}
			result.AddWarning(NewSemanticDedupSummaryWarning(dedupeResult.RemovedCount, "schema"))
		}
	}

	result.Stats = parser.GetDocumentStats(joined)

	// Apply reference rewriting if schemas were renamed
	if result.rewriter != nil {
		if err := result.rewriter.RewriteDocument(joined); err != nil {
			return nil, fmt.Errorf("joiner: failed to rewrite references after schema renames: %w", err)
		}
	}

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
	if err := j.mergeCallbacks(target.Callbacks, source.Callbacks, componentStrategy, ctx, result); err != nil {
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
			if result.rewriter == nil {
				result.rewriter = NewSchemaRewriter()
			}
			result.rewriter.RegisterRename(name, effectiveName, result.OASVersion)

			line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", name))
			result.AddWarning(NewNamespacePrefixWarning(name, effectiveName, "schema", ctx.filePath, line, col))
		}

		if _, exists := target[effectiveName]; exists {
			// Handle collision based on strategy
			result.CollisionCount++

			// Invoke collision handler if configured
			if j.shouldInvokeHandler(CollisionTypeSchema) {
				collision := CollisionContext{
					Type:               CollisionTypeSchema,
					Name:               effectiveName,
					JSONPath:           fmt.Sprintf("$.components.schemas.%s", effectiveName),
					LeftSource:         result.firstFilePath,
					LeftLocation:       j.getLocationPtr(result.firstFilePath, fmt.Sprintf("$.components.schemas.%s", effectiveName)),
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
					applied, err := j.applySchemaResolution(collision, resolution, target, result, ctx, sourceGraph)
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
				switch j.config.EquivalenceMode {
				case "shallow":
					mode = EquivalenceModeShallow
				case "deep":
					mode = EquivalenceModeDeep
				}

				if mode != EquivalenceModeNone {
					eqResult := CompareSchemas(target[effectiveName], schema, mode)
					if eqResult.Equivalent {
						// Schemas are equivalent, keep existing and skip
						line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
						result.AddWarning(NewSchemaDedupWarning(effectiveName, "schema", ctx.filePath, line, col))
						j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, "deduplicated", "")
						continue
					}
					// Not equivalent, fall back to default strategy or fail
					return fmt.Errorf("schema '%s' collision: not equivalent, deduplicate strategy requires identical schemas (found %d differences)", effectiveName, len(eqResult.Differences))
				}
				return fmt.Errorf("schema '%s' collision: deduplicate strategy requires equivalence mode to be 'shallow' or 'deep'", effectiveName)

			case StrategyRenameLeft:
				// Rename the existing (left) schema and keep the new (right) schema under original name
				// Use namespace prefix if available for the left source, otherwise use template
				leftPrefix := j.getNamespacePrefix(result.firstFilePath)
				var newName string
				if leftPrefix != "" {
					newName = j.generatePrefixedSchemaName(effectiveName, leftPrefix)
				} else {
					// Pass nil for graph since we don't have the original document's graph readily available
					newName = j.generateRenamedSchemaName(effectiveName, result.firstFilePath, 0, nil)
				}

				// Move existing schema to new name
				target[newName] = target[effectiveName]

				// Add new schema under original name
				target[effectiveName] = schema

				// Register rename for reference rewriting (will be applied at end of join)
				if result.rewriter == nil {
					result.rewriter = NewSchemaRewriter()
				}
				result.rewriter.RegisterRename(effectiveName, newName, result.OASVersion)

				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
				result.AddWarning(NewSchemaRenamedWarning(effectiveName, newName, "schema", ctx.filePath, line, col, true))
				j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, "renamed", newName)

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

				// Add new schema under renamed name
				target[newName] = schema

				// Keep existing schema under original name (no change needed)

				// Register rename for reference rewriting
				if result.rewriter == nil {
					result.rewriter = NewSchemaRewriter()
				}
				result.rewriter.RegisterRename(effectiveName, newName, result.OASVersion)

				line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
				result.AddWarning(NewSchemaRenamedWarning(effectiveName, newName, "schema", ctx.filePath, line, col, false))
				j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, "renamed", newName)

			default:
				// Handle existing strategies (accept-left, accept-right, fail, fail-on-paths)
				if err := j.handleCollision(effectiveName, "components.schemas", strategy, result.firstFilePath, ctx.filePath); err != nil {
					return err
				}
				if j.shouldOverwrite(strategy) {
					target[effectiveName] = schema
					line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
					result.AddWarning(NewSchemaCollisionWarning(effectiveName, "overwritten", "components.schemas", result.firstFilePath, ctx.filePath, line, col))
					j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, "kept-right", "")
				} else {
					line, col := j.getLocation(ctx.filePath, fmt.Sprintf("$.components.schemas.%s", effectiveName))
					result.AddWarning(NewSchemaCollisionWarning(effectiveName, "kept from first document", "components.schemas", result.firstFilePath, ctx.filePath, line, col))
					j.recordCollisionEvent(result, effectiveName, result.firstFilePath, ctx.filePath, strategy, "kept-left", "")
				}
			}
		} else {
			target[effectiveName] = schema
		}
	}
	return nil
}

// applySchemaResolution applies a CollisionResolution to a schema collision.
// Returns true if the resolution was fully handled, false if strategy should still be applied.
func (j *Joiner) applySchemaResolution(
	collision CollisionContext,
	resolution CollisionResolution,
	target map[string]*parser.Schema,
	result *JoinResult,
	ctx documentContext,
	sourceGraph *RefGraph,
) (bool, error) {
	// Record message as warning if provided
	if resolution.Message != "" {
		line, col := j.getLocation(ctx.filePath, collision.JSONPath)
		result.AddWarning(NewHandlerResolutionWarning(collision.JSONPath, resolution.Message, ctx.filePath, line, col))
	}

	schema, ok := collision.RightValue.(*parser.Schema)
	if !ok {
		return false, fmt.Errorf("collision handler: RightValue is not a *parser.Schema")
	}

	switch resolution.Action {
	case ResolutionContinue:
		// Delegate to configured strategy
		return false, nil

	case ResolutionAcceptLeft:
		// Keep existing (left), discard incoming (right)
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "kept-left", "")
		return true, nil

	case ResolutionAcceptRight:
		// Replace with incoming (right)
		target[collision.Name] = schema
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "kept-right", "")
		return true, nil

	case ResolutionRename:
		// Rename right schema
		newName := j.generateRenamedSchemaName(collision.Name, ctx.filePath, ctx.docIndex, sourceGraph)
		target[newName] = schema
		if result.rewriter == nil {
			result.rewriter = NewSchemaRewriter()
		}
		result.rewriter.RegisterRename(collision.Name, newName, result.OASVersion)
		line, col := j.getLocation(ctx.filePath, collision.JSONPath)
		result.AddWarning(NewSchemaRenamedWarning(collision.Name, newName, "schema", ctx.filePath, line, col, false))
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "renamed", newName)
		return true, nil

	case ResolutionDeduplicate:
		// Keep left, discard right (treat as equivalent)
		line, col := j.getLocation(ctx.filePath, collision.JSONPath)
		result.AddWarning(NewSchemaDedupWarning(collision.Name, "schema", ctx.filePath, line, col))
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "deduplicated", "")
		return true, nil

	case ResolutionFail:
		// Return error with handler's message
		msg := resolution.Message
		if msg == "" {
			msg = fmt.Sprintf("schema collision on %q rejected by handler", collision.Name)
		}
		return true, fmt.Errorf("collision handler: %s", msg)

	case ResolutionCustom:
		if resolution.CustomValue == nil {
			return true, fmt.Errorf("collision handler: ResolutionCustom requires CustomValue")
		}
		customSchema, ok := resolution.CustomValue.(*parser.Schema)
		if !ok {
			return true, fmt.Errorf("collision handler: CustomValue must be *parser.Schema for schema collisions")
		}
		target[collision.Name] = customSchema
		j.recordCollisionEvent(result, collision.Name, collision.LeftSource, collision.RightSource, collision.ConfiguredStrategy, "custom", "")
		return true, nil

	default:
		return false, fmt.Errorf("unknown resolution action: %d", resolution.Action)
	}
}

// Helper functions for merging specific component types
func (j *Joiner) mergeResponses(target, source map[string]*parser.Response, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.responses", strategy, ctx, result)
}

func (j *Joiner) mergeParameters(target, source map[string]*parser.Parameter, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.parameters", strategy, ctx, result)
}

func (j *Joiner) mergeExamples(target, source map[string]*parser.Example, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.examples", strategy, ctx, result)
}

func (j *Joiner) mergeRequestBodies(target, source map[string]*parser.RequestBody, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.requestBodies", strategy, ctx, result)
}

func (j *Joiner) mergeHeaders(target, source map[string]*parser.Header, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.headers", strategy, ctx, result)
}

func (j *Joiner) mergeSecuritySchemes(target, source map[string]*parser.SecurityScheme, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.securitySchemes", strategy, ctx, result)
}

func (j *Joiner) mergeLinks(target, source map[string]*parser.Link, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.links", strategy, ctx, result)
}

func (j *Joiner) mergeCallbacks(target, source map[string]*parser.Callback, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.callbacks", strategy, ctx, result)
}

func (j *Joiner) mergePathItems(target, source map[string]*parser.PathItem, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	return mergeMap(j, target, source, "components.pathItems", strategy, ctx, result)
}

// mergeMap is a generic helper function to merge component maps
func mergeMap[T any](j *Joiner, target, source map[string]T, section string, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	for name, item := range source {
		if _, exists := target[name]; exists {
			if err := j.handleCollision(name, section, strategy, result.firstFilePath, ctx.filePath); err != nil {
				return err
			}
			result.CollisionCount++
			if j.shouldOverwrite(strategy) {
				target[name] = item
			}
		} else {
			target[name] = item
		}
	}
	return nil
}
