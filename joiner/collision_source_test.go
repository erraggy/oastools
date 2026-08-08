package joiner

import (
	"fmt"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// threeDocOAS3 builds a document that collides with its siblings on one entry in
// every section a collision can be reported for. Each value carries the source
// name, so the side a collision reports is identifiable.
func threeDocOAS3(name string) parser.ParseResult {
	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.1.0",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/shared": &parser.PathItem{Description: name, Get: &parser.Operation{OperationID: name}},
			},
			Webhooks: map[string]*parser.PathItem{
				"Shared": {Description: name},
			},
			Components: &parser.Components{
				Schemas:    map[string]*parser.Schema{"Shared": {Type: "object", Description: name}},
				Responses:  map[string]*parser.Response{"Shared": {Description: name}},
				Parameters: map[string]*parser.Parameter{"Shared": {Name: "q", In: "query", Description: name}},
			},
			OASVersion: parser.OASVersion310,
		},
		Version: "3.1.0", OASVersion: parser.OASVersion310,
		SourcePath: name, SourceFormat: parser.SourceFormatJSON,
	}
}

// collisionSources joins a, b and c under accept-right, so each document
// replaces the last, and returns what each collision reported as its left side
// keyed by "section/incoming document".
func collisionSources(t *testing.T, opts ...Option) map[string]string {
	t.Helper()
	seen := map[string]string{}
	base := []Option{
		WithParsed(threeDocOAS3("a"), threeDocOAS3("b"), threeDocOAS3("c")),
		WithDefaultStrategy(StrategyAcceptRight),
		WithPathStrategy(StrategyAcceptRight),
		WithSchemaStrategy(StrategyAcceptRight),
		WithComponentStrategy(StrategyAcceptRight),
		WithCollisionHandler(func(c CollisionContext) (CollisionResolution, error) {
			seen[string(c.Type)+"/"+c.RightSource] = c.LeftSource
			return ContinueWithStrategy(), nil
		}),
	}
	_, err := JoinWithOptions(append(base, opts...)...)
	require.NoError(t, err)
	return seen
}

// TestCollisionReportsContributingLeftSource covers #490. Every section took its
// left source from the first document, so once a later document replaced a
// value, the next collision named a document whose value was no longer there.
func TestCollisionReportsContributingLeftSource(t *testing.T) {
	seen := collisionSources(t)

	for _, section := range []string{
		string(CollisionTypePath),
		string(CollisionTypeWebhook),
		string(CollisionTypeSchema),
		string(CollisionTypeResponse),
		string(CollisionTypeParameter),
	} {
		// b collides with a, which is still the contributor.
		assert.Equal(t, "a", seen[section+"/b"], "%s: b's collision", section)
		// c collides with b, because accept-right put b's value there.
		assert.Equal(t, "b", seen[section+"/c"], "%s: c's collision", section)
	}
}

// TestCollisionLeftValueMatchesLeftSource is the property that matters: the
// document named as the left side is the one whose value is being handed over.
func TestCollisionLeftValueMatchesLeftSource(t *testing.T) {
	var mismatches []string
	_, err := JoinWithOptions(
		WithParsed(threeDocOAS3("a"), threeDocOAS3("b"), threeDocOAS3("c")),
		WithDefaultStrategy(StrategyAcceptRight),
		WithPathStrategy(StrategyAcceptRight),
		WithSchemaStrategy(StrategyAcceptRight),
		WithComponentStrategy(StrategyAcceptRight),
		WithCollisionHandler(func(c CollisionContext) (CollisionResolution, error) {
			var describes string
			switch v := c.LeftValue.(type) {
			case *parser.Schema:
				describes = v.Description
			case *parser.Response:
				describes = v.Description
			case *parser.Parameter:
				describes = v.Description
			case *parser.PathItem:
				describes = v.Description
			default:
				return ContinueWithStrategy(), nil
			}
			if describes != c.LeftSource {
				mismatches = append(mismatches,
					string(c.Type)+": LeftSource="+c.LeftSource+" but LeftValue is from "+describes)
			}
			return ContinueWithStrategy(), nil
		}),
	)
	require.NoError(t, err)
	assert.Empty(t, mismatches)
}

// TestCollisionReportRecordsContributingSource checks the collision report, which
// reads the same source as the handler.
func TestCollisionReportRecordsContributingSource(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(threeDocOAS3("a"), threeDocOAS3("b"), threeDocOAS3("c")),
		WithDefaultStrategy(StrategyAcceptRight),
		WithPathStrategy(StrategyAcceptRight),
		WithSchemaStrategy(StrategyAcceptRight),
		WithComponentStrategy(StrategyAcceptRight),
		WithCollisionReport(true),
	)
	require.NoError(t, err)
	require.NotNil(t, res.CollisionDetails)

	lefts := map[string][]string{}
	for _, e := range res.CollisionDetails.Events {
		lefts[e.RightSource] = append(lefts[e.RightSource], e.LeftSource)
	}
	require.NotEmpty(t, lefts["c"], "c's collisions should be recorded")
	for _, left := range lefts["c"] {
		assert.Equal(t, "b", left, "c collided with b's value, not a's")
	}
}

// TestCollisionSourceSectionsAreIndependent checks the section is part of the
// key: two sections holding the same name track their contributors separately.
func TestCollisionSourceSectionsAreIndependent(t *testing.T) {
	// a contributes the schema, b contributes the response, and neither
	// document has the other's entry, so nothing collides until c arrives.
	doc := func(name string, schema, response bool) parser.ParseResult {
		components := &parser.Components{}
		if schema {
			components.Schemas = map[string]*parser.Schema{"Shared": {Type: "object", Description: name}}
		}
		if response {
			components.Responses = map[string]*parser.Response{"Shared": {Description: name}}
		}
		return parser.ParseResult{
			Document: &parser.OAS3Document{
				OpenAPI:    "3.1.0",
				Info:       &parser.Info{Title: name, Version: "1.0.0"},
				Paths:      parser.Paths{"/" + name: &parser.PathItem{Get: &parser.Operation{}}},
				Components: components,
				OASVersion: parser.OASVersion310,
			},
			Version: "3.1.0", OASVersion: parser.OASVersion310,
			SourcePath: name, SourceFormat: parser.SourceFormatJSON,
		}
	}

	seen := map[string]string{}
	_, err := JoinWithOptions(
		WithParsed(doc("a", true, false), doc("b", false, true), doc("c", true, true)),
		WithDefaultStrategy(StrategyAcceptRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithSchemaStrategy(StrategyAcceptRight),
		WithComponentStrategy(StrategyAcceptRight),
		WithCollisionHandler(func(c CollisionContext) (CollisionResolution, error) {
			seen[string(c.Type)] = c.LeftSource
			return ContinueWithStrategy(), nil
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, "a", seen[string(CollisionTypeSchema)], "the schema came from a")
	assert.Equal(t, "b", seen[string(CollisionTypeResponse)], "the response came from b")
}

// threeDocOAS2 is the OAS 2 counterpart, covering the definitions path, which
// merges separately from the component maps, and the three sections OAS 2 puts
// at the document root.
func threeDocOAS2(name string) parser.ParseResult {
	return parser.ParseResult{
		Document: &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/shared": &parser.PathItem{Description: name, Get: &parser.Operation{OperationID: name}},
			},
			Definitions: map[string]*parser.Schema{"Shared": {Type: "object", Description: name}},
			Parameters:  map[string]*parser.Parameter{"Shared": {Name: "q", In: "query", Description: name}},
			Responses:   map[string]*parser.Response{"Shared": {Description: name}},
			SecurityDefinitions: map[string]*parser.SecurityScheme{
				"Shared": {Type: "apiKey", Name: name, In: "header"},
			},
			OASVersion: parser.OASVersion20,
		},
		Version: "2.0", OASVersion: parser.OASVersion20,
		SourcePath: name, SourceFormat: parser.SourceFormatJSON,
	}
}

// TestCollisionReportsContributingLeftSourceOAS2 is the OAS 2 counterpart of
// TestCollisionReportsContributingLeftSource. Definitions merge through
// mergeOAS2Definitions rather than mergeMap, so the path is separate.
func TestCollisionReportsContributingLeftSourceOAS2(t *testing.T) {
	seen := map[string]string{}
	values := map[string]string{}
	_, err := JoinWithOptions(
		WithParsed(threeDocOAS2("a"), threeDocOAS2("b"), threeDocOAS2("c")),
		WithDefaultStrategy(StrategyAcceptRight),
		WithPathStrategy(StrategyAcceptRight),
		WithSchemaStrategy(StrategyAcceptRight),
		WithComponentStrategy(StrategyAcceptRight),
		WithCollisionHandler(func(c CollisionContext) (CollisionResolution, error) {
			key := string(c.Type) + "/" + c.RightSource
			seen[key] = c.LeftSource
			switch v := c.LeftValue.(type) {
			case *parser.Schema:
				values[key] = v.Description
			case *parser.Parameter:
				values[key] = v.Description
			case *parser.Response:
				values[key] = v.Description
			case *parser.SecurityScheme:
				values[key] = v.Name
			case *parser.PathItem:
				values[key] = v.Description
			}
			return ContinueWithStrategy(), nil
		}),
	)
	require.NoError(t, err)

	for _, section := range []string{
		string(CollisionTypeSchema),
		string(CollisionTypeParameter),
		string(CollisionTypeResponse),
		string(CollisionTypeSecurityScheme),
		string(CollisionTypePath),
	} {
		assert.Equal(t, "a", seen[section+"/b"], "%s: b's collision", section)
		assert.Equal(t, "b", seen[section+"/c"], "%s: c's collision", section)
	}

	// And the named source is the one whose value is on the left.
	for key, source := range seen {
		if got, ok := values[key]; ok {
			assert.Equal(t, source, got, "%s: LeftSource and LeftValue disagree", key)
		}
	}
}

// parseWithSourceMap parses a document with locations recorded, so a collision's
// LeftLocation can be traced back to the document it came from. Each document
// pads its Shared schema to a different line.
func parseWithSourceMap(t *testing.T, name string, pad int) parser.ParseResult {
	t.Helper()
	var filler string
	for i := range pad {
		filler += fmt.Sprintf("    Pad%d%s:\n      type: string\n", i, name)
	}
	spec := fmt.Sprintf(`openapi: 3.0.3
info:
  title: %s
  version: 1.0.0
paths: {}
components:
  schemas:
%s    Shared:
      type: object
      description: %s
`, name, filler, name)

	res, err := parser.ParseWithOptions(
		parser.WithBytes([]byte(spec)),
		parser.WithSourceMap(true),
	)
	require.NoError(t, err)
	require.NotNil(t, res.SourceMap, "the parse should record locations")
	res.SourcePath = name
	return *res
}

// TestCollisionReportsContributingLeftLocation covers the other half of the
// context: the location has to come from the same document as the source, or a
// caller following it lands in a file that no longer holds the value.
func TestCollisionReportsContributingLeftLocation(t *testing.T) {
	docs := []parser.ParseResult{
		parseWithSourceMap(t, "a", 0),
		parseWithSourceMap(t, "b", 2),
		parseWithSourceMap(t, "c", 4),
	}
	const schemaPath = "$.components.schemas.Shared"
	lineOf := map[string]int{}
	maps := map[string]*parser.SourceMap{}
	for _, d := range docs {
		lineOf[d.SourcePath] = d.SourceMap.Get(schemaPath).Line
		maps[d.SourcePath] = d.SourceMap
	}
	require.NotEqual(t, lineOf["a"], lineOf["b"], "the fixtures must differ in position")

	j := New(JoinerConfig{
		DefaultStrategy:   StrategyAcceptRight,
		PathStrategy:      StrategyAcceptRight,
		SchemaStrategy:    StrategyAcceptRight,
		ComponentStrategy: StrategyAcceptRight,
	})
	j.SourceMaps = maps

	seenLine := map[string]int{}
	seenSource := map[string]string{}
	j.collisionHandler = func(c CollisionContext) (CollisionResolution, error) {
		if c.Type == CollisionTypeSchema && c.LeftLocation != nil {
			seenLine[c.RightSource] = c.LeftLocation.Line
			seenSource[c.RightSource] = c.LeftSource
		}
		return ContinueWithStrategy(), nil
	}

	_, err := j.JoinParsed(docs)
	require.NoError(t, err)

	assert.Equal(t, lineOf["a"], seenLine["b"], "b collided with a's value")
	assert.Equal(t, lineOf["b"], seenLine["c"], "c collided with b's value")

	// The location and the source name the same document.
	assert.Equal(t, "a", seenSource["b"])
	assert.Equal(t, "b", seenSource["c"])
}

// TestWebhookHandlerOverwriteRecordsSource covers the webhook path a handler
// takes: accepting the right value has to move the contributor with it, or the
// next collision names the document that was replaced.
func TestWebhookHandlerOverwriteRecordsSource(t *testing.T) {
	seen := map[string]string{}
	_, err := JoinWithOptions(
		WithParsed(threeDocOAS3("a"), threeDocOAS3("b"), threeDocOAS3("c")),
		WithDefaultStrategy(StrategyAcceptLeft),
		WithPathStrategy(StrategyAcceptLeft),
		WithSchemaStrategy(StrategyAcceptLeft),
		WithComponentStrategy(StrategyAcceptLeft),
		// The handler, not the strategy, is what accepts the incoming webhook.
		WithCollisionHandlerFor(func(c CollisionContext) (CollisionResolution, error) {
			seen[c.RightSource] = c.LeftSource
			return AcceptRight(), nil
		}, CollisionTypeWebhook),
	)
	require.NoError(t, err)

	assert.Equal(t, "a", seen["b"], "b collided with a")
	assert.Equal(t, "b", seen["c"], "the handler accepted b, so c collides with b")
}
