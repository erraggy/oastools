package differ

import (
	"testing"

	"github.com/erraggy/oastools/internal/severity"
	"github.com/erraggy/oastools/parser"
)

func TestBreakingRulesConfig_getRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *BreakingRulesConfig
		key      RuleKey
		wantRule *BreakingChangeRule
	}{
		{
			name:     "nil config returns nil",
			config:   nil,
			key:      RuleKey{Category: CategoryOperation, ChangeType: ChangeTypeRemoved},
			wantRule: nil,
		},
		{
			name:   "empty config returns nil",
			config: &BreakingRulesConfig{},
			key:    RuleKey{Category: CategoryOperation, ChangeType: ChangeTypeRemoved},
		},
		{
			name: "operation removed rule found",
			config: &BreakingRulesConfig{
				Operation: &OperationRules{
					Removed: &BreakingChangeRule{Severity: SeverityPtr(SeverityWarning)},
				},
			},
			key:      RuleKey{Category: CategoryOperation, ChangeType: ChangeTypeRemoved},
			wantRule: &BreakingChangeRule{Severity: SeverityPtr(SeverityWarning)},
		},
		{
			name: "operation operationId modified rule found",
			config: &BreakingRulesConfig{
				Operation: &OperationRules{
					OperationIDModified: &BreakingChangeRule{Ignore: true},
				},
			},
			key:      RuleKey{Category: CategoryOperation, ChangeType: ChangeTypeModified, SubType: "operationId"},
			wantRule: &BreakingChangeRule{Ignore: true},
		},
		{
			name: "schema type changed rule found",
			config: &BreakingRulesConfig{
				Schema: &SchemaRules{
					TypeChanged: &BreakingChangeRule{Severity: SeverityPtr(SeverityCritical)},
				},
			},
			key:      RuleKey{Category: CategorySchema, ChangeType: ChangeTypeModified, SubType: "type"},
			wantRule: &BreakingChangeRule{Severity: SeverityPtr(SeverityCritical)},
		},
		{
			name: "extension removed rule found",
			config: &BreakingRulesConfig{
				Extension: &ExtensionRules{
					Removed: &BreakingChangeRule{Ignore: true},
				},
			},
			key:      RuleKey{Category: CategoryExtension, ChangeType: ChangeTypeRemoved},
			wantRule: &BreakingChangeRule{Ignore: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.config.getRule(tc.key)
			if tc.wantRule == nil {
				if got != nil {
					t.Errorf("expected nil rule, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected rule, got nil")
			}
			if tc.wantRule.Ignore != got.Ignore {
				t.Errorf("Ignore: want %v, got %v", tc.wantRule.Ignore, got.Ignore)
			}
			if tc.wantRule.Severity != nil && got.Severity != nil {
				if *tc.wantRule.Severity != *got.Severity {
					t.Errorf("Severity: want %v, got %v", *tc.wantRule.Severity, *got.Severity)
				}
			}
		})
	}
}

func TestBreakingChangeRule_ApplyRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		rule            *BreakingChangeRule
		defaultSeverity Severity
		wantSeverity    Severity
		wantIgnore      bool
	}{
		{
			name:            "nil rule uses default",
			rule:            nil,
			defaultSeverity: SeverityError,
			wantSeverity:    SeverityError,
			wantIgnore:      false,
		},
		{
			name:            "ignore rule returns ignore",
			rule:            &BreakingChangeRule{Ignore: true},
			defaultSeverity: SeverityError,
			wantSeverity:    0,
			wantIgnore:      true,
		},
		{
			name:            "severity override",
			rule:            &BreakingChangeRule{Severity: SeverityPtr(SeverityInfo)},
			defaultSeverity: SeverityError,
			wantSeverity:    SeverityInfo,
			wantIgnore:      false,
		},
		{
			name:            "no override uses default",
			rule:            &BreakingChangeRule{},
			defaultSeverity: SeverityWarning,
			wantSeverity:    SeverityWarning,
			wantIgnore:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotSeverity, gotIgnore := tc.rule.ApplyRule(tc.defaultSeverity)
			if gotSeverity != tc.wantSeverity {
				t.Errorf("Severity: want %v, got %v", tc.wantSeverity, gotSeverity)
			}
			if gotIgnore != tc.wantIgnore {
				t.Errorf("Ignore: want %v, got %v", tc.wantIgnore, gotIgnore)
			}
		})
	}
}

func TestDiff_WithBreakingRules(t *testing.T) {
	t.Parallel()

	// Create source and target specs with operationId change
	source := createTestSpec("getUsers")
	target := createTestSpec("listUsers")

	tests := []struct {
		name         string
		rules        *BreakingRulesConfig
		wantChanges  int
		wantSeverity Severity
	}{
		{
			name:         "no rules - default severity",
			rules:        nil,
			wantChanges:  1,
			wantSeverity: SeverityWarning, // Default for operationId change
		},
		{
			name: "downgrade operationId change to info",
			rules: &BreakingRulesConfig{
				Operation: &OperationRules{
					OperationIDModified: &BreakingChangeRule{Severity: SeverityPtr(SeverityInfo)},
				},
			},
			wantChanges:  1,
			wantSeverity: SeverityInfo,
		},
		{
			name: "upgrade operationId change to error",
			rules: &BreakingRulesConfig{
				Operation: &OperationRules{
					OperationIDModified: &BreakingChangeRule{Severity: SeverityPtr(SeverityError)},
				},
			},
			wantChanges:  1,
			wantSeverity: SeverityError,
		},
		{
			name: "ignore operationId changes",
			rules: &BreakingRulesConfig{
				Operation: &OperationRules{
					OperationIDModified: &BreakingChangeRule{Ignore: true},
				},
			},
			wantChanges: 0, // Change should be filtered out
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := New()
			d.Mode = ModeBreaking
			d.BreakingRules = tc.rules

			result, err := d.DiffParsed(source, target)
			if err != nil {
				t.Fatalf("DiffParsed failed: %v", err)
			}

			// Filter to only operationId changes
			var opIdChanges []Change
			for _, c := range result.Changes {
				if c.Category == CategoryOperation && c.Message == `operationId changed from "getUsers" to "listUsers"` {
					opIdChanges = append(opIdChanges, c)
				}
			}

			if len(opIdChanges) != tc.wantChanges {
				t.Errorf("want %d operationId changes, got %d", tc.wantChanges, len(opIdChanges))
			}

			if tc.wantChanges > 0 && len(opIdChanges) > 0 {
				if opIdChanges[0].Severity != tc.wantSeverity {
					t.Errorf("want severity %v, got %v", tc.wantSeverity, opIdChanges[0].Severity)
				}
			}
		})
	}
}

func TestDiffWithOptions_BreakingRules(t *testing.T) {
	t.Parallel()

	source := createTestSpec("getUsers")
	target := createTestSpec("listUsers")

	result, err := DiffWithOptions(
		WithSourceParsed(source),
		WithTargetParsed(target),
		WithMode(ModeBreaking),
		WithBreakingRules(&BreakingRulesConfig{
			Operation: &OperationRules{
				OperationIDModified: &BreakingChangeRule{Severity: SeverityPtr(SeverityInfo)},
			},
		}),
	)
	if err != nil {
		t.Fatalf("DiffWithOptions failed: %v", err)
	}

	// Find the operationId change
	var found bool
	for _, c := range result.Changes {
		if c.Category == CategoryOperation && c.Message == `operationId changed from "getUsers" to "listUsers"` {
			found = true
			if c.Severity != SeverityInfo {
				t.Errorf("want severity Info, got %v", c.Severity)
			}
		}
	}
	if !found {
		t.Error("operationId change not found in results")
	}
}

func TestPresetRules(t *testing.T) {
	t.Parallel()

	t.Run("DefaultRules returns empty config", func(t *testing.T) {
		t.Parallel()
		rules := DefaultRules()
		if rules == nil {
			t.Fatal("DefaultRules returned nil")
		}
		// Default rules should have all nil fields
		if rules.Operation != nil {
			t.Error("expected nil Operation rules")
		}
	})

	t.Run("StrictRules elevates warnings", func(t *testing.T) {
		t.Parallel()
		rules := StrictRules()
		if rules == nil || rules.Operation == nil {
			t.Fatal("StrictRules returned incomplete config")
		}
		if rules.Operation.OperationIDModified == nil {
			t.Fatal("OperationIDModified rule not set")
		}
		if *rules.Operation.OperationIDModified.Severity != severity.SeverityError {
			t.Errorf("expected Error severity, got %v", *rules.Operation.OperationIDModified.Severity)
		}
	})

	t.Run("LenientRules downgrades errors", func(t *testing.T) {
		t.Parallel()
		rules := LenientRules()
		if rules == nil || rules.Schema == nil {
			t.Fatal("LenientRules returned incomplete config")
		}
		if rules.Schema.EnumValueRemoved == nil {
			t.Fatal("EnumValueRemoved rule not set")
		}
		if *rules.Schema.EnumValueRemoved.Severity != severity.SeverityWarning {
			t.Errorf("expected Warning severity, got %v", *rules.Schema.EnumValueRemoved.Severity)
		}
	})
}

// createTestSpec creates a minimal OAS 3.0 spec for testing.
func createTestSpec(operationID string) parser.ParseResult {
	return parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info: &parser.Info{
				Title:   "Test API",
				Version: "v1",
			},
			Paths: map[string]*parser.PathItem{
				"/users": {
					Get: &parser.Operation{
						OperationID: operationID,
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"200": {Description: "OK"},
							},
						},
					},
				},
			},
		},
	}
}
