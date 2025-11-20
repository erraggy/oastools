package joiner

import "github.com/erraggy/oastools/parser"

// copyInfo creates a shallow copy of an Info object
func copyInfo(info *parser.Info) *parser.Info {
	if info == nil {
		return nil
	}
	copied := *info
	if info.Contact != nil {
		contact := *info.Contact
		copied.Contact = &contact
	}
	if info.License != nil {
		license := *info.License
		copied.License = &license
	}
	return &copied
}

// copyServers creates a copy of a servers slice
func copyServers(servers []*parser.Server) []*parser.Server {
	if servers == nil {
		return nil
	}
	result := make([]*parser.Server, len(servers))
	for i, server := range servers {
		if server != nil {
			copied := *server
			// Deep copy variables map
			if server.Variables != nil {
				copied.Variables = make(map[string]parser.ServerVariable)
				for k, v := range server.Variables {
					// Deep copy ServerVariable fields (Enum slice and Extra map)
					varCopy := parser.ServerVariable{
						Default:     v.Default,
						Description: v.Description,
					}
					if v.Enum != nil {
						varCopy.Enum = make([]string, len(v.Enum))
						copy(varCopy.Enum, v.Enum)
					}
					if v.Extra != nil {
						varCopy.Extra = make(map[string]any)
						for ek, ev := range v.Extra {
							varCopy.Extra[ek] = ev
						}
					}
					copied.Variables[k] = varCopy
				}
			}
			result[i] = &copied
		}
	}
	return result
}

// copyTags creates a copy of a tags slice
func copyTags(tags []*parser.Tag) []*parser.Tag {
	if tags == nil {
		return nil
	}
	result := make([]*parser.Tag, len(tags))
	for i, tag := range tags {
		if tag != nil {
			copied := *tag
			if tag.ExternalDocs != nil {
				docs := *tag.ExternalDocs
				copied.ExternalDocs = &docs
			}
			result[i] = &copied
		}
	}
	return result
}

// copyExternalDocs creates a copy of an ExternalDocs object
func copyExternalDocs(docs *parser.ExternalDocs) *parser.ExternalDocs {
	if docs == nil {
		return nil
	}
	copied := *docs
	return &copied
}

// copySecurityRequirements creates a copy of security requirements
func copySecurityRequirements(reqs []parser.SecurityRequirement) []parser.SecurityRequirement {
	if reqs == nil {
		return nil
	}
	result := make([]parser.SecurityRequirement, len(reqs))
	for i, req := range reqs {
		copied := make(parser.SecurityRequirement)
		for k, v := range req {
			scopes := make([]string, len(v))
			copy(scopes, v)
			copied[k] = scopes
		}
		result[i] = copied
	}
	return result
}

// copyStringSlice creates a copy of a string slice
func copyStringSlice(slice []string) []string {
	if slice == nil {
		return nil
	}
	result := make([]string, len(slice))
	copy(result, slice)
	return result
}

// mergeTags merges two tag slices, deduplicating by name if configured
func (j *Joiner) mergeTags(existing, new []*parser.Tag) []*parser.Tag {
	if !j.config.DeduplicateTags {
		return append(existing, new...)
	}

	// Build a map of existing tag names
	tagMap := make(map[string]*parser.Tag)
	for _, tag := range existing {
		if tag != nil {
			tagMap[tag.Name] = tag
		}
	}

	// Add new tags if they don't exist
	result := existing
	for _, tag := range new {
		if tag != nil {
			if _, exists := tagMap[tag.Name]; !exists {
				result = append(result, tag)
				tagMap[tag.Name] = tag
			}
		}
	}

	return result
}
