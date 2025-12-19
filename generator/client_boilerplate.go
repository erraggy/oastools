// client_boilerplate.go contains shared client code generation templates
// used by both OAS2 and OAS3 generators.

package generator

import (
	"bytes"
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// writeClientStruct writes the Client struct definition to the buffer.
func writeClientStruct(buf *bytes.Buffer) {
	buf.WriteString("// Client is the API client.\n")
	buf.WriteString("type Client struct {\n")
	buf.WriteString("\t// BaseURL is the base URL for API requests.\n")
	buf.WriteString("\tBaseURL string\n")
	buf.WriteString("\t// HTTPClient is the HTTP client to use for requests.\n")
	buf.WriteString("\tHTTPClient *http.Client\n")
	buf.WriteString("\t// UserAgent is the User-Agent header value for requests.\n")
	buf.WriteString("\tUserAgent string\n")
	buf.WriteString("\t// RequestEditors are functions that can modify requests before sending.\n")
	buf.WriteString("\tRequestEditors []RequestEditorFn\n")
	buf.WriteString("}\n\n")
}

// writeClientTypes writes the shared client type definitions.
func writeClientTypes(buf *bytes.Buffer) {
	buf.WriteString("// RequestEditorFn is a function that can modify an HTTP request.\n")
	buf.WriteString("type RequestEditorFn func(ctx context.Context, req *http.Request) error\n\n")

	buf.WriteString("// ClientOption is a function that configures a Client.\n")
	buf.WriteString("type ClientOption func(*Client) error\n\n")
}

// writeClientConstructor writes the NewClient constructor function.
func writeClientConstructor(buf *bytes.Buffer, info *parser.Info) {
	defaultUserAgent := buildDefaultUserAgent(info)
	buf.WriteString("// NewClient creates a new API client.\n")
	buf.WriteString("func NewClient(baseURL string, opts ...ClientOption) (*Client, error) {\n")
	buf.WriteString("\tc := &Client{\n")
	buf.WriteString("\t\tBaseURL:    strings.TrimSuffix(baseURL, \"/\"),\n")
	buf.WriteString("\t\tHTTPClient: http.DefaultClient,\n")
	_, _ = fmt.Fprintf(buf, "\t\tUserAgent:  %q,\n", defaultUserAgent)
	buf.WriteString("\t}\n")
	buf.WriteString("\tfor _, opt := range opts {\n")
	buf.WriteString("\t\tif err := opt(c); err != nil {\n")
	buf.WriteString("\t\t\treturn nil, err\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\treturn c, nil\n")
	buf.WriteString("}\n\n")
}

// writeClientOptions writes the standard client option functions.
func writeClientOptions(buf *bytes.Buffer) {
	buf.WriteString("// WithHTTPClient sets the HTTP client.\n")
	buf.WriteString("func WithHTTPClient(client *http.Client) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.HTTPClient = client\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	buf.WriteString("// WithRequestEditor adds a request editor function.\n")
	buf.WriteString("func WithRequestEditor(fn RequestEditorFn) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.RequestEditors = append(c.RequestEditors, fn)\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	buf.WriteString("// WithUserAgent sets the User-Agent header value.\n")
	buf.WriteString("func WithUserAgent(ua string) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.UserAgent = ua\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")
}

// writeClientBoilerplate writes all standard client boilerplate code.
// This includes the struct, types, constructor, and options.
func writeClientBoilerplate(buf *bytes.Buffer, info *parser.Info) {
	writeClientStruct(buf)
	writeClientTypes(buf)
	writeClientConstructor(buf, info)
	writeClientOptions(buf)
}
