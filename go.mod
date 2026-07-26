module github.com/erraggy/oastools

go 1.25.8

require (
	github.com/modelcontextprotocol/go-sdk v1.6.1
	github.com/stretchr/testify v1.11.1
	// Pinned deliberately. rc.5/rc.6 restructured loading and dumping into a
	// three-stage pipeline, which costs +12-16% allocations on every parse and
	// +35% allocations, +17% bytes and +18% time on marshal, buying only ~5%
	// less total memory on parse. Do not bump without re-running the parser,
	// joiner and builder benchmarks. Re-evaluate when v4.0.0 final ships.
	go.yaml.in/yaml/v4 v4.0.0-rc.4
	golang.org/x/text v0.40.0
	golang.org/x/tools v0.48.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
