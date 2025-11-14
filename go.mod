// GENEALOGIX Specification
// Official specification and tools for the GENEALOGIX family archive format.
// Provides JSON schemas, validation tools, and examples for genealogical data.
module github.com/genealogix/spec

go 1.25

require (
	github.com/xeipuuv/gojsonschema v1.2.0
	// YAML parsing for CLI tool
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
)

// CLI tool for GENEALOGIX archives
// Provides: glx init, glx validate, glx check-schemas
// See: ./glx/main.go
