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
	"reflect"
	"strings"
	"testing"
)

// NoteList stands in for the go-glx NoteList type; the checker detects it by
// name, so the stand-in must share the name to exercise that path.
type NoteList []string

// DateString stands in for the go-glx DateString type (underlying kind string).
type DateString string

// compareFixture parses a single-file schema and compares goVal's type against
// its root object, returning the raw findings.
func compareFixture(t *testing.T, entity string, goVal any, schemaJSON string) []finding {
	t.Helper()
	set, err := newSchemaSet(map[string][]byte{"test.schema.json": []byte(schemaJSON)})
	if err != nil {
		t.Fatalf("newSchemaSet: %v", err)
	}
	node, loc, err := set.resolve("test.schema.json", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	c := newChecker(set, "go-glx/types.go")
	c.compareStruct(entity, reflect.TypeOf(goVal), node, loc, true)

	return c.findings
}

func hasFinding(findings []finding, sev severity, substr string) bool {
	for i := range findings {
		if findings[i].Severity == sev && strings.Contains(findings[i].Message, substr) {
			return true
		}
	}

	return false
}

func TestCompareClean(t *testing.T) {
	type clean struct {
		Title string         `yaml:"title"`
		Date  DateString     `yaml:"date,omitempty"`
		Notes NoteList       `yaml:"notes,omitempty"`
		Props map[string]any `yaml:"properties,omitempty"`
	}
	schema := `{
		"type": "object", "required": ["title"], "additionalProperties": false,
		"properties": {
			"title": {"type": "string"},
			"date": {"type": "string"},
			"notes": {"oneOf": [{"type":"string"},{"type":"array","items":{"type":"string"}}]},
			"properties": {"type": "object", "additionalProperties": true}
		}
	}`
	if got := compareFixture(t, "Clean", clean{}, schema); len(got) != 0 {
		t.Fatalf("expected no findings, got %+v", got)
	}
}

func TestMissingRequiredField(t *testing.T) {
	type s struct {
		Title string `yaml:"title"`
	}
	schema := `{"type":"object","required":["title","kind"],"additionalProperties":false,
		"properties":{"title":{"type":"string"},"kind":{"type":"string"}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityCritical, `required property "kind"`) {
		t.Fatalf("expected critical for missing required field, got %+v", got)
	}
}

func TestMissingOptionalField(t *testing.T) {
	type s struct {
		Title string `yaml:"title"`
	}
	schema := `{"type":"object","required":["title"],"additionalProperties":false,
		"properties":{"title":{"type":"string"},"note":{"type":"string"}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityMajor, `optional property "note"`) {
		t.Fatalf("expected major for missing optional field, got %+v", got)
	}
}

func TestExtraGoFieldClosed(t *testing.T) {
	type s struct {
		Title string `yaml:"title"`
		Extra string `yaml:"extra,omitempty"`
	}
	schema := `{"type":"object","required":["title"],"additionalProperties":false,
		"properties":{"title":{"type":"string"}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityCritical, "additionalProperties:false") {
		t.Fatalf("expected critical for extra field in closed object, got %+v", got)
	}
}

func TestExtraGoFieldOpen(t *testing.T) {
	type s struct {
		Title string `yaml:"title"`
		Extra string `yaml:"extra,omitempty"`
	}
	// No additionalProperties:false → extra Go field is not drift.
	schema := `{"type":"object","required":["title"],
		"properties":{"title":{"type":"string"}}}`
	if got := compareFixture(t, "S", s{}, schema); len(got) != 0 {
		t.Fatalf("expected no findings for extra field in open object, got %+v", got)
	}
}

func TestRequiredHasOmitempty(t *testing.T) {
	type s struct {
		Title string `yaml:"title,omitempty"` // wrong: required field must not omitempty
	}
	schema := `{"type":"object","required":["title"],"additionalProperties":false,
		"properties":{"title":{"type":"string"}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityCritical, "remove omitempty") {
		t.Fatalf("expected critical remove-omitempty, got %+v", got)
	}
}

func TestOptionalMissingOmitempty(t *testing.T) {
	type s struct {
		Title string `yaml:"title"`
		Note  string `yaml:"note"` // wrong: optional field should omitempty
	}
	schema := `{"type":"object","required":["title"],"additionalProperties":false,
		"properties":{"title":{"type":"string"},"note":{"type":"string"}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityMajor, "add omitempty") {
		t.Fatalf("expected major add-omitempty, got %+v", got)
	}
}

func TestTypeMismatch(t *testing.T) {
	type s struct {
		Tags string `yaml:"tags,omitempty"` // wrong: schema says array
	}
	schema := `{"type":"object","additionalProperties":false,
		"properties":{"tags":{"type":"array","items":{"type":"string"}}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityCritical, `defines type "array"`) {
		t.Fatalf("expected critical type mismatch, got %+v", got)
	}
}

func TestNoteListMatchesOneOf(t *testing.T) {
	type s struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}
	schema := `{"type":"object","additionalProperties":false,
		"properties":{"notes":{"oneOf":[{"type":"string"},{"type":"array","items":{"type":"string"}}]}}}`
	if got := compareFixture(t, "S", s{}, schema); len(got) != 0 {
		t.Fatalf("NoteList should match oneOf[string,array], got %+v", got)
	}
}

func TestNoteListMismatch(t *testing.T) {
	type s struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}
	schema := `{"type":"object","additionalProperties":false,
		"properties":{"notes":{"type":"string"}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityCritical, "NoteList") {
		t.Fatalf("expected NoteList type mismatch, got %+v", got)
	}
}

func TestNestedStructRecursion(t *testing.T) {
	type inner struct {
		Person string `yaml:"person"`
	}
	type outer struct {
		Items []inner `yaml:"items"`
	}
	// inner item is missing required "role".
	schema := `{"type":"object","required":["items"],"additionalProperties":false,
		"properties":{"items":{"type":"array","items":{
			"type":"object","required":["person","role"],"additionalProperties":false,
			"properties":{"person":{"type":"string"},"role":{"type":"string"}}}}}}`
	got := compareFixture(t, "Outer", outer{}, schema)
	if !hasFinding(got, severityCritical, `required property "role"`) {
		t.Fatalf("expected nested missing-field finding, got %+v", got)
	}
}

func TestPointerAndFloat(t *testing.T) {
	type s struct {
		Lat *float64 `yaml:"lat,omitempty"`
	}
	schema := `{"type":"object","additionalProperties":false,
		"properties":{"lat":{"type":"number"}}}`
	if got := compareFixture(t, "S", s{}, schema); len(got) != 0 {
		t.Fatalf("*float64 should match number, got %+v", got)
	}
}

func TestEntityRefOneOfUnion(t *testing.T) {
	type entityRef struct {
		Person string `yaml:"person,omitempty"`
		Event  string `yaml:"event,omitempty"`
	}
	type s struct {
		Subject entityRef `yaml:"subject"`
	}
	// oneOf of two single-field objects → union {person,event}, matches struct.
	schema := `{"type":"object","required":["subject"],"additionalProperties":false,
		"properties":{"subject":{"oneOf":[
			{"type":"object","required":["person"],"additionalProperties":false,"properties":{"person":{"type":"string"}}},
			{"type":"object","required":["event"],"additionalProperties":false,"properties":{"event":{"type":"string"}}}
		]}}}`
	if got := compareFixture(t, "S", s{}, schema); len(got) != 0 {
		t.Fatalf("EntityRef union should be clean, got %+v", got)
	}
}

func TestEntityRefOneOfExtraBranch(t *testing.T) {
	type entityRef struct {
		Person string `yaml:"person,omitempty"`
	}
	type s struct {
		Subject entityRef `yaml:"subject"`
	}
	schema := `{"type":"object","required":["subject"],"additionalProperties":false,
		"properties":{"subject":{"oneOf":[
			{"type":"object","required":["person"],"additionalProperties":false,"properties":{"person":{"type":"string"}}},
			{"type":"object","required":["place"],"additionalProperties":false,"properties":{"place":{"type":"string"}}}
		]}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityMajor, `property "place"`) {
		t.Fatalf("expected missing 'place' field from oneOf union, got %+v", got)
	}
}
