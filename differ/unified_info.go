package differ

import (
	"fmt"
	"reflect"

	"github.com/erraggy/oastools/parser"
)

// diffExtrasUnified compares Extra maps (specification extensions with x- prefix)
func (d *Differ) diffExtrasUnified(source, target map[string]any, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed extensions
	for key, sourceValue := range source {
		targetValue, exists := target[key]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, key), ChangeTypeRemoved, CategoryExtension,
				SeverityInfo, sourceValue, nil, fmt.Sprintf("extension %q removed", key))
			continue
		}

		// Check if value changed
		if !reflect.DeepEqual(sourceValue, targetValue) {
			d.addChange(result, fmt.Sprintf("%s.%s", path, key), ChangeTypeModified, CategoryExtension,
				SeverityInfo, sourceValue, targetValue, fmt.Sprintf("extension %q modified", key))
		}
	}

	// Find added extensions
	for key, targetValue := range target {
		if _, exists := source[key]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, key), ChangeTypeAdded, CategoryExtension,
				SeverityInfo, nil, targetValue, fmt.Sprintf("extension %q added", key))
		}
	}
}

// diffStringSlicesUnified compares string slices with mode-appropriate severity
func (d *Differ) diffStringSlicesUnified(source, target []string, path string, category ChangeCategory, itemName string, result *DiffResult) {
	sourceMap := make(map[string]struct{})
	for _, item := range source {
		sourceMap[item] = struct{}{}
	}

	targetMap := make(map[string]struct{})
	for _, item := range target {
		targetMap[item] = struct{}{}
	}

	// Find removed items
	for item := range sourceMap {
		if _, ok := targetMap[item]; !ok {
			d.addChange(result, path, ChangeTypeRemoved, category,
				SeverityWarning, item, nil, fmt.Sprintf("%s %q removed", itemName, item))
		}
	}

	// Find added items
	for item := range targetMap {
		if _, ok := sourceMap[item]; !ok {
			d.addChange(result, path, ChangeTypeAdded, category,
				SeverityInfo, nil, item, fmt.Sprintf("%s %q added", itemName, item))
		}
	}
}

// diffTagUnified compares individual Tag objects
func (d *Differ) diffTagUnified(source, target *parser.Tag, path string, result *DiffResult) {
	if source.Description != target.Description {
		d.addChange(result, path+".description", ChangeTypeModified, CategoryInfo,
			SeverityInfo, source.Description, target.Description, "tag description changed")
	}

	// Compare Tag extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffTagsUnified compares Tag slices
func (d *Differ) diffTagsUnified(source, target []*parser.Tag, path string, result *DiffResult) {
	sourceMap := make(map[string]*parser.Tag)
	for _, tag := range source {
		sourceMap[tag.Name] = tag
	}

	targetMap := make(map[string]*parser.Tag)
	for _, tag := range target {
		targetMap[tag.Name] = tag
	}

	// Find removed tags
	for name, sourceTag := range sourceMap {
		targetTag, exists := targetMap[name]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s[%s]", path, name), ChangeTypeRemoved, CategoryInfo,
				SeverityInfo, nil, nil, fmt.Sprintf("tag %q removed", name))
			continue
		}

		// Compare tag details
		d.diffTagUnified(sourceTag, targetTag, fmt.Sprintf("%s[%s]", path, name), result)
	}

	// Find added tags
	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s[%s]", path, name), ChangeTypeAdded, CategoryInfo,
				SeverityInfo, nil, nil, fmt.Sprintf("tag %q added", name))
		}
	}
}

// diffServerUnified compares individual Server objects
func (d *Differ) diffServerUnified(source, target *parser.Server, path string, result *DiffResult) {
	// In simple mode, always report description changes
	// In breaking mode, only report if source had a description
	if source.Description != target.Description {
		if d.Mode == ModeSimple || source.Description != "" {
			d.addChange(result, path+".description", ChangeTypeModified, CategoryServer,
				SeverityInfo, source.Description, target.Description, "server description changed")
		}
	}

	// Compare Server extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffServersUnified compares Server slices (OAS 3.x)
func (d *Differ) diffServersUnified(source, target []*parser.Server, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Build maps by URL for easier comparison
	sourceMap := make(map[string]*parser.Server)
	for _, srv := range source {
		sourceMap[srv.URL] = srv
	}

	targetMap := make(map[string]*parser.Server)
	for _, srv := range target {
		targetMap[srv.URL] = srv
	}

	// Find removed servers
	for url, sourceSrv := range sourceMap {
		targetSrv, exists := targetMap[url]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s[%s]", path, url), ChangeTypeRemoved, CategoryServer,
				SeverityWarning, url, nil, fmt.Sprintf("server %q removed", url))
			continue
		}

		// Compare server details if both exist
		d.diffServerUnified(sourceSrv, targetSrv, fmt.Sprintf("%s[%s]", path, url), result)
	}

	// Find added servers
	for url := range targetMap {
		if _, exists := sourceMap[url]; !exists {
			d.addChange(result, fmt.Sprintf("%s[%s]", path, url), ChangeTypeAdded, CategoryServer,
				SeverityInfo, nil, url, fmt.Sprintf("server %q added", url))
		}
	}
}

// diffInfoUnified compares Info objects
func (d *Differ) diffInfoUnified(source, target *parser.Info, path string, result *DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil {
		d.addChange(result, path, ChangeTypeAdded, CategoryInfo,
			SeverityInfo, nil, target, "info object added")
		return
	}

	if target == nil {
		d.addChange(result, path, ChangeTypeRemoved, CategoryInfo,
			SeverityInfo, source, nil, "info object removed")
		return
	}

	// Compare fields
	if source.Title != target.Title {
		d.addChange(result, path+".title", ChangeTypeModified, CategoryInfo,
			SeverityInfo, source.Title, target.Title, fmt.Sprintf("title changed from %q to %q", source.Title, target.Title))
	}

	if source.Version != target.Version {
		d.addChange(result, path+".version", ChangeTypeModified, CategoryInfo,
			SeverityInfo, source.Version, target.Version, fmt.Sprintf("API version changed from %q to %q", source.Version, target.Version))
	}

	// In simple mode, always report description changes
	// In breaking mode, only report if source had a description
	if source.Description != target.Description {
		if d.Mode == ModeSimple || source.Description != "" {
			d.addChange(result, path+".description", ChangeTypeModified, CategoryInfo,
				SeverityInfo, source.Description, target.Description, "description changed")
		}
	}

	// Compare Info extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffSecuritySchemeUnified compares individual SecurityScheme objects
func (d *Differ) diffSecuritySchemeUnified(source, target *parser.SecurityScheme, path string, result *DiffResult) {
	if source.Type != target.Type {
		d.addChange(result, path+".type", ChangeTypeModified, CategorySecurity,
			SeverityError, source.Type, target.Type, fmt.Sprintf("security scheme type changed from %q to %q", source.Type, target.Type))
	}

	// Compare SecurityScheme extensions
	d.diffExtrasUnified(source.Extra, target.Extra, path, result)
}

// diffSecuritySchemesUnified compares security scheme maps
func (d *Differ) diffSecuritySchemesUnified(source, target map[string]*parser.SecurityScheme, path string, result *DiffResult) {
	// Find removed schemes
	for name, sourceScheme := range source {
		targetScheme, exists := target[name]
		if !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeRemoved, CategorySecurity,
				SeverityError, nil, nil, fmt.Sprintf("security scheme %q removed", name))
			continue
		}

		// Compare security scheme details
		d.diffSecuritySchemeUnified(sourceScheme, targetScheme, fmt.Sprintf("%s.%s", path, name), result)
	}

	// Find added schemes
	for name := range target {
		if _, exists := source[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeAdded, CategorySecurity,
				SeverityWarning, nil, nil, fmt.Sprintf("security scheme %q added", name))
		}
	}
}

// diffWebhooksUnified compares webhook maps (OAS 3.1+)
func (d *Differ) diffWebhooksUnified(source, target map[string]*parser.PathItem, path string, result *DiffResult) {
	if len(source) == 0 && len(target) == 0 {
		return
	}

	// Find removed webhooks
	for name := range source {
		if _, exists := target[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeRemoved, CategoryEndpoint,
				SeverityError, nil, nil, fmt.Sprintf("webhook %q removed", name))
		}
	}

	// Find added webhooks
	for name := range target {
		if _, exists := source[name]; !exists {
			d.addChange(result, fmt.Sprintf("%s.%s", path, name), ChangeTypeAdded, CategoryEndpoint,
				SeverityInfo, nil, nil, fmt.Sprintf("webhook %q added", name))
		}
	}
}
