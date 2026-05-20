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
//
// Out is the diagnostic-output stream that --quiet may silence.
// MachineOut is the machine-consumable stream (created entity IDs, JSON
// results for shell capture) that --quiet does NOT silence — without this,
// `id=$(glx add … --quiet)` would yield an empty id.
// ErrOut is the error stream and is never silenced.
type IOStreams struct {
	Out        io.Writer
	MachineOut io.Writer
	ErrOut     io.Writer
}

// quietOutput is set by the --quiet/-q persistent flag registered on rootCmd
// in cli_commands.go. When true, SystemIOStreams discards stdout while leaving
// stderr connected to os.Stderr so errors are still surfaced.
var quietOutput bool

// SystemIOStreams returns IOStreams connected to os.Stdout and os.Stderr.
// If the --quiet/-q flag was passed, Out is replaced with io.Discard so
// migrated runners that write diagnostic output via Out are silenced.
// MachineOut is always os.Stdout so the entity ID echoed by `glx add` and
// other machine-consumable outputs survive --quiet for shell capture.
func SystemIOStreams() *IOStreams {
	out := io.Writer(os.Stdout)
	if quietOutput {
		out = io.Discard
	}

	return &IOStreams{Out: out, MachineOut: os.Stdout, ErrOut: os.Stderr}
}

// TestIOStreams returns IOStreams backed by buffers for testing. The
// diagnostic Out and machine-consumable MachineOut share the same buffer so
// existing tests asserting against Out continue to see the entity-ID echo;
// dedicated --quiet tests rebind Out to io.Discard and assert separately on
// MachineOut.
func TestIOStreams() (*IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	return &IOStreams{Out: out, MachineOut: out, ErrOut: errOut}, out, errOut
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
