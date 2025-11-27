package parser

// DocumentStats contains statistical information about an OAS document
type DocumentStats struct {
	PathCount      int // Number of paths defined
	OperationCount int // Total number of operations across all paths
	SchemaCount    int // Number of schemas/definitions
}

// GetDocumentStats returns statistics for a parsed OAS document
func GetDocumentStats(doc any) DocumentStats {
	stats := DocumentStats{}

	switch d := doc.(type) {
	case *OAS2Document:
		stats.PathCount = len(d.Paths)
		stats.OperationCount = countOAS2Operations(d.Paths)
		stats.SchemaCount = len(d.Definitions)
	case *OAS3Document:
		stats.PathCount = len(d.Paths)
		stats.OperationCount = countOAS3Operations(d.Paths)
		if d.Components != nil && d.Components.Schemas != nil {
			stats.SchemaCount = len(d.Components.Schemas)
		}
	}

	return stats
}

// countOAS2Operations counts the total number of operations in OAS 2.0 paths
func countOAS2Operations(paths Paths) int {
	count := 0
	for _, pathItem := range paths {
		if pathItem == nil {
			continue
		}
		count += countPathItemOperations(pathItem)
	}
	return count
}

// countOAS3Operations counts the total number of operations in OAS 3.x paths
func countOAS3Operations(paths Paths) int {
	count := 0
	for _, pathItem := range paths {
		if pathItem == nil {
			continue
		}
		count += countPathItemOperations(pathItem)
	}
	return count
}

// countPathItemOperations counts operations in a single PathItem
func countPathItemOperations(pathItem *PathItem) int {
	count := 0
	if pathItem.Get != nil {
		count++
	}
	if pathItem.Put != nil {
		count++
	}
	if pathItem.Post != nil {
		count++
	}
	if pathItem.Delete != nil {
		count++
	}
	if pathItem.Options != nil {
		count++
	}
	if pathItem.Head != nil {
		count++
	}
	if pathItem.Patch != nil {
		count++
	}
	if pathItem.Trace != nil {
		count++
	}
	return count
}
