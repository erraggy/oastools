package paramutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erraggy/oastools/internal/paramutil"
	"github.com/erraggy/oastools/parser"
)

func TestNeedsRequiredTrue(t *testing.T) {
	tests := []struct {
		name  string
		param *parser.Parameter
		want  bool
	}{
		{
			name:  "path parameter without required",
			param: &parser.Parameter{Name: "userId", In: parser.ParamInPath},
			want:  true,
		},
		{
			name:  "path parameter with required true",
			param: &parser.Parameter{Name: "userId", In: parser.ParamInPath, Required: true},
			want:  false,
		},
		{
			name:  "query parameter without required",
			param: &parser.Parameter{Name: "verbose", In: parser.ParamInQuery},
			want:  false,
		},
		{
			name:  "header parameter without required",
			param: &parser.Parameter{Name: "X-Trace", In: parser.ParamInHeader},
			want:  false,
		},
		{
			name:  "cookie parameter without required",
			param: &parser.Parameter{Name: "session", In: parser.ParamInCookie},
			want:  false,
		},
		{
			// A reference carries no fields of its own; the defect belongs to the
			// definition it names, which callers check separately.
			name:  "reference to a path parameter definition",
			param: &parser.Parameter{Ref: "#/components/parameters/UserId"},
			want:  false,
		},
		{
			// A $ref with siblings is not valid OAS, but the ref still wins: the
			// target's own required field is what the document ends up using.
			name:  "reference with in path sibling",
			param: &parser.Parameter{Ref: "#/components/parameters/UserId", In: parser.ParamInPath},
			want:  false,
		},
		{
			name:  "missing in field",
			param: &parser.Parameter{Name: "userId"},
			want:  false,
		},
		{
			name:  "nil parameter",
			param: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, paramutil.NeedsRequiredTrue(tt.param))
		})
	}
}
