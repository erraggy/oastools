package generator

import (
	"sort"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// FileSplitter analyzes OpenAPI documents and determines how to split generated
// code across multiple files for large APIs.
type FileSplitter struct {
	// MaxLinesPerFile is the estimated maximum lines per file before splitting.
	// Default: 2000, 0 = no limit
	MaxLinesPerFile int

	// MaxTypesPerFile is the maximum types per file before splitting.
	// Default: 200, 0 = no limit
	MaxTypesPerFile int

	// MaxOperationsPerFile is the maximum operations per file before splitting.
	// Default: 100, 0 = no limit
	MaxOperationsPerFile int

	// SplitByTag enables splitting by operation tags.
	// Default: true
	SplitByTag bool

	// SplitByPathPrefix enables splitting by path prefix as a fallback.
	// Default: true
	SplitByPathPrefix bool
}

// NewFileSplitter creates a new FileSplitter with default settings.
func NewFileSplitter() *FileSplitter {
	return &FileSplitter{
		MaxLinesPerFile:      2000,
		MaxTypesPerFile:      200,
		MaxOperationsPerFile: 100,
		SplitByTag:           true,
		SplitByPathPrefix:    true,
	}
}

// SplitPlan represents the plan for splitting generated files.
type SplitPlan struct {
	// NeedsSplit is true if the document should be split into multiple files.
	NeedsSplit bool

	// Groups contains the file groups to generate.
	Groups []FileGroup

	// SharedTypes contains type names that are used across multiple groups
	// and should be placed in the main types.go file.
	SharedTypes []string

	// TotalOperations is the total number of operations in the document.
	TotalOperations int

	// TotalTypes is the total number of types/schemas in the document.
	TotalTypes int

	// EstimatedLines is the estimated total lines of generated code.
	EstimatedLines int
}

// FileGroup represents a group of operations and types that will be
// generated into a single file.
type FileGroup struct {
	// Name is the group name (e.g., "users", "mail").
	// Used for file naming: client_users.go, types_mail.go, etc.
	Name string

	// DisplayName is the human-readable name for documentation.
	DisplayName string

	// Operations contains the operation IDs in this group.
	Operations []string

	// Types contains the type names specific to this group.
	Types []string

	// IsShared is true if this group represents shared types (types.go).
	IsShared bool

	// EstimatedLines is the estimated lines of code for this group.
	EstimatedLines int

	// Tag is the original tag name (if grouped by tag).
	Tag string

	// PathPrefix is the path prefix (if grouped by path).
	PathPrefix string
}

// OperationInfo contains information about an operation for grouping purposes.
type OperationInfo struct {
	// OperationID is the unique operation identifier.
	OperationID string

	// Path is the URL path for the operation.
	Path string

	// Method is the HTTP method (GET, POST, etc.).
	Method string

	// Tags are the tags associated with this operation.
	Tags []string

	// ReferencedTypes contains the type names referenced by this operation.
	ReferencedTypes []string

	// EstimatedLines is the estimated lines of generated code.
	EstimatedLines int
}

// TypeInfo contains information about a type/schema for grouping purposes.
type TypeInfo struct {
	// Name is the type name.
	Name string

	// ReferencedBy contains operation IDs that reference this type.
	ReferencedBy []string

	// References contains type names this type references.
	References []string

	// EstimatedLines is the estimated lines of generated code.
	EstimatedLines int
}

// AnalyzeOAS3 analyzes an OAS 3.x document and returns a split plan.
func (fs *FileSplitter) AnalyzeOAS3(doc *parser.OAS3Document) *SplitPlan {
	plan := &SplitPlan{
		Groups:      make([]FileGroup, 0),
		SharedTypes: make([]string, 0),
	}

	// Count operations
	operations := fs.extractOAS3Operations(doc)
	plan.TotalOperations = len(operations)

	// Count types
	if doc.Components != nil && doc.Components.Schemas != nil {
		plan.TotalTypes = len(doc.Components.Schemas)
	}

	// Estimate lines (rough heuristic: 30 lines per operation, 15 lines per type)
	plan.EstimatedLines = plan.TotalOperations*30 + plan.TotalTypes*15

	// Determine if splitting is needed
	plan.NeedsSplit = fs.needsSplit(plan.TotalOperations, plan.TotalTypes, plan.EstimatedLines)

	if !plan.NeedsSplit {
		// No splitting needed - create single group
		plan.Groups = []FileGroup{{
			Name:           "",
			DisplayName:    "All",
			Operations:     fs.getOperationIDs(operations),
			Types:          fs.getSchemaNames(doc),
			IsShared:       false,
			EstimatedLines: plan.EstimatedLines,
		}}
		return plan
	}

	// Group operations and build file groups using shared helpers
	groups := fs.groupOperations(operations)
	typeUsage := fs.analyzeTypeUsage(doc, groups)
	fs.buildFileGroupsFromGroups(groups, typeUsage, plan)

	return plan
}

// AnalyzeOAS2 analyzes an OAS 2.0 document and returns a split plan.
func (fs *FileSplitter) AnalyzeOAS2(doc *parser.OAS2Document) *SplitPlan {
	plan := &SplitPlan{
		Groups:      make([]FileGroup, 0),
		SharedTypes: make([]string, 0),
	}

	// Count operations
	operations := fs.extractOAS2Operations(doc)
	plan.TotalOperations = len(operations)

	// Count types
	if doc.Definitions != nil {
		plan.TotalTypes = len(doc.Definitions)
	}

	// Estimate lines
	plan.EstimatedLines = plan.TotalOperations*30 + plan.TotalTypes*15

	// Determine if splitting is needed
	plan.NeedsSplit = fs.needsSplit(plan.TotalOperations, plan.TotalTypes, plan.EstimatedLines)

	if !plan.NeedsSplit {
		// No splitting needed - create single group
		plan.Groups = []FileGroup{{
			Name:           "",
			DisplayName:    "All",
			Operations:     fs.getOperationIDs(operations),
			Types:          fs.getOAS2SchemaNames(doc),
			IsShared:       false,
			EstimatedLines: plan.EstimatedLines,
		}}
		return plan
	}

	// Group operations and build file groups using shared helpers
	groups := fs.groupOperations(operations)
	typeUsage := fs.analyzeOAS2TypeUsage(doc, groups)
	fs.buildFileGroupsFromGroups(groups, typeUsage, plan)

	return plan
}

// needsSplit determines if the document should be split based on thresholds.
func (fs *FileSplitter) needsSplit(operations, types, lines int) bool {
	if fs.MaxOperationsPerFile > 0 && operations > fs.MaxOperationsPerFile {
		return true
	}
	if fs.MaxTypesPerFile > 0 && types > fs.MaxTypesPerFile {
		return true
	}
	if fs.MaxLinesPerFile > 0 && lines > fs.MaxLinesPerFile {
		return true
	}
	return false
}

// groupOperations groups operations using the configured strategies (tag, path prefix, alphabetical).
// This is shared between OAS 2.0 and OAS 3.x analysis.
func (fs *FileSplitter) groupOperations(operations []*OperationInfo) map[string][]*OperationInfo {
	var groups map[string][]*OperationInfo
	if fs.SplitByTag {
		groups = fs.groupByTag(operations)
	}

	// Fall back to path prefix if no tags or not enabled
	if len(groups) <= 1 && fs.SplitByPathPrefix {
		groups = fs.groupByPathPrefix(operations)
	}

	// If still only one group, fall back to alphabetical chunking
	if len(groups) <= 1 {
		groups = fs.groupAlphabetically(operations, fs.MaxOperationsPerFile)
	}

	return groups
}

// buildFileGroupsFromGroups builds file groups from grouped operations and type usage info.
// This is shared between OAS 2.0 and OAS 3.x analysis.
func (fs *FileSplitter) buildFileGroupsFromGroups(groups map[string][]*OperationInfo, typeUsage map[string]*typeUsageInfo, plan *SplitPlan) {
	// Build file groups
	sortedGroupNames := make([]string, 0, len(groups))
	for name := range groups {
		sortedGroupNames = append(sortedGroupNames, name)
	}
	sort.Strings(sortedGroupNames)

	// Track method names across all groups to avoid duplicates.
	// When two operations from different tags normalize to the same method name,
	// only the first occurrence (alphabetically by group name) gets included.
	assignedMethods := make(map[string]bool)

	for _, groupName := range sortedGroupNames {
		ops := groups[groupName]
		group := FileGroup{
			Name:        fs.sanitizeGroupName(groupName),
			DisplayName: groupName,
			Operations:  make([]string, 0, len(ops)),
			Types:       make([]string, 0),
			Tag:         groupName,
		}

		for _, op := range ops {
			// Store the transformed Go method name to match code generator expectations
			methodName := operationInfoToMethodName(op)
			// Skip if this method name was already assigned to another group
			if assignedMethods[methodName] {
				continue
			}
			assignedMethods[methodName] = true
			group.Operations = append(group.Operations, methodName)
		}

		// Add group-specific types
		for typeName, usage := range typeUsage {
			if len(usage.groups) == 1 && usage.groups[0] == groupName {
				group.Types = append(group.Types, typeName)
			}
		}
		sort.Strings(group.Types)

		group.EstimatedLines = len(group.Operations)*30 + len(group.Types)*15
		plan.Groups = append(plan.Groups, group)
	}

	// Collect shared types (used by multiple groups)
	for typeName, usage := range typeUsage {
		if len(usage.groups) > 1 || len(usage.groups) == 0 {
			plan.SharedTypes = append(plan.SharedTypes, typeName)
		}
	}
	sort.Strings(plan.SharedTypes)
}

// extractOAS3Operations extracts operation info from an OAS 3.x document.
func (fs *FileSplitter) extractOAS3Operations(doc *parser.OAS3Document) []*OperationInfo {
	operations := make([]*OperationInfo, 0)

	if doc.Paths == nil {
		return operations
	}

	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Iterate over each HTTP method
		methodOps := map[string]*parser.Operation{
			"get":     pathItem.Get,
			"put":     pathItem.Put,
			"post":    pathItem.Post,
			"delete":  pathItem.Delete,
			"options": pathItem.Options,
			"head":    pathItem.Head,
			"patch":   pathItem.Patch,
			"trace":   pathItem.Trace,
		}

		for method, op := range methodOps {
			if op == nil {
				continue
			}

			opID := op.OperationID
			if opID == "" {
				opID = operationToMethodName(op, path, method)
			}

			info := &OperationInfo{
				OperationID:     opID,
				Path:            path,
				Method:          method,
				Tags:            op.Tags,
				ReferencedTypes: make([]string, 0),
				EstimatedLines:  30,
			}

			operations = append(operations, info)
		}
	}

	// Sort by operation ID for deterministic output
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].OperationID < operations[j].OperationID
	})

	return operations
}

// extractOAS2Operations extracts operation info from an OAS 2.0 document.
func (fs *FileSplitter) extractOAS2Operations(doc *parser.OAS2Document) []*OperationInfo {
	operations := make([]*OperationInfo, 0)

	if doc.Paths == nil {
		return operations
	}

	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Iterate over each HTTP method
		methodOps := map[string]*parser.Operation{
			"get":     pathItem.Get,
			"put":     pathItem.Put,
			"post":    pathItem.Post,
			"delete":  pathItem.Delete,
			"options": pathItem.Options,
			"head":    pathItem.Head,
			"patch":   pathItem.Patch,
		}

		for method, op := range methodOps {
			if op == nil {
				continue
			}

			opID := op.OperationID
			if opID == "" {
				opID = operationToMethodName(op, path, method)
			}

			info := &OperationInfo{
				OperationID:     opID,
				Path:            path,
				Method:          method,
				Tags:            op.Tags,
				ReferencedTypes: make([]string, 0),
				EstimatedLines:  30,
			}

			operations = append(operations, info)
		}
	}

	// Sort by operation ID for deterministic output
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].OperationID < operations[j].OperationID
	})

	return operations
}

// groupByTag groups operations by their first tag.
func (fs *FileSplitter) groupByTag(operations []*OperationInfo) map[string][]*OperationInfo {
	groups := make(map[string][]*OperationInfo)

	for _, op := range operations {
		tag := "default"
		if len(op.Tags) > 0 {
			tag = op.Tags[0]
		}

		groups[tag] = append(groups[tag], op)
	}

	return groups
}

// groupByPathPrefix groups operations by their first path segment.
func (fs *FileSplitter) groupByPathPrefix(operations []*OperationInfo) map[string][]*OperationInfo {
	groups := make(map[string][]*OperationInfo)

	for _, op := range operations {
		prefix := fs.extractPathPrefix(op.Path)
		groups[prefix] = append(groups[prefix], op)
	}

	return groups
}

// extractPathPrefix extracts the first path segment for grouping.
func (fs *FileSplitter) extractPathPrefix(path string) string {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Get first segment
	parts := strings.SplitN(path, "/", 2)
	if len(parts) > 0 && parts[0] != "" {
		// Remove any path parameters
		segment := parts[0]
		if strings.HasPrefix(segment, "{") {
			return "default"
		}
		return segment
	}

	return "default"
}

// groupAlphabetically groups operations alphabetically by operation ID.
func (fs *FileSplitter) groupAlphabetically(operations []*OperationInfo, maxPerGroup int) map[string][]*OperationInfo {
	if maxPerGroup <= 0 {
		maxPerGroup = 100
	}

	groups := make(map[string][]*OperationInfo)
	groupNum := 1
	currentGroup := make([]*OperationInfo, 0, maxPerGroup)

	for _, op := range operations {
		currentGroup = append(currentGroup, op)

		if len(currentGroup) >= maxPerGroup {
			groupName := fs.getGroupNameFromOps(currentGroup)
			groups[groupName] = currentGroup
			currentGroup = make([]*OperationInfo, 0, maxPerGroup)
			groupNum++
		}
	}

	// Don't forget the last group
	if len(currentGroup) > 0 {
		groupName := fs.getGroupNameFromOps(currentGroup)
		groups[groupName] = currentGroup
	}

	return groups
}

// getGroupNameFromOps generates a group name from the first and last operation.
func (fs *FileSplitter) getGroupNameFromOps(ops []*OperationInfo) string {
	if len(ops) == 0 {
		return "misc"
	}
	if len(ops) == 1 {
		return ops[0].OperationID
	}

	// Use first letter range (e.g., "a_m" for operations starting with a-m)
	first := strings.ToLower(ops[0].OperationID[:1])
	last := strings.ToLower(ops[len(ops)-1].OperationID[:1])

	if first == last {
		return first
	}
	return first + "_" + last
}

// typeUsageInfo tracks which groups use a type.
type typeUsageInfo struct {
	groups []string
}

// getOperationByMethod returns the operation for a given HTTP method from a PathItem.
func (fs *FileSplitter) getOperationByMethod(pathItem *parser.PathItem, method string) *parser.Operation {
	switch strings.ToLower(method) {
	case "get":
		return pathItem.Get
	case "put":
		return pathItem.Put
	case "post":
		return pathItem.Post
	case "delete":
		return pathItem.Delete
	case "options":
		return pathItem.Options
	case "head":
		return pathItem.Head
	case "patch":
		return pathItem.Patch
	case "trace":
		return pathItem.Trace
	default:
		return nil
	}
}

// analyzeTypeUsage analyzes which groups use which types.
func (fs *FileSplitter) analyzeTypeUsage(doc *parser.OAS3Document, groups map[string][]*OperationInfo) map[string]*typeUsageInfo {
	usage := make(map[string]*typeUsageInfo)

	if doc.Components == nil || doc.Components.Schemas == nil {
		return usage
	}

	// Initialize all types
	for typeName := range doc.Components.Schemas {
		usage[typeName] = &typeUsageInfo{groups: make([]string, 0)}
	}

	// For each group, find which types are used by operations in that group
	for groupName, ops := range groups {
		usedTypes := make(map[string]bool)

		for _, op := range ops {
			// Find the actual operation in the document
			if doc.Paths == nil {
				continue
			}
			pathItem, ok := doc.Paths[op.Path]
			if !ok || pathItem == nil {
				continue
			}

			opObj := fs.getOperationByMethod(pathItem, op.Method)
			if opObj == nil {
				continue
			}

			// Collect types from request body, responses, and parameters
			fs.collectOAS3OperationTypes(opObj, usedTypes)
		}

		// Mark types as used by this group
		for typeName := range usedTypes {
			if info, ok := usage[typeName]; ok {
				info.groups = append(info.groups, groupName)
			}
		}
	}

	return usage
}

// collectOAS3OperationTypes collects type references from an operation.
func (fs *FileSplitter) collectOAS3OperationTypes(op *parser.Operation, usedTypes map[string]bool) {
	// Request body
	if op.RequestBody != nil {
		if op.RequestBody.Content != nil {
			for _, mediaType := range op.RequestBody.Content {
				if mediaType.Schema != nil {
					fs.collectSchemaRefs(mediaType.Schema, usedTypes)
				}
			}
		}
	}

	// Responses
	if op.Responses != nil {
		// Check default response
		if op.Responses.Default != nil && op.Responses.Default.Content != nil {
			for _, mediaType := range op.Responses.Default.Content {
				if mediaType.Schema != nil {
					fs.collectSchemaRefs(mediaType.Schema, usedTypes)
				}
			}
		}
		// Check status code responses
		for _, resp := range op.Responses.Codes {
			if resp != nil && resp.Content != nil {
				for _, mediaType := range resp.Content {
					if mediaType.Schema != nil {
						fs.collectSchemaRefs(mediaType.Schema, usedTypes)
					}
				}
			}
		}
	}

	// Parameters
	for _, param := range op.Parameters {
		if param != nil && param.Schema != nil {
			fs.collectSchemaRefs(param.Schema, usedTypes)
		}
	}
}

// collectSchemaRefs collects schema references from a schema.
func (fs *FileSplitter) collectSchemaRefs(schema *parser.Schema, usedTypes map[string]bool) {
	if schema == nil {
		return
	}

	if schema.Ref != "" {
		typeName := fs.extractRefName(schema.Ref)
		if typeName != "" {
			usedTypes[typeName] = true
		}
	}

	// Array items (can be *Schema or bool in OAS 3.1+)
	if schema.Items != nil {
		if itemsSchema, ok := schema.Items.(*parser.Schema); ok {
			fs.collectSchemaRefs(itemsSchema, usedTypes)
		}
	}

	// Object properties
	for _, prop := range schema.Properties {
		fs.collectSchemaRefs(prop, usedTypes)
	}

	// AllOf, OneOf, AnyOf
	for _, s := range schema.AllOf {
		fs.collectSchemaRefs(s, usedTypes)
	}
	for _, s := range schema.OneOf {
		fs.collectSchemaRefs(s, usedTypes)
	}
	for _, s := range schema.AnyOf {
		fs.collectSchemaRefs(s, usedTypes)
	}

	// AdditionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		if additionalSchema, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			fs.collectSchemaRefs(additionalSchema, usedTypes)
		}
	}
}

// extractRefName extracts the type name from a $ref string.
func (fs *FileSplitter) extractRefName(ref string) string {
	// Handle OAS 3.x refs: #/components/schemas/TypeName
	if name, ok := strings.CutPrefix(ref, "#/components/schemas/"); ok {
		return name
	}
	// Handle OAS 2.0 refs: #/definitions/TypeName
	if name, ok := strings.CutPrefix(ref, "#/definitions/"); ok {
		return name
	}
	return ""
}

// analyzeOAS2TypeUsage analyzes type usage for OAS 2.0 documents.
func (fs *FileSplitter) analyzeOAS2TypeUsage(doc *parser.OAS2Document, groups map[string][]*OperationInfo) map[string]*typeUsageInfo {
	usage := make(map[string]*typeUsageInfo)

	if doc.Definitions == nil {
		return usage
	}

	// Initialize all types
	for typeName := range doc.Definitions {
		usage[typeName] = &typeUsageInfo{groups: make([]string, 0)}
	}

	// For each group, find which types are used
	for groupName, ops := range groups {
		usedTypes := make(map[string]bool)

		for _, op := range ops {
			if doc.Paths == nil {
				continue
			}
			pathItem, ok := doc.Paths[op.Path]
			if !ok || pathItem == nil {
				continue
			}

			opObj := fs.getOperationByMethod(pathItem, op.Method)
			if opObj == nil {
				continue
			}

			fs.collectOAS2OperationTypes(opObj, usedTypes)
		}

		for typeName := range usedTypes {
			if info, ok := usage[typeName]; ok {
				info.groups = append(info.groups, groupName)
			}
		}
	}

	return usage
}

// collectOAS2OperationTypes collects type references from an OAS 2.0 operation.
func (fs *FileSplitter) collectOAS2OperationTypes(op *parser.Operation, usedTypes map[string]bool) {
	// Parameters (including body)
	for _, param := range op.Parameters {
		if param != nil {
			if param.Schema != nil {
				fs.collectSchemaRefs(param.Schema, usedTypes)
			}
		}
	}

	// Responses (OAS 2.0 uses Schema directly, not Content)
	if op.Responses != nil {
		// Check default response
		if op.Responses.Default != nil && op.Responses.Default.Schema != nil {
			fs.collectSchemaRefs(op.Responses.Default.Schema, usedTypes)
		}
		// Check status code responses
		for _, resp := range op.Responses.Codes {
			if resp != nil && resp.Schema != nil {
				fs.collectSchemaRefs(resp.Schema, usedTypes)
			}
		}
	}
}

// getOperationIDs extracts operation IDs from a list of operations.
func (fs *FileSplitter) getOperationIDs(operations []*OperationInfo) []string {
	ids := make([]string, len(operations))
	for i, op := range operations {
		ids[i] = op.OperationID
	}
	return ids
}

// getSchemaNames extracts schema names from an OAS 3.x document.
func (fs *FileSplitter) getSchemaNames(doc *parser.OAS3Document) []string {
	if doc.Components == nil || doc.Components.Schemas == nil {
		return nil
	}

	names := make([]string, 0, len(doc.Components.Schemas))
	for name := range doc.Components.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// getOAS2SchemaNames extracts schema names from an OAS 2.0 document.
func (fs *FileSplitter) getOAS2SchemaNames(doc *parser.OAS2Document) []string {
	if doc.Definitions == nil {
		return nil
	}

	names := make([]string, 0, len(doc.Definitions))
	for name := range doc.Definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sanitizeGroupName converts a group name to a valid Go filename suffix.
func (fs *FileSplitter) sanitizeGroupName(name string) string {
	// Convert to lowercase snake_case
	name = strings.ToLower(name)

	// Replace spaces and hyphens with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Remove any non-alphanumeric characters except underscores
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}

	name = result.String()

	// Collapse multiple underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// Trim leading/trailing underscores
	name = strings.Trim(name, "_")

	if name == "" {
		return "misc"
	}

	return name
}

// GroupNameToTypeName converts a group name to a PascalCase type prefix.
func GroupNameToTypeName(name string) string {
	return toTypeName(name)
}
