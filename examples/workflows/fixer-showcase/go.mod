module fixer-showcase

go 1.26.11

require github.com/erraggy/oastools v1.66.0

require (
	go.yaml.in/yaml/v4 v4.0.0-rc.4 // indirect
	golang.org/x/text v0.44.0 // indirect
)

replace github.com/erraggy/oastools => ../../../
