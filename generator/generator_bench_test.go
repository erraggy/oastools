package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/parser"
)

func BenchmarkGenerateTypes(b *testing.B) {
	// Create a spec with multiple schemas
	spec := `openapi: "3.0.0"
info:
  title: Benchmark API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        tag:
          type: string
        category:
          $ref: '#/components/schemas/Category'
    Category:
      type: object
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
    Error:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
    Status:
      type: string
      enum:
        - available
        - pending
        - sold
`
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench-api.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		b.Fatalf("failed to write temp file: %v", err)
	}

	// Parse once outside the loop
	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}

	g := New()
	g.PackageName = "benchapi"
	g.GenerateTypes = true

	for b.Loop() {
		_, err := g.GenerateParsed(*parseResult)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateClient(b *testing.B) {
	spec := `openapi: "3.0.0"
info:
  title: Benchmark API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '201':
          description: Created
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: A pet
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
    put:
      operationId: updatePet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '200':
          description: Updated
    delete:
      operationId: deletePet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '204':
          description: Deleted
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench-client.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		b.Fatalf("failed to write temp file: %v", err)
	}

	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}

	g := New()
	g.PackageName = "benchapi"
	g.GenerateClient = true

	for b.Loop() {
		_, err := g.GenerateParsed(*parseResult)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateServer(b *testing.B) {
	spec := `openapi: "3.0.0"
info:
  title: Benchmark API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        '200':
          description: A list of pets
    post:
      operationId: createPet
      responses:
        '201':
          description: Created
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: A pet
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench-server.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		b.Fatalf("failed to write temp file: %v", err)
	}

	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}

	g := New()
	g.PackageName = "benchapi"
	g.GenerateServer = true

	for b.Loop() {
		_, err := g.GenerateParsed(*parseResult)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateAll(b *testing.B) {
	spec := `openapi: "3.0.0"
info:
  title: Benchmark API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '201':
          description: Created
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: A pet
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        category:
          $ref: '#/components/schemas/Category'
    Category:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench-all.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		b.Fatalf("failed to write temp file: %v", err)
	}

	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}

	g := New()
	g.PackageName = "benchapi"
	g.GenerateTypes = true
	g.GenerateClient = true
	g.GenerateServer = true

	for b.Loop() {
		_, err := g.GenerateParsed(*parseResult)
		if err != nil {
			b.Fatal(err)
		}
	}
}
