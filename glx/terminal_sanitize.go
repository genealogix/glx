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
	"strings"
	"unicode/utf8"
)

// isTerminalControl reports whether r is a control character that a terminal
// would interpret rather than display: the ASCII C0 range (except tab and
// newline, which are ordinary layout), DEL, and the C1 range U+0080–U+009F
// (which includes CSI and OSC introducers). ESC (0x1b) is the usual carrier
// for color, cursor-movement, screen-clear and OSC hyperlink sequences.
func isTerminalControl(r rune) bool {
	if r == '\t' || r == '\n' {
		return false
	}

	return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f)
}

// sanitizeForTerminal returns s with terminal control characters replaced by
// their visible Go-style hex escape (for example ESC becomes `\x1b`), so a
// string taken from an archive — a person's name, a place name, a note —
// cannot inject control sequences into human-readable CLI output. Tab and
// newline are preserved. Bytes that are not valid UTF-8 are passed through
// unchanged. Applying this at the output boundary keeps YAML values
// untouched and keeps JSON output (already escaped by encoding/json) as-is.
func sanitizeForTerminal(s string) string {
	if !strings.ContainsFunc(s, isTerminalControl) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			b.WriteByte(s[i])
		case isTerminalControl(r):
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteString(s[i : i+size])
		}
		i += size
	}

	return b.String()
}
