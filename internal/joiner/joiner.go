package joiner

// Joiner handles joining of multiple OpenAPI specifications
type Joiner struct {
	// TODO: Add configuration options
}

// New creates a new Joiner instance
func New() *Joiner {
	return &Joiner{}
}

// Join joins multiple OpenAPI specifications into a single document
func (m *Joiner) Join(specPaths []string, outputPath string) error {
	// TODO: Implement join logic
	return nil
}
