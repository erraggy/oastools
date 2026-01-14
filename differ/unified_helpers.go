package differ

import (
	"fmt"
	"strconv"
	"strings"
)

// isCompatibleTypeChange checks if a type change is compatible (widening)
func isCompatibleTypeChange(oldType, newType string) bool {
	// integer -> number is a widening conversion (compatible)
	if oldType == "integer" && newType == "number" {
		return true
	}
	return false
}

// isSuccessCode checks if a status code is a success code (2xx)
func isSuccessCode(code string) bool {
	if strings.HasPrefix(code, "2") {
		return true
	}
	// Check if it's a numeric 2xx code
	if codeNum, err := strconv.Atoi(code); err == nil {
		return codeNum >= 200 && codeNum < 300
	}
	return false
}

// isErrorCode checks if a status code is an error code (4xx or 5xx)
func isErrorCode(code string) bool {
	if strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5") {
		return true
	}
	// Check if it's a numeric 4xx or 5xx code
	if codeNum, err := strconv.Atoi(code); err == nil {
		return codeNum >= 400
	}
	return false
}

// anyToString converts any value to a string representation
func anyToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprint(val)
	}
}

// severityWithRule returns severity based on diff mode and breaking rules.
// If a rule is configured for the given key, it may override the default severity.
// Returns the severity and whether the change should be ignored.
func (d *Differ) severityWithRule(defaultSeverity Severity, key RuleKey) (Severity, bool) {
	if d.Mode != ModeBreaking {
		return 0, false
	}
	rule := d.BreakingRules.getRule(key)
	return rule.ApplyRule(defaultSeverity)
}

// severityConditionalWithRule returns severity based on mode, condition, and breaking rules.
func (d *Differ) severityConditionalWithRule(condition bool, severityIfTrue, severityIfFalse Severity, key RuleKey) (Severity, bool) {
	if d.Mode != ModeBreaking {
		return 0, false
	}
	defaultSeverity := severityIfFalse
	if condition {
		defaultSeverity = severityIfTrue
	}
	rule := d.BreakingRules.getRule(key)
	return rule.ApplyRule(defaultSeverity)
}

// addChange is a helper to append a change with mode-appropriate severity.
func (d *Differ) addChange(result *DiffResult, path string, changeType ChangeType, category ChangeCategory, breakingSeverity Severity, oldValue, newValue any, message string) {
	d.addChangeWithKey(result, path, changeType, category, breakingSeverity, oldValue, newValue, message, "")
}

// addChangeWithKey is like addChange but allows specifying a subtype for rule lookup.
// The subType provides additional context (e.g., "operationId", "required") for
// fine-grained rule matching.
func (d *Differ) addChangeWithKey(result *DiffResult, path string, changeType ChangeType, category ChangeCategory, breakingSeverity Severity, oldValue, newValue any, message string, subType string) {
	key := RuleKey{Category: category, ChangeType: changeType, SubType: subType}
	sev, ignore := d.severityWithRule(breakingSeverity, key)
	if ignore {
		return
	}
	change := Change{
		Path:     path,
		Type:     changeType,
		Category: category,
		Severity: sev,
		OldValue: oldValue,
		NewValue: newValue,
		Message:  message,
	}
	d.populateChangeLocation(&change, changeType)
	result.Changes = append(result.Changes, change)
}

// addChangeConditional is a helper that picks severity based on a condition.
func (d *Differ) addChangeConditional(result *DiffResult, path string, changeType ChangeType, category ChangeCategory, condition bool, severityIfTrue, severityIfFalse Severity, oldValue, newValue any, message string) {
	d.addChangeConditionalWithKey(result, path, changeType, category, condition, severityIfTrue, severityIfFalse, oldValue, newValue, message, "")
}

// addChangeConditionalWithKey is like addChangeConditional but allows specifying a subtype.
func (d *Differ) addChangeConditionalWithKey(result *DiffResult, path string, changeType ChangeType, category ChangeCategory, condition bool, severityIfTrue, severityIfFalse Severity, oldValue, newValue any, message string, subType string) {
	key := RuleKey{Category: category, ChangeType: changeType, SubType: subType}
	sev, ignore := d.severityConditionalWithRule(condition, severityIfTrue, severityIfFalse, key)
	if ignore {
		return
	}
	change := Change{
		Path:     path,
		Type:     changeType,
		Category: category,
		Severity: sev,
		OldValue: oldValue,
		NewValue: newValue,
		Message:  message,
	}
	d.populateChangeLocation(&change, changeType)
	result.Changes = append(result.Changes, change)
}
