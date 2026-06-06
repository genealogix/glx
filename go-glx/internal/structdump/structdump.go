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

// Package structdump deterministically extracts the entity/vocabulary
// collections and their struct fields from go-glx/types.go using the Go AST.
//
// It exists so the check-* drift tooling never hand-maintains a list of
// entities/fields: the GLXFile struct is the single source of truth. The
// deterministic spec/code-drift checks consume Collections (which yaml key
// maps to which Go type) and Fields (name, yaml tag, source line) instead of
// asking an LLM to read types.go and hope.
//
// Pure: Extract parses caller-provided source bytes and performs NO filesystem
// I/O, per the go-glx no-I/O rule — the caller reads the file.
package structdump

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
)

// ErrNoGLXFile is returned by Extract when the source has no GLXFile struct.
var ErrNoGLXFile = errors.New("GLXFile struct not found")

// Field is one struct field: its Go name, its yaml tag key (sans options like
// ",omitempty"), and the 1-based line it is declared on.
type Field struct {
	Name    string
	YAMLTag string
	Line    int
}

// TypeInfo is a struct type declared in the file.
type TypeInfo struct {
	Name   string
	Line   int
	Fields []Field
}

// Collection is one `map[string]*X` field of GLXFile: the yaml key it
// serializes under and the Go entity/vocabulary type X it points at.
type Collection struct {
	YAMLKey string
	GoType  string
	Line    int
}

// Dump is the extracted view of types.go.
type Dump struct {
	// Collections is every map[string]*X field of GLXFile, in declaration order.
	Collections []Collection
	// Types is every struct type declared in the file, keyed by Go type name.
	Types map[string]TypeInfo
}

// CollectionType returns the TypeInfo a collection points at (e.g. "persons" -> Person).
func (d *Dump) CollectionType(yamlKey string) (TypeInfo, bool) {
	for _, c := range d.Collections {
		if c.YAMLKey == yamlKey {
			t, ok := d.Types[c.GoType]

			return t, ok
		}
	}

	return TypeInfo{}, false
}

// Extract parses the given Go source (the contents of go-glx/types.go, read by
// the caller) and returns its struct types and the GLXFile collections.
// filename is used only for error messages and source positions.
func Extract(filename string, src []byte) (*Dump, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	d := &Dump{Types: map[string]TypeInfo{}}
	var glxFile *ast.StructType

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if ts.Name.Name == "GLXFile" {
				glxFile = st
			}
			d.Types[ts.Name.Name] = TypeInfo{
				Name:   ts.Name.Name,
				Line:   fset.Position(ts.Pos()).Line,
				Fields: fieldsOf(st, fset),
			}
		}
	}

	if glxFile == nil {
		return nil, fmt.Errorf("%s: %w", filename, ErrNoGLXFile)
	}
	d.Collections = collectionsOf(glxFile, fset)

	return d, nil
}

// fieldsOf returns the serialized named fields of a struct with their yaml tag
// keys. Embedded fields, unexported fields, and fields with no yaml tag or
// `yaml:"-"` are skipped — they are not serialized, so they are not schema
// fields the drift tooling should compare against.
func fieldsOf(st *ast.StructType, fset *token.FileSet) []Field {
	var out []Field
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue // embedded field
		}
		tag := yamlKey(f.Tag)
		if tag == "" || tag == "-" {
			continue // not serialized
		}
		line := fset.Position(f.Pos()).Line
		for _, n := range f.Names {
			if !n.IsExported() {
				continue // unexported fields are not serialized
			}
			out = append(out, Field{Name: n.Name, YAMLTag: tag, Line: line})
		}
	}

	return out
}

// collectionsOf returns the map[string]*X fields of GLXFile that serialize
// under a yaml key (skips no-tag / `yaml:"-"` fields).
func collectionsOf(st *ast.StructType, fset *token.FileSet) []Collection {
	var out []Collection
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue
		}
		goType, ok := mapStringPointerElem(f.Type)
		if !ok {
			continue // not a map[string]*X collection (e.g. *Metadata, validation)
		}
		key := yamlKey(f.Tag)
		if key == "" || key == "-" {
			continue // not serialized under a yaml key
		}
		out = append(out, Collection{
			YAMLKey: key,
			GoType:  goType,
			Line:    fset.Position(f.Pos()).Line,
		})
	}

	return out
}

// mapStringPointerElem reports whether expr is `map[string]*Ident` and returns Ident.
func mapStringPointerElem(expr ast.Expr) (string, bool) {
	m, ok := expr.(*ast.MapType)
	if !ok {
		return "", false
	}
	if key, ok := m.Key.(*ast.Ident); !ok || key.Name != "string" {
		return "", false
	}
	star, ok := m.Value.(*ast.StarExpr)
	if !ok {
		return "", false
	}
	id, ok := star.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	return id.Name, true
}

// yamlKey parses a struct tag literal and returns the yaml key without options.
func yamlKey(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	raw := strings.Trim(tag.Value, "`")
	v := reflect.StructTag(raw).Get("yaml")
	if v == "" {
		return ""
	}

	return strings.TrimSpace(strings.SplitN(v, ",", 2)[0])
}

// SortedTypeNames returns the declared type names in stable order (for tests/output).
func (d *Dump) SortedTypeNames() []string {
	names := make([]string, 0, len(d.Types))
	for n := range d.Types {
		names = append(names, n)
	}
	sort.Strings(names)

	return names
}
