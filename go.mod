// GENEALOGIX Specification
// Official specification and tools for the GENEALOGIX family archive format.
// Provides JSON schemas, validation tools, and examples for genealogical data.
module github.com/genealogix/glx

go 1.26.0

toolchain go1.26.1

require (
	github.com/xeipuuv/gojsonschema v1.2.0
	// YAML parsing for CLI tool
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/brianvoe/gofakeit/v7 v7.14.1
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/stretchr/testify v1.11.1
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
)

require golang.org/x/text v0.35.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
)

// CLI tool for GENEALOGIX archives
// See: ./glx/main.go
