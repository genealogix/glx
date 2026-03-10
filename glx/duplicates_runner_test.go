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
)

func TestFindDuplicates_Integration_TextOutput(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	writeTestPersonFull(t, dir, "person-d-lane", "D Lane", "1815", "place-va")
	writeTestPersonFull(t, dir, "person-daniel-lane", "Daniel Lane", "1815", "place-va")

	err := findDuplicates(dir, 0.4, "", false)
	assert.NoError(t, err)
}

func TestFindDuplicates_Integration_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	writeTestPersonFull(t, dir, "person-d-lane", "D Lane", "1815", "place-va")
	writeTestPersonFull(t, dir, "person-daniel-lane", "Daniel Lane", "1815", "place-va")

	err := findDuplicates(dir, 0.4, "", true)
	assert.NoError(t, err)
}

func TestFindDuplicates_Integration_PersonFilter(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	writeTestPersonFull(t, dir, "person-d-lane", "D Lane", "1815", "place-va")
	writeTestPersonFull(t, dir, "person-daniel-lane", "Daniel Lane", "1815", "place-va")
	writeTestPerson(t, dir, "person-john-smith", "John Smith", "1830")

	err := findDuplicates(dir, 0.4, "person-d-lane", false)
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

func writeTestPerson(t *testing.T, dir, id, name, born string) {
	t.Helper()
	yaml := "persons:\n  " + id + ":\n    properties:\n      name: \"" + name + "\"\n      born_on: \"" + born + "\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", id+".glx"), []byte(yaml), 0o644))
}

func writeTestPersonFull(t *testing.T, dir, id, name, born, bornAt string) {
	t.Helper()
	yaml := "persons:\n  " + id + ":\n    properties:\n      name: \"" + name + "\"\n      born_on: \"" + born + "\"\n      born_at: \"" + bornAt + "\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", id+".glx"), []byte(yaml), 0o644))
}
