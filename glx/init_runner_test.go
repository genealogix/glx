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

	glxlib "github.com/genealogix/glx/go-glx"
)

// TestStructToMapNilInput verifies a nil/empty source returns an empty map
// without error rather than panicking.
func TestStructToMapNilInput(t *testing.T) {
	m, err := structToMap(map[string]*glxlib.Person(nil))
	require.NoError(t, err)
	assert.Empty(t, m)
}

// TestStructToMapRoundTrip verifies a populated typed map round-trips into
// `map[string]any` shape preserving keys.
func TestStructToMapRoundTrip(t *testing.T) {
	src := map[string]*glxlib.Person{
		"person-1": {Properties: map[string]any{"primary_name": "Test One"}},
		"person-2": {Properties: map[string]any{"primary_name": "Test Two"}},
	}
	m, err := structToMap(src)
	require.NoError(t, err)
	assert.Contains(t, m, "person-1")
	assert.Contains(t, m, "person-2")
}

// TestWriteTestDataFile verifies a single entity is written under
// `<dir>/<id>.glx` with the correct YAML envelope and that callers receive
// a wrapped error on write failure (not a panic).
func TestWriteTestDataFile(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "persons"), 0o755))

	chdirTo(t, tmp)

	err := writeTestDataFile("persons", "person-test", map[string]any{
		"primary_name": "Test Person",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmp, "persons", "person-test.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "persons:")
	assert.Contains(t, string(data), "person-test:")
	assert.Contains(t, string(data), "Test Person")
}

// TestWriteTestDataFileBadDir verifies that a missing destination directory
// surfaces a wrapped error referencing the file path rather than panicking.
func TestWriteTestDataFileBadDir(t *testing.T) {
	tmp := t.TempDir()
	chdirTo(t, tmp)

	err := writeTestDataFile("does/not/exist", "person-x", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does/not/exist")
}

// chdirTo cds to dir and restores the previous cwd at test teardown.
func chdirTo(t *testing.T, dir string) {
	t.Helper()
	t.Chdir(dir)
}
