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
	"fmt"
	"io"
	"os"
)

// The helpers below are the only way CLI code writes human-readable text to
// the process's stdout and stderr without going through IOStreams. Each one
// formats exactly as its fmt counterpart does and then passes the rendered
// text through sanitizeForTerminal, so archive-controlled values (names,
// places, notes, IDs, filenames) cannot inject terminal control sequences
// (#925, #1208). TestNoBareTerminalPrints fails on any bare fmt.Print* call,
// or fmt.Fprint* to os.Stdout or os.Stderr, outside this file.
//
// Unlike SystemIOStreams().Out, they ignore --quiet: they are a drop-in for
// the fmt calls of runners not yet on IOStreams, and keep those runners'
// behavior apart from the sanitization. os.Stdout and os.Stderr are read at
// call time, so tests that swap them still capture the output.

// termPrintf is fmt.Printf with the output sanitized for the terminal.
func termPrintf(format string, args ...any) {
	fmt.Fprint(os.Stdout, sanitizeForTerminal(fmt.Sprintf(format, args...)))
}

// termPrintln is fmt.Println with the output sanitized for the terminal.
func termPrintln(args ...any) {
	fmt.Fprint(os.Stdout, sanitizeForTerminal(fmt.Sprintln(args...)))
}

// termPrint is fmt.Print with the output sanitized for the terminal.
func termPrint(args ...any) {
	fmt.Fprint(os.Stdout, sanitizeForTerminal(fmt.Sprint(args...)))
}

// termErrorf is fmt.Fprintf(os.Stderr, ...) with the output sanitized for
// the terminal.
func termErrorf(format string, args ...any) {
	fmt.Fprint(os.Stderr, sanitizeForTerminal(fmt.Sprintf(format, args...)))
}

// termErrorln is fmt.Fprintln(os.Stderr, ...) with the output sanitized for
// the terminal.
func termErrorln(args ...any) {
	fmt.Fprint(os.Stderr, sanitizeForTerminal(fmt.Sprintln(args...)))
}

// terminalWriter is an io.Writer over stdout or stderr that sanitizes each
// write for the terminal. It is for passing to a helper that takes a writer
// for human-readable diagnostics (a migration warning, a verbose progress
// line), so the raw stream is never forwarded where archive-controlled text
// is printed. Each write is sanitized on its own, so a caller must render a
// line in one write (as fmt.Fprintf does) rather than byte by byte.
type terminalWriter struct{ stderr bool }

func (w terminalWriter) Write(p []byte) (int, error) {
	dst := os.Stdout
	if w.stderr {
		dst = os.Stderr
	}
	if _, err := dst.WriteString(sanitizeForTerminal(string(p))); err != nil {
		return 0, err
	}

	return len(p), nil
}

// termStdout returns a writer to stdout that sanitizes for the terminal.
func termStdout() io.Writer { return terminalWriter{} }

// termStderr returns a writer to stderr that sanitizes for the terminal.
func termStderr() io.Writer { return terminalWriter{stderr: true} }

// machinePrintln writes s and a newline to stdout byte for byte. It is for
// machine-readable output (--format json and the like) that a consumer
// parses, which must stay faithful to the data; encoding/json already
// escapes the C0 range. Never pass it human-readable text.
func machinePrintln(s string) {
	fmt.Fprintln(os.Stdout, s)
}
