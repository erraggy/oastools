package overlay

import (
	"encoding/json"
	"fmt"

	"github.com/erraggy/oastools/internal/jsonpath"
	"github.com/erraggy/oastools/parser"
	"go.yaml.in/yaml/v4"
)

// Applier applies overlays to OpenAPI documents.
type Applier struct {
	// StrictTargets causes Apply to return an error if any target matches no nodes.
	StrictTargets bool
}

// NewApplier creates a new Applier with default settings.
func NewApplier() *Applier {
	return &Applier{
		StrictTargets: false,
	}
}

// Apply applies an overlay to an OpenAPI specification file.
//
// The function parses the specification, applies the overlay transformations,
// and returns the result. Actions are applied sequentially in order.
func (a *Applier) Apply(specPath, overlayPath string) (*ApplyResult, error) {
	// Parse the specification
	p := parser.New()
	parseResult, err := p.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("overlay: failed to parse specification: %w", err)
	}

	// Parse the overlay
	o, err := ParseOverlayFile(overlayPath)
	if err != nil {
		return nil, err
	}

	return a.ApplyParsed(parseResult, o)
}

// ApplyParsed applies an overlay to an already-parsed specification.
//
// This method is useful when you have already parsed the specification
// or want to apply multiple overlays to the same document.
func (a *Applier) ApplyParsed(spec *parser.ParseResult, o *Overlay) (*ApplyResult, error) {
	// Validate overlay first
	if errs := Validate(o); len(errs) > 0 {
		return nil, errs[0] // Return first validation error
	}

	// Deep copy the document to avoid modifying the original
	doc := deepCopy(spec.Document)

	result := &ApplyResult{
		Document:     doc,
		SourceFormat: spec.SourceFormat,
	}

	// Apply each action sequentially
	for i, action := range o.Actions {
		change, err := a.applyAction(doc, action, i)
		if err != nil {
			if a.StrictTargets {
				return nil, err
			}
			result.AddWarning(&ApplyWarning{
				Category:    WarnActionError,
				ActionIndex: i,
				Target:      action.Target,
				Message:     "action execution failed",
				Cause:       err,
			})
			result.ActionsSkipped++
			continue
		}

		if change.MatchCount == 0 {
			if a.StrictTargets {
				return nil, fmt.Errorf("overlay: action[%d] target %q matched no nodes", i, action.Target)
			}
			result.AddWarning(&ApplyWarning{
				Category:    WarnNoMatch,
				ActionIndex: i,
				Target:      action.Target,
				Message:     "target matched no nodes",
			})
			result.ActionsSkipped++
			continue
		}

		result.Changes = append(result.Changes, *change)
		result.ActionsApplied++
	}

	return result, nil
}

// DryRun previews overlay application without modifying the document.
//
// This method evaluates the overlay against the specification and returns
// information about what changes would be made, without actually applying them.
// Useful for previewing changes before committing to them.
func (a *Applier) DryRun(spec *parser.ParseResult, o *Overlay) (*DryRunResult, error) {
	// Validate overlay first
	if errs := Validate(o); len(errs) > 0 {
		return nil, errs[0]
	}

	// Work with a copy to avoid any side effects
	doc := deepCopy(spec.Document)

	result := &DryRunResult{}

	for i, action := range o.Actions {
		change, err := a.previewAction(doc, action, i)
		if err != nil {
			result.AddWarning(&ApplyWarning{
				Category:    WarnActionError,
				ActionIndex: i,
				Target:      action.Target,
				Message:     "action preview failed",
				Cause:       err,
			})
			result.WouldSkip++
			continue
		}

		if change.MatchCount == 0 {
			result.AddWarning(&ApplyWarning{
				Category:    WarnNoMatch,
				ActionIndex: i,
				Target:      action.Target,
				Message:     "target would match no nodes",
			})
			result.WouldSkip++
			continue
		}

		result.Changes = append(result.Changes, *change)
		result.WouldApply++
	}

	return result, nil
}

// previewAction previews what a single action would do.
func (a *Applier) previewAction(doc any, action Action, index int) (*ProposedChange, error) {
	path, err := jsonpath.Parse(action.Target)
	if err != nil {
		return nil, &ApplyError{
			ActionIndex: index,
			Target:      action.Target,
			Cause:       fmt.Errorf("invalid JSONPath: %w", err),
		}
	}

	change := &ProposedChange{
		ActionIndex: index,
		Target:      action.Target,
		Description: action.Description,
	}

	// Get matches
	matches := path.Get(doc)
	change.MatchCount = len(matches)

	// Determine operation type
	if action.Remove {
		change.Operation = "remove"
	} else if action.Update != nil {
		// Peek at first match to determine operation type
		if len(matches) > 0 {
			switch target := matches[0].(type) {
			case map[string]any:
				if _, ok := action.Update.(map[string]any); ok {
					change.Operation = "update"
				} else {
					change.Operation = "replace"
				}
			case []any:
				change.Operation = "append"
				_ = target // use variable
			default:
				change.Operation = "replace"
			}
		} else {
			change.Operation = "update" // Default for no matches
		}
	}

	return change, nil
}

// DryRunWithOptions previews overlay application using functional options.
//
// Example:
//
//	result, err := overlay.DryRunWithOptions(
//	    overlay.WithSpecFilePath("openapi.yaml"),
//	    overlay.WithOverlayFilePath("changes.yaml"),
//	)
//	for _, change := range result.Changes {
//	    fmt.Printf("Would %s %d nodes at %s\n", change.Operation, change.MatchCount, change.Target)
//	}
func DryRunWithOptions(opts ...Option) (*DryRunResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("overlay: invalid options: %w", err)
	}

	spec, o, err := loadInputs(cfg)
	if err != nil {
		return nil, err
	}

	a := &Applier{StrictTargets: cfg.strictTargets}
	return a.DryRun(spec, o)
}

// applyAction applies a single action to the document.
func (a *Applier) applyAction(doc any, action Action, index int) (*ChangeRecord, error) {
	path, err := jsonpath.Parse(action.Target)
	if err != nil {
		return nil, &ApplyError{
			ActionIndex: index,
			Target:      action.Target,
			Cause:       fmt.Errorf("invalid JSONPath: %w", err),
		}
	}

	record := &ChangeRecord{
		ActionIndex: index,
		Target:      action.Target,
	}

	// Remove takes precedence over Update (per spec)
	if action.Remove {
		record.Operation = "remove"
		matches := path.Get(doc)
		record.MatchCount = len(matches)

		if len(matches) > 0 {
			if _, err := path.Remove(doc); err != nil {
				return nil, &ApplyError{
					ActionIndex: index,
					Target:      action.Target,
					Cause:       err,
				}
			}
		}
		return record, nil
	}

	// Update operation
	if action.Update != nil {
		matches := path.Get(doc)
		record.MatchCount = len(matches)

		if len(matches) > 0 {
			err := path.Modify(doc, func(elem any) any {
				switch target := elem.(type) {
				case map[string]any:
					if update, ok := action.Update.(map[string]any); ok {
						record.Operation = "update"
						return mergeDeep(target, update)
					}
					// If update is not a map, replace the value
					record.Operation = "replace"
					return action.Update
				case []any:
					record.Operation = "append"
					return append(target, action.Update)
				default:
					// For scalar values, replace
					record.Operation = "replace"
					return action.Update
				}
			})

			if err != nil {
				return nil, &ApplyError{
					ActionIndex: index,
					Target:      action.Target,
					Cause:       err,
				}
			}
		}
		return record, nil
	}

	return record, nil
}

// mergeDeep performs a deep merge of source into target.
//
// Properties from source are recursively merged into target:
//   - Same-name properties are replaced
//   - New properties are added
//   - Nested objects are merged recursively
func mergeDeep(target, source map[string]any) map[string]any {
	for key, srcVal := range source {
		if targetVal, exists := target[key]; exists {
			targetMap, targetIsMap := targetVal.(map[string]any)
			srcMap, srcIsMap := srcVal.(map[string]any)
			if targetIsMap && srcIsMap {
				mergeDeep(targetMap, srcMap)
				continue
			}
		}
		target[key] = srcVal
	}
	return target
}

// deepCopy creates a deep copy of a document using recursive copying.
//
// This ensures the original document is not modified during overlay application.
// Unlike JSON marshal/unmarshal, this preserves exact types and float precision.
func deepCopy(doc any) any {
	return deepCopyValue(doc)
}

// deepCopyValue recursively copies a value.
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = deepCopyValue(v)
		}
		return result

	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = deepCopyValue(v)
		}
		return result

	case string, bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		// Primitives are immutable, return as-is
		return val

	// Parser types need to be converted to map[string]any for JSONPath navigation.
	// JSONPath can only traverse unstructured map types, not typed Go structs.
	case *parser.OAS3Document, *parser.OAS2Document:
		return typedDocToMap(val)

	default:
		// For unknown types, return as-is (they may be immutable or
		// the caller may not need a deep copy for this type)
		return val
	}
}

// typedDocToMap converts a typed parser document to map[string]any.
// This is necessary because JSONPath can only navigate unstructured maps.
func typedDocToMap(doc any) map[string]any {
	data, err := json.Marshal(doc)
	if err != nil {
		// Fallback: return empty map if marshal fails
		return make(map[string]any)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return make(map[string]any)
	}
	return result
}

// ParseOverlaySingle is a helper that returns an overlay from either an instance or a file path.
//
// This is useful for packages that integrate with overlay support and need to handle
// both pre-parsed overlays and overlay file paths. If the overlay instance is provided,
// it is returned directly. If only the file path is provided, the overlay is parsed from the file.
// Returns nil if neither is provided.
func ParseOverlaySingle(o *Overlay, file *string) (*Overlay, error) {
	if o != nil {
		return o, nil
	}
	if file != nil && *file != "" {
		return ParseOverlayFile(*file)
	}
	return nil, nil
}

// ReparseDocument re-parses an overlaid document to restore typed structures.
//
// After overlay application, the document becomes a map[string]any.
// Packages that need typed documents (*parser.OAS2Document or *parser.OAS3Document)
// can use this function to serialize to YAML and re-parse to restore the typed structure.
// The original ParseResult's metadata (SourcePath, SourceFormat) is preserved.
func ReparseDocument(original *parser.ParseResult, doc any) (*parser.ParseResult, error) {
	// Serialize the document to YAML
	data, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("overlay: failed to marshal document: %w", err)
	}

	// Re-parse with the parser to get typed document
	p := parser.New()
	p.ValidateStructure = true

	result, err := p.ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("overlay: failed to reparse document: %w", err)
	}

	// Preserve original metadata
	result.SourcePath = original.SourcePath
	result.SourceFormat = original.SourceFormat

	return result, nil
}
