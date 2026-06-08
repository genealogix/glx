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

// typesGoPath is go-glx/types.go relative to tools/driftcheck/.
const typesGoPath = "../../go-glx/types.go"

func TestAttachLines(t *testing.T) {
	findings := []finding{
		{Entity: "Person", YamlTag: "notes", Field: "Notes"},     // matches by yaml tag
		{Entity: "Person", YamlTag: "nope", Field: "Properties"}, // tag miss -> matches by Go field name
		{Entity: "Nonexistent", YamlTag: "x", Field: "Y"},        // no match -> Line stays 0
	}
	if err := attachLines(findings, typesGoPath); err != nil {
		t.Fatalf("attachLines: %v", err)
	}
	if findings[0].Line <= 0 {
		t.Errorf("Person.notes: expected a source line, got %d", findings[0].Line)
	}
	if findings[1].Line <= 0 {
		t.Errorf("Person.Properties (by field name): expected a source line, got %d", findings[1].Line)
	}
	if findings[2].Line != 0 {
		t.Errorf("unknown entity: expected Line 0, got %d", findings[2].Line)
	}
}

func TestAttachLinesReadError(t *testing.T) {
	if err := attachLines(nil, "/no/such/types.go"); err == nil {
		t.Error("expected an error when the types file cannot be read")
	}
}
