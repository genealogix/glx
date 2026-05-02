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

package glx

import (
	"reflect"
	"testing"
)

// TestNewExternalIDEntry locks the public contract of the helper used by both
// GEDCOM EXID import (via buildExternalIDEntry) and first-party callers such
// as the CLI `glx link` command. Any change here is a public-API change for
// library consumers who type-switch on the return value.
func TestNewExternalIDEntry(t *testing.T) {
	t.Run("empty typeURI returns the bare value string", func(t *testing.T) {
		got := NewExternalIDEntry("1:1:C4H8-2DW2", "")
		want := "1:1:C4H8-2DW2"
		if got != want {
			t.Errorf("got %v (%T), want %q (string)", got, got, want)
		}
	})

	t.Run("non-empty typeURI returns structured value/fields.type map", func(t *testing.T) {
		got := NewExternalIDEntry("1:1:C4H8-2DW2", "https://www.familysearch.org/ark:/61903/")
		want := map[string]any{
			"value": "1:1:C4H8-2DW2",
			"fields": map[string]any{
				"type": "https://www.familysearch.org/ark:/61903/",
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})

	t.Run("empty value preserved (not replaced)", func(t *testing.T) {
		got := NewExternalIDEntry("", "")
		if got != "" {
			t.Errorf("got %v, want empty string", got)
		}
	})

	t.Run("empty value with typeURI still yields a structured map", func(t *testing.T) {
		got := NewExternalIDEntry("", "https://example.com/authority")
		m, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", got)
		}
		if m["value"] != "" {
			t.Errorf("value: got %v, want empty string", m["value"])
		}
		fields, _ := m["fields"].(map[string]any)
		if fields["type"] != "https://example.com/authority" {
			t.Errorf("fields.type: got %v", fields["type"])
		}
	})
}

// TestBuildExternalIDEntry_DelegatesToNewExternalIDEntry verifies that the
// GEDCOM-import-side wrapper produces identical output to the exported helper
// for equivalent inputs. This prevents silent divergence between the two
// paths if either is refactored.
func TestBuildExternalIDEntry_DelegatesToNewExternalIDEntry(t *testing.T) {
	t.Run("EXID with TYPE matches structured NewExternalIDEntry output", func(t *testing.T) {
		rec := &GEDCOMRecord{
			Value: "1:1:C4H8-2DW2",
			SubRecords: []*GEDCOMRecord{
				{Tag: GedcomTagType, Value: "https://www.familysearch.org/ark:/61903/"},
			},
		}
		got := buildExternalIDEntry(rec)
		want := NewExternalIDEntry("1:1:C4H8-2DW2", "https://www.familysearch.org/ark:/61903/")
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})

	t.Run("EXID without TYPE matches bare-string NewExternalIDEntry output", func(t *testing.T) {
		rec := &GEDCOMRecord{Value: "1:1:C4H8-2DW2"}
		got := buildExternalIDEntry(rec)
		want := NewExternalIDEntry("1:1:C4H8-2DW2", "")
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})
}
