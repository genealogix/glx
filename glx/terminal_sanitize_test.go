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
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeForTerminal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii", "William Shakespeare", "William Shakespeare"},
		{"unicode preserved", "Charlotte Brontë — Zoë", "Charlotte Brontë — Zoë"},
		{"tab and newline preserved", "a\tb\nc", "a\tb\nc"},
		{"clear screen sequence", "Virginia\x1b[2J\x1b[H", `Virginia\x1b[2J\x1b[H`},
		{"osc hyperlink", "\x1b]8;;https://evil.example\x07click\x1b]8;;\x07", `\x1b]8;;https://evil.example\x07click\x1b]8;;\x07`},
		{"carriage return", "ok\rNO ISSUES", `ok\x0dNO ISSUES`},
		{"delete", "a\x7fb", `a\x7fb`},
		{"c1 csi", "a\u009bb", `a\x9bb`},
		{"backspace", "a\bb", `a\x08b`},
		{"invalid utf8 escaped", "a\xffb", `a\xffb`},
		{"raw c1 csi byte escaped", "a\x9b[2Jb", `a\x9b[2Jb`},
		{"raw c1 byte with no valid control rune", "\x9b[2J", `\x9b[2J`},
		{"truncated multibyte escaped", "Bront\xc3", `Bront\xc3`},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeForTerminal(tt.in))
		})
	}
}

func TestSanitizeForTerminal_NeverEmitsControlBytes(t *testing.T) {
	var all strings.Builder
	for i := range 0x100 {
		all.WriteRune(rune(i))
	}
	got := sanitizeForTerminal(all.String())

	assert.False(t, strings.ContainsFunc(got, isTerminalControl))
	assert.Contains(t, got, "\t")
	assert.Contains(t, got, "\n")
}

// TestSanitizeForTerminal_NeverEmitsRawHighBytes feeds every single byte
// value as a raw byte (not a rune), so 0x80–0xff arrive as invalid UTF-8
// rather than as encoded C1 code points. None may survive: a raw 0x9b is a
// single-byte CSI to a terminal that honors eight-bit controls.
func TestSanitizeForTerminal_NeverEmitsRawHighBytes(t *testing.T) {
	raw := make([]byte, 0, 0x100)
	for i := range 0x100 {
		raw = append(raw, byte(i))
	}
	got := sanitizeForTerminal(string(raw))

	assert.True(t, utf8.ValidString(got))
	assert.False(t, strings.ContainsFunc(got, isTerminalControl))
	assert.Contains(t, got, `\x9b`)
	assert.Contains(t, got, `\xff`)
}

func TestIOStreams_SanitizeOutAndErrOut(t *testing.T) {
	streams, out, errOut := TestIOStreams()
	name := "Mary\x1b[2J\x1b[H"

	streams.Printf("Person: %s\n", name)
	streams.Println(name)
	streams.Errorf("warning: %s\n", name)

	assert.NotContains(t, out.String(), "\x1b")
	assert.NotContains(t, errOut.String(), "\x1b")
	assert.Equal(t, "Person: Mary\\x1b[2J\\x1b[H\nMary\\x1b[2J\\x1b[H\n", out.String())
	assert.Equal(t, "warning: Mary\\x1b[2J\\x1b[H\n", errOut.String())
}

func TestIOStreams_PrintfPlainTextUnchanged(t *testing.T) {
	streams, out, _ := TestIOStreams()
	streams.Printf("Validated %d files.\n", 3)
	streams.Println("✅ Archive is valid.")

	assert.Equal(t, "Validated 3 files.\n✅ Archive is valid.\n", out.String())
}

func TestPrintIssue_SanitizesArchiveText(t *testing.T) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	printIssue(&AnalysisIssue{
		Category: "consistency",
		Severity: "medium",
		Person:   "person-\x1b[31mevil",
		Message:  "born in Virginia\x1b[2J\x1b[H",
	})
	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	assert.NotContains(t, buf.String(), "\x1b")
	assert.Contains(t, buf.String(), `person-\x1b[31mevil`)
	assert.Contains(t, buf.String(), `Virginia\x1b[2J\x1b[H`)
}
