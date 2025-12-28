package overlay

import (
	"errors"
	"testing"
)

func TestApplyWarning_String(t *testing.T) {
	tests := []struct {
		name    string
		warning *ApplyWarning
		want    string
	}{
		{
			name: "with cause",
			warning: &ApplyWarning{
				Category:    WarnActionError,
				ActionIndex: 2,
				Target:      "$.paths['/users']",
				Cause:       errors.New("invalid JSONPath"),
			},
			want: `action[2] target "$.paths['/users']": invalid JSONPath`,
		},
		{
			name: "with message",
			warning: &ApplyWarning{
				Category:    WarnNoMatch,
				ActionIndex: 0,
				Target:      "$.info.contact",
				Message:     "target matched 0 nodes",
			},
			want: `action[0] target "$.info.contact": target matched 0 nodes`,
		},
		{
			name: "category fallback",
			warning: &ApplyWarning{
				Category:    WarnNoMatch,
				ActionIndex: 5,
				Target:      "$.paths",
			},
			want: `action[5] target "$.paths": no_match`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.warning.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyWarning_HasLocation(t *testing.T) {
	tests := []struct {
		name        string
		actionIndex int
		want        bool
	}{
		{"zero index", 0, true},
		{"positive index", 5, true},
		{"negative index", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &ApplyWarning{ActionIndex: tt.actionIndex}
			if got := w.HasLocation(); got != tt.want {
				t.Errorf("HasLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyWarning_Location(t *testing.T) {
	w := &ApplyWarning{ActionIndex: 3}

	got := w.Location()
	want := "action[3]"

	if got != want {
		t.Errorf("Location() = %q, want %q", got, want)
	}
}

func TestApplyWarnings_Strings(t *testing.T) {
	warnings := ApplyWarnings{
		{ActionIndex: 0, Target: "$.a", Message: "first"},
		{ActionIndex: 1, Target: "$.b", Message: "second"},
	}

	got := warnings.Strings()

	if len(got) != 2 {
		t.Fatalf("len(Strings()) = %d, want 2", len(got))
	}
	// Verify format matches String() output
	if got[0] != warnings[0].String() {
		t.Errorf("Strings()[0] = %q, want %q", got[0], warnings[0].String())
	}
	if got[1] != warnings[1].String() {
		t.Errorf("Strings()[1] = %q, want %q", got[1], warnings[1].String())
	}
}

func TestApplyResult_AddWarning(t *testing.T) {
	result := &ApplyResult{}

	w := &ApplyWarning{
		Category:    WarnNoMatch,
		ActionIndex: 0,
		Target:      "$.info",
		Message:     "no match",
	}

	result.AddWarning(w)

	// Check StructuredWarnings
	if len(result.StructuredWarnings) != 1 {
		t.Fatalf("StructuredWarnings len = %d, want 1", len(result.StructuredWarnings))
	}
	if result.StructuredWarnings[0] != w {
		t.Error("StructuredWarnings[0] != original warning")
	}

	// Check legacy Warnings (backward compatibility)
	if len(result.Warnings) != 1 {
		t.Fatalf("Warnings len = %d, want 1", len(result.Warnings))
	}
	if result.Warnings[0] != w.String() {
		t.Errorf("Warnings[0] = %q, want %q", result.Warnings[0], w.String())
	}
}

func TestDryRunResult_AddWarning(t *testing.T) {
	result := &DryRunResult{}

	w := &ApplyWarning{
		Category:    WarnActionError,
		ActionIndex: 2,
		Target:      "$.servers",
		Cause:       errors.New("invalid path"),
	}

	result.AddWarning(w)

	// Check StructuredWarnings
	if len(result.StructuredWarnings) != 1 {
		t.Fatalf("StructuredWarnings len = %d, want 1", len(result.StructuredWarnings))
	}
	if result.StructuredWarnings[0] != w {
		t.Error("StructuredWarnings[0] != original warning")
	}

	// Check legacy Warnings (backward compatibility)
	if len(result.Warnings) != 1 {
		t.Fatalf("Warnings len = %d, want 1", len(result.Warnings))
	}
	if result.Warnings[0] != w.String() {
		t.Errorf("Warnings[0] = %q, want %q", result.Warnings[0], w.String())
	}
}

func TestWarningCategories(t *testing.T) {
	// Ensure categories have expected values for stability
	if WarnNoMatch != "no_match" {
		t.Errorf("WarnNoMatch = %q, want %q", WarnNoMatch, "no_match")
	}
	if WarnActionError != "action_error" {
		t.Errorf("WarnActionError = %q, want %q", WarnActionError, "action_error")
	}
}

func TestApplyWarning_Unwrap(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		w := &ApplyWarning{Cause: cause}

		got := w.Unwrap()
		if got == nil || got.Error() != cause.Error() {
			t.Errorf("Unwrap() = %v, want %v", got, cause)
		}
	})

	t.Run("without cause", func(t *testing.T) {
		w := &ApplyWarning{}

		if got := w.Unwrap(); got != nil {
			t.Errorf("Unwrap() = %v, want nil", got)
		}
	})

	t.Run("Unwrap enables error inspection", func(t *testing.T) {
		targetErr := errors.New("target")
		w := &ApplyWarning{Cause: targetErr}

		// ApplyWarning is not an error itself, but Unwrap() enables
		// error chain inspection when the warning is converted to an error
		got := w.Unwrap()
		if got == nil || got.Error() != targetErr.Error() {
			t.Error("Unwrap should return the wrapped error")
		}
	})
}

func TestNewNoMatchWarning(t *testing.T) {
	w := NewNoMatchWarning(3, "$.paths")

	if w.Category != WarnNoMatch {
		t.Errorf("Category = %v, want %v", w.Category, WarnNoMatch)
	}
	if w.ActionIndex != 3 {
		t.Errorf("ActionIndex = %d, want 3", w.ActionIndex)
	}
	if w.Target != "$.paths" {
		t.Errorf("Target = %q, want %q", w.Target, "$.paths")
	}
	if w.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestNewActionErrorWarning(t *testing.T) {
	cause := errors.New("invalid path")
	w := NewActionErrorWarning(5, "$.info", cause)

	if w.Category != WarnActionError {
		t.Errorf("Category = %v, want %v", w.Category, WarnActionError)
	}
	if w.ActionIndex != 5 {
		t.Errorf("ActionIndex = %d, want 5", w.ActionIndex)
	}
	if w.Cause == nil || w.Cause.Error() != cause.Error() {
		t.Error("Cause should be the provided error")
	}
}

func TestApplyWarnings_ByCategory(t *testing.T) {
	warnings := ApplyWarnings{
		{Category: WarnNoMatch, Target: "$.a"},
		{Category: WarnActionError, Target: "$.b"},
		{Category: WarnNoMatch, Target: "$.c"},
	}

	noMatchWarnings := warnings.ByCategory(WarnNoMatch)

	if len(noMatchWarnings) != 2 {
		t.Fatalf("ByCategory(WarnNoMatch) len = %d, want 2", len(noMatchWarnings))
	}
	for _, w := range noMatchWarnings {
		if w.Category != WarnNoMatch {
			t.Errorf("unexpected category %v in filtered results", w.Category)
		}
	}
}

func TestApplyWarnings_ByCategory_WithNil(t *testing.T) {
	warnings := ApplyWarnings{
		{Category: WarnNoMatch, Target: "$.a"},
		nil,
		{Category: WarnNoMatch, Target: "$.c"},
	}

	noMatchWarnings := warnings.ByCategory(WarnNoMatch)

	if len(noMatchWarnings) != 2 {
		t.Fatalf("ByCategory with nil elements should skip nil, got len = %d", len(noMatchWarnings))
	}
}
