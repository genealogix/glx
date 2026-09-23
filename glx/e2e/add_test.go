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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdd_ProgressLineIsSingular pins the user-facing shape of the "Adding"
// line. It used to interpolate the plural EntityType — the YAML key and
// directory name — straight against the ID ("Adding persons person-x"), which
// reads as a list rather than a sentence. See issue #1276.
func TestAdd_ProgressLineIsSingular(t *testing.T) {
	archive := copyExample(t, "basic-family")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "person",
			args: []string{"add", "person", "--given", "Michael David", "--surname", "Hollnagel"},
			want: "Adding person: person-michael-david-hollnagel",
		},
		{
			name: "place",
			args: []string{"add", "place", "--name", "Liepen", "--type", "locality"},
			want: "Adding place: place-liepen",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runGLX(t, archive, tc.args...)

			require.Equal(t, 0, res.exitCode, res.stderr)
			assert.Contains(t, res.stdout, tc.want+"\n")
		})
	}
}

// TestAdd_LastStdoutLineIsTheBareID guards the documented contract for
// `id=$(glx add …)`: the progress line may change wording, the ID echo may
// not. See issue #1276.
func TestAdd_LastStdoutLineIsTheBareID(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, archive, "add", "person", "--given", "Anna", "--surname", "Jungk")

	require.Equal(t, 0, res.exitCode, res.stderr)
	lines := strings.Split(strings.TrimRight(res.stdout, "\n"), "\n")
	assert.Equal(t, "person-anna-jungk", lines[len(lines)-1])
}

// TestAdd_QuietDropsTheProgressLineButKeepsTheID checks that the progress line
// is diagnostic output (suppressed by --quiet) while the ID echo survives for
// shell capture. See issue #1276.
func TestAdd_QuietDropsTheProgressLineButKeepsTheID(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, archive, "--quiet", "add", "person", "--given", "Quiet", "--surname", "Person")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.NotContains(t, res.stdout, "Adding")
	assert.Equal(t, "person-quiet-person", strings.TrimSpace(res.stdout))
}
