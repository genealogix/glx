module github.com/genealogix/glx/ci-tools

go 1.27.0

toolchain go1.27.1

tool (
	github.com/hmarr/codeowners/cmd/codeowners
	golang.org/x/vuln/cmd/govulncheck
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/hmarr/codeowners v1.2.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/telemetry v0.0.0-20260811182544-a038080d80e5 // indirect
	golang.org/x/tools v0.49.0 // indirect
	golang.org/x/vuln v1.3.0 // indirect
)
