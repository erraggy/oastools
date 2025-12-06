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

// BenchmarkBuilderNew benchmarks builder creation
func BenchmarkBuilderNew(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = New(parser.OASVersion320)
	}
}

// BenchmarkBuilderSetInfo benchmarks setting basic info
func BenchmarkBuilderSetInfo(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			SetDescription("A test API")
	}
}

// BenchmarkSchemaFrom benchmarks schema generation from various types
func BenchmarkSchemaFrom(b *testing.B) {
	b.Run("Primitive", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			bldr := New(parser.OASVersion320)
			_ = bldr.generateSchema(string(""))
		}
	})

	b.Run("Struct", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			bldr := New(parser.OASVersion320)
			_ = bldr.generateSchema(BenchmarkUser{})
		}
	})

	b.Run("NestedStruct", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			bldr := New(parser.OASVersion320)
			_ = bldr.generateSchema(BenchmarkOrder{})
		}
	})

	b.Run("Slice", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			bldr := New(parser.OASVersion320)
			_ = bldr.generateSchema([]BenchmarkUser{})
		}
	})

	b.Run("Map", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			bldr := New(parser.OASVersion320)
			_ = bldr.generateSchema(map[string]BenchmarkUser{})
		}
	})
}

// BenchmarkBuilderAddOperation benchmarks adding operations with various configurations
func BenchmarkBuilderAddOperation(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = New(parser.OASVersion320).
				SetTitle("Test API").
				SetVersion("1.0.0").
				AddOperation(http.MethodGet, "/users",
					WithOperationID("listUsers"),
					WithResponse(http.StatusOK, []BenchmarkUser{}),
				)
		}
	})

	b.Run("WithParams", func(b *testing.B) {
		b.ReportAllocs()
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
	})

	b.Run("WithRequestBody", func(b *testing.B) {
		b.ReportAllocs()
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
	})
}

// BenchmarkBuilderBuild benchmarks building a complete API spec
func BenchmarkBuilderBuild(b *testing.B) {
	b.ReportAllocs()
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

// BenchmarkBuilderMarshal benchmarks marshaling built specs
func BenchmarkBuilderMarshal(b *testing.B) {
	spec := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users",
			WithOperationID("listUsers"),
			WithResponse(http.StatusOK, []BenchmarkUser{}),
		)

	b.Run("YAML", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := spec.MarshalYAML()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("JSON", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := spec.MarshalJSON()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkOASTag benchmarks OAS tag parsing and application
func BenchmarkOASTag(b *testing.B) {
	b.Run("Apply", func(b *testing.B) {
		schema := &parser.Schema{Type: "string"}
		tag := "description=Test description,minLength=1,maxLength=100,pattern=^[a-z]+$"

		b.ReportAllocs()
		for b.Loop() {
			_ = applyOASTag(schema, tag)
		}
	})

	b.Run("Parse", func(b *testing.B) {
		tag := "description=Test description,minLength=1,maxLength=100,pattern=^[a-z]+$,format=email,enum=a|b|c"

		b.ReportAllocs()
		for b.Loop() {
			_ = parseOASTag(tag)
		}
	})
}

// BenchmarkBuilderFormParams benchmarks form parameter handling
func BenchmarkBuilderFormParams(b *testing.B) {
	b.Run("OAS2", func(b *testing.B) {
		b.ReportAllocs()
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
	})

	b.Run("OAS3", func(b *testing.B) {
		b.ReportAllocs()
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
	})
}
