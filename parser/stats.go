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
		stats.OperationCount = countOperations(d.Paths)
		stats.SchemaCount = len(d.Definitions)
	case *OAS3Document:
		stats.PathCount = len(d.Paths)
		stats.OperationCount = countOperations(d.Paths)

		// Count webhook operations (OAS 3.1+)
		if len(d.Webhooks) > 0 {
			for _, pathItem := range d.Webhooks {
				if pathItem != nil {
					stats.OperationCount += countPathItemOperations(pathItem)
				}
			}
		}

		if d.Components != nil && d.Components.Schemas != nil {
			stats.SchemaCount = len(d.Components.Schemas)
		}
	}

	return stats
}

// countOperations counts the total number of operations in a path collection
func countOperations(paths Paths) int {
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
