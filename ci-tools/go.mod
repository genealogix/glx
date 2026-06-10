module github.com/genealogix/glx/ci-tools

go 1.26.0

toolchain go1.26.1

tool (
	github.com/hmarr/codeowners/cmd/codeowners
	golang.org/x/vuln/cmd/govulncheck
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/hmarr/codeowners v1.2.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/telemetry v0.0.0-20260421165255-392afab6f40e // indirect
	golang.org/x/tools v0.44.0 // indirect
	golang.org/x/vuln v1.3.0 // indirect
)
