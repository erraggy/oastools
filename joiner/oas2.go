package joiner

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// joinOAS2Documents joins multiple OAS 2.0 (Swagger) documents
func (j *Joiner) joinOAS2Documents(docs []parser.ParseResult) (*JoinResult, error) {
	// Start with a copy of the first document
	baseDoc := docs[0].Document.(*parser.OAS2Document)

	result := &JoinResult{
		Version:       docs[0].Version,
		OASVersion:    docs[0].OASVersion,
		SourceFormat:  docs[0].SourceFormat,
		Warnings:      make([]string, 0),
		firstFilePath: docs[0].SourcePath,
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
		oas2Doc := doc.Document.(*parser.OAS2Document)
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
	return result, nil
}

// mergeOAS2Document merges a single OAS2 document into the joined document
func (j *Joiner) mergeOAS2Document(joined *parser.OAS2Document, oas2Doc *parser.OAS2Document, ctx documentContext, result *JoinResult) error {
	// Merge paths
	if err := j.mergeOAS2Paths(joined, oas2Doc, ctx, result); err != nil {
		return err
	}

	// Merge definitions (schemas)
	if err := j.mergeOAS2Definitions(joined, oas2Doc, ctx, result); err != nil {
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
	for path, pathItem := range source.Paths {
		if _, exists := joined.Paths[path]; exists {
			if err := j.handleCollision(path, "paths", pathStrategy, result.firstFilePath, ctx.filePath); err != nil {
				return err
			}
			result.CollisionCount++
			if j.shouldOverwrite(pathStrategy) {
				joined.Paths[path] = pathItem
				result.Warnings = append(result.Warnings, fmt.Sprintf("path '%s' at paths.%s overwritten: source %s", path, path, ctx.filePath))
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("path '%s' at paths.%s kept from %s (collision with %s)", path, path, result.firstFilePath, ctx.filePath))
			}
		} else {
			joined.Paths[path] = pathItem
		}
	}
	return nil
}

// mergeOAS2Definitions merges definitions (schemas) from source document
func (j *Joiner) mergeOAS2Definitions(joined, source *parser.OAS2Document, ctx documentContext, result *JoinResult) error {
	schemaStrategy := j.getEffectiveStrategy(j.config.SchemaStrategy)
	for name, schema := range source.Definitions {
		if _, exists := joined.Definitions[name]; exists {
			if err := j.handleCollision(name, "definitions", schemaStrategy, result.firstFilePath, ctx.filePath); err != nil {
				return err
			}
			result.CollisionCount++
			if j.shouldOverwrite(schemaStrategy) {
				joined.Definitions[name] = schema
				result.Warnings = append(result.Warnings, fmt.Sprintf("definition '%s' at definitions.%s overwritten: source %s", name, name, ctx.filePath))
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("definition '%s' at definitions.%s kept from %s (collision with %s)", name, name, result.firstFilePath, ctx.filePath))
			}
		} else {
			joined.Definitions[name] = schema
		}
	}
	return nil
}

// mergeOAS2Components merges parameters, responses, and security definitions
func (j *Joiner) mergeOAS2Components(joined, source *parser.OAS2Document, ctx documentContext, result *JoinResult) error {
	componentStrategy := j.getEffectiveStrategy(j.config.ComponentStrategy)

	if err := mergeMap(j, joined.Parameters, source.Parameters, "parameters", componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := mergeMap(j, joined.Responses, source.Responses, "responses", componentStrategy, ctx, result); err != nil {
		return err
	}
	if err := mergeMap(j, joined.SecurityDefinitions, source.SecurityDefinitions, "securityDefinitions", componentStrategy, ctx, result); err != nil {
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
			result.Warnings = append(result.Warnings, fmt.Sprintf("host '%s' from %s ignored (using first document's host: '%s')", source.Host, ctx.filePath, joined.Host))
		}
		if source.BasePath != "" && source.BasePath != joined.BasePath {
			result.Warnings = append(result.Warnings, fmt.Sprintf("basePath '%s' from %s ignored (using first document's basePath: '%s')", source.BasePath, ctx.filePath, joined.BasePath))
		}

		// Info object is always taken from the first document
		// Additional info sections from subsequent documents are ignored
	}
}

// mergeUniqueStrings merges two string slices, removing duplicates
func (j *Joiner) mergeUniqueStrings(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))

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
