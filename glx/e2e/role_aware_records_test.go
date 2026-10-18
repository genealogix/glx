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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The reproductions of #1370, #1363 and #1362, run end to end against one
// parish-register archive built with glx add.
func TestRoleAwareRecords_ParishRegister(t *testing.T) {
	root := t.TempDir()
	res := runGLX(t, root, "init", "parish", "--no-git")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	archive := filepath.Join(root, "parish")

	for _, args := range [][]string{
		{"add", "place", "--id", "pl-hre", "--name", "Heiliges Römisches Reich", "--type", "country"},
		{"add", "place", "--id", "pl-amt", "--name", "Amt Hüttenberg", "--type", "district", "--parent", "pl-hre"},
		{"add", "place", "--id", "pl-hd", "--name", "Landgrafschaft Hessen-Darmstadt", "--type", "state", "--parent", "pl-hre"},
		{"add", "place", "--id", "pl-nk", "--name", "Niederkleen", "--type", "village", "--parent", "pl-amt"},
		{"add", "place", "--id", "pl-pg", "--name", "Pohl-Göns", "--type", "village", "--parent", "pl-amt"},
		{"add", "place", "--id", "pl-bz", "--name", "Butzbach", "--type", "town", "--parent", "pl-hd"},
		{"add", "person", "--id", "p-jg", "--given", "Johann Georg", "--surname", "Schöpff", "--sex", "male"},
		{"add", "person", "--id", "p-bride", "--given", "Elisabeth", "--surname", "Roth", "--sex", "female"},
		{"add", "person", "--id", "p-father", "--given", "Wörner", "--surname", "Roth", "--sex", "male"},
		{"add", "person", "--id", "p-child", "--given", "Michell", "--surname", "Schöpff", "--sex", "male"},
		{"add", "event", "--id", "ev-birth", "--type", "birth", "--date", "ABT 1621", "--place", "pl-nk", "--principal", "p-jg"},
		{
			"add", "event", "--id", "ev-marr", "--type", "marriage", "--date", "1644-02-26", "--place", "pl-pg",
			"--participant", "p-jg:groom", "--participant", "p-bride:bride", "--participant", "p-father:parent",
		},
		{
			"add", "event", "--id", "ev-bapt", "--type", "baptism", "--date", "1645-10-12", "--place", "pl-pg",
			"--principal", "p-child", "--participant", "p-jg:parent",
		},
		{
			"add", "event", "--id", "ev-burial-child", "--type", "burial", "--date", "1646-08-07", "--place", "pl-bz",
			"--principal", "p-child", "--participant", "p-jg:parent",
		},
		{
			"add", "event", "--id", "ev-death-bride", "--type", "death", "--date", "1688-05-04", "--place", "pl-pg",
			"--principal", "p-bride", "--participant", "p-jg:spouse",
		},
		{"add", "event", "--id", "ev-death", "--type", "death", "--date", "1689-12-27", "--place", "pl-pg", "--principal", "p-jg"},
	} {
		res := runGLX(t, archive, args...)
		require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	}

	migrations := runGLX(t, archive, "migrations", "p-jg")
	require.Equal(t, 0, migrations.exitCode, migrations.stdout+migrations.stderr)
	assert.Contains(t, migrations.stdout, "Niederkleen → Pohl-Göns (ABT 1621 – February 26, 1644)")
	assert.Contains(t, migrations.stdout, `[not counted as a move: role "parent" on a burial places the deceased, not this person]`)
	assert.NotContains(t, migrations.stdout, "Landgrafschaft Hessen-Darmstadt →")

	timeline := runGLX(t, archive, "timeline", "p-jg")
	require.Equal(t, 0, timeline.exitCode, timeline.stdout+timeline.stderr)
	assert.Contains(t, timeline.stdout, "Burial of Michell Schöpff (parent)")
	assert.Contains(t, timeline.stdout, "Death of Elisabeth Roth (spouse)")

	summary := runGLX(t, archive, "summary", "p-jg")
	require.Equal(t, 0, summary.exitCode, summary.stdout+summary.stderr)
	assert.Contains(t, summary.stdout, "He died on December 27, 1689")
	assert.NotContains(t, summary.stdout, "Wörner Roth", "the bride's father is not a spouse")

	father := runGLX(t, archive, "summary", "p-father")
	require.Equal(t, 0, father.exitCode, father.stdout+father.stderr)
	assert.Contains(t, father.stdout, "Spouse:           (none)")
}
