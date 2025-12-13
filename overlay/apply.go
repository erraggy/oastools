package overlay

import (
	"encoding/json"
	"fmt"

	"github.com/erraggy/oastools/internal/jsonpath"
	"github.com/erraggy/oastools/parser"
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
	doc, err := deepCopy(spec.Document)
	if err != nil {
		return nil, fmt.Errorf("overlay: failed to copy document: %w", err)
	}

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
			result.Warnings = append(result.Warnings, err.Error())
			result.ActionsSkipped++
			continue
		}

		if change.MatchCount == 0 {
			if a.StrictTargets {
				return nil, fmt.Errorf("overlay: action[%d] target %q matched no nodes", i, action.Target)
			}
			result.Warnings = append(result.Warnings, fmt.Sprintf("action[%d] target %q matched no nodes", i, action.Target))
			result.ActionsSkipped++
			continue
		}

		result.Changes = append(result.Changes, *change)
		result.ActionsApplied++
	}

	return result, nil
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

// deepCopy creates a deep copy of a document using JSON marshaling.
//
// This ensures the original document is not modified during overlay application.
func deepCopy(doc any) (any, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}

	var copy any
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}

	return copy, nil
}
