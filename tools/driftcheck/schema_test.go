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
	"testing"
)

func TestResolveCrossFileAndPointer(t *testing.T) {
	files := map[string][]byte{
		"glx-file.schema.json": []byte(`{
			"type":"object",
			"properties":{
				"persons":{"type":"object","patternProperties":{"^x$":{"$ref":"person.schema.json"}}},
				"event_types":{"type":"object","patternProperties":{"^x$":{"$ref":"vocabularies/v.schema.json#/definitions/Def"}}}
			}
		}`),
		"person.schema.json": []byte(`{"type":"object","properties":{"name":{"type":"string"}}}`),
		"vocabularies/v.schema.json": []byte(`{
			"definitions":{"Def":{"type":"object","properties":{"label":{"type":"string"}}}}
		}`),
	}
	set, err := newSchemaSet(files)
	if err != nil {
		t.Fatalf("newSchemaSet: %v", err)
	}

	// Cross-file ref relative to root dir.
	node, loc, err := set.resolve("person.schema.json", "")
	if err != nil {
		t.Fatalf("resolve person: %v", err)
	}
	if node.Properties["name"] == nil || loc.file != "person.schema.json" {
		t.Fatalf("unexpected person node/loc: %+v %+v", node, loc)
	}

	// Ref with a JSON pointer into a subdirectory file, resolved relative to
	// the referencing file's directory.
	node, loc, err = set.resolve("vocabularies/v.schema.json#/definitions/Def", "glx-file.schema.json")
	if err != nil {
		t.Fatalf("resolve vocab def: %v", err)
	}
	if node.Properties["label"] == nil {
		t.Fatalf("expected Def with label, got %+v", node)
	}
	if loc.file != "vocabularies/v.schema.json" || loc.pointer != "/definitions/Def" {
		t.Fatalf("unexpected loc: %+v", loc)
	}
}

func TestResolveMissingFile(t *testing.T) {
	set, err := newSchemaSet(map[string][]byte{"a.schema.json": []byte(`{"type":"object"}`)})
	if err != nil {
		t.Fatalf("newSchemaSet: %v", err)
	}
	if _, _, err := set.resolve("missing.schema.json", ""); err == nil {
		t.Fatal("expected error resolving missing file")
	}
}

func TestAdditionalProperties(t *testing.T) {
	cases := []struct {
		name       string
		raw        string
		wantClosed bool
		wantSchema bool
	}{
		{"false", `{"additionalProperties": false}`, true, false},
		{"true", `{"additionalProperties": true}`, false, false},
		{"schema", `{"additionalProperties": {"$ref":"x"}}`, false, true},
		{"absent", `{}`, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set, err := newSchemaSet(map[string][]byte{"a.schema.json": []byte(tc.raw)})
			if err != nil {
				t.Fatalf("newSchemaSet: %v", err)
			}
			n := set.roots["a.schema.json"]
			if got := n.additionalPropsClosed(); got != tc.wantClosed {
				t.Errorf("additionalPropsClosed = %v, want %v", got, tc.wantClosed)
			}
			if got := n.additionalPropsSchema() != nil; got != tc.wantSchema {
				t.Errorf("additionalPropsSchema present = %v, want %v", got, tc.wantSchema)
			}
		})
	}
}
