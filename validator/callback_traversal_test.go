package validator

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/erraggy/oastools/parser"
)

// callbackCarrier is a parameter holding one violation for each walk below:
// `example` beside `examples` for the Example rules, and an enum disagreeing with
// its type for the schema rules.
func callbackCarrier() []*parser.Parameter {
	return []*parser.Parameter{{
		Name:     "q",
		In:       "query",
		Example:  "one",
		Examples: map[string]*parser.Example{"other": {Value: "two"}},
		Schema:   &parser.Schema{Type: "string", Enum: []any{1}},
	}}
}

// callbackChain returns the head of a chain of `links` distinct path items, each
// reaching the next through a callback on its `get`. Nothing repeats, so the
// visited set never fires and only the depth bound stops the walk.
func callbackChain(links int) *parser.PathItem {
	head := &parser.PathItem{Parameters: callbackCarrier()}
	current := head
	for range links - 1 {
		next := &parser.PathItem{Parameters: callbackCarrier()}
		callback := parser.Callback{"next": next}
		current.Get = &parser.Operation{
			Callbacks: map[string]*parser.Callback{"chain": &callback},
		}
		current = next
	}
	return head
}

// cyclicCallbackItem returns a path item whose `cycling` operations each lead
// back to it through a callback, so the graph branches at every step.
//
// Hand-built because a parsed document cannot close that loop: `$ref` resolution
// and YAML aliases both copy rather than share, so no two positions hold the same
// Path Item. ValidateParsed takes the caller's.
func cyclicCallbackItem(cycling int) *parser.PathItem {
	item := &parser.PathItem{Parameters: callbackCarrier()}
	callback := parser.Callback{"loop": item}
	cycle := map[string]*parser.Callback{"c": &callback}

	setters := []func(*parser.Operation){
		func(op *parser.Operation) { item.Get = op },
		func(op *parser.Operation) { item.Post = op },
		func(op *parser.Operation) { item.Put = op },
	}
	for _, set := range setters[:cycling] {
		set(&parser.Operation{Callbacks: cycle})
	}
	return item
}

// walkFromPathItem runs one of the two callback walks over a path item and
// returns how many errors it reported.
var walkFromPathItem = map[string]func(*parser.PathItem) int{
	// The Example and `querystring` traversal.
	"traversePathItem": func(item *parser.PathItem) int {
		result := &ValidationResult{}
		New().validateOAS3TraversalPathItem(item, "paths./a", parser.OASVersion310,
			parser.GetOperations(item, parser.OASVersion310), result)
		return len(result.Errors)
	},
	// The schema traversal, entered at an operation rather than a path item.
	"validatePathItemSchemas": func(item *parser.PathItem) int {
		result := &ValidationResult{}
		New().validateOAS3OperationSchemas(item.Get, "paths./a.get", result)
		return len(result.Errors)
	},
}

// TestCallbackWalksTerminateOnACycle pins the visited set both callback walks
// carry. A Callback Object holds path items whose operations hold callbacks, so
// the two can point at each other, and a depth bound alone does not contain that:
// a path item with more than one operation leading back to it branches, so the
// walk goes exponential in depth long before the bound is reached. Removing
// either set hangs the corresponding case rather than failing it.
func TestCallbackWalksTerminateOnACycle(t *testing.T) {
	for name, walk := range walkFromPathItem {
		t.Run(name, func(t *testing.T) {
			for _, cycling := range []int{1, 2, 3} {
				item := cyclicCallbackItem(cycling)
				var got int
				runsWithin(t, 15*time.Second, "fan-out "+strconv.Itoa(cycling),
					func() { got = walk(item) })
				assert.Equal(t, 1, got,
					"fan-out %d: the path item should be walked once however many operations lead back to it", cycling)
			}
		})
	}
}

// TestCallbackWalksStopAtTheDepthBound pins maxCallbackNestingDepth, which the
// visited set does not subsume: a chain of distinct path items repeats nothing,
// so only the counter stops it.
func TestCallbackWalksStopAtTheDepthBound(t *testing.T) {
	const links = maxCallbackNestingDepth + 150

	for name, walk := range walkFromPathItem {
		t.Run(name, func(t *testing.T) {
			// The head sits at depth 0, so the bound admits one more link than it
			// names, and each link carries one violation.
			assert.Equal(t, maxCallbackNestingDepth+1, walk(callbackChain(links)),
				"the walk should stop at the nesting bound rather than following all %d links", links)
		})
	}
}
