// GENEALOGIX Specification
// Official specification and tools for the GENEALOGIX family archive format.
// Provides JSON schemas, validation tools, and examples for genealogical data.
module github.com/genealogix/spec

go 1.23

require (
	// YAML parsing for CLI tool
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// CLI tool for GENEALOGIX archives
// Provides: glx init, glx validate, glx check-schemas
// See: ./glx/main.go
