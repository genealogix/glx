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
	err = exportToJSONLD(glxPath, outPath, false)
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
	require.NoError(t, exportToJSONLD(glxPath, nestedOut, false))

	_, err = os.Stat(nestedOut)
	require.NoError(t, err, "nested directories should be created")
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
