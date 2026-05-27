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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestEnsureJSONLDExtension(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "no extension", path: "output", want: "output.jsonld"},
		{name: "has .jsonld", path: "output.jsonld", want: "output.jsonld"},
		{name: "has .JSONLD uppercase", path: "output.JSONLD", want: "output.JSONLD"},
		{name: "has .json (different)", path: "output.json", want: "output.json.jsonld"},
		{name: "nested path", path: "dir/sub/file", want: "dir/sub/file.jsonld"},
		{name: "nested with .jsonld", path: "dir/sub/file.jsonld", want: "dir/sub/file.jsonld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ensureJSONLDExtension(tt.path))
		})
	}
}

func TestExportToJSONLD_RoundtripShape(t *testing.T) {
	tmpDir := t.TempDir()
	glxPath := filepath.Join(tmpDir, "archive.glx")

	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	outPath := filepath.Join(tmpDir, "shakespeare")
	err = exportToJSONLD(glxPath, outPath, false, false)
	require.NoError(t, err)

	finalPath := outPath + ".jsonld"
	data, err := os.ReadFile(finalPath)
	require.NoError(t, err, "export should have written .jsonld file")
	require.NotEmpty(t, data)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc), "output must be valid JSON")
	require.Contains(t, doc, "@context")
	require.Contains(t, doc, "@graph")

	graph, ok := doc["@graph"].([]any)
	require.True(t, ok, "@graph must be an array")
	require.NotEmpty(t, graph, "Shakespeare archive should produce non-empty graph")

	// Spot-check that at least one Person node is present and that its @id
	// uses the #person- fragment scheme.
	foundPerson := false
	for _, raw := range graph {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if nodeType, _ := node["@type"].(string); nodeType == "Person" {
			foundPerson = true
			id, _ := node["@id"].(string)
			assert.Contains(t, id, "#person-", "person @id should use #person- prefix")

			break
		}
	}
	require.True(t, foundPerson, "expected at least one Person node in graph")
}

func TestExportToJSONLD_CreatesNestedOutputDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	glxPath := filepath.Join(tmpDir, "archive.glx")

	err := importGEDCOM("testdata/gedcom/7.0/minimal-valid/minimal70.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	nestedOut := filepath.Join(tmpDir, "nested", "subdir", "out.jsonld")
	require.NoError(t, exportToJSONLD(glxPath, nestedOut, false, false))

	_, err = os.Stat(nestedOut)
	require.NoError(t, err, "nested directories should be created")
}

func TestExportToJSONLD_InputNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	err := exportToJSONLD(filepath.Join(tmpDir, "nonexistent.glx"), filepath.Join(tmpDir, "out.jsonld"), false, false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInputNotFound)
}

func TestExportToJSONLD_VerboseMode(t *testing.T) {
	tmpDir := t.TempDir()
	glxPath := filepath.Join(tmpDir, "archive.glx")

	err := importGEDCOM("testdata/gedcom/7.0/minimal-valid/minimal70.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	outPath := filepath.Join(tmpDir, "verbose-out.jsonld")
	require.NoError(t, exportToJSONLD(glxPath, outPath, true, false))

	_, err = os.Stat(outPath)
	require.NoError(t, err)
}

// privacyTestArchive builds a synthetic archive with one clearly-dead person
// (birth 1850, death 1920) and one clearly-living person (birth 2010, no
// death). It mirrors the fixture in TestExportToGEDCOM_PrivatizeLiving so the
// JSON-LD privacy guarantee is verified against the same data.
func privacyTestArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-old": {
				Properties: map[string]any{
					"name": map[string]any{
						"value":  "Ezekiel Privacytest",
						"fields": map[string]any{"given": "Ezekiel", "surname": "Privacytest"},
					},
					"sex": "male",
				},
			},
			"person-young": {
				Properties: map[string]any{
					"name": map[string]any{
						"value":  "Modern Currentperson",
						"fields": map[string]any{"given": "Modern", "surname": "Currentperson"},
					},
					"sex":        "male",
					"occupation": "software engineer",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-old": {
				Type: glxlib.EventTypeBirth,
				Date: "1850",
				Participants: []glxlib.Participant{
					{Person: "person-old", Role: glxlib.ParticipantRolePrincipal},
				},
			},
			"event-death-old": {
				Type: glxlib.EventTypeDeath,
				Date: "1920",
				Participants: []glxlib.Participant{
					{Person: "person-old", Role: glxlib.ParticipantRolePrincipal},
				},
			},
			"event-birth-young": {
				Type: glxlib.EventTypeBirth,
				Date: "2010",
				Participants: []glxlib.Participant{
					{Person: "person-young", Role: glxlib.ParticipantRolePrincipal},
				},
			},
		},
	}
}

// findGraphNode returns the @graph node with the given @id, or nil if absent.
func findGraphNode(graph []any, id string) map[string]any {
	for _, raw := range graph {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if nodeID, _ := node["@id"].(string); nodeID == id {
			return node
		}
	}

	return nil
}

func TestExportToJSONLD_PrivatizeLiving(t *testing.T) {
	tmpDir := t.TempDir()

	glxPath := filepath.Join(tmpDir, "archive.glx")
	require.NoError(t, writeSingleFileArchive(glxPath, privacyTestArchive(), false), "setup: write archive")

	outPath := filepath.Join(tmpDir, "output.jsonld")
	require.NoError(t, exportToJSONLD(glxPath, outPath, false, true))

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	out := string(data)

	// Raw-text guarantees: no field of the living person leaks anywhere in the
	// document (Person node, participant nodes, @context, etc.).
	require.NotContains(t, out, "Modern", "living person's given name should not leak")
	require.NotContains(t, out, "Currentperson", "living person's surname should not leak")
	require.NotContains(t, out, "software engineer", "living person's occupation should not leak")
	require.NotContains(t, out, "2010", "living person's birth year should not leak")

	// Dead person's data is preserved.
	require.Contains(t, out, "Ezekiel", "dead person's name should be preserved")
	require.Contains(t, out, "1850", "dead person's birth year should be preserved")
	require.Contains(t, out, "1920", "dead person's death year should be preserved")

	// Structural guarantee: the living Person node is redacted to exactly the
	// canonical {name:"Living", glx:living:true} shape with no PII-bearing keys.
	// structuredName({value:"Living"}) collapses to the string "Living", so the
	// emitted name is a plain string rather than a nested object.
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	graph, ok := doc["@graph"].([]any)
	require.True(t, ok, "@graph must be an array")

	young := findGraphNode(graph, "#person-person-young")
	require.NotNil(t, young, "living person node should be present in graph")
	assert.Equal(t, "Living", young["name"], `living person's name should be the literal string "Living"`)
	assert.Equal(t, true, young["glx:living"], "living person should carry glx:living true")
	for _, leakKey := range []string{"givenName", "familyName", "hasOccupation", "glx:occupation", "glx:sex"} {
		assert.NotContains(t, young, leakKey, "redacted living person should not expose %q", leakKey)
	}

	// The dead person keeps their structured name fields.
	old := findGraphNode(graph, "#person-person-old")
	require.NotNil(t, old, "dead person node should be present in graph")
	assert.Equal(t, "Ezekiel", old["givenName"], "dead person's given name should be preserved")
}

func TestRunExport_PrivatizeLivingJSONLD(t *testing.T) {
	tmpDir := t.TempDir()

	glxPath := filepath.Join(tmpDir, "archive.glx")
	require.NoError(t, writeSingleFileArchive(glxPath, privacyTestArchive(), false), "setup: write archive")

	// Drive the dispatcher exactly as Cobra would, with --privatize-living set,
	// to prove runExport forwards exportPrivatizeLiving to the JSON-LD path (a
	// direct exportToJSONLD call would not catch a wiring miss in runExport).
	prevFormat, prevOutput, prevPrivatize := exportFormat, exportOutput, exportPrivatizeLiving
	t.Cleanup(func() {
		exportFormat, exportOutput, exportPrivatizeLiving = prevFormat, prevOutput, prevPrivatize
	})

	outPath := filepath.Join(tmpDir, "out.jsonld")
	exportFormat = ExportFormatJSONLD
	exportOutput = outPath
	exportPrivatizeLiving = true

	require.NoError(t, runExport(nil, []string{glxPath}))

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	out := string(data)
	require.NotContains(t, out, "Modern", "dispatcher must forward --privatize-living to JSON-LD export")
	require.NotContains(t, out, "software engineer", "living person's occupation should not leak via the dispatcher path")
	require.Contains(t, out, "Living", `living person should be redacted to "Living"`)
}

func TestRunExport_DispatchesToJSONLDFormat(t *testing.T) {
	tmpDir := t.TempDir()
	glxPath := filepath.Join(tmpDir, "archive.glx")

	err := importGEDCOM("testdata/gedcom/7.0/minimal-valid/minimal70.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Drive the runExport dispatcher by setting the package-level flag vars
	// the same way Cobra would have populated them.
	previousFormat := exportFormat
	previousOutput := exportOutput
	t.Cleanup(func() {
		exportFormat = previousFormat
		exportOutput = previousOutput
	})

	outPath := filepath.Join(tmpDir, "out.jsonld")
	exportFormat = ExportFormatJSONLD
	exportOutput = outPath

	require.NoError(t, runExport(nil, []string{glxPath}))

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	require.Contains(t, doc, "@context")
}
