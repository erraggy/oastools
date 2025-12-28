package generator

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Shared Client Code Generation Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// pathParam represents a path parameter with its original name and Go variable name.
type pathParam struct {
	name    string // Original parameter name from spec (e.g., "user_id")
	varName string // Go variable name (e.g., "userId")
}

// writeMethodDoc writes the documentation comment for a client method.
func writeMethodDoc(buf *bytes.Buffer, op *parser.Operation, methodName, method, path string) {
	if op.Summary != "" {
		buf.WriteString(formatMultilineComment(op.Summary, methodName, ""))
	} else if op.Description != "" {
		buf.WriteString(formatMultilineComment(op.Description, methodName, ""))
	} else {
		fmt.Fprintf(buf, "// %s calls %s %s\n", methodName, strings.ToUpper(method), path)
	}
	if op.Deprecated {
		buf.WriteString("// Deprecated: This operation is deprecated.\n")
	}
}

// writeMethodSignature writes the method signature line.
func writeMethodSignature(buf *bytes.Buffer, methodName string, params []string, responseType string) {
	fmt.Fprintf(buf, "func (c *Client) %s(%s) (%s, error) {\n", methodName, strings.Join(params, ", "), responseType)
}

// writeURLBuilding writes the URL path construction code.
func writeURLBuilding(buf *bytes.Buffer, path string, pathParams []pathParam) {
	buf.WriteString("\tpath := ")
	if len(pathParams) > 0 {
		buf.WriteString("fmt.Sprintf(\"")
		pathTemplate := path
		for _, pp := range pathParams {
			pathTemplate = strings.ReplaceAll(pathTemplate, "{"+pp.name+"}", "%v")
		}
		buf.WriteString(pathTemplate)
		buf.WriteString("\"")
		for _, pp := range pathParams {
			buf.WriteString(", " + pp.varName)
		}
		buf.WriteString(")\n")
	} else {
		fmt.Fprintf(buf, "%q\n", path)
	}
}

// writeQueryStringBuilding writes the query string construction code.
func writeQueryStringBuilding(buf *bytes.Buffer, queryParams []*parser.Parameter) {
	if len(queryParams) == 0 {
		return
	}
	buf.WriteString("\tquery := make(url.Values)\n")
	buf.WriteString("\tif params != nil {\n")
	for _, param := range queryParams {
		paramName := toFieldName(param.Name)
		if param.Required {
			fmt.Fprintf(buf, "\t\tquery.Set(%q, fmt.Sprintf(\"%%v\", params.%s))\n", param.Name, paramName)
		} else {
			fmt.Fprintf(buf, "\t\tif params.%s != nil {\n", paramName)
			fmt.Fprintf(buf, "\t\t\tquery.Set(%q, fmt.Sprintf(\"%%v\", *params.%s))\n", param.Name, paramName)
			buf.WriteString("\t\t}\n")
		}
	}
	buf.WriteString("\t}\n")
	buf.WriteString("\tif len(query) > 0 {\n")
	buf.WriteString("\t\tpath += \"?\" + query.Encode()\n")
	buf.WriteString("\t}\n")
}

// writeRequestCreation writes the HTTP request creation code.
func writeRequestCreation(buf *bytes.Buffer, hasBody bool, method, responseType string) {
	if hasBody {
		buf.WriteString("\tbodyData, err := json.Marshal(body)\n")
		buf.WriteString("\tif err != nil {\n")
		fmt.Fprintf(buf, "\t\treturn %s, fmt.Errorf(\"marshal request body: %%w\", err)\n", zeroValue(responseType))
		buf.WriteString("\t}\n")
		fmt.Fprintf(buf, "\treq, err := http.NewRequestWithContext(ctx, %q, c.BaseURL+path, bytes.NewReader(bodyData))\n", strings.ToUpper(method))
	} else {
		fmt.Fprintf(buf, "\treq, err := http.NewRequestWithContext(ctx, %q, c.BaseURL+path, nil)\n", strings.ToUpper(method))
	}
	buf.WriteString("\tif err != nil {\n")
	fmt.Fprintf(buf, "\t\treturn %s, fmt.Errorf(\"create request: %%w\", err)\n", zeroValue(responseType))
	buf.WriteString("\t}\n")
}

// writeRequestHeaders writes the header setting code.
func writeRequestHeaders(buf *bytes.Buffer, hasBody bool, contentType string) {
	if hasBody {
		fmt.Fprintf(buf, "\treq.Header.Set(\"Content-Type\", %q)\n", contentType)
	}
	buf.WriteString("\treq.Header.Set(\"Accept\", \"application/json\")\n")
	buf.WriteString("\tif c.UserAgent != \"\" {\n")
	buf.WriteString("\t\treq.Header.Set(\"User-Agent\", c.UserAgent)\n")
	buf.WriteString("\t}\n")
}

// writeRequestEditors writes the request editor application code.
func writeRequestEditors(buf *bytes.Buffer, responseType string) {
	buf.WriteString("\tfor _, editor := range c.RequestEditors {\n")
	buf.WriteString("\t\tif err := editor(ctx, req); err != nil {\n")
	fmt.Fprintf(buf, "\t\t\treturn %s, fmt.Errorf(\"request editor: %%w\", err)\n", zeroValue(responseType))
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")
}

// writeRequestExecution writes the request execution and error handling code.
func writeRequestExecution(buf *bytes.Buffer, responseType string) {
	buf.WriteString("\tresp, err := c.HTTPClient.Do(req)\n")
	buf.WriteString("\tif err != nil {\n")
	fmt.Fprintf(buf, "\t\treturn %s, fmt.Errorf(\"execute request: %%w\", err)\n", zeroValue(responseType))
	buf.WriteString("\t}\n")
	buf.WriteString("\tdefer resp.Body.Close()\n")
}

// writeErrorResponseHandling writes the error response handling code.
func writeErrorResponseHandling(buf *bytes.Buffer, responseType string) {
	buf.WriteString("\tif resp.StatusCode >= 400 {\n")
	buf.WriteString("\t\tbody, _ := io.ReadAll(resp.Body)\n")
	fmt.Fprintf(buf, "\t\treturn %s, &APIError{StatusCode: resp.StatusCode, Body: body}\n", zeroValue(responseType))
	buf.WriteString("\t}\n")
}

// writeResponseParsing writes the response body parsing code.
func writeResponseParsing(buf *bytes.Buffer, responseType string) {
	if responseType != "" && responseType != httpResponseType {
		if strings.HasPrefix(responseType, "*") {
			fmt.Fprintf(buf, "\tvar result %s\n", responseType[1:])
			buf.WriteString("\tif err := json.NewDecoder(resp.Body).Decode(&result); err != nil {\n")
			fmt.Fprintf(buf, "\t\treturn %s, fmt.Errorf(\"decode response: %%w\", err)\n", zeroValue(responseType))
			buf.WriteString("\t}\n")
			buf.WriteString("\treturn &result, nil\n")
		} else {
			fmt.Fprintf(buf, "\tvar result %s\n", responseType)
			buf.WriteString("\tif err := json.NewDecoder(resp.Body).Decode(&result); err != nil {\n")
			fmt.Fprintf(buf, "\t\treturn %s, fmt.Errorf(\"decode response: %%w\", err)\n", zeroValue(responseType))
			buf.WriteString("\t}\n")
			buf.WriteString("\treturn result, nil\n")
		}
	} else {
		buf.WriteString("\treturn resp, nil\n")
	}
	buf.WriteString("}\n\n")
}

// writeParamsStruct writes the query parameters struct definition.
// paramToGoType is a callback to get the Go type for a parameter (version-specific).
func writeParamsStruct(buf *bytes.Buffer, methodName string, queryParams []*parser.Parameter, paramToGoType func(*parser.Parameter) string) {
	if len(queryParams) == 0 {
		return
	}
	fmt.Fprintf(buf, "// %sParams contains query parameters for %s.\n", methodName, methodName)
	fmt.Fprintf(buf, "type %sParams struct {\n", methodName)
	for _, param := range queryParams {
		goType := paramToGoType(param)
		fieldName := toFieldName(param.Name)
		if param.Description != "" {
			fmt.Fprintf(buf, "\t// %s\n", cleanDescription(param.Description))
		}
		if !param.Required {
			fmt.Fprintf(buf, "\t%s *%s `json:%q`\n", fieldName, goType, param.Name+",omitempty")
		} else {
			fmt.Fprintf(buf, "\t%s %s `json:%q`\n", fieldName, goType, param.Name)
		}
	}
	buf.WriteString("}\n\n")
}

// writeClientMethod writes all the shared client method code.
// This is identical between OAS 2.0 and OAS 3.x generators.
func writeClientMethod(buf *bytes.Buffer, op *parser.Operation, methodName, method, path string,
	params []string, pathParams []pathParam, queryParams []*parser.Parameter,
	hasBody bool, contentType, responseType string, paramToGoType func(*parser.Parameter) string) {
	writeMethodDoc(buf, op, methodName, method, path)
	writeMethodSignature(buf, methodName, params, responseType)
	writeURLBuilding(buf, path, pathParams)
	writeQueryStringBuilding(buf, queryParams)
	writeRequestCreation(buf, hasBody, method, responseType)
	writeRequestHeaders(buf, hasBody, contentType)
	writeRequestEditors(buf, responseType)
	writeRequestExecution(buf, responseType)
	writeErrorResponseHandling(buf, responseType)
	writeResponseParsing(buf, responseType)
	writeParamsStruct(buf, methodName, queryParams, paramToGoType)
}
