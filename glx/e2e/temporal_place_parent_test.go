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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// temporalParentArchive stages basic-family with Springfield's parent made
// temporal (#225): Illinois Territory until statehood on 3 Dec 1818, then
// Illinois. A settler born in 1815 falls in the territorial period.
func temporalParentArchive(t *testing.T) string {
	t.Helper()
	archive := copyExample(t, "basic-family")
	write := func(rel, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(archive, rel), []byte(content), 0o644))
	}
	write(filepath.Join("places", "place-illinois-territory.glx"), `places:
  place-illinois-territory:
    name: "Illinois Territory"
    parent: place-united-states
`)
	write(filepath.Join("places", "place-springfield.glx"), `places:
  place-springfield:
    name: "Springfield"
    type: city
    parent:
      - {value: place-illinois-territory, date: "FROM 1809 TO 1818-12-02"}
      - {value: place-illinois, date: "FROM 1818-12-03"}
`)
	write(filepath.Join("persons", "person-early-settler.glx"), `persons:
  person-early-settler:
    properties:
      name: "Early Settler"
`)
	write(filepath.Join("events", "event-birth-1815.glx"), `events:
  event-birth-1815:
    type: birth
    date: "1815-06-15"
    place: place-springfield
    participants:
      - person: person-early-settler
        role: subject
`)

	return archive
}

func TestTemporalPlaceParent_Validates(t *testing.T) {
	archive := temporalParentArchive(t)

	res := runGLX(t, archive, "validate", ".")

	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.NotContains(t, res.stdout+res.stderr, "parent")
}

func TestTemporalPlaceParent_ValidateRejectsBadShapeAndRefs(t *testing.T) {
	archive := temporalParentArchive(t)
	path := filepath.Join(archive, "places", "place-springfield.glx")

	require.NoError(t, os.WriteFile(path, []byte(`places:
  place-springfield:
    name: "Springfield"
    parent:
      - {value: place-illinois, when: "1818"}
`), 0o644))
	res := runGLX(t, archive, "validate", ".")
	assert.NotEqual(t, 0, res.exitCode, "an unknown key in a parent entry fails the schema")

	require.NoError(t, os.WriteFile(path, []byte(`places:
  place-springfield:
    name: "Springfield"
    parent:
      - {value: place-nowhere, date: "TO 1818"}
      - {value: place-illinois, date: "FROM 1818"}
`), 0o644))
	res = runGLX(t, archive, "validate", ".")
	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stdout+res.stderr, "place-nowhere")
}

func TestTemporalPlaceParent_GEDCOMExportUsesEventDate(t *testing.T) {
	archive := temporalParentArchive(t)
	work := t.TempDir()

	res := runGLX(t, work, "export", archive, "-o", "family.ged", "--format", "551")

	require.Equal(t, 0, res.exitCode, res.stderr)
	ged, err := os.ReadFile(filepath.Join(work, "family.ged"))
	require.NoError(t, err)
	assert.Contains(t, string(ged), "2 DATE 15 JUN 1815\n2 PLAC Springfield, Illinois Territory, United States\n")
	assert.Contains(t, string(ged), "2 DATE 15 JUN 1875\n2 PLAC Springfield, Illinois, United States\n")
}

// join writes the temporal list back out, and the joined file still validates.
func TestTemporalPlaceParent_JoinSplitRoundTrip(t *testing.T) {
	archive := temporalParentArchive(t)
	work := t.TempDir()

	require.Equal(t, 0, runGLX(t, work, "join", archive, "family.glx").exitCode)
	joined, err := os.ReadFile(filepath.Join(work, "family.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(joined), "- value: place-illinois-territory\n")
	assert.Contains(t, string(joined), `date: "FROM 1818-12-03"`)
	res := runGLX(t, work, "validate", "family.glx")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	require.Equal(t, 0, runGLX(t, work, "split", "family.glx", "split").exitCode)
	diff := runGLX(t, work, "diff", archive, "split")
	require.Equal(t, 0, diff.exitCode, diff.stderr)
	assert.Contains(t, diff.stdout, "No changes.")
}
