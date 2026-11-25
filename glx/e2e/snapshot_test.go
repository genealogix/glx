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
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateSnapshots rewrites the CLI output snapshots from the current binary:
//
//	go test ./glx/e2e/ -run TestReadCommands_OutputSnapshots -update
//
// Review the resulting diff: every changed line is an output change a user,
// a script or a documentation example will see.
var updateSnapshots = flag.Bool("update", false, "rewrite CLI output snapshots from current output")

// snapshotDir holds one file per read command and output format.
const snapshotDir = "testdata/snapshots"

// TestReadCommands_OutputSnapshots pins the complete stdout of every read
// command in readCases, in its text form and, where it has one, its JSON
// form, run against basic-family (#301). The other read-command tests check
// one line each; this one fails on any change to the output's wording,
// layout, order or JSON shape, so a change is made on purpose and reviewed.
func TestReadCommands_OutputSnapshots(t *testing.T) {
	archive := copyExample(t, "basic-family")

	for _, tc := range readCases {
		variants := []struct {
			suffix string
			args   []string
		}{{".txt", tc.args}}
		if tc.jsonArgs != nil {
			variants = append(variants, struct {
				suffix string
				args   []string
			}{".json", append(append([]string{}, tc.args...), tc.jsonArgs...)})
		}

		for _, v := range variants {
			name := strings.ReplaceAll(tc.name, " ", "-") + v.suffix
			t.Run(name, func(t *testing.T) {
				res := runGLX(t, archive, v.args...)
				require.Equal(t, 0, res.exitCode, res.stderr)
				assertSnapshot(t, name, res.stdout)
			})
		}
	}
}

// assertSnapshot compares got against the named snapshot, or rewrites the
// snapshot under -update. Line endings are normalized on both sides, so a
// checkout that converts the snapshot files to CRLF still compares equal.
func assertSnapshot(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join(snapshotDir, name)
	got = strings.ReplaceAll(got, "\r\n", "\n")
	if *updateSnapshots {
		require.NoError(t, os.MkdirAll(snapshotDir, 0o755))
		require.NoError(t, os.WriteFile(path, []byte(got), 0o600))

		return
	}

	want, err := os.ReadFile(path)
	require.NoError(t, err, "missing snapshot %s: run go test ./glx/e2e/ -run TestReadCommands_OutputSnapshots -update", path)
	assert.Equal(t, strings.ReplaceAll(string(want), "\r\n", "\n"), got,
		"output of %s changed; if the change is intended, rerun with -update and review the snapshot diff", name)
}
