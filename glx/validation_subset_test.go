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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSubsetArchive lays out a directory archive with no vocabularies/
// directory: its event and source types come from the standard vocabularies.
func writeSubsetArchive(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, "persons", "person-1.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")
	writeSkipTestFile(t, filepath.Join(root, "places", "place-1.glx"),
		"places:\n  place-1:\n    name: Leeds\n")
	writeSkipTestFile(t, filepath.Join(root, "events", "event-1.glx"),
		"events:\n  event-1:\n    type: birth\n    place: place-1\n    participants:\n      - person: person-1\n")

	return root
}

// `glx validate events` checks events/ against the archive around it (#1206).
func TestValidatePaths_EntityDirectoryUsesArchiveContext(t *testing.T) {
	root := writeSubsetArchive(t)
	events := filepath.Join(root, "events")

	t.Run("references and types resolve", func(t *testing.T) {
		streams, out, errOut := newTestStreams()
		require.NoError(t, validatePaths(streams, []string{events}), errOut.String())
		assert.Contains(t, out.String(), "resolved against the archive")
		assert.Contains(t, out.String(), "Validated 1 files.")
	})

	t.Run("a broken reference in the selection is reported, one outside it is not", func(t *testing.T) {
		writeSkipTestFile(t, filepath.Join(root, "events", "event-2.glx"),
			"events:\n  event-2:\n    type: death\n    place: place-missing\n    participants:\n      - person: person-1\n")
		writeSkipTestFile(t, filepath.Join(root, "assertions", "assertion-1.glx"),
			"assertions:\n  assertion-1:\n    subject:\n      person: person-1\n    property: occupation\n    value: smith\n    citations:\n      - citation-missing\n")
		streams, _, errOut := newTestStreams()
		require.ErrorIs(t, validatePaths(streams, []string{events}), ErrValidationFailed)
		assert.Contains(t, errOut.String(), "place-missing")
		assert.NotContains(t, errOut.String(), "citation-missing", "assertions/ was not selected")
	})
}

// A directory that is not inside an archive is validated on its own, as
// before: nothing around it is guessed at as context.
func TestValidatePaths_DirectoryOutsideAnArchiveHasNoContext(t *testing.T) {
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, "notes", "event-1.glx"),
		"events:\n  event-1:\n    type: birth\n    participants:\n      - person: person-1\n")
	streams, out, errOut := newTestStreams()
	require.Error(t, validatePaths(streams, []string{filepath.Join(root, "notes")}))
	assert.NotContains(t, out.String(), "resolved against the archive")
	assert.Contains(t, errOut.String(), "person-1")
}

// A directory archive without a vocabulary file accepts the standard
// vocabulary's values, as `glx add` and single-file validation do (#1355).
func TestValidatePaths_DirectoryArchiveFallsBackToStandardVocabularies(t *testing.T) {
	root := writeSubsetArchive(t)
	writeSkipTestFile(t, filepath.Join(root, "sources", "source-1.glx"),
		"sources:\n  source-1:\n    title: Births\n    type: vital_record\n")
	streams, out, errOut := newTestStreams()
	require.NoError(t, validatePaths(streams, []string{root}), errOut.String())
	assert.Contains(t, out.String(), "Archive is valid")

	t.Run("a vocabulary the archive defines stays authoritative", func(t *testing.T) {
		writeSkipTestFile(t, filepath.Join(root, "vocabularies", "source-types.glx"),
			"source_types:\n  diary:\n    label: Diary\n")
		streams, _, errOut := newTestStreams()
		require.Error(t, validatePaths(streams, []string{root}))
		assert.Contains(t, errOut.String(), "vital_record")
	})
}

// IDs differing only by case collide (#696), and a duplicate ID names both
// files (#1232).
func TestValidatePaths_IDCollisionsNameTheirFiles(t *testing.T) {
	t.Run("case-insensitive collision", func(t *testing.T) {
		root := t.TempDir()
		writeSkipTestFile(t, filepath.Join(root, "persons", "a.glx"), "persons:\n  Person-A:\n    properties:\n      primary_name: Alice\n")
		writeSkipTestFile(t, filepath.Join(root, "persons", "b.glx"), "persons:\n  person-a:\n    properties:\n      primary_name: Bob\n")
		streams, _, errOut := newTestStreams()
		require.Error(t, validatePaths(streams, []string{root}))
		assert.Contains(t, errOut.String(), `"Person-A" and "person-a" differ only by case`)
	})

	t.Run("exact duplicate", func(t *testing.T) {
		root := t.TempDir()
		writeSkipTestFile(t, filepath.Join(root, "persons", "a.glx"), "persons:\n  person-a:\n    properties:\n      primary_name: Alice\n")
		writeSkipTestFile(t, filepath.Join(root, "persons", "b.glx"), "persons:\n  person-a:\n    properties:\n      primary_name: Bob\n")
		streams, _, errOut := newTestStreams()
		require.Error(t, validatePaths(streams, []string{root}))
		assert.Contains(t, errOut.String(), "conflict persons ID: person-a (persons")
		assert.Contains(t, errOut.String(), "a.glx vs persons")
	})
}
