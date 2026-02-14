package mcpserver

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// specInput represents the three ways an OAS spec can be provided to a tool.
// Exactly one of File, URL, or Content must be set.
type specInput struct {
	File    string `json:"file,omitempty"    jsonschema:"Path to an OAS file on disk"`
	URL     string `json:"url,omitempty"     jsonschema:"URL to fetch an OAS document from"`
	Content string `json:"content,omitempty" jsonschema:"Inline OAS document content (JSON or YAML)"`
}

// resolve parses the spec from whichever input was provided.
// Additional parser options can be passed to customize parsing behavior.
func (s specInput) resolve(extraOpts ...parser.Option) (*parser.ParseResult, error) {
	count := 0
	if s.File != "" {
		count++
	}
	if s.URL != "" {
		count++
	}
	if s.Content != "" {
		count++
	}
	if count != 1 {
		return nil, fmt.Errorf("exactly one of file, url, or content must be provided (got %d)", count)
	}

	var opts []parser.Option
	switch {
	case s.File != "":
		opts = append(opts, parser.WithFilePath(s.File))
	case s.URL != "":
		opts = append(opts, parser.WithFilePath(s.URL))
	case s.Content != "":
		opts = append(opts, parser.WithReader(strings.NewReader(s.Content)))
	}
	opts = append(opts, extraOpts...)

	return parser.ParseWithOptions(opts...)
}
