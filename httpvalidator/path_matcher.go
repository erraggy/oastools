package httpvalidator

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// PathMatcher handles matching request paths against OpenAPI path templates.
// It converts path templates like "/pets/{petId}" into regex patterns and
// extracts parameter values from actual request paths.
type PathMatcher struct {
	// template is the original OAS path template (e.g., "/pets/{petId}")
	template string

	// regex is the compiled pattern for matching
	regex *regexp.Regexp

	// paramNames are the parameter names in order of appearance
	paramNames []string

	// specificity is used for sorting matchers (higher = more specific)
	specificity int
}

// NewPathMatcher creates a PathMatcher from an OpenAPI path template.
// The template should be in the format "/path/{param}/more/{param2}".
//
// Returns an error if the template is malformed (e.g., unclosed braces).
func NewPathMatcher(template string) (*PathMatcher, error) {
	if template == "" {
		return nil, fmt.Errorf("path template cannot be empty")
	}

	var regexBuf strings.Builder
	regexBuf.WriteString("^")

	paramNames := []string{}
	specificity := 0

	i := 0
	for i < len(template) {
		if template[i] == '{' {
			// Find closing brace
			end := strings.Index(template[i:], "}")
			if end == -1 {
				return nil, fmt.Errorf("unclosed path parameter at position %d in template %q", i, template)
			}

			paramName := template[i+1 : i+end]
			if paramName == "" {
				return nil, fmt.Errorf("empty path parameter at position %d in template %q", i, template)
			}

			// Check for duplicate parameter names
			for _, existing := range paramNames {
				if existing == paramName {
					return nil, fmt.Errorf("duplicate path parameter %q in template %q", paramName, template)
				}
			}

			paramNames = append(paramNames, paramName)

			// Use named capture group matching any non-slash characters
			// This follows RFC 3986 which says path segments are separated by /
			regexBuf.WriteString("([^/]+)")

			i += end + 1
			// Parameters reduce specificity (exact matches are more specific)
			specificity--
		} else {
			// Escape regex special characters
			c := template[i]
			if strings.ContainsRune(`\.+*?()|[]{}^$`, rune(c)) {
				regexBuf.WriteByte('\\')
			}
			regexBuf.WriteByte(c)
			i++

			// Non-parameter characters increase specificity
			if c != '/' {
				specificity++
			}
		}
	}

	regexBuf.WriteString("$")

	regex, err := regexp.Compile(regexBuf.String())
	if err != nil {
		return nil, fmt.Errorf("failed to compile path pattern for template %q: %w", template, err)
	}

	return &PathMatcher{
		template:    template,
		regex:       regex,
		paramNames:  paramNames,
		specificity: specificity,
	}, nil
}

// Match checks if the given path matches this template and extracts parameters.
// Returns true and a map of parameter names to values if the path matches.
// Returns false and nil if the path does not match.
func (pm *PathMatcher) Match(path string) (bool, map[string]string) {
	matches := pm.regex.FindStringSubmatch(path)
	if matches == nil {
		return false, nil
	}

	// First match is the full string, subsequent matches are capture groups
	if len(matches) != len(pm.paramNames)+1 {
		return false, nil
	}

	params := make(map[string]string, len(pm.paramNames))
	for i, name := range pm.paramNames {
		params[name] = matches[i+1]
	}

	return true, params
}

// Template returns the original path template.
func (pm *PathMatcher) Template() string {
	return pm.template
}

// ParamNames returns the list of parameter names in order of appearance.
func (pm *PathMatcher) ParamNames() []string {
	return pm.paramNames
}

// PathMatcherSet manages a collection of path matchers and finds the best match
// for a given request path according to OpenAPI specification precedence rules.
type PathMatcherSet struct {
	// matchers is the list of matchers sorted by specificity
	matchers []*PathMatcher
}

// NewPathMatcherSet creates a new PathMatcherSet from a list of path templates.
// The matchers are sorted by specificity so that more specific paths match first.
func NewPathMatcherSet(templates []string) (*PathMatcherSet, error) {
	matchers := make([]*PathMatcher, 0, len(templates))

	for _, template := range templates {
		matcher, err := NewPathMatcher(template)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, matcher)
	}

	// Sort by specificity (highest first), then by template length (longest first),
	// then alphabetically for stability
	sort.Slice(matchers, func(i, j int) bool {
		if matchers[i].specificity != matchers[j].specificity {
			return matchers[i].specificity > matchers[j].specificity
		}
		if len(matchers[i].template) != len(matchers[j].template) {
			return len(matchers[i].template) > len(matchers[j].template)
		}
		return matchers[i].template < matchers[j].template
	})

	return &PathMatcherSet{matchers: matchers}, nil
}

// Match finds the best matching path template for the given request path.
// Returns the matched template, extracted parameters, and whether a match was found.
//
// According to OpenAPI spec, paths are matched in order of specificity:
// 1. Exact matches before parameterized paths
// 2. More specific templates before less specific ones
// 3. Longer paths before shorter paths
func (pms *PathMatcherSet) Match(path string) (template string, params map[string]string, found bool) {
	for _, matcher := range pms.matchers {
		if matched, params := matcher.Match(path); matched {
			return matcher.template, params, true
		}
	}
	return "", nil, false
}

// Templates returns all path templates in the set.
func (pms *PathMatcherSet) Templates() []string {
	templates := make([]string, len(pms.matchers))
	for i, m := range pms.matchers {
		templates[i] = m.template
	}
	return templates
}
