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

// These tests cover the element/value recursion and per-symbol attribution
// fixes made in response to the PR #969 Copilot review (compareSlice and
// compareMapValue recursing into scalar element/value types, and findings
// being attributed to the nested Go type rather than the parent entity).

func TestSliceScalarElementMismatch(t *testing.T) {
	type s struct {
		Tags []string `yaml:"tags,omitempty"`
	}
	// []string element vs schema items {type: integer} — must be flagged.
	schema := `{"type":"object","additionalProperties":false,
		"properties":{"tags":{"type":"array","items":{"type":"integer"}}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityCritical, `defines type "integer"`) {
		t.Fatalf("expected scalar slice-element type mismatch, got %+v", got)
	}
}

func TestSliceScalarElementMatch(t *testing.T) {
	type s struct {
		Tags []string `yaml:"tags,omitempty"`
	}
	schema := `{"type":"object","additionalProperties":false,
		"properties":{"tags":{"type":"array","items":{"type":"string"}}}}`
	if got := compareFixture(t, "S", s{}, schema); len(got) != 0 {
		t.Fatalf("[]string vs items string should be clean, got %+v", got)
	}
}

func TestMapScalarValueMismatch(t *testing.T) {
	type s struct {
		M map[string]int `yaml:"m,omitempty"`
	}
	// map value is int, but the per-value subschema is typed string.
	schema := `{"type":"object","additionalProperties":false,
		"properties":{"m":{"type":"object","additionalProperties":{"type":"string"}}}}`
	got := compareFixture(t, "S", s{}, schema)
	if !hasFinding(got, severityCritical, `defines type "string"`) {
		t.Fatalf("expected scalar map-value type mismatch, got %+v", got)
	}
}

// TestNestedFindingUsesOwnTypeName guards the per-symbol identity contract:
// drift inside a nested struct must be attributed to that struct's Go type,
// not the parent entity it was reached through.
func TestNestedFindingUsesOwnTypeName(t *testing.T) {
	type leaf struct {
		Code string `yaml:"code"`
	}
	type holder struct {
		Item leaf `yaml:"item"`
	}
	// holder.item requires code+extra; leaf lacks "extra".
	schema := `{"type":"object","required":["item"],"additionalProperties":false,
		"properties":{"item":{"type":"object","required":["code","extra"],"additionalProperties":false,
			"properties":{"code":{"type":"string"},"extra":{"type":"string"}}}}}`
	got := compareFixture(t, "holder", holder{}, schema)

	var found bool
	for i := range got {
		switch got[i].Symbol {
		case "leaf.extra":
			found = true
			if got[i].Entity != "leaf" {
				t.Errorf("nested finding Entity = %q, want leaf", got[i].Entity)
			}
		case "holder.extra":
			t.Errorf("nested finding wrongly attributed to parent: %+v", got[i])
		}
	}
	if !found {
		t.Fatalf("expected a leaf.extra finding, got %+v", got)
	}
}
