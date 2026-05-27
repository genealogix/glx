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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCountLegacySourceDescriptions(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want int
	}{
		{
			name: "top-level description counts",
			yaml: "sources:\n  source-1:\n    title: T\n    description: legacy\n",
			want: 1,
		},
		{
			name: "properties.description does not count",
			yaml: "sources:\n  source-1:\n    title: T\n    properties:\n      description: prop\n",
			want: 0,
		},
		{
			name: "both present: legacy top-level still counted (must be removed on save)",
			yaml: "sources:\n  source-1:\n    title: T\n    description: legacy\n    properties:\n      description: prop\n",
			want: 1,
		},
		{
			name: "no sources block (e.g. vocabulary file)",
			yaml: "source_properties:\n  description:\n    label: Source Description\n    value_type: string\n",
			want: 0,
		},
		{
			name: "multiple sources, only those with top-level description",
			yaml: "sources:\n  s1:\n    title: A\n    description: d1\n  s2:\n    title: B\n    description: d2\n  s3:\n    title: C\n",
			want: 2,
		},
		{
			name: "malformed yaml returns 0",
			yaml: "sources: [",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countLegacySourceDescriptions([]byte(tt.yaml)))
		})
	}
}

func TestMigrateSourceDescriptions_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	content := "sources:\n  source-1:\n    title: Parish Register\n    description: Baptisms\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	assert.Equal(t, 1, migrateSourceDescriptions(path, false))
}

func TestMigrateSourceDescriptions_MultiFileDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "source-1.glx"),
		[]byte("sources:\n  source-1:\n    title: A\n    description: legacy\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "source-2.glx"),
		[]byte("sources:\n  source-2:\n    title: B\n    properties:\n      description: prop\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "source-3.glx"),
		[]byte("sources:\n  source-3:\n    title: C\n    description: also-legacy\n"), 0o644))

	// Two of the three sources carry a legacy top-level description; the one
	// with an explicit properties.description is not counted.
	assert.Equal(t, 2, migrateSourceDescriptions(dir, true))
}

func TestMigrateSourceDescriptions_ErrorsReturnZero(t *testing.T) {
	// An unreadable single file and an unreadable directory both degrade
	// gracefully to 0 (a real archive was already loaded before this runs).
	assert.Equal(t, 0, migrateSourceDescriptions(filepath.Join(t.TempDir(), "missing.glx"), false))
	assert.Equal(t, 0, migrateSourceDescriptions(filepath.Join(t.TempDir(), "missing-dir"), true))
}

// TestMigrateArchive_SourceDescriptionToProperty_EndToEnd exercises the full
// flag path: a legacy archive is loaded (the shim folds description into
// properties), counted, and re-saved in the new on-disk form.
func TestMigrateArchive_SourceDescriptionToProperty_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	content := "sources:\n  source-1:\n    title: Parish Register\n    description: Baptisms, marriages, burials\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	migrateSourceDescriptionToProperty = true
	defer func() { migrateSourceDescriptionToProperty = false }()

	require.NoError(t, migrateArchive(path))

	out, err := os.ReadFile(path)
	require.NoError(t, err)

	var doc struct {
		Sources map[string]struct {
			Description string         `yaml:"description"`
			Properties  map[string]any `yaml:"properties"`
		} `yaml:"sources"`
	}
	require.NoError(t, yaml.Unmarshal(out, &doc))

	src := doc.Sources["source-1"]
	assert.Empty(t, src.Description, "legacy top-level description should be gone after migration")
	assert.Equal(t, "Baptisms, marriages, burials", src.Properties["description"],
		"description should now live under properties")

	// Idempotent: a second scan finds nothing left to migrate.
	assert.Equal(t, 0, migrateSourceDescriptions(path, false))
}

// A source with BOTH a legacy top-level description and an explicit
// properties.description must still be re-saved so the schema-invalid top-level
// key is removed from disk; the explicit property is preserved (no-clobber).
func TestMigrateArchive_SourceDescriptionToProperty_RemovesLegacyDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	content := "sources:\n  source-1:\n    title: T\n    description: legacy\n    properties:\n      description: explicit\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	migrateSourceDescriptionToProperty = true
	defer func() { migrateSourceDescriptionToProperty = false }()
	require.NoError(t, migrateArchive(path))

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		Sources map[string]struct {
			Description string         `yaml:"description"`
			Properties  map[string]any `yaml:"properties"`
		} `yaml:"sources"`
	}
	require.NoError(t, yaml.Unmarshal(out, &doc))

	src := doc.Sources["source-1"]
	assert.Empty(t, src.Description, "schema-invalid top-level description must be removed from disk")
	assert.Equal(t, "explicit", src.Properties["description"], "explicit properties.description is preserved (no-clobber)")
	assert.Equal(t, 0, migrateSourceDescriptions(path, false), "idempotent after migration")
}
