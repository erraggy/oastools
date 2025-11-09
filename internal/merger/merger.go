package merger

// Merger handles merging of multiple OpenAPI specifications
type Merger struct {
	// TODO: Add configuration options
}

// New creates a new Merger instance
func New() *Merger {
	return &Merger{}
}

// Merge merges multiple OpenAPI specifications into a single document
func (m *Merger) Merge(specPaths []string, outputPath string) error {
	// TODO: Implement merge logic
	return nil
}
