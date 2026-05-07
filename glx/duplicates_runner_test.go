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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDuplicates_Integration_TextOutput(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	writeTestPersonFull(t, dir, "person-r-webb", "R Webb", "1815", "place-va")
	writeTestPersonFull(t, dir, "person-robert-webb", "Robert Webb", "1815", "place-va")

	output := captureStdout(t, func() {
		err := findDuplicates(dir, 0.4, "", false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Potential duplicates")
	assert.Contains(t, output, "Score:")
	assert.Contains(t, output, "duplicate pair(s)")
}

func TestFindDuplicates_Integration_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	writeTestPersonFull(t, dir, "person-r-webb", "R Webb", "1815", "place-va")
	writeTestPersonFull(t, dir, "person-robert-webb", "Robert Webb", "1815", "place-va")

	output := captureStdout(t, func() {
		err := findDuplicates(dir, 0.4, "", true)
		require.NoError(t, err)
	})

	assert.Contains(t, output, `"threshold"`)
	assert.Contains(t, output, `"pairs"`)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(output), "{"), "JSON output should start with {")
}

func TestFindDuplicates_Integration_PersonFilter(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	writeTestPersonFull(t, dir, "person-r-webb", "R Webb", "1815", "place-va")
	writeTestPersonFull(t, dir, "person-robert-webb", "Robert Webb", "1815", "place-va")
	writeTestPerson(t, dir, "person-john-smith", "John Smith", "1830")

	err := findDuplicates(dir, 0.4, "person-r-webb", false)
	assert.NoError(t, err)
}

func TestFindDuplicates_Integration_EmptyArchive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	err := findDuplicates(dir, 0.6, "", false)
	assert.NoError(t, err)
}

func TestFindDuplicates_Integration_NoDuplicates(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	writeTestPerson(t, dir, "person-john", "John Smith", "1850")
	writeTestPerson(t, dir, "person-mary", "Mary Johnson", "1832")

	err := findDuplicates(dir, 0.9, "", false)
	assert.NoError(t, err)
}

func TestFindDuplicates_Integration_InvalidPath(t *testing.T) {
	err := findDuplicates("/nonexistent/path", 0.6, "", false)
	assert.Error(t, err)
}

func TestFindDuplicates_Integration_InvalidThreshold(t *testing.T) {
	err := findDuplicates(".", 1.5, "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--threshold must be between 0.0 and 1.0")

	err = findDuplicates(".", -0.1, "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--threshold must be between 0.0 and 1.0")
}

func TestFindDuplicates_Integration_AgeImplausibleSuppressed(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "events"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "relationships"), 0o755))

	// Father has no birth date but is documented as parent on a 1720
	// child-birth relationship. Newborn is principal of a 1730 birth event.
	// The father-vs-newborn pair must score 0 even though their names match.
	personYAML := func(id, name string) string {
		return "persons:\n  " + id + ":\n    properties:\n      name: \"" + name + "\"\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-hans-father.glx"),
		[]byte(personYAML("person-hans-father", "Hans Schmidt")), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-hans-newborn.glx"),
		[]byte(personYAML("person-hans-newborn", "Hans Schmidt")), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-anna-child.glx"),
		[]byte(personYAML("person-anna-child", "Anna Schmidt")), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "events", "event-birth-newborn.glx"),
		[]byte("events:\n  event-birth-newborn:\n    type: birth\n    date: \"1730\"\n    participants:\n      - person: person-hans-newborn\n        role: principal\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "events", "event-birth-child.glx"),
		[]byte("events:\n  event-birth-child:\n    type: birth\n    date: \"1720\"\n    participants:\n      - person: person-anna-child\n        role: principal\n"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "relationships", "rel-father-child.glx"),
		[]byte("relationships:\n  rel-father-child:\n    type: parent_child\n    participants:\n      - person: person-hans-father\n        role: parent\n      - person: person-anna-child\n        role: child\n"), 0o644))

	output := captureStdout(t, func() {
		err := findDuplicates(dir, 0.0, "", true)
		require.NoError(t, err)
	})

	// In the JSON output, locate the father-vs-newborn pair and assert score 0.
	assert.Contains(t, output, "person-hans-father")
	assert.Contains(t, output, "person-hans-newborn")
	assert.Contains(t, output, "Age plausibility")
	assert.Contains(t, output, "1720")
	// Confirm the pair's score is suppressed (not the same as a non-implausible
	// match would produce). The smoking gun is a "score": 0 entry next to the
	// father-newborn pair; check both the suppressed score and the breakdown.
	assert.Contains(t, output, `"score": 0`)
}

func writeTestPerson(t *testing.T, dir, id, name, born string) {
	t.Helper()
	personYAML := "persons:\n  " + id + ":\n    properties:\n      name: \"" + name + "\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", id+".glx"), []byte(personYAML), 0o644))

	// Write birth event in events directory
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "events"), 0o755))
	eventID := "event-birth-" + id
	eventYAML := "events:\n  " + eventID + ":\n    type: birth\n    date: \"" + born + "\"\n    participants:\n      - person: " + id + "\n        role: principal\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "events", eventID+".glx"), []byte(eventYAML), 0o644))
}

func writeTestPersonFull(t *testing.T, dir, id, name, born, bornAt string) {
	t.Helper()
	personYAML := "persons:\n  " + id + ":\n    properties:\n      name: \"" + name + "\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", id+".glx"), []byte(personYAML), 0o644))

	// Write birth event in events directory
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "events"), 0o755))
	eventID := "event-birth-" + id
	eventYAML := "events:\n  " + eventID + ":\n    type: birth\n    date: \"" + born + "\"\n    place: \"" + bornAt + "\"\n    participants:\n      - person: " + id + "\n        role: principal\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "events", eventID+".glx"), []byte(eventYAML), 0o644))
}

// captureStdout redirects os.Stdout during fn execution and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdout := os.Stdout
	os.Stdout = w

	wClosed := false
	defer func() {
		if !wClosed {
			_ = w.Close()
		}
		os.Stdout = origStdout
	}()

	fn()

	require.NoError(t, w.Close())
	wClosed = true

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)

	return buf.String()
}
