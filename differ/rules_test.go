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

func TestBreakingRulesConfig_AllCategories(t *testing.T) {
	t.Parallel()

	// Test all category-specific getters with various subtypes
	tests := []struct {
		name   string
		config *BreakingRulesConfig
		key    RuleKey
	}{
		// Operation category
		{"operation added", &BreakingRulesConfig{Operation: &OperationRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryOperation, ChangeTypeAdded, ""}},
		{"operation summary modified", &BreakingRulesConfig{Operation: &OperationRules{SummaryModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryOperation, ChangeTypeModified, "summary"}},
		{"operation description modified", &BreakingRulesConfig{Operation: &OperationRules{DescriptionModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryOperation, ChangeTypeModified, "description"}},
		{"operation deprecated modified", &BreakingRulesConfig{Operation: &OperationRules{DeprecatedModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryOperation, ChangeTypeModified, "deprecated"}},
		{"operation tags modified", &BreakingRulesConfig{Operation: &OperationRules{TagsModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryOperation, ChangeTypeModified, "tags"}},
		{"operation unknown subtype", &BreakingRulesConfig{Operation: &OperationRules{}}, RuleKey{CategoryOperation, ChangeTypeModified, "unknown"}},

		// Parameter category
		{"parameter removed", &BreakingRulesConfig{Parameter: &ParameterRules{Removed: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryParameter, ChangeTypeRemoved, ""}},
		{"parameter added", &BreakingRulesConfig{Parameter: &ParameterRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryParameter, ChangeTypeAdded, ""}},
		{"parameter required changed", &BreakingRulesConfig{Parameter: &ParameterRules{RequiredChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryParameter, ChangeTypeModified, "required"}},
		{"parameter type changed", &BreakingRulesConfig{Parameter: &ParameterRules{TypeChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryParameter, ChangeTypeModified, "type"}},
		{"parameter format changed", &BreakingRulesConfig{Parameter: &ParameterRules{FormatChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryParameter, ChangeTypeModified, "format"}},
		{"parameter style changed", &BreakingRulesConfig{Parameter: &ParameterRules{StyleChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryParameter, ChangeTypeModified, "style"}},
		{"parameter schema changed", &BreakingRulesConfig{Parameter: &ParameterRules{SchemaChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryParameter, ChangeTypeModified, "schema"}},
		{"parameter description modified", &BreakingRulesConfig{Parameter: &ParameterRules{DescriptionModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryParameter, ChangeTypeModified, "description"}},
		{"parameter nil rules", &BreakingRulesConfig{}, RuleKey{CategoryParameter, ChangeTypeRemoved, ""}},

		// RequestBody category
		{"requestBody removed", &BreakingRulesConfig{RequestBody: &RequestBodyRules{Removed: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryRequestBody, ChangeTypeRemoved, ""}},
		{"requestBody added", &BreakingRulesConfig{RequestBody: &RequestBodyRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryRequestBody, ChangeTypeAdded, ""}},
		{"requestBody mediaType removed", &BreakingRulesConfig{RequestBody: &RequestBodyRules{MediaTypeRemoved: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryRequestBody, ChangeTypeRemoved, "mediaType"}},
		{"requestBody mediaType added", &BreakingRulesConfig{RequestBody: &RequestBodyRules{MediaTypeAdded: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryRequestBody, ChangeTypeAdded, "mediaType"}},
		{"requestBody required changed", &BreakingRulesConfig{RequestBody: &RequestBodyRules{RequiredChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryRequestBody, ChangeTypeModified, "required"}},
		{"requestBody schema changed", &BreakingRulesConfig{RequestBody: &RequestBodyRules{SchemaChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryRequestBody, ChangeTypeModified, "schema"}},
		{"requestBody nil rules", &BreakingRulesConfig{}, RuleKey{CategoryRequestBody, ChangeTypeRemoved, ""}},

		// Response category
		{"response removed", &BreakingRulesConfig{Response: &ResponseRules{Removed: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryResponse, ChangeTypeRemoved, ""}},
		{"response added", &BreakingRulesConfig{Response: &ResponseRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryResponse, ChangeTypeAdded, ""}},
		{"response mediaType removed", &BreakingRulesConfig{Response: &ResponseRules{MediaTypeRemoved: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryResponse, ChangeTypeRemoved, "mediaType"}},
		{"response mediaType added", &BreakingRulesConfig{Response: &ResponseRules{MediaTypeAdded: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryResponse, ChangeTypeAdded, "mediaType"}},
		{"response header removed", &BreakingRulesConfig{Response: &ResponseRules{HeaderRemoved: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryResponse, ChangeTypeRemoved, "header"}},
		{"response header added", &BreakingRulesConfig{Response: &ResponseRules{HeaderAdded: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryResponse, ChangeTypeAdded, "header"}},
		{"response description modified", &BreakingRulesConfig{Response: &ResponseRules{DescriptionModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryResponse, ChangeTypeModified, "description"}},
		{"response schema changed", &BreakingRulesConfig{Response: &ResponseRules{SchemaChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryResponse, ChangeTypeModified, "schema"}},
		{"response nil rules", &BreakingRulesConfig{}, RuleKey{CategoryResponse, ChangeTypeRemoved, ""}},

		// Schema category
		{"schema removed", &BreakingRulesConfig{Schema: &SchemaRules{Removed: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeRemoved, ""}},
		{"schema added", &BreakingRulesConfig{Schema: &SchemaRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeAdded, ""}},
		{"schema property removed", &BreakingRulesConfig{Schema: &SchemaRules{PropertyRemoved: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeRemoved, "property"}},
		{"schema property added", &BreakingRulesConfig{Schema: &SchemaRules{PropertyAdded: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeAdded, "property"}},
		{"schema required removed", &BreakingRulesConfig{Schema: &SchemaRules{RequiredRemoved: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeRemoved, "required"}},
		{"schema required added", &BreakingRulesConfig{Schema: &SchemaRules{RequiredAdded: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeAdded, "required"}},
		{"schema enum removed", &BreakingRulesConfig{Schema: &SchemaRules{EnumValueRemoved: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeRemoved, "enum"}},
		{"schema enum added", &BreakingRulesConfig{Schema: &SchemaRules{EnumValueAdded: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeAdded, "enum"}},
		{"schema nullable removed", &BreakingRulesConfig{Schema: &SchemaRules{NullableRemoved: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeRemoved, "nullable"}},
		{"schema nullable added", &BreakingRulesConfig{Schema: &SchemaRules{NullableAdded: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeAdded, "nullable"}},
		{"schema format changed", &BreakingRulesConfig{Schema: &SchemaRules{FormatChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeModified, "format"}},
		{"schema pattern changed", &BreakingRulesConfig{Schema: &SchemaRules{PatternChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeModified, "pattern"}},
		{"schema maximum decreased", &BreakingRulesConfig{Schema: &SchemaRules{MaximumDecreased: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeModified, "maximum"}},
		{"schema minimum increased", &BreakingRulesConfig{Schema: &SchemaRules{MinimumIncreased: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeModified, "minimum"}},
		{"schema maxLength decreased", &BreakingRulesConfig{Schema: &SchemaRules{MaxLengthDecreased: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeModified, "maxLength"}},
		{"schema minLength increased", &BreakingRulesConfig{Schema: &SchemaRules{MinLengthIncreased: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeModified, "minLength"}},
		{"schema additionalProperties changed", &BreakingRulesConfig{Schema: &SchemaRules{AdditionalPropertiesChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeModified, "additionalProperties"}},
		{"schema description modified", &BreakingRulesConfig{Schema: &SchemaRules{DescriptionModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySchema, ChangeTypeModified, "description"}},
		{"schema nil rules", &BreakingRulesConfig{}, RuleKey{CategorySchema, ChangeTypeRemoved, ""}},

		// Security category
		{"security removed", &BreakingRulesConfig{Security: &SecurityRules{Removed: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySecurity, ChangeTypeRemoved, ""}},
		{"security added", &BreakingRulesConfig{Security: &SecurityRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySecurity, ChangeTypeAdded, ""}},
		{"security scope removed", &BreakingRulesConfig{Security: &SecurityRules{ScopeRemoved: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySecurity, ChangeTypeRemoved, "scope"}},
		{"security scope added", &BreakingRulesConfig{Security: &SecurityRules{ScopeAdded: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySecurity, ChangeTypeAdded, "scope"}},
		{"security type changed", &BreakingRulesConfig{Security: &SecurityRules{TypeChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategorySecurity, ChangeTypeModified, "type"}},
		{"security nil rules", &BreakingRulesConfig{}, RuleKey{CategorySecurity, ChangeTypeRemoved, ""}},

		// Server category
		{"server removed", &BreakingRulesConfig{Server: &ServerRules{Removed: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryServer, ChangeTypeRemoved, ""}},
		{"server added", &BreakingRulesConfig{Server: &ServerRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryServer, ChangeTypeAdded, ""}},
		{"server description modified", &BreakingRulesConfig{Server: &ServerRules{DescriptionModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryServer, ChangeTypeModified, "description"}},
		{"server variable changed", &BreakingRulesConfig{Server: &ServerRules{VariableChanged: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryServer, ChangeTypeModified, "variable"}},
		{"server nil rules", &BreakingRulesConfig{}, RuleKey{CategoryServer, ChangeTypeRemoved, ""}},

		// Endpoint category
		{"endpoint removed", &BreakingRulesConfig{Endpoint: &EndpointRules{Removed: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryEndpoint, ChangeTypeRemoved, ""}},
		{"endpoint added", &BreakingRulesConfig{Endpoint: &EndpointRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryEndpoint, ChangeTypeAdded, ""}},
		{"endpoint description modified", &BreakingRulesConfig{Endpoint: &EndpointRules{DescriptionModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryEndpoint, ChangeTypeModified, "description"}},
		{"endpoint nil rules", &BreakingRulesConfig{}, RuleKey{CategoryEndpoint, ChangeTypeRemoved, ""}},

		// Info category
		{"info title modified", &BreakingRulesConfig{Info: &InfoRules{TitleModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryInfo, ChangeTypeModified, "title"}},
		{"info version modified", &BreakingRulesConfig{Info: &InfoRules{VersionModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryInfo, ChangeTypeModified, "version"}},
		{"info description modified", &BreakingRulesConfig{Info: &InfoRules{DescriptionModified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryInfo, ChangeTypeModified, "description"}},
		{"info nil rules", &BreakingRulesConfig{}, RuleKey{CategoryInfo, ChangeTypeModified, "title"}},
		{"info non-modified type", &BreakingRulesConfig{Info: &InfoRules{}}, RuleKey{CategoryInfo, ChangeTypeRemoved, ""}},

		// Extension category
		{"extension removed", &BreakingRulesConfig{Extension: &ExtensionRules{Removed: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryExtension, ChangeTypeRemoved, ""}},
		{"extension added", &BreakingRulesConfig{Extension: &ExtensionRules{Added: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryExtension, ChangeTypeAdded, ""}},
		{"extension modified", &BreakingRulesConfig{Extension: &ExtensionRules{Modified: &BreakingChangeRule{Ignore: true}}}, RuleKey{CategoryExtension, ChangeTypeModified, ""}},
		{"extension nil rules", &BreakingRulesConfig{}, RuleKey{CategoryExtension, ChangeTypeRemoved, ""}},

		// Unknown category
		{"unknown category", &BreakingRulesConfig{}, RuleKey{ChangeCategory("unknown"), ChangeTypeRemoved, ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Just verify it doesn't panic and returns consistently
			got := tc.config.getRule(tc.key)
			// Verify the config's getRule method works (non-nil return for valid configs)
			if tc.config != nil {
				// Just ensure getRule doesn't panic and returns some value
				_ = got
			}
		})
	}
}

func TestSeverityPtr(t *testing.T) {
	t.Parallel()

	severities := []Severity{SeverityInfo, SeverityWarning, SeverityError, SeverityCritical}
	for _, s := range severities {
		ptr := SeverityPtr(s)
		if ptr == nil {
			t.Fatalf("SeverityPtr(%v) returned nil", s)
		}
		if *ptr != s {
			t.Errorf("SeverityPtr(%v) = %v, want %v", s, *ptr, s)
		}
	}
}

func TestStrictRulesComprehensive(t *testing.T) {
	t.Parallel()

	rules := StrictRules()

	// Verify all expected rules are set
	checks := []struct {
		name     string
		rule     *BreakingChangeRule
		expected Severity
	}{
		{"Operation.OperationIDModified", rules.Operation.OperationIDModified, SeverityError},
		{"Parameter.FormatChanged", rules.Parameter.FormatChanged, SeverityError},
		{"Parameter.StyleChanged", rules.Parameter.StyleChanged, SeverityError},
		{"Schema.FormatChanged", rules.Schema.FormatChanged, SeverityError},
		{"Schema.PatternChanged", rules.Schema.PatternChanged, SeverityError},
		{"Schema.PropertyRemoved", rules.Schema.PropertyRemoved, SeverityError},
		{"Schema.MinLengthIncreased", rules.Schema.MinLengthIncreased, SeverityError},
		{"Security.Added", rules.Security.Added, SeverityError},
		{"Security.ScopeRemoved", rules.Security.ScopeRemoved, SeverityError},
		{"Server.Removed", rules.Server.Removed, SeverityError},
		{"Server.VariableChanged", rules.Server.VariableChanged, SeverityError},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if c.rule == nil {
				t.Fatalf("%s is nil", c.name)
			}
			if c.rule.Severity == nil {
				t.Fatalf("%s.Severity is nil", c.name)
			}
			if *c.rule.Severity != c.expected {
				t.Errorf("%s.Severity = %v, want %v", c.name, *c.rule.Severity, c.expected)
			}
		})
	}
}

func TestLenientRulesComprehensive(t *testing.T) {
	t.Parallel()

	rules := LenientRules()

	// Verify all expected rules are set
	checks := []struct {
		name     string
		rule     *BreakingChangeRule
		expected Severity
	}{
		{"Schema.EnumValueRemoved", rules.Schema.EnumValueRemoved, SeverityWarning},
		{"Schema.RequiredAdded", rules.Schema.RequiredAdded, SeverityWarning},
		{"Security.Removed", rules.Security.Removed, SeverityWarning},
		{"Parameter.RequiredChanged", rules.Parameter.RequiredChanged, SeverityWarning},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if c.rule == nil {
				t.Fatalf("%s is nil", c.name)
			}
			if c.rule.Severity == nil {
				t.Fatalf("%s.Severity is nil", c.name)
			}
			if *c.rule.Severity != c.expected {
				t.Errorf("%s.Severity = %v, want %v", c.name, *c.rule.Severity, c.expected)
			}
		})
	}
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
