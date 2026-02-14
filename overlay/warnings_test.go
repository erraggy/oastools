package overlay

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, tt.want, tt.warning.String())
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
			assert.Equal(t, tt.want, w.HasLocation())
		})
	}
}

func TestApplyWarning_Location(t *testing.T) {
	w := &ApplyWarning{ActionIndex: 3}

	assert.Equal(t, "action[3]", w.Location())
}

func TestApplyWarnings_Strings(t *testing.T) {
	warnings := ApplyWarnings{
		{ActionIndex: 0, Target: "$.a", Message: "first"},
		{ActionIndex: 1, Target: "$.b", Message: "second"},
	}

	got := warnings.Strings()

	require.Len(t, got, 2)
	// Verify format matches String() output
	assert.Equal(t, warnings[0].String(), got[0])
	assert.Equal(t, warnings[1].String(), got[1])
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
	require.Len(t, result.StructuredWarnings, 1)
	assert.Equal(t, w, result.StructuredWarnings[0])

	// Check legacy Warnings (backward compatibility)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, w.String(), result.Warnings[0])
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
	require.Len(t, result.StructuredWarnings, 1)
	assert.Equal(t, w, result.StructuredWarnings[0])

	// Check legacy Warnings (backward compatibility)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, w.String(), result.Warnings[0])
}

func TestWarningCategories(t *testing.T) {
	// Ensure categories have expected values for stability
	assert.Equal(t, OverlayWarningCategory("no_match"), WarnNoMatch)
	assert.Equal(t, OverlayWarningCategory("action_error"), WarnActionError)
}

func TestApplyWarning_Unwrap(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		w := &ApplyWarning{Cause: cause}

		got := w.Unwrap()
		assert.Equal(t, cause, got)
	})

	t.Run("without cause", func(t *testing.T) {
		w := &ApplyWarning{}

		assert.Nil(t, w.Unwrap())
	})

	t.Run("Unwrap enables error inspection", func(t *testing.T) {
		targetErr := errors.New("target")
		w := &ApplyWarning{Cause: targetErr}

		// ApplyWarning is not an error itself, but Unwrap() enables
		// error chain inspection when the warning is converted to an error
		got := w.Unwrap()
		require.NotNil(t, got, "Unwrap should return the wrapped error")
		assert.Equal(t, targetErr.Error(), got.Error())
	})
}

func TestNewNoMatchWarning(t *testing.T) {
	w := NewNoMatchWarning(3, "$.paths")

	assert.Equal(t, WarnNoMatch, w.Category)
	assert.Equal(t, 3, w.ActionIndex)
	assert.Equal(t, "$.paths", w.Target)
	assert.NotEmpty(t, w.Message)
}

func TestNewActionErrorWarning(t *testing.T) {
	cause := errors.New("invalid path")
	w := NewActionErrorWarning(5, "$.info", cause)

	assert.Equal(t, WarnActionError, w.Category)
	assert.Equal(t, 5, w.ActionIndex)
	assert.Equal(t, cause, w.Cause)
}

func TestApplyWarnings_ByCategory(t *testing.T) {
	warnings := ApplyWarnings{
		{Category: WarnNoMatch, Target: "$.a"},
		{Category: WarnActionError, Target: "$.b"},
		{Category: WarnNoMatch, Target: "$.c"},
	}

	noMatchWarnings := warnings.ByCategory(WarnNoMatch)

	require.Len(t, noMatchWarnings, 2)
	for _, w := range noMatchWarnings {
		assert.Equal(t, WarnNoMatch, w.Category)
	}
}

func TestApplyWarnings_ByCategory_WithNil(t *testing.T) {
	warnings := ApplyWarnings{
		{Category: WarnNoMatch, Target: "$.a"},
		nil,
		{Category: WarnNoMatch, Target: "$.c"},
	}

	noMatchWarnings := warnings.ByCategory(WarnNoMatch)

	require.Len(t, noMatchWarnings, 2, "ByCategory with nil elements should skip nil")
}
