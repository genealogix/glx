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

package structdump

import (
	"os"
	"testing"
)

const typesGo = "../../types.go"

// loadTypes reads types.go (I/O lives in the test, not the package) and extracts it.
func loadTypes(t *testing.T) *Dump {
	t.Helper()
	src, err := os.ReadFile(typesGo)
	if err != nil {
		t.Fatalf("read %s: %v", typesGo, err)
	}
	d, err := Extract(typesGo, src)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	return d
}

func TestExtractCollections(t *testing.T) {
	d := loadTypes(t)

	// Known GLXFile collections (yaml key -> Go type). These are stable anchors.
	want := map[string]string{
		"persons":           "Person",
		"events":            "Event",
		"event_types":       "VocabularyEntry",
		"person_properties": "PropertyDefinition",
	}
	got := map[string]string{}
	for _, c := range d.Collections {
		got[c.YAMLKey] = c.GoType
		if c.Line <= 0 {
			t.Errorf("collection %q has non-positive line %d", c.YAMLKey, c.Line)
		}
		if c.YAMLKey == "" || c.YAMLKey == "-" {
			t.Errorf("collection with empty/ignored yaml key leaked in: %+v", c)
		}
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("collection %q: got Go type %q, want %q", k, got[k], v)
		}
	}

	// The *Metadata field and the unexported validation field are NOT collections.
	for _, c := range d.Collections {
		if c.YAMLKey == "metadata" || c.GoType == "Metadata" {
			t.Errorf("non-collection field leaked in as a collection: %+v", c)
		}
	}
}

func TestReferentialIntegrity(t *testing.T) {
	d := loadTypes(t)
	if len(d.Collections) == 0 {
		t.Fatal("no collections extracted")
	}
	for _, c := range d.Collections {
		if _, ok := d.Types[c.GoType]; !ok {
			t.Errorf("collection %q points at Go type %q which was not extracted", c.YAMLKey, c.GoType)
		}
	}
}

func TestFields(t *testing.T) {
	d := loadTypes(t)
	person, ok := d.CollectionType("persons")
	if !ok {
		t.Fatal(`CollectionType("persons") not found`)
	}
	tags := map[string]int{}
	for _, f := range person.Fields {
		tags[f.YAMLTag] = f.Line
		if f.YAMLTag == "" || f.YAMLTag == "-" {
			t.Errorf("Person field with empty/ignored yaml tag leaked in: %+v", f)
		}
	}
	for _, tag := range []string{"properties", "notes"} {
		if tags[tag] == 0 {
			t.Errorf("Person missing field with yaml tag %q (have %v)", tag, tags)
		}
	}
}

// TestSkipsUntaggedAndDashFields proves fields with no yaml tag or `yaml:"-"`
// are excluded — they are not serialized, so the drift tooling must not treat
// them as schema fields.
func TestSkipsUntaggedAndDashFields(t *testing.T) {
	src := []byte("package p\n" +
		"type GLXFile struct {\n" +
		"\tPersons map[string]*Person `yaml:\"persons,omitempty\"`\n" +
		"}\n" +
		"type Person struct {\n" +
		"\tKept     string `yaml:\"kept\"`\n" +
		"\tIgnored  string `yaml:\"-\"`\n" +
		"\tUntagged string\n" +
		"}\n")
	d, err := Extract("synthetic.go", src)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	got := make([]string, 0, len(d.Types["Person"].Fields))
	for _, f := range d.Types["Person"].Fields {
		got = append(got, f.YAMLTag)
	}
	if len(got) != 1 || got[0] != "kept" {
		t.Errorf("expected only [kept], got %v (untagged / yaml:\"-\" fields must be skipped)", got)
	}
}
