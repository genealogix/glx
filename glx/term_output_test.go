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
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStd runs fn with os.Stdout and os.Stderr redirected to pipes and
// returns what each received.
func captureStd(t *testing.T, fn func()) (string, string) {
	t.Helper()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()

	fn()

	_ = outW.Close()
	_ = errW.Close()
	out, _ := io.ReadAll(outR)
	errOut, _ := io.ReadAll(errR)

	return string(out), string(errOut)
}

func TestTermHelpers_SanitizeArchiveText(t *testing.T) {
	const evil = "Jane\x1b[2J\u009b31mDoe"
	const shown = `Jane\x1b[2J\x9b31mDoe`

	out, errOut := captureStd(t, func() {
		termPrintf("name: %s\n", evil)
		termPrintln("name:", evil)
		termPrint("name: ", evil, "\n")
		termErrorf("Warning: %s\n", evil)
		termErrorln("Warning:", evil)
	})

	wantOut := "name: " + shown + "\n" + "name: " + shown + "\n" + "name: " + shown + "\n"
	if out != wantOut {
		t.Errorf("stdout = %q, want %q", out, wantOut)
	}
	wantErr := "Warning: " + shown + "\n" + "Warning: " + shown + "\n"
	if errOut != wantErr {
		t.Errorf("stderr = %q, want %q", errOut, wantErr)
	}
}

func TestTermHelpers_FormatLikeFmt(t *testing.T) {
	// Println spaces every operand; Print spaces only between non-strings.
	out, _ := captureStd(t, func() {
		termPrintln("a", 1, "b")
		termPrint("a", "b", 1, 2, "\n")
		termPrintf("%-5s|%3d\t|\n", "x", 7)
	})
	want := "a 1 b\nab1 2\nx    |  7\t|\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestMachinePrintln_ByteFaithful(t *testing.T) {
	// DEL and C1 characters pass through encoding/json unescaped; machine
	// output must keep them so a consumer parses the archive's real value.
	const data = "{\"name\":\"Jane\u0085\x7f\"}"
	out, _ := captureStd(t, func() { machinePrintln(data) })
	if out != data+"\n" {
		t.Errorf("got %q, want %q", out, data+"\n")
	}
}

// TestNoBareTerminalPrints keeps every CLI write to the process's stdout and
// stderr on a sanitized path (#1208): IOStreams, or the term* helpers in
// term_output.go. A bare fmt.Print*, or fmt.Fprint* to os.Stdout or
// os.Stderr, would print archive-controlled text with its terminal control
// sequences intact.
func TestNoBareTerminalPrints(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	allowed := map[string]bool{
		"term_output.go":       true, // the sanitized helpers themselves
		"iostreams.go":         true, // the IOStreams boundary
		"terminal_sanitize.go": true,
	}

	fset := token.NewFileSet()
	var bad []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") || allowed[name] {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !isIdent(fn.X, "fmt") {
				return true
			}
			switch fn.Sel.Name {
			case "Print", "Printf", "Println":
				bad = append(bad, fset.Position(call.Pos()).String()+": fmt."+fn.Sel.Name)
			case "Fprint", "Fprintf", "Fprintln":
				if len(call.Args) > 0 && isStdStream(call.Args[0]) {
					bad = append(bad, fset.Position(call.Pos()).String()+": fmt."+fn.Sel.Name+" to a std stream")
				}
			}

			return true
		})
	}

	for _, b := range bad {
		t.Errorf("%s: write through IOStreams or a term* helper from term_output.go instead (#1208)", b)
	}
}

func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)

	return ok && id.Name == name
}

func isStdStream(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)

	return ok && isIdent(sel.X, "os") && (sel.Sel.Name == "Stdout" || sel.Sel.Name == "Stderr")
}
