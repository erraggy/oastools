package generator

import "testing"

// TestGetTemplates tests the lazy template loading functionality.
func TestGetTemplates(t *testing.T) {
	// First call should initialize templates
	tmpl, err := getTemplates()
	if err != nil {
		t.Fatalf("getTemplates() returned error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("getTemplates() returned nil template")
	}

	// Second call should return cached templates (sync.Once)
	tmpl2, err := getTemplates()
	if err != nil {
		t.Fatalf("second getTemplates() call returned error: %v", err)
	}
	if tmpl2 != tmpl {
		t.Error("expected same template instance from sync.Once")
	}

	// Verify some expected templates exist
	expectedTemplates := []string{"client.go.tmpl", "types.go.tmpl", "router.go.tmpl"}
	for _, name := range expectedTemplates {
		if tmpl.Lookup(name) == nil {
			t.Errorf("expected template %q to exist", name)
		}
	}
}
