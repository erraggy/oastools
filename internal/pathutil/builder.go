package pathutil

import (
	"strconv"
	"strings"
)

// PathBuilder provides efficient incremental path construction.
// Uses push/pop semantics to avoid allocations during traversal.
// The full string is only materialized when String() is called.
type PathBuilder struct {
	segments []string
	length   int // Pre-calculated length for String() allocation
}

// Push adds a segment to the path.
func (p *PathBuilder) Push(segment string) {
	p.segments = append(p.segments, segment)
	if len(p.segments) > 1 {
		p.length++ // For dot separator
	}
	p.length += len(segment)
}

// PushIndex adds an array index segment: "[0]", "[1]", etc.
func (p *PathBuilder) PushIndex(i int) {
	seg := "[" + strconv.Itoa(i) + "]"
	p.segments = append(p.segments, seg)
	p.length += len(seg) // No dot separator for brackets
}

// Pop removes the last segment.
func (p *PathBuilder) Pop() {
	if len(p.segments) == 0 {
		return
	}
	last := p.segments[len(p.segments)-1]
	p.segments = p.segments[:len(p.segments)-1]
	p.length -= len(last)
	// Remove dot separator if this wasn't the first segment and wasn't a bracket
	if len(p.segments) > 0 && (len(last) == 0 || last[0] != '[') {
		p.length--
	}
}

// Reset clears the builder for reuse.
func (p *PathBuilder) Reset() {
	p.segments = p.segments[:0]
	p.length = 0
}

// String materializes the full path. Only call when the path is needed.
func (p *PathBuilder) String() string {
	if len(p.segments) == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(p.length)
	b.WriteString(p.segments[0])
	for _, seg := range p.segments[1:] {
		if len(seg) > 0 && seg[0] == '[' {
			b.WriteString(seg)
		} else {
			b.WriteByte('.')
			b.WriteString(seg)
		}
	}
	return b.String()
}
