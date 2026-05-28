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

func TestCountLegacyMediaDescriptions(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want int
	}{
		{
			name: "top-level description counts",
			yaml: "media:\n  media-1:\n    uri: media/files/x.jpg\n    description: legacy\n",
			want: 1,
		},
		{
			name: "properties.description does not count",
			yaml: "media:\n  media-1:\n    uri: media/files/x.jpg\n    properties:\n      description: prop\n",
			want: 0,
		},
		{
			name: "both present: legacy top-level still counted (must be removed on save)",
			yaml: "media:\n  media-1:\n    uri: media/files/x.jpg\n    description: legacy\n    properties:\n      description: prop\n",
			want: 1,
		},
		{
			name: "no media block (e.g. vocabulary file)",
			yaml: "media_properties:\n  description:\n    label: Media Description\n    value_type: string\n",
			want: 0,
		},
		{
			name: "multiple media, only those with top-level description",
			yaml: "media:\n  m1:\n    uri: a.jpg\n    description: d1\n  m2:\n    uri: b.jpg\n    description: d2\n  m3:\n    uri: c.jpg\n",
			want: 2,
		},
		{
			name: "malformed yaml returns 0",
			yaml: "media: [",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countLegacyMediaDescriptions([]byte(tt.yaml)))
		})
	}
}

func TestMigrateMediaDescriptions_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	content := "media:\n  media-1:\n    uri: media/files/portrait.jpg\n    description: Studio portrait\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	assert.Equal(t, 1, migrateMediaDescriptions(path, false))
}

func TestMigrateMediaDescriptions_MultiFileDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "media-1.glx"),
		[]byte("media:\n  media-1:\n    uri: a.jpg\n    description: legacy\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "media-2.glx"),
		[]byte("media:\n  media-2:\n    uri: b.jpg\n    properties:\n      description: prop\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "media-3.glx"),
		[]byte("media:\n  media-3:\n    uri: c.jpg\n    description: also-legacy\n"), 0o644))

	// Two of the three media entities carry a legacy top-level description; the
	// one with an explicit properties.description is not counted.
	assert.Equal(t, 2, migrateMediaDescriptions(dir, true))
}

func TestMigrateMediaDescriptions_ErrorsReturnZero(t *testing.T) {
	// An unreadable single file and an unreadable directory both degrade
	// gracefully to 0 (a real archive was already loaded before this runs).
	assert.Equal(t, 0, migrateMediaDescriptions(filepath.Join(t.TempDir(), "missing.glx"), false))
	assert.Equal(t, 0, migrateMediaDescriptions(filepath.Join(t.TempDir(), "missing-dir"), true))
}

// TestMigrateArchive_MediaDescriptionToProperty_EndToEnd exercises the full
// flag path: a legacy archive is loaded (the shim folds description into
// properties), counted, and re-saved in the new on-disk form.
func TestMigrateArchive_MediaDescriptionToProperty_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	content := "media:\n  media-1:\n    uri: media/files/portrait.jpg\n    description: Studio portrait taken in Leeds\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	migrateMediaDescriptionToProperty = true
	defer func() { migrateMediaDescriptionToProperty = false }()

	require.NoError(t, migrateArchive(path))

	out, err := os.ReadFile(path)
	require.NoError(t, err)

	var doc struct {
		Media map[string]struct {
			Description string         `yaml:"description"`
			Properties  map[string]any `yaml:"properties"`
		} `yaml:"media"`
	}
	require.NoError(t, yaml.Unmarshal(out, &doc))

	m := doc.Media["media-1"]
	assert.Empty(t, m.Description, "legacy top-level description should be gone after migration")
	assert.Equal(t, "Studio portrait taken in Leeds", m.Properties["description"],
		"description should now live under properties")

	// Idempotent: a second scan finds nothing left to migrate.
	assert.Equal(t, 0, migrateMediaDescriptions(path, false))
}

// A media entity with BOTH a legacy top-level description and an explicit
// properties.description must still be re-saved so the schema-invalid top-level
// key is removed from disk; the explicit property is preserved (no-clobber).
func TestMigrateArchive_MediaDescriptionToProperty_RemovesLegacyDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	content := "media:\n  media-1:\n    uri: media/files/x.jpg\n    description: legacy\n    properties:\n      description: explicit\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	migrateMediaDescriptionToProperty = true
	defer func() { migrateMediaDescriptionToProperty = false }()
	require.NoError(t, migrateArchive(path))

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		Media map[string]struct {
			Description string         `yaml:"description"`
			Properties  map[string]any `yaml:"properties"`
		} `yaml:"media"`
	}
	require.NoError(t, yaml.Unmarshal(out, &doc))

	m := doc.Media["media-1"]
	assert.Empty(t, m.Description, "schema-invalid top-level description must be removed from disk")
	assert.Equal(t, "explicit", m.Properties["description"], "explicit properties.description is preserved (no-clobber)")
	assert.Equal(t, 0, migrateMediaDescriptions(path, false), "idempotent after migration")
}
