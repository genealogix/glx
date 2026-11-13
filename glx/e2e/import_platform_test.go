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

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ancestryExport = `0 HEAD
1 SOUR Ancestry.com Family Trees
2 NAME Ancestry.com Member Trees
2 CORP Ancestry.com
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 DEAT
2 DATE 1952
1 _DCAUSE Pneumonia
1 _MILT US Army
2 DATE 1918
2 SOUR @S1@
3 _APID 1,6482::4512345
0 @S1@ SOUR
1 TITL Draft Cards
0 TRLR
`

func writeGEDCOM(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "tree.ged")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestImport_ListPlatforms(t *testing.T) {
	res := runGLX(t, t.TempDir(), "import", "--list-platforms")
	require.Equal(t, 0, res.exitCode, res.stderr)
	for _, name := range []string{"ancestry", "findmypast", "ftm", "legacy", "myheritage", "rootsmagic", "generic"} {
		assert.Contains(t, res.stdout, name)
	}

	assertExitWithStderr(t, runGLX(t, t.TempDir(), "import", "--list-platforms", "tree.ged"), "unknown command")
}

func TestImport_DetectsAncestryAndMapsItsTags(t *testing.T) {
	work := t.TempDir()
	ged := writeGEDCOM(t, work, ancestryExport)

	res := runGLX(t, work, "import", ged, "-o", "archive", "-v")
	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Platform: Ancestry export")
	assert.Contains(t, res.stdout, "_MILT: 1 rewritten")
	assert.Contains(t, res.stdout, "_DCAUSE: 1 rewritten")
	assert.Contains(t, res.stdout, "_APID: 1 rewritten")

	archive := filepath.Join(work, "archive")
	events := readArchiveDir(t, filepath.Join(archive, "events"))
	assert.Contains(t, events, "type: military_service")
	assert.Contains(t, events, "cause: Pneumonia")
	citations := readArchiveDir(t, filepath.Join(archive, "citations"))
	assert.Contains(t, citations, "https://www.ancestry.com/discoveryui-content/view/4512345:6482")
	assertArchiveValid(t, archive)
}

func TestImport_ForcedPlatform(t *testing.T) {
	work := t.TempDir()
	ged := writeGEDCOM(t, work, strings.Replace(ancestryExport, "1 SOUR Ancestry.com Family Trees\n2 NAME Ancestry.com Member Trees\n2 CORP Ancestry.com\n", "1 SOUR OTHER\n", 1))

	res := runGLX(t, work, "import", ged, "-o", "generic", "-v")
	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "no platform-specific handling")

	res = runGLX(t, work, "import", ged, "-o", "forced", "-v", "--platform", "ancestry")
	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Platform: Ancestry export")
	assert.Contains(t, readArchiveDir(t, filepath.Join(work, "forced", "events")), "type: military_service")

	assertExitWithStderr(t, runGLX(t, work, "import", ged, "-o", "bad", "--platform", "geni"), "unknown GEDCOM platform")
	assert.NoDirExists(t, filepath.Join(work, "bad"))
}

// readArchiveDir concatenates every file in dir.
func readArchiveDir(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var all strings.Builder
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err)
		all.Write(data)
	}

	return all.String()
}
