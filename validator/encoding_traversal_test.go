package validator

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/erraggy/oastools/parser"
)

// cyclicEncoding returns an Encoding Object whose nested encodings all lead back
// to it, so the graph is a cycle that branches `cycling` ways at every step. Its
// header carries one violation for each walk below: `example` beside `examples`
// for the Example rules, and an enum disagreeing with its type for the schema
// rules.
//
// Hand-built because no parsed document can reach the same Encoding Object twice.
// Encoding is not a referenceable component, and the two routes that could share
// a pointer do not: `$ref` resolution and YAML aliases each produce a copy.
// ValidateParsed takes the caller's document.
func cyclicEncoding(cycling int) *parser.Encoding {
	enc := &parser.Encoding{
		Headers: map[string]*parser.Header{
			"X-Trace": {
				Example:  "one",
				Examples: map[string]*parser.Example{"other": {Value: "two"}},
				Schema:   &parser.Schema{Type: "string", Enum: []any{1}},
			},
		},
	}
	nested := make(map[string]*parser.Encoding, cycling)
	for i := range cycling {
		nested[string(rune('a'+i))] = enc
	}
	enc.Encoding = nested
	return enc
}

// encodingChain returns the head of a chain of `links` distinct Encoding Objects,
// each nesting the next. Nothing repeats, so the visited set never fires and only
// the depth bound stops the walk.
func encodingChain(links int) *parser.Encoding {
	head := &parser.Encoding{}
	current := head
	for range links - 1 {
		next := &parser.Encoding{
			Headers: map[string]*parser.Header{
				"X-Trace": {
					Example:  "one",
					Examples: map[string]*parser.Example{"other": {Value: "two"}},
					Schema:   &parser.Schema{Type: "string", Enum: []any{1}},
				},
			},
		}
		current.Encoding = map[string]*parser.Encoding{"next": next}
		current = next
	}
	return head
}

// runsWithin fails the test if walk has not returned by limit. A walk that lost
// its visited set does not fail an assertion, it stops returning, so the only
// report is this one: what names the case that hung.
func runsWithin(t *testing.T, limit time.Duration, what string, walk func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		walk()
	}()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("%s: the walk did not terminate; the visited set is gone", what)
	}
}

// gateEncodingDoc wraps a media type in a pre-3.2 document, so the version gate
// applies and its own Encoding walk runs.
func gateEncodingDoc(mt *parser.MediaType) *parser.OAS3Document {
	return &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: parser.Paths{"/a": {Get: &parser.Operation{
			RequestBody: &parser.RequestBody{
				Content: map[string]*parser.MediaType{"multipart/form-data": mt},
			},
		}}},
		OASVersion: parser.OASVersion303,
	}
}

// encodingWalks is every Encoding walk the validator carries. Nothing in the type
// system connects them, so they are listed here and each test runs all three.
var encodingWalks = []struct {
	name string
	// run walks one media type and returns how many errors it reported.
	run func(*parser.MediaType) int
	// wantCycle is the error count for a cycle, which every fan-out shares: the
	// encoding is visited once however many nested keys lead back to it.
	wantCycle int
	// wantChain is the error count for a chain longer than the bound. The two
	// example walks report on a nested encoding's own header, so the deepest
	// admitted encoding contributes nothing and the count is the bound. The gate
	// reports the nesting field on the parent instead, so it counts one more.
	wantChain int
}{
	{
		name: "visitEncodingExamples",
		run: func(mt *parser.MediaType) int {
			result := &ValidationResult{}
			New().visitMediaTypeExamples(mt, "paths./a.get.requestBody.content.x", result)
			return len(result.Errors)
		},
		// The header's `example` beside its `examples`.
		wantCycle: 1,
		wantChain: maxEncodingNestingDepth,
	},
	{
		name: "validateEncodingSchemas",
		run: func(mt *parser.MediaType) int {
			result := &ValidationResult{}
			New().validateMediaTypeSchemas(mt, "paths./a.get.requestBody.content.x", result)
			return len(result.Errors)
		},
		// The header schema's enum disagreeing with its type.
		wantCycle: 1,
		wantChain: maxEncodingNestingDepth,
	},
	{
		name: "oas32Gate.encoding",
		run: func(mt *parser.MediaType) int {
			result := &ValidationResult{}
			New().validateOAS32FieldsNotYetIntroduced(gateEncodingDoc(mt), result)
			return len(result.Errors)
		},
		// The nested `encoding` field, which 3.0.3 predates.
		wantCycle: 1,
		wantChain: maxEncodingNestingDepth + 1,
	},
}

// TestEncodingWalksTerminateOnACycle covers the three Encoding walks the
// validator carries. Each recurses through the nesting 3.2 added, and each needs
// its own visited set: a depth bound alone does not contain a cycle whose
// encoding nests more than once, because the walk branches and goes exponential
// in depth long before the bound is reached.
//
// One test over all three because nothing in the type system connects them, so a
// guard added to one is not inherited by the others. Issue #431 has the
// measurements.
func TestEncodingWalksTerminateOnACycle(t *testing.T) {
	for _, walk := range encodingWalks {
		t.Run(walk.name, func(t *testing.T) {
			for _, cycling := range []int{1, 2, 3} {
				mt := &parser.MediaType{
					Encoding: map[string]*parser.Encoding{"field": cyclicEncoding(cycling)},
				}
				var got int
				runsWithin(t, 15*time.Second, "fan-out "+strconv.Itoa(cycling),
					func() { got = walk.run(mt) })
				assert.Equal(t, walk.wantCycle, got,
					"fan-out %d: the encoding should be walked once however many nested keys lead back to it", cycling)
			}
		})
	}
}

// TestEncodingWalksStopAtTheDepthBound pins the bound, which the visited set does
// not subsume: a chain of distinct encodings repeats nothing, so only the counter
// stops it.
//
// The head sits at depth 0 and carries no header, so the bound admits one more
// encoding than it names.
func TestEncodingWalksStopAtTheDepthBound(t *testing.T) {
	const links = maxEncodingNestingDepth + 150

	for _, walk := range encodingWalks {
		t.Run(walk.name, func(t *testing.T) {
			mt := &parser.MediaType{
				Encoding: map[string]*parser.Encoding{"field": encodingChain(links)},
			}
			assert.Equal(t, walk.wantChain, walk.run(mt),
				"the walk should stop at the nesting bound rather than following all %d links", links)
		})
	}
}
