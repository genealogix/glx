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
			// Progress is diagnostic output on stderr; stdout is the ID alone.
			assert.Contains(t, res.stderr, tc.want+"\n")
			assert.NotContains(t, res.stdout, "Adding")
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

// Each add subcommand run against basic-family creates exactly one file named
// after the new ID and touches nothing else. The IDs are derived from the
// flags, so they are part of the contract being checked.
func TestAdd_EachSubcommandCreatesExactlyOneFile(t *testing.T) {
	cases := []struct {
		name string
		args []string
		id   string
		dir  string
	}{
		{"person", []string{"person", "--given", "Jane", "--surname", "Doe", "--sex", "female"}, "person-jane-doe", "persons"},
		{"place", []string{"place", "--name", "Chatham", "--type", "city", "--parent", "place-illinois"}, "place-chatham", "places"},
		{"event", []string{"event", "--type", "death", "--date", "1920", "--principal", "person-robert-thompson", "--place", "place-springfield"}, "event-death-robert-thompson", "events"},
		{"repository", []string{"repository", "--name", "State Archive", "--type", "archive"}, "repository-state-archive", "repositories"},
		{"source", []string{"source", "--title", "Illinois Deaths", "--type", "vital_record"}, "source-illinois-deaths", "sources"},
		{"citation", []string{"citation", "--source", "source-sangamon-births", "--id", "citation-e2e", "--locator", "p. 9"}, "citation-e2e", "citations"},
		{"relationship", []string{"relationship", "--type", "parent_child", "--parent", "person-robert-thompson", "--child", "person-mary-thompson"}, "relationship-parent-child-robert-thompson-mary-thompson", "relationships"},
		{"assertion", []string{"assertion", "--subject-person", "person-mary-thompson", "--property", "sex", "--value", "female", "--citation", "citation-robert-birth"}, "assertion-mary-thompson-sex", "assertions"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := copyExample(t, "basic-family")
			before := snapshotTree(t, archive)

			res := runGLX(t, archive, append([]string{"add"}, tc.args...)...)

			require.Equal(t, 0, res.exitCode, res.stderr)
			assert.Equal(t, tc.id+"\n", res.stdout, "stdout must be the created ID and nothing else")
			assert.Contains(t, res.stderr, "Adding "+tc.name+": "+tc.id)
			diff := diffTrees(before, snapshotTree(t, archive))
			assert.Empty(t, diff.changed)
			assert.Equal(t, []string{tc.dir + "/" + tc.id + ".glx"}, diff.created)
			assert.Empty(t, diff.removed)
			assertArchiveValid(t, archive)
		})
	}
}

// The documented `pid=$(glx add person …)` capture: stdout carries only the ID
// with and without --quiet, and --quiet silences the progress line.
func TestAdd_StdoutIsOnlyTheID(t *testing.T) {
	archive := copyExample(t, "basic-family")

	plain := runGLX(t, archive, "add", "person", "--given", "Anna", "--surname", "Müller")
	require.Equal(t, 0, plain.exitCode, plain.stderr)
	assert.Equal(t, "person-anna-mueller\n", plain.stdout)

	quiet := runGLX(t, archive, "add", "person", "--given", "Otto", "--surname", "Müller", "--quiet")
	require.Equal(t, 0, quiet.exitCode, quiet.stderr)
	assert.Equal(t, "person-otto-mueller\n", quiet.stdout)
	assert.Empty(t, quiet.stderr, "--quiet silences the progress line")
}

func TestAdd_ArchiveFlagFromOutside(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, t.TempDir(), "add", "person", "--given", "Jane", "--surname", "Doe", "--archive", archive)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.FileExists(t, filepath.Join(archive, "persons", "person-jane-doe.glx"))
}

func TestAdd_DryRunWritesNothing(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "add", "person", "--given", "Jane", "--surname", "Doe", "--dry-run")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Equal(t, "person-jane-doe\n", res.stdout, "a dry run still reports the ID it would create")
	assert.Contains(t, res.stderr, "dry run")
	assertTreeUnchanged(t, before, archive)
}

// A second add of the same derived ID does not overwrite: it picks the next
// free suffix. An explicit --id that is taken is refused unless --force.
func TestAdd_IDCollisions(t *testing.T) {
	archive := copyExample(t, "basic-family")
	first := runGLX(t, archive, "add", "person", "--given", "Jane", "--surname", "Doe")
	require.Equal(t, 0, first.exitCode, first.stderr)

	second := runGLX(t, archive, "add", "person", "--given", "Jane", "--surname", "Doe")
	require.Equal(t, 0, second.exitCode, second.stderr)
	assert.Equal(t, "person-jane-doe-2\n", second.stdout)

	before := snapshotTree(t, archive)
	taken := runGLX(t, archive, "add", "person", "--id", "person-jane-doe", "--given", "Other")
	assertExitWithStderr(t, taken, "already exists (use --force to overwrite)")
	assertTreeUnchanged(t, before, archive)

	forced := runGLX(t, archive, "add", "person", "--id", "person-jane-doe", "--given", "Other", "--force")
	require.Equal(t, 0, forced.exitCode, forced.stderr)
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Equal(t, []string{"persons/person-jane-doe.glx"}, diff.changed)
	assert.Empty(t, diff.created)
}

func TestAdd_InvalidInputFailsWithoutWriting(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown vocabulary value", []string{"event", "--type", "bogus_type"}, `event_types="bogus_type"`},
		{"dangling reference", []string{"event", "--type", "death", "--principal", "person-nobody"}, `persons "person-nobody"`},
		{"stray positional argument", []string{"person", "extra", "--given", "A"}, `unknown command "extra" for "glx add person"`},
		{"unknown subcommand", []string{"bogus"}, `unknown command "bogus" for "glx add"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, archive, append([]string{"add"}, tc.args...)...), tc.want)
		})
	}
	assertTreeUnchanged(t, before, archive)
}

func TestAdd_MissingArchivePathFails(t *testing.T) {
	res := runGLX(t, t.TempDir(), "add", "person", "--given", "A", "--archive", "does-not-exist")

	assertExitWithStderr(t, res, "does-not-exist")
	assert.Empty(t, res.stdout, "no ID is echoed when nothing was created")
}

func TestAdd_ChainedCaptureBuildsValidArchive(t *testing.T) {
	// The workflow the add help documents: capture each ID and feed it to the
	// next command.
	archive := copyExample(t, "basic-family")
	capture := func(args ...string) string {
		t.Helper()
		res := runGLX(t, archive, append([]string{"add"}, args...)...)
		require.Equal(t, 0, res.exitCode, res.stderr)

		return strings.TrimSpace(res.stdout)
	}

	person := capture("person", "--given", "Johann", "--surname", "Jungk", "--sex", "male")
	place := capture("place", "--name", "Enkirch", "--type", "town")
	event := capture("event", "--type", "christening", "--date", "1725-02-25", "--place", place, "--principal", person)

	data, err := os.ReadFile(filepath.Join(archive, "events", event+".glx"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "person: "+person)
	assert.Contains(t, string(data), "place: "+place)
	assertArchiveValid(t, archive)
}
