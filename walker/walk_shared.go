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

// handleRef processes a $ref if ref tracking is enabled.
// It calls the ref handler if set, and returns Stop if the handler requests it.
func (w *Walker) handleRef(ref string, jsonPath string, nodeType RefNodeType, state *walkState) Action {
	if !w.trackRefs || ref == "" {
		return Continue
	}

	refInfo := &RefInfo{
		Ref:        ref,
		SourcePath: jsonPath,
		NodeType:   nodeType,
	}

	if w.onRef != nil {
		wc := state.buildContext(jsonPath)
		wc.CurrentRef = refInfo
		action := w.onRef(wc, refInfo)
		releaseContext(wc)
		if action == Stop {
			w.stopped = true
			return Stop
		}
	}

	return Continue
}

// walkParameter walks a Parameter.
func (w *Walker) walkParameter(param *parser.Parameter, basePath string, state *walkState) error {
	if param == nil {
		return nil
	}

	// Check for $ref
	if w.handleRef(param.Ref, basePath, RefNodeParameter, state) == Stop {
		return nil
	}

	continueToChildren := true
	if w.onParameter != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onParameter(wc, param))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Push parameter as parent for nested nodes
	state.pushParent(param, basePath)
	defer state.popParent()

	// Schema (OAS 3.x)
	if param.Schema != nil {
		schemaState := state.clone()
		schemaState.name = "" // Clear name for nested schemas
		if err := w.walkSchema(param.Schema, basePath+".schema", 0, schemaState); err != nil {
			return err
		}
	}

	// Content (OAS 3.x)
	if param.Content != nil {
		if err := w.walkContent(param.Content, basePath+".content", state); err != nil {
			return err
		}
	}

	// Examples
	if param.Examples != nil {
		w.walkExamples(param.Examples, basePath+".examples", state)
	}

	return nil
}

// walkHeaders walks a map of Headers.
func (w *Walker) walkHeaders(headers map[string]*parser.Header, basePath string, state *walkState) error {
	for _, name := range sortedMapKeys(headers) {
		if w.stopped {
			return nil
		}
		header := headers[name]
		if header != nil {
			if err := w.walkHeader(name, header, basePath+"['"+name+"']", state); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkHeader walks a single Header.
func (w *Walker) walkHeader(name string, header *parser.Header, basePath string, state *walkState) error {
	headerState := state.clone()
	headerState.name = name

	// Check for $ref
	if w.handleRef(header.Ref, basePath, RefNodeHeader, headerState) == Stop {
		return nil
	}

	continueToChildren := true
	if w.onHeader != nil {
		wc := headerState.buildContext(basePath)
		continueToChildren = w.handleAction(w.onHeader(wc, header))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Push header as parent for nested nodes
	headerState.pushParent(header, basePath)
	defer headerState.popParent()

	// Schema
	if header.Schema != nil {
		schemaState := headerState.clone()
		schemaState.name = "" // Clear name for nested schemas
		if err := w.walkSchema(header.Schema, basePath+".schema", 0, schemaState); err != nil {
			return err
		}
	}

	// Content
	if header.Content != nil {
		if err := w.walkContent(header.Content, basePath+".content", headerState); err != nil {
			return err
		}
	}

	// Examples
	if header.Examples != nil {
		w.walkExamples(header.Examples, basePath+".examples", headerState)
	}

	return nil
}

// walkContent walks a map of MediaTypes.
func (w *Walker) walkContent(content map[string]*parser.MediaType, basePath string, state *walkState) error {
	for _, mtName := range sortedMapKeys(content) {
		if w.stopped {
			return nil
		}
		mt := content[mtName]
		if mt != nil {
			if err := w.walkMediaType(mtName, mt, basePath+"['"+mtName+"']", state); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkMediaType walks a single MediaType.
func (w *Walker) walkMediaType(name string, mt *parser.MediaType, basePath string, state *walkState) error {
	mtState := state.clone()
	mtState.name = name

	continueToChildren := true
	if w.onMediaType != nil {
		wc := mtState.buildContext(basePath)
		continueToChildren = w.handleAction(w.onMediaType(wc, mt))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Push media type as parent for nested nodes
	mtState.pushParent(mt, basePath)
	defer mtState.popParent()

	// Schema
	if mt.Schema != nil {
		schemaState := mtState.clone()
		schemaState.name = "" // Clear name for nested schemas
		if err := w.walkSchema(mt.Schema, basePath+".schema", 0, schemaState); err != nil {
			return err
		}
	}

	// Examples
	if mt.Examples != nil {
		w.walkExamples(mt.Examples, basePath+".examples", mtState)
	}

	return nil
}

// walkExamples walks a map of Examples.
func (w *Walker) walkExamples(examples map[string]*parser.Example, basePath string, state *walkState) {
	for _, name := range sortedMapKeys(examples) {
		if w.stopped {
			return
		}
		ex := examples[name]
		if ex == nil {
			continue
		}

		exPath := basePath + "['" + name + "']"
		exState := state.clone()
		exState.name = name

		// Check for $ref
		if w.handleRef(ex.Ref, exPath, RefNodeExample, exState) == Stop {
			return
		}

		if w.onExample != nil {
			wc := exState.buildContext(exPath)
			w.handleAction(w.onExample(wc, ex))
			releaseContext(wc)
		}
	}
}

// walkSchema walks a Schema and all its nested schemas.
func (w *Walker) walkSchema(schema *parser.Schema, basePath string, depth int, state *walkState) error {
	if schema == nil {
		return nil
	}

	// Check for $ref before anything else
	if w.handleRef(schema.Ref, basePath, RefNodeSchema, state) == Stop {
		return nil
	}

	// Check depth limit
	if depth > w.maxDepth {
		if w.onSchemaSkipped != nil {
			wc := state.buildContext(basePath)
			w.onSchemaSkipped(wc, "depth", schema)
			releaseContext(wc)
		}
		return nil
	}

	// Check for cycle
	if w.visitedSchemas[schema] {
		if w.onSchemaSkipped != nil {
			wc := state.buildContext(basePath)
			w.onSchemaSkipped(wc, "cycle", schema)
			releaseContext(wc)
		}
		return nil
	}

	w.visitedSchemas[schema] = true
	defer delete(w.visitedSchemas, schema)

	// Call pre-visit handler
	if w.onSchema != nil {
		wc := state.buildContext(basePath)
		continueToChildren := w.handleAction(w.onSchema(wc, schema))
		releaseContext(wc)
		if !continueToChildren {
			if w.stopped {
				return nil
			}
			return nil // SkipChildren - don't call post handler
		}
	}

	// Push schema as parent for nested schemas
	state.pushParent(schema, basePath)
	defer state.popParent()

	// Walk nested schemas in groups - clear name for nested schemas
	nestedState := state.clone()
	nestedState.name = ""

	if err := w.walkSchemaProperties(schema, basePath, depth, nestedState); err != nil {
		return err
	}
	if err := w.walkSchemaArrayKeywords(schema, basePath, depth, nestedState); err != nil {
		return err
	}
	if err := w.walkSchemaComposition(schema, basePath, depth, nestedState); err != nil {
		return err
	}
	if err := w.walkSchemaConditionals(schema, basePath, depth, nestedState); err != nil {
		return err
	}
	if err := w.walkSchemaMisc(schema, basePath, depth, nestedState); err != nil {
		return err
	}

	// Call post-visit handler after children (but before popParent)
	if w.onSchemaPost != nil && !w.stopped {
		wc := state.buildContext(basePath)
		w.onSchemaPost(wc, schema)
		releaseContext(wc)
	}

	return nil
}

// walkSchemaProperties walks object-related schema keywords.
func (w *Walker) walkSchemaProperties(schema *parser.Schema, basePath string, depth int, state *walkState) error {
	// Properties
	for _, name := range sortedMapKeys(schema.Properties) {
		if w.stopped {
			return nil
		}
		if prop := schema.Properties[name]; prop != nil {
			propState := state.clone()
			propState.name = name
			if err := w.walkSchema(prop, basePath+".properties['"+name+"']", depth+1, propState); err != nil {
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
			if err := w.walkSchema(prop, basePath+".patternProperties['"+pattern+"']", depth+1, state); err != nil {
				return err
			}
		}
	}

	// AdditionalProperties (can be *Schema, bool, or map[string]any (which may contain a $ref key))
	switch addProps := schema.AdditionalProperties.(type) {
	case *parser.Schema:
		if err := w.walkSchema(addProps, basePath+".additionalProperties", depth+1, state); err != nil {
			return err
		}
	case map[string]any:
		if w.trackMapRefs {
			if ref, ok := addProps["$ref"].(string); ok && ref != "" {
				if w.handleRef(ref, basePath+".additionalProperties", RefNodeSchema, state) == Stop {
					return nil
				}
			}
		}
	}

	// UnevaluatedProperties (can be *Schema, bool, or map[string]any (which may contain a $ref key))
	switch uProps := schema.UnevaluatedProperties.(type) {
	case *parser.Schema:
		if err := w.walkSchema(uProps, basePath+".unevaluatedProperties", depth+1, state); err != nil {
			return err
		}
	case map[string]any:
		if w.trackMapRefs {
			if ref, ok := uProps["$ref"].(string); ok && ref != "" {
				if w.handleRef(ref, basePath+".unevaluatedProperties", RefNodeSchema, state) == Stop {
					return nil
				}
			}
		}
	}

	// PropertyNames
	if schema.PropertyNames != nil {
		if err := w.walkSchema(schema.PropertyNames, basePath+".propertyNames", depth+1, state); err != nil {
			return err
		}
	}

	// DependentSchemas
	for _, name := range sortedMapKeys(schema.DependentSchemas) {
		if w.stopped {
			return nil
		}
		if ds := schema.DependentSchemas[name]; ds != nil {
			if err := w.walkSchema(ds, basePath+".dependentSchemas['"+name+"']", depth+1, state); err != nil {
				return err
			}
		}
	}

	return nil
}

// walkSchemaArrayKeywords walks array-related schema keywords.
func (w *Walker) walkSchemaArrayKeywords(schema *parser.Schema, basePath string, depth int, state *walkState) error {
	// Items (can be *Schema, bool, or map[string]any (which may contain a $ref key))
	switch items := schema.Items.(type) {
	case *parser.Schema:
		if err := w.walkSchema(items, basePath+".items", depth+1, state); err != nil {
			return err
		}
	case map[string]any:
		if w.trackMapRefs {
			if ref, ok := items["$ref"].(string); ok && ref != "" {
				if w.handleRef(ref, basePath+".items", RefNodeSchema, state) == Stop {
					return nil
				}
			}
		}
	}

	// AdditionalItems (can be *Schema, bool, or map[string]any (which may contain a $ref key))
	switch addItems := schema.AdditionalItems.(type) {
	case *parser.Schema:
		if err := w.walkSchema(addItems, basePath+".additionalItems", depth+1, state); err != nil {
			return err
		}
	case map[string]any:
		if w.trackMapRefs {
			if ref, ok := addItems["$ref"].(string); ok && ref != "" {
				if w.handleRef(ref, basePath+".additionalItems", RefNodeSchema, state) == Stop {
					return nil
				}
			}
		}
	}

	// PrefixItems (OAS 3.1+)
	for i, prefixItem := range schema.PrefixItems {
		if w.stopped {
			return nil
		}
		if prefixItem != nil {
			if err := w.walkSchema(prefixItem, fmt.Sprintf("%s.prefixItems[%d]", basePath, i), depth+1, state); err != nil {
				return err
			}
		}
	}

	// UnevaluatedItems (can be *Schema, bool, or map[string]any (which may contain a $ref key))
	switch uItems := schema.UnevaluatedItems.(type) {
	case *parser.Schema:
		if err := w.walkSchema(uItems, basePath+".unevaluatedItems", depth+1, state); err != nil {
			return err
		}
	case map[string]any:
		if w.trackMapRefs {
			if ref, ok := uItems["$ref"].(string); ok && ref != "" {
				if w.handleRef(ref, basePath+".unevaluatedItems", RefNodeSchema, state) == Stop {
					return nil
				}
			}
		}
	}

	// Contains
	if schema.Contains != nil {
		if err := w.walkSchema(schema.Contains, basePath+".contains", depth+1, state); err != nil {
			return err
		}
	}

	return nil
}

// walkSchemaComposition walks allOf/anyOf/oneOf/not keywords.
func (w *Walker) walkSchemaComposition(schema *parser.Schema, basePath string, depth int, state *walkState) error {
	// AllOf
	for i, sub := range schema.AllOf {
		if w.stopped {
			return nil
		}
		if sub != nil {
			if err := w.walkSchema(sub, fmt.Sprintf("%s.allOf[%d]", basePath, i), depth+1, state); err != nil {
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
			if err := w.walkSchema(sub, fmt.Sprintf("%s.anyOf[%d]", basePath, i), depth+1, state); err != nil {
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
			if err := w.walkSchema(sub, fmt.Sprintf("%s.oneOf[%d]", basePath, i), depth+1, state); err != nil {
				return err
			}
		}
	}

	// Not
	if schema.Not != nil {
		if err := w.walkSchema(schema.Not, basePath+".not", depth+1, state); err != nil {
			return err
		}
	}

	return nil
}

// walkSchemaConditionals walks if/then/else keywords.
func (w *Walker) walkSchemaConditionals(schema *parser.Schema, basePath string, depth int, state *walkState) error {
	if schema.If != nil {
		if err := w.walkSchema(schema.If, basePath+".if", depth+1, state); err != nil {
			return err
		}
	}
	if schema.Then != nil {
		if err := w.walkSchema(schema.Then, basePath+".then", depth+1, state); err != nil {
			return err
		}
	}
	if schema.Else != nil {
		if err := w.walkSchema(schema.Else, basePath+".else", depth+1, state); err != nil {
			return err
		}
	}
	return nil
}

// walkSchemaMisc walks miscellaneous schema keywords.
func (w *Walker) walkSchemaMisc(schema *parser.Schema, basePath string, depth int, state *walkState) error {
	// ContentSchema
	if schema.ContentSchema != nil {
		if err := w.walkSchema(schema.ContentSchema, basePath+".contentSchema", depth+1, state); err != nil {
			return err
		}
	}

	// $defs (Defs)
	for _, name := range sortedMapKeys(schema.Defs) {
		if w.stopped {
			return nil
		}
		if def := schema.Defs[name]; def != nil {
			defState := state.clone()
			defState.name = name
			if err := w.walkSchema(def, basePath+".$defs['"+name+"']", depth+1, defState); err != nil {
				return err
			}
		}
	}

	return nil
}
