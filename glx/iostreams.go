// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"io"
	"os"
)

// IOStreams provides the standard output streams for CLI commands.
// Modeled after kubectl's genericiooptions.IOStreams pattern.
type IOStreams struct {
	Out    io.Writer
	ErrOut io.Writer
}

// SystemIOStreams returns IOStreams connected to os.Stdout and os.Stderr.
func SystemIOStreams() *IOStreams {
	return &IOStreams{Out: os.Stdout, ErrOut: os.Stderr}
}

// TestIOStreams returns IOStreams backed by buffers for testing.
func TestIOStreams() (streams *IOStreams, out *bytes.Buffer, errOut *bytes.Buffer) {
	out = &bytes.Buffer{}
	errOut = &bytes.Buffer{}
	streams = &IOStreams{Out: out, ErrOut: errOut}

	return streams, out, errOut
}
