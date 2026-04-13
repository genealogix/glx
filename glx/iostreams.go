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
	"fmt"
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
func TestIOStreams() (*IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	return &IOStreams{Out: out, ErrOut: errOut}, out, errOut
}

// Printf writes a formatted string to the standard output stream.
func (s *IOStreams) Printf(format string, args ...any) {
	fmt.Fprintf(s.Out, format, args...) //nolint:errcheck // CLI output
}

// Println writes a line to the standard output stream.
func (s *IOStreams) Println(msg string) {
	fmt.Fprintln(s.Out, msg) //nolint:errcheck // CLI output
}

// Errorf writes a formatted string to the error output stream.
func (s *IOStreams) Errorf(format string, args ...any) {
	fmt.Fprintf(s.ErrOut, format, args...) //nolint:errcheck // CLI output
}
