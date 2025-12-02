package builder

import (
	"net/http"
	"testing"
	"time"

	"github.com/erraggy/oastools/parser"
)

// Test types for benchmarks
type BenchmarkUser struct {
	ID        int64     `json:"id" oas:"description=Unique identifier"`
	Name      string    `json:"name" oas:"minLength=1,maxLength=100"`
	Email     string    `json:"email" oas:"format=email"`
	Age       int       `json:"age,omitempty" oas:"minimum=0,maximum=150"`
	CreatedAt time.Time `json:"created_at" oas:"readOnly=true"`
}

type BenchmarkAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code" oas:"pattern=^[0-9]{5}$"`
	Country string `json:"country"`
}

type BenchmarkOrder struct {
	ID         int64            `json:"id"`
	CustomerID int64            `json:"customer_id"`
	Items      []BenchmarkItem  `json:"items"`
	Total      float64          `json:"total"`
	Status     string           `json:"status" oas:"enum=pending|processing|shipped|delivered"`
	CreatedAt  time.Time        `json:"created_at"`
	Address    BenchmarkAddress `json:"address"`
}

type BenchmarkItem struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type BenchmarkError struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		_ = New(parser.OASVersion320)
	}
}

func BenchmarkBuilder_SetInfo(b *testing.B) {
	for b.Loop() {
		_ = New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			SetDescription("A test API")
	}
}

func BenchmarkSchemaFrom_Primitive(b *testing.B) {
	for b.Loop() {
		bldr := New(parser.OASVersion320)
		_ = bldr.generateSchema(string(""))
	}
}

func BenchmarkSchemaFrom_Struct(b *testing.B) {
	for b.Loop() {
		bldr := New(parser.OASVersion320)
		_ = bldr.generateSchema(BenchmarkUser{})
	}
}

func BenchmarkSchemaFrom_NestedStruct(b *testing.B) {
	for b.Loop() {
		bldr := New(parser.OASVersion320)
		_ = bldr.generateSchema(BenchmarkOrder{})
	}
}

func BenchmarkSchemaFrom_Slice(b *testing.B) {
	for b.Loop() {
		bldr := New(parser.OASVersion320)
		_ = bldr.generateSchema([]BenchmarkUser{})
	}
}

func BenchmarkSchemaFrom_Map(b *testing.B) {
	for b.Loop() {
		bldr := New(parser.OASVersion320)
		_ = bldr.generateSchema(map[string]BenchmarkUser{})
	}
}

func BenchmarkBuilder_AddOperation_Simple(b *testing.B) {
	for b.Loop() {
		_ = New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/users",
				WithOperationID("listUsers"),
				WithResponse(http.StatusOK, []BenchmarkUser{}),
			)
	}
}

func BenchmarkBuilder_AddOperation_WithParams(b *testing.B) {
	for b.Loop() {
		_ = New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/users/{id}",
				WithOperationID("getUser"),
				WithPathParam("id", int64(0)),
				WithQueryParam("include", string("")),
				WithHeaderParam("X-Request-ID", string("")),
				WithResponse(http.StatusOK, BenchmarkUser{}),
				WithResponse(http.StatusNotFound, BenchmarkError{}),
			)
	}
}

func BenchmarkBuilder_AddOperation_WithRequestBody(b *testing.B) {
	for b.Loop() {
		_ = New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/orders",
				WithOperationID("createOrder"),
				WithRequestBody("application/json", BenchmarkOrder{},
					WithRequired(true),
				),
				WithResponse(http.StatusCreated, BenchmarkOrder{}),
				WithResponse(http.StatusBadRequest, BenchmarkError{}),
			)
	}
}

func BenchmarkBuilder_Build(b *testing.B) {
	for b.Loop() {
		spec := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddServer("https://api.example.com",
				WithServerDescription("Production server"),
			).
			AddTag("users", WithTagDescription("User operations")).
			AddOperation(http.MethodGet, "/users",
				WithOperationID("listUsers"),
				WithTags("users"),
				WithQueryParam("limit", int32(0)),
				WithQueryParam("offset", int32(0)),
				WithResponse(http.StatusOK, []BenchmarkUser{}),
			).
			AddOperation(http.MethodGet, "/users/{id}",
				WithOperationID("getUser"),
				WithTags("users"),
				WithPathParam("id", int64(0)),
				WithResponse(http.StatusOK, BenchmarkUser{}),
				WithResponse(http.StatusNotFound, BenchmarkError{}),
			).
			AddOperation(http.MethodPost, "/users",
				WithOperationID("createUser"),
				WithTags("users"),
				WithRequestBody("application/json", BenchmarkUser{},
					WithRequired(true),
				),
				WithResponse(http.StatusCreated, BenchmarkUser{}),
			)

		_, err := spec.BuildOAS3()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuilder_MarshalYAML(b *testing.B) {
	spec := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users",
			WithOperationID("listUsers"),
			WithResponse(http.StatusOK, []BenchmarkUser{}),
		)

	for b.Loop() {
		_, err := spec.MarshalYAML()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuilder_MarshalJSON(b *testing.B) {
	spec := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users",
			WithOperationID("listUsers"),
			WithResponse(http.StatusOK, []BenchmarkUser{}),
		)

	for b.Loop() {
		_, err := spec.MarshalJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkApplyOASTag(b *testing.B) {
	schema := &parser.Schema{Type: "string"}
	tag := "description=Test description,minLength=1,maxLength=100,pattern=^[a-z]+$"

	for b.Loop() {
		_ = applyOASTag(schema, tag)
	}
}

func BenchmarkParseOASTag(b *testing.B) {
	tag := "description=Test description,minLength=1,maxLength=100,pattern=^[a-z]+$,format=email,enum=a|b|c"

	for b.Loop() {
		_ = parseOASTag(tag)
	}
}

func BenchmarkBuilder_AddOperation_WithFormParams_OAS2(b *testing.B) {
	for b.Loop() {
		_ = New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/login",
				WithOperationID("login"),
				WithFormParam("username", string(""),
					WithParamRequired(true),
					WithParamMinLength(3),
				),
				WithFormParam("password", string(""),
					WithParamRequired(true),
					WithParamMinLength(8),
				),
				WithFormParam("remember_me", bool(false),
					WithParamDefault(false),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)
	}
}

func BenchmarkBuilder_AddOperation_WithFormParams_OAS3(b *testing.B) {
	for b.Loop() {
		_ = New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/login",
				WithOperationID("login"),
				WithFormParam("username", string(""),
					WithParamRequired(true),
					WithParamMinLength(3),
				),
				WithFormParam("password", string(""),
					WithParamRequired(true),
					WithParamMinLength(8),
				),
				WithFormParam("remember_me", bool(false),
					WithParamDefault(false),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)
	}
}
