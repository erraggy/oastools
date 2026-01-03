package walker

import (
	"fmt"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// sortedMapKeys returns sorted keys from any map with string keys.
func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// walkParameter walks a Parameter.
func (w *Walker) walkParameter(param *parser.Parameter, basePath string) error {
	if param == nil {
		return nil
	}

	continueToChildren := true
	if w.onParameter != nil {
		continueToChildren = w.handleAction(w.onParameter(param, basePath))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Schema (OAS 3.x)
	if param.Schema != nil {
		if err := w.walkSchema(param.Schema, basePath+".schema", 0); err != nil {
			return err
		}
	}

	// Content (OAS 3.x)
	if param.Content != nil {
		if err := w.walkContent(param.Content, basePath+".content"); err != nil {
			return err
		}
	}

	// Examples
	if param.Examples != nil {
		if err := w.walkExamples(param.Examples, basePath+".examples"); err != nil {
			return err
		}
	}

	return nil
}

// walkHeaders walks a map of Headers.
func (w *Walker) walkHeaders(headers map[string]*parser.Header, basePath string) error {
	for _, name := range sortedMapKeys(headers) {
		if w.stopped {
			return nil
		}
		header := headers[name]
		if header != nil {
			if err := w.walkHeader(name, header, basePath+"['"+name+"']"); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkHeader walks a single Header.
func (w *Walker) walkHeader(name string, header *parser.Header, basePath string) error {
	continueToChildren := true
	if w.onHeader != nil {
		continueToChildren = w.handleAction(w.onHeader(name, header, basePath))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Schema
	if header.Schema != nil {
		if err := w.walkSchema(header.Schema, basePath+".schema", 0); err != nil {
			return err
		}
	}

	// Content
	if header.Content != nil {
		if err := w.walkContent(header.Content, basePath+".content"); err != nil {
			return err
		}
	}

	// Examples
	if header.Examples != nil {
		if err := w.walkExamples(header.Examples, basePath+".examples"); err != nil {
			return err
		}
	}

	return nil
}

// walkContent walks a map of MediaTypes.
func (w *Walker) walkContent(content map[string]*parser.MediaType, basePath string) error {
	for _, mtName := range sortedMapKeys(content) {
		if w.stopped {
			return nil
		}
		mt := content[mtName]
		if mt != nil {
			if err := w.walkMediaType(mtName, mt, basePath+"['"+mtName+"']"); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkMediaType walks a single MediaType.
func (w *Walker) walkMediaType(name string, mt *parser.MediaType, basePath string) error {
	continueToChildren := true
	if w.onMediaType != nil {
		continueToChildren = w.handleAction(w.onMediaType(name, mt, basePath))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Schema
	if mt.Schema != nil {
		if err := w.walkSchema(mt.Schema, basePath+".schema", 0); err != nil {
			return err
		}
	}

	// Examples
	if mt.Examples != nil {
		if err := w.walkExamples(mt.Examples, basePath+".examples"); err != nil {
			return err
		}
	}

	return nil
}

// walkExamples walks a map of Examples.
func (w *Walker) walkExamples(examples map[string]*parser.Example, basePath string) error {
	for _, name := range sortedMapKeys(examples) {
		if w.stopped {
			return nil
		}
		ex := examples[name]
		if ex != nil && w.onExample != nil {
			w.handleAction(w.onExample(name, ex, basePath+"['"+name+"']"))
		}
	}
	return nil
}

// walkSchema walks a Schema and all its nested schemas.
func (w *Walker) walkSchema(schema *parser.Schema, basePath string, depth int) error {
	if schema == nil {
		return nil
	}

	// Check depth limit
	if depth > w.maxDepth {
		if w.onSchemaSkipped != nil {
			w.onSchemaSkipped("depth", schema, basePath)
		}
		return nil
	}

	// Check for cycle
	if w.visitedSchemas[schema] {
		if w.onSchemaSkipped != nil {
			w.onSchemaSkipped("cycle", schema, basePath)
		}
		return nil
	}

	w.visitedSchemas[schema] = true
	defer delete(w.visitedSchemas, schema)

	// Call handler
	if w.onSchema != nil {
		if !w.handleAction(w.onSchema(schema, basePath)) {
			if w.stopped {
				return nil
			}
			return nil // SkipChildren
		}
	}

	// Walk nested schemas in groups
	if err := w.walkSchemaProperties(schema, basePath, depth); err != nil {
		return err
	}
	if err := w.walkSchemaArrayKeywords(schema, basePath, depth); err != nil {
		return err
	}
	if err := w.walkSchemaComposition(schema, basePath, depth); err != nil {
		return err
	}
	if err := w.walkSchemaConditionals(schema, basePath, depth); err != nil {
		return err
	}
	return w.walkSchemaMisc(schema, basePath, depth)
}

// walkSchemaProperties walks object-related schema keywords.
func (w *Walker) walkSchemaProperties(schema *parser.Schema, basePath string, depth int) error {
	// Properties
	for _, name := range sortedMapKeys(schema.Properties) {
		if w.stopped {
			return nil
		}
		if prop := schema.Properties[name]; prop != nil {
			if err := w.walkSchema(prop, basePath+".properties['"+name+"']", depth+1); err != nil {
				return err
			}
		}
	}

	// PatternProperties
	for _, pattern := range sortedMapKeys(schema.PatternProperties) {
		if w.stopped {
			return nil
		}
		if prop := schema.PatternProperties[pattern]; prop != nil {
			if err := w.walkSchema(prop, basePath+".patternProperties['"+pattern+"']", depth+1); err != nil {
				return err
			}
		}
	}

	// AdditionalProperties (can be *Schema or bool)
	if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
		if err := w.walkSchema(addProps, basePath+".additionalProperties", depth+1); err != nil {
			return err
		}
	}

	// UnevaluatedProperties (can be *Schema or bool)
	if uProps, ok := schema.UnevaluatedProperties.(*parser.Schema); ok {
		if err := w.walkSchema(uProps, basePath+".unevaluatedProperties", depth+1); err != nil {
			return err
		}
	}

	// PropertyNames
	if schema.PropertyNames != nil {
		if err := w.walkSchema(schema.PropertyNames, basePath+".propertyNames", depth+1); err != nil {
			return err
		}
	}

	// DependentSchemas
	for _, name := range sortedMapKeys(schema.DependentSchemas) {
		if w.stopped {
			return nil
		}
		if ds := schema.DependentSchemas[name]; ds != nil {
			if err := w.walkSchema(ds, basePath+".dependentSchemas['"+name+"']", depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// walkSchemaArrayKeywords walks array-related schema keywords.
func (w *Walker) walkSchemaArrayKeywords(schema *parser.Schema, basePath string, depth int) error {
	// Items (can be *Schema or bool)
	if items, ok := schema.Items.(*parser.Schema); ok {
		if err := w.walkSchema(items, basePath+".items", depth+1); err != nil {
			return err
		}
	}

	// AdditionalItems (can be *Schema or bool)
	if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
		if err := w.walkSchema(addItems, basePath+".additionalItems", depth+1); err != nil {
			return err
		}
	}

	// PrefixItems (OAS 3.1+)
	for i, prefixItem := range schema.PrefixItems {
		if w.stopped {
			return nil
		}
		if prefixItem != nil {
			if err := w.walkSchema(prefixItem, fmt.Sprintf("%s.prefixItems[%d]", basePath, i), depth+1); err != nil {
				return err
			}
		}
	}

	// UnevaluatedItems (can be *Schema or bool)
	if uItems, ok := schema.UnevaluatedItems.(*parser.Schema); ok {
		if err := w.walkSchema(uItems, basePath+".unevaluatedItems", depth+1); err != nil {
			return err
		}
	}

	// Contains
	if schema.Contains != nil {
		if err := w.walkSchema(schema.Contains, basePath+".contains", depth+1); err != nil {
			return err
		}
	}

	return nil
}

// walkSchemaComposition walks allOf/anyOf/oneOf/not keywords.
func (w *Walker) walkSchemaComposition(schema *parser.Schema, basePath string, depth int) error {
	// AllOf
	for i, sub := range schema.AllOf {
		if w.stopped {
			return nil
		}
		if sub != nil {
			if err := w.walkSchema(sub, fmt.Sprintf("%s.allOf[%d]", basePath, i), depth+1); err != nil {
				return err
			}
		}
	}

	// AnyOf
	for i, sub := range schema.AnyOf {
		if w.stopped {
			return nil
		}
		if sub != nil {
			if err := w.walkSchema(sub, fmt.Sprintf("%s.anyOf[%d]", basePath, i), depth+1); err != nil {
				return err
			}
		}
	}

	// OneOf
	for i, sub := range schema.OneOf {
		if w.stopped {
			return nil
		}
		if sub != nil {
			if err := w.walkSchema(sub, fmt.Sprintf("%s.oneOf[%d]", basePath, i), depth+1); err != nil {
				return err
			}
		}
	}

	// Not
	if schema.Not != nil {
		if err := w.walkSchema(schema.Not, basePath+".not", depth+1); err != nil {
			return err
		}
	}

	return nil
}

// walkSchemaConditionals walks if/then/else keywords.
func (w *Walker) walkSchemaConditionals(schema *parser.Schema, basePath string, depth int) error {
	if schema.If != nil {
		if err := w.walkSchema(schema.If, basePath+".if", depth+1); err != nil {
			return err
		}
	}
	if schema.Then != nil {
		if err := w.walkSchema(schema.Then, basePath+".then", depth+1); err != nil {
			return err
		}
	}
	if schema.Else != nil {
		if err := w.walkSchema(schema.Else, basePath+".else", depth+1); err != nil {
			return err
		}
	}
	return nil
}

// walkSchemaMisc walks miscellaneous schema keywords.
func (w *Walker) walkSchemaMisc(schema *parser.Schema, basePath string, depth int) error {
	// ContentSchema
	if schema.ContentSchema != nil {
		if err := w.walkSchema(schema.ContentSchema, basePath+".contentSchema", depth+1); err != nil {
			return err
		}
	}

	// $defs (Defs)
	for _, name := range sortedMapKeys(schema.Defs) {
		if w.stopped {
			return nil
		}
		if def := schema.Defs[name]; def != nil {
			if err := w.walkSchema(def, basePath+".$defs['"+name+"']", depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}
