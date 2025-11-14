package main

import _ "embed"

//go:embed defaults/.gitignore
var defaultGitignore []byte

//go:embed defaults/README.md
var defaultReadme []byte

