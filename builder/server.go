package builder

import (
	"github.com/erraggy/oastools/parser"
)

// serverConfig holds configuration for server building.
type serverConfig struct {
	description string
	variables   map[string]parser.ServerVariable
}

// ServerOption configures a server.
type ServerOption func(*serverConfig)

// WithServerDescription sets the server description.
func WithServerDescription(desc string) ServerOption {
	return func(cfg *serverConfig) {
		cfg.description = desc
	}
}

// serverVariableConfig holds configuration for server variable building.
type serverVariableConfig struct {
	enum        []string
	description string
}

// ServerVariableOption configures a server variable.
type ServerVariableOption func(*serverVariableConfig)

// WithServerVariableEnum sets the enum values for a server variable.
func WithServerVariableEnum(values ...string) ServerVariableOption {
	return func(cfg *serverVariableConfig) {
		cfg.enum = values
	}
}

// WithServerVariableDescription sets the description for a server variable.
func WithServerVariableDescription(desc string) ServerVariableOption {
	return func(cfg *serverVariableConfig) {
		cfg.description = desc
	}
}

// WithServerVariable adds a variable to the server.
func WithServerVariable(name, defaultValue string, opts ...ServerVariableOption) ServerOption {
	return func(cfg *serverConfig) {
		varCfg := &serverVariableConfig{}
		for _, opt := range opts {
			opt(varCfg)
		}

		if cfg.variables == nil {
			cfg.variables = make(map[string]parser.ServerVariable)
		}

		cfg.variables[name] = parser.ServerVariable{
			Default:     defaultValue,
			Enum:        varCfg.enum,
			Description: varCfg.description,
		}
	}
}

// AddServer adds a server to the specification.
func (b *Builder) AddServer(url string, opts ...ServerOption) *Builder {
	cfg := &serverConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	server := &parser.Server{
		URL:         url,
		Description: cfg.description,
		Variables:   cfg.variables,
	}

	b.servers = append(b.servers, server)
	return b
}
