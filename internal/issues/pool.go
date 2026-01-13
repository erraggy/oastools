package issues

import (
	"strings"
	"sync"
)

var stringBuilderPool = sync.Pool{
	New: func() any {
		return new(strings.Builder)
	},
}

// getStringBuilder retrieves a builder from the pool and resets it.
func getStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// putStringBuilder returns a builder to the pool.
func putStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}
	stringBuilderPool.Put(sb)
}

// FormatPath efficiently formats a JSON path from segments.
func FormatPath(segments ...string) string {
	if len(segments) == 0 {
		return ""
	}
	if len(segments) == 1 {
		return segments[0]
	}

	sb := getStringBuilder()
	for i, seg := range segments {
		if i > 0 {
			sb.WriteByte('.')
		}
		sb.WriteString(seg)
	}
	result := sb.String()
	putStringBuilder(sb)
	return result
}
