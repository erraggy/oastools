package joiner

import (
	"fmt"
	"strconv"

	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

// Option is a function that configures a join operation
type Option func(*joinConfig) error

// joinConfig holds configuration for a join operation
type joinConfig struct {
	// Input sources (variadic, requires at least 2 total)
	filePaths  []string
	parsedDocs []parser.ParseResult

	// Configuration options (nil means use default from DefaultConfig)
	defaultStrategy   *CollisionStrategy
	pathStrategy      *CollisionStrategy
	schemaStrategy    *CollisionStrategy
	componentStrategy *CollisionStrategy
	deduplicateTags   *bool
	mergeArrays       *bool

	// Advanced collision strategies configuration
	renameTemplate         *string
	namespacePrefix        map[string]string
	alwaysApplyPrefix      *bool
	equivalenceMode        *string
	collisionReport        *bool
	semanticDeduplication  *bool
	operationContext       *bool
	primaryOperationPolicy *PrimaryOperationPolicy

	// Source location tracking
	sourceMaps map[string]*parser.SourceMap // Maps file paths to their SourceMaps

	// Collision handler configuration
	collisionHandler      CollisionHandler       // Handler called on collisions (nil if not configured)
	collisionHandlerTypes map[CollisionType]bool // Which collision types invoke the handler (nil/empty means all)

	// Overlay integration options
	preJoinOverlays     []*overlay.Overlay          // Applied to all specs before joining
	preJoinOverlayFiles []string                    // File paths for pre-join overlays
	postJoinOverlay     *overlay.Overlay            // Applied to result after joining
	postJoinOverlayFile *string                     // File path for post-join overlay
	specOverlays        map[string]*overlay.Overlay // Per-spec overlays
	specOverlayFiles    map[string]string           // Per-spec overlay file paths
}

// JoinWithOptions joins multiple OpenAPI specifications using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// When overlay options are provided, the join process follows these steps:
//  1. Parse all input specifications
//  2. Apply pre-join overlays to all specs (in order specified)
//  3. Apply per-spec overlays to their respective specs
//  4. Perform the join operation
//  5. Apply post-join overlay to the merged result
//
// Example:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths("api1.yaml", "api2.yaml"),
//	    joiner.WithPathStrategy(joiner.StrategyAcceptLeft),
//	    joiner.WithPreJoinOverlayFile("normalize.yaml"),
//	    joiner.WithPostJoinOverlayFile("enhance.yaml"),
//	)
func JoinWithOptions(opts ...Option) (*JoinResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("joiner: invalid options: %w", err)
	}

	// Build JoinerConfig from options (use defaults for nil values)
	defaults := DefaultConfig()
	joinerCfg := JoinerConfig{
		DefaultStrategy:       valueOrDefault(cfg.defaultStrategy, defaults.DefaultStrategy),
		PathStrategy:          valueOrDefault(cfg.pathStrategy, defaults.PathStrategy),
		SchemaStrategy:        valueOrDefault(cfg.schemaStrategy, defaults.SchemaStrategy),
		ComponentStrategy:     valueOrDefault(cfg.componentStrategy, defaults.ComponentStrategy),
		DeduplicateTags:       boolValueOrDefault(cfg.deduplicateTags, defaults.DeduplicateTags),
		MergeArrays:           boolValueOrDefault(cfg.mergeArrays, defaults.MergeArrays),
		RenameTemplate:        stringValueOrDefault(cfg.renameTemplate, defaults.RenameTemplate),
		NamespacePrefix:       mapValueOrDefault(cfg.namespacePrefix, defaults.NamespacePrefix),
		AlwaysApplyPrefix:     boolValueOrDefault(cfg.alwaysApplyPrefix, defaults.AlwaysApplyPrefix),
		EquivalenceMode:       stringValueOrDefault(cfg.equivalenceMode, defaults.EquivalenceMode),
		CollisionReport:       boolValueOrDefault(cfg.collisionReport, defaults.CollisionReport),
		SemanticDeduplication: boolValueOrDefault(cfg.semanticDeduplication, defaults.SemanticDeduplication),
	}
	if cfg.operationContext != nil {
		joinerCfg.OperationContext = *cfg.operationContext
	}
	if cfg.primaryOperationPolicy != nil {
		joinerCfg.PrimaryOperationPolicy = *cfg.primaryOperationPolicy
	}

	j := New(joinerCfg)

	// Set SourceMaps if provided
	if cfg.sourceMaps != nil {
		j.SourceMaps = cfg.sourceMaps
	}

	// Set collision handler if provided
	if cfg.collisionHandler != nil {
		j.collisionHandler = cfg.collisionHandler
		j.collisionHandlerTypes = cfg.collisionHandlerTypes
	}

	// Check if any overlays are configured
	hasOverlays := len(cfg.preJoinOverlays) > 0 ||
		len(cfg.preJoinOverlayFiles) > 0 ||
		cfg.postJoinOverlay != nil ||
		cfg.postJoinOverlayFile != nil ||
		len(cfg.specOverlays) > 0 ||
		len(cfg.specOverlayFiles) > 0

	// Fast path: no overlays configured, use original logic
	if !hasOverlays {
		return joinWithoutOverlays(j, cfg)
	}

	// Slow path: overlays require us to parse, transform, then join
	return joinWithOverlays(j, cfg)
}

// joinWithoutOverlays handles the original join logic without overlay processing
func joinWithoutOverlays(j *Joiner, cfg *joinConfig) (*JoinResult, error) {
	// Route to appropriate join method based on input sources
	if len(cfg.filePaths) > 0 && len(cfg.parsedDocs) == 0 {
		return j.Join(cfg.filePaths)
	}
	if len(cfg.parsedDocs) > 0 && len(cfg.filePaths) == 0 {
		return j.JoinParsed(cfg.parsedDocs)
	}
	// Mixed: parse file paths and append to parsed docs
	allDocs := make([]parser.ParseResult, 0, len(cfg.parsedDocs)+len(cfg.filePaths))
	allDocs = append(allDocs, cfg.parsedDocs...)

	p := parser.New()
	for _, path := range cfg.filePaths {
		result, err := p.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("joiner: failed to parse %s: %w", path, err)
		}
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("joiner: %s has %d parse error(s)", path, len(result.Errors))
		}
		allDocs = append(allDocs, *result)
	}
	return j.JoinParsed(allDocs)
}

// joinWithOverlays handles join with overlay processing
func joinWithOverlays(j *Joiner, cfg *joinConfig) (*JoinResult, error) {
	// Step 1: Parse all overlay files
	preOverlays, err := parseOverlayList(cfg.preJoinOverlays, cfg.preJoinOverlayFiles)
	if err != nil {
		return nil, fmt.Errorf("joiner: pre-join overlay: %w", err)
	}

	postOverlay, err := overlay.ParseOverlaySingle(cfg.postJoinOverlay, cfg.postJoinOverlayFile)
	if err != nil {
		return nil, fmt.Errorf("joiner: post-join overlay: %w", err)
	}

	specOverlays, err := mergeSpecOverlays(cfg.specOverlays, cfg.specOverlayFiles)
	if err != nil {
		return nil, fmt.Errorf("joiner: spec overlay: %w", err)
	}

	// Step 2: Parse all input documents
	allDocs := make([]parser.ParseResult, 0, len(cfg.parsedDocs)+len(cfg.filePaths))
	docIdentifiers := make([]string, 0, len(cfg.parsedDocs)+len(cfg.filePaths))

	// Add pre-parsed documents
	for i, doc := range cfg.parsedDocs {
		allDocs = append(allDocs, doc)
		docIdentifiers = append(docIdentifiers, strconv.Itoa(i))
	}

	// Parse file paths
	p := parser.New()
	for _, path := range cfg.filePaths {
		result, err := p.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("joiner: failed to parse %s: %w", path, err)
		}
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("joiner: %s has %d parse error(s)", path, len(result.Errors))
		}
		allDocs = append(allDocs, *result)
		docIdentifiers = append(docIdentifiers, path)
	}

	// Step 3: Apply pre-join overlays to all documents
	// After overlay application, we must re-parse to restore typed documents
	applier := overlay.NewApplier()
	for i := range allDocs {
		needsReparse := false
		for _, o := range preOverlays {
			result, err := applier.ApplyParsed(&allDocs[i], o)
			if err != nil {
				return nil, fmt.Errorf("joiner: applying pre-join overlay to doc %d: %w", i, err)
			}
			allDocs[i].Document = result.Document
			needsReparse = true
		}

		// Check for spec-specific overlay
		if o, ok := specOverlays[docIdentifiers[i]]; ok {
			result, err := applier.ApplyParsed(&allDocs[i], o)
			if err != nil {
				return nil, fmt.Errorf("joiner: applying spec overlay to %s: %w", docIdentifiers[i], err)
			}
			allDocs[i].Document = result.Document
			needsReparse = true
		}

		// Re-parse to restore typed document if overlays were applied
		if needsReparse {
			reparsed, err := overlay.ReparseDocument(&allDocs[i], allDocs[i].Document)
			if err != nil {
				return nil, fmt.Errorf("joiner: failed to reparse doc %d after overlay: %w", i, err)
			}
			allDocs[i] = *reparsed
		}
	}

	// Step 5: Perform the join
	joinResult, err := j.JoinParsed(allDocs)
	if err != nil {
		return nil, err
	}

	// Step 6: Apply post-join overlay
	if postOverlay != nil {
		postResult, err := applier.ApplyParsed(&parser.ParseResult{
			Document:     joinResult.Document,
			SourceFormat: joinResult.SourceFormat,
		}, postOverlay)
		if err != nil {
			return nil, fmt.Errorf("joiner: applying post-join overlay: %w", err)
		}
		joinResult.Document = postResult.Document
	}

	return joinResult, nil
}

// parseOverlayList parses and combines overlay instances and files
func parseOverlayList(overlays []*overlay.Overlay, files []string) ([]*overlay.Overlay, error) {
	result := make([]*overlay.Overlay, 0, len(overlays)+len(files))
	result = append(result, overlays...)

	for _, path := range files {
		o, err := overlay.ParseOverlayFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		result = append(result, o)
	}

	return result, nil
}

// mergeSpecOverlays combines spec overlay instances and files
func mergeSpecOverlays(overlays map[string]*overlay.Overlay, files map[string]string) (map[string]*overlay.Overlay, error) {
	if len(overlays) == 0 && len(files) == 0 {
		return nil, nil
	}

	result := make(map[string]*overlay.Overlay)

	// Copy existing overlays
	for k, v := range overlays {
		result[k] = v
	}

	// Parse and add file overlays (files override instances if same key)
	for spec, path := range files {
		o, err := overlay.ParseOverlayFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse overlay for %s: %w", spec, err)
		}
		result[spec] = o
	}

	return result, nil
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*joinConfig, error) {
	cfg := &joinConfig{
		filePaths:  make([]string, 0),
		parsedDocs: make([]parser.ParseResult, 0),
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate at least 2 documents total
	totalDocs := len(cfg.filePaths) + len(cfg.parsedDocs)
	if totalDocs < 2 {
		return nil, fmt.Errorf("joiner: at least 2 documents are required for joining, got %d", totalDocs)
	}

	return cfg, nil
}

// Helper functions for option defaults
func valueOrDefault(ptr *CollisionStrategy, defaultVal CollisionStrategy) CollisionStrategy {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

func boolValueOrDefault(ptr *bool, defaultVal bool) bool {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

func stringValueOrDefault(ptr *string, defaultVal string) string {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

func mapValueOrDefault(m map[string]string, defaultVal map[string]string) map[string]string {
	if m == nil {
		return defaultVal
	}
	return m
}

// WithFilePaths specifies file paths as input sources
func WithFilePaths(paths ...string) Option {
	return func(cfg *joinConfig) error {
		cfg.filePaths = append(cfg.filePaths, paths...)
		return nil
	}
}

// WithParsed specifies parsed ParseResults as input sources
func WithParsed(docs ...parser.ParseResult) Option {
	return func(cfg *joinConfig) error {
		cfg.parsedDocs = append(cfg.parsedDocs, docs...)
		return nil
	}
}

// WithConfig applies an entire JoinerConfig struct
// This is useful for reusing existing configurations or loading from files
func WithConfig(config JoinerConfig) Option {
	return func(cfg *joinConfig) error {
		cfg.defaultStrategy = &config.DefaultStrategy
		cfg.pathStrategy = &config.PathStrategy
		cfg.schemaStrategy = &config.SchemaStrategy
		cfg.componentStrategy = &config.ComponentStrategy
		cfg.deduplicateTags = &config.DeduplicateTags
		cfg.mergeArrays = &config.MergeArrays
		cfg.renameTemplate = &config.RenameTemplate
		cfg.namespacePrefix = config.NamespacePrefix
		cfg.alwaysApplyPrefix = &config.AlwaysApplyPrefix
		cfg.equivalenceMode = &config.EquivalenceMode
		cfg.collisionReport = &config.CollisionReport
		cfg.semanticDeduplication = &config.SemanticDeduplication
		cfg.operationContext = &config.OperationContext
		cfg.primaryOperationPolicy = &config.PrimaryOperationPolicy
		return nil
	}
}

// WithDefaultStrategy sets the global collision strategy
func WithDefaultStrategy(strategy CollisionStrategy) Option {
	return func(cfg *joinConfig) error {
		cfg.defaultStrategy = &strategy
		return nil
	}
}

// WithPathStrategy sets the collision strategy for paths
func WithPathStrategy(strategy CollisionStrategy) Option {
	return func(cfg *joinConfig) error {
		cfg.pathStrategy = &strategy
		return nil
	}
}

// WithSchemaStrategy sets the collision strategy for schemas/definitions
func WithSchemaStrategy(strategy CollisionStrategy) Option {
	return func(cfg *joinConfig) error {
		cfg.schemaStrategy = &strategy
		return nil
	}
}

// WithComponentStrategy sets the collision strategy for components
func WithComponentStrategy(strategy CollisionStrategy) Option {
	return func(cfg *joinConfig) error {
		cfg.componentStrategy = &strategy
		return nil
	}
}

// WithDeduplicateTags enables or disables tag deduplication
// Default: true
func WithDeduplicateTags(enabled bool) Option {
	return func(cfg *joinConfig) error {
		cfg.deduplicateTags = &enabled
		return nil
	}
}

// WithMergeArrays enables or disables array merging (servers, security, etc.)
// Default: true
func WithMergeArrays(enabled bool) Option {
	return func(cfg *joinConfig) error {
		cfg.mergeArrays = &enabled
		return nil
	}
}

// WithRenameTemplate sets the Go template for renamed schema names
// Default: "{{.Name}}_{{.Source}}"
// Available variables: {{.Name}}, {{.Source}}, {{.Index}}, {{.Suffix}}
func WithRenameTemplate(template string) Option {
	return func(cfg *joinConfig) error {
		cfg.renameTemplate = &template
		return nil
	}
}

// WithNamespacePrefix adds a namespace prefix mapping for a source file.
// When schemas from a source file collide (or when AlwaysApplyPrefix is true),
// the prefix is applied to schema names: e.g., "User" -> "Users_User"
// Can be called multiple times to add multiple mappings.
func WithNamespacePrefix(sourcePath, prefix string) Option {
	return func(cfg *joinConfig) error {
		if cfg.namespacePrefix == nil {
			cfg.namespacePrefix = make(map[string]string)
		}
		cfg.namespacePrefix[sourcePath] = prefix
		return nil
	}
}

// WithAlwaysApplyPrefix enables or disables applying namespace prefix to all schemas,
// not just those that collide. When false (default), prefix is only applied on collision.
func WithAlwaysApplyPrefix(enabled bool) Option {
	return func(cfg *joinConfig) error {
		cfg.alwaysApplyPrefix = &enabled
		return nil
	}
}

// WithEquivalenceMode sets the schema comparison mode for deduplication
// Valid values: "none", "shallow", "deep"
// Default: "none"
func WithEquivalenceMode(mode string) Option {
	return func(cfg *joinConfig) error {
		cfg.equivalenceMode = &mode
		return nil
	}
}

// WithCollisionReport enables or disables detailed collision reporting
// Default: false
func WithCollisionReport(enabled bool) Option {
	return func(cfg *joinConfig) error {
		cfg.collisionReport = &enabled
		return nil
	}
}

// WithSemanticDeduplication enables or disables semantic schema deduplication.
// When enabled, after merging all documents, the joiner identifies semantically
// identical schemas and consolidates them to a single canonical schema.
// The canonical name is selected alphabetically (e.g., "Address" beats "Location").
// All references to duplicate schemas are rewritten to the canonical name.
// Default: false
func WithSemanticDeduplication(enabled bool) Option {
	return func(cfg *joinConfig) error {
		cfg.semanticDeduplication = &enabled
		return nil
	}
}

// WithOperationContext enables operation-aware context in rename templates.
// When enabled, the joiner builds a reference graph for each document to
// populate operation-derived fields like Path, Method, OperationID, and Tags.
// This enables templates like "{{.OperationID | pascalCase}}{{.Name}}".
//
// Limitation: Only schemas from the RIGHT (incoming) document receive operation
// context. The LEFT (base) document's schemas do not have their references traced,
// so RenameContext fields like Path, Method, OperationID, and Tags will be empty
// for base document schemas.
func WithOperationContext(enabled bool) Option {
	return func(cfg *joinConfig) error {
		cfg.operationContext = &enabled
		return nil
	}
}

// WithPrimaryOperationPolicy sets the policy for selecting the primary operation
// when a schema is referenced by multiple operations. The primary operation's
// context is used for the single-value fields (Path, Method, OperationID, Tags).
// Aggregate fields (AllPaths, AllMethods, etc.) always contain all references.
func WithPrimaryOperationPolicy(policy PrimaryOperationPolicy) Option {
	return func(cfg *joinConfig) error {
		cfg.primaryOperationPolicy = &policy
		return nil
	}
}

// WithSourceMaps provides SourceMaps for all input documents.
// The map keys should match the file paths used when parsing (e.g., ParseResult.SourcePath).
// When provided, collision errors and events include line/column information
// for both sides of the collision, enabling precise error reporting.
func WithSourceMaps(maps map[string]*parser.SourceMap) Option {
	return func(cfg *joinConfig) error {
		cfg.sourceMaps = maps
		return nil
	}
}

// WithCollisionHandler registers a handler called when collisions are detected.
// The handler receives full context and can resolve, observe, or delegate.
// If the handler returns an error, it's logged as a warning and the configured
// strategy is used instead.
//
// By default, the handler is called for all collision types. Use
// WithCollisionHandlerFor to handle specific types only.
func WithCollisionHandler(handler CollisionHandler) Option {
	return func(cfg *joinConfig) error {
		if handler == nil {
			return fmt.Errorf("collision handler cannot be nil")
		}
		cfg.collisionHandler = handler
		cfg.collisionHandlerTypes = nil // nil/empty means all types
		return nil
	}
}

// WithCollisionHandlerFor registers a handler for specific collision types only.
// Collisions of other types use the configured strategy without invoking the handler.
func WithCollisionHandlerFor(handler CollisionHandler, types ...CollisionType) Option {
	return func(cfg *joinConfig) error {
		if handler == nil {
			return fmt.Errorf("collision handler cannot be nil")
		}
		if len(types) == 0 {
			return fmt.Errorf("at least one collision type must be specified")
		}
		cfg.collisionHandler = handler
		cfg.collisionHandlerTypes = make(map[CollisionType]bool, len(types))
		for _, t := range types {
			cfg.collisionHandlerTypes[t] = true
		}
		return nil
	}
}
