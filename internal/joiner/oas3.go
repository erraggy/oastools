package joiner

import (
	"fmt"

	"github.com/erraggy/oastools/internal/parser"
)

// joinOAS3Documents joins multiple OAS 3.x documents
func (j *Joiner) joinOAS3Documents(docs []*parser.ParseResult, contexts []documentContext) (*JoinResult, error) {
	// Start with a copy of the first document
	baseDoc := docs[0].Document.(*parser.OAS3Document)

	result := &JoinResult{
		Version:       docs[0].Version,
		OASVersion:    docs[0].OASVersion,
		Warnings:      make([]string, 0),
		firstFilePath: contexts[0].filePath,
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
		oas3Doc := doc.Document.(*parser.OAS3Document)
		ctx := contexts[i]

		// Merge paths
		pathStrategy := j.getEffectiveStrategy(j.config.PathStrategy)
		for path, pathItem := range oas3Doc.Paths {
			if _, exists := joined.Paths[path]; exists {
				if err := j.handleCollision(path, "paths", pathStrategy, result.firstFilePath, ctx.filePath); err != nil {
					return nil, err
				}
				result.CollisionCount++
				if j.shouldOverwrite(pathStrategy) {
					joined.Paths[path] = pathItem
					result.Warnings = append(result.Warnings, fmt.Sprintf("path '%s' overwritten: %s -> %s", path, result.firstFilePath, ctx.filePath))
				} else {
					result.Warnings = append(result.Warnings, fmt.Sprintf("path '%s' kept from %s (collision with %s)", path, result.firstFilePath, ctx.filePath))
				}
			} else {
				joined.Paths[path] = pathItem
			}
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
					result.Warnings = append(result.Warnings, fmt.Sprintf("webhook '%s' overwritten: %s -> %s", name, result.firstFilePath, ctx.filePath))
				} else {
					result.Warnings = append(result.Warnings, fmt.Sprintf("webhook '%s' kept from %s (collision with %s)", name, result.firstFilePath, ctx.filePath))
				}
			} else {
				joined.Webhooks[name] = webhook
			}
		}

		// Merge components
		if oas3Doc.Components != nil {
			if err := j.mergeOAS3Components(joined.Components, oas3Doc.Components, ctx, result); err != nil {
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
	return result, nil
}

// mergeOAS3Components merges components from source into target
func (j *Joiner) mergeOAS3Components(target, source *parser.Components, ctx documentContext, result *JoinResult) error {
	schemaStrategy := j.getEffectiveStrategy(j.config.SchemaStrategy)
	componentStrategy := j.getEffectiveStrategy(j.config.ComponentStrategy)

	// Merge schemas with detailed warnings
	if err := j.mergeSchemas(target.Schemas, source.Schemas, schemaStrategy, ctx, result); err != nil {
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
func (j *Joiner) mergeSchemas(target, source map[string]*parser.Schema, strategy CollisionStrategy, ctx documentContext, result *JoinResult) error {
	for name, schema := range source {
		if _, exists := target[name]; exists {
			if err := j.handleCollision(name, "components.schemas", strategy, result.firstFilePath, ctx.filePath); err != nil {
				return err
			}
			result.CollisionCount++
			if j.shouldOverwrite(strategy) {
				target[name] = schema
				result.Warnings = append(result.Warnings, fmt.Sprintf("schema '%s' at components.schemas.%s overwritten: source %s", name, name, ctx.filePath))
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("schema '%s' at components.schemas.%s kept from %s (collision with %s)", name, name, result.firstFilePath, ctx.filePath))
			}
		} else {
			target[name] = schema
		}
	}
	return nil
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
