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
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addTestCommand attaches cmd to rootCmd for the length of the test. It is
// marked by hand because markRunStartOnce may already have walked the tree.
func addTestCommand(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	markRunStart(cmd)
	rootCmd.AddCommand(cmd)
	t.Cleanup(func() { rootCmd.RemoveCommand(cmd) })
}

func TestRun_MisuseIsBadInvocationWithUsage(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown flag", []string{"validate", "--no-such-flag"}, "unknown flag: --no-such-flag"},
		{"unknown command", []string{"no-such-command"}, `unknown command "no-such-command"`},
		{"unknown subcommand of a group", []string{"add", "no-such-entity"}, `unknown command "no-such-entity"`},
		{"missing positional arguments", []string{"rename"}, "accepts 2 arg(s), received 0"},
		// Cobra checks required flags after the persistent pre-run hooks, so
		// this case is the one a PersistentPreRun-based check got wrong.
		{"missing required flag", []string{"add", "study"}, `required flag(s) "title" not set`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := run(tc.args, &stderr)

			assert.Equal(t, exitBadInvocation, code, stderr.String())
			assert.Contains(t, stderr.String(), "Usage:")
			assert.Contains(t, stderr.String(), tc.want)
		})
	}
}

func TestRun_RuntimeErrorIsExitOneWithoutUsage(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"validate", filepath.Join(t.TempDir(), "missing")}, &stderr)

	assert.Equal(t, 1, code, stderr.String())
	assert.NotContains(t, stderr.String(), "Usage:")
}

func TestRun_SilentExitPrintsNothing(t *testing.T) {
	addTestCommand(t, &cobra.Command{
		Use:    "test-silent-exit",
		Hidden: true,
		RunE: func(*cobra.Command, []string) error {
			return &silentExitError{code: 7}
		},
	})

	var stderr bytes.Buffer
	code := run([]string{"test-silent-exit"}, &stderr)

	assert.Equal(t, 7, code)
	assert.Empty(t, stderr.String())
}

func TestRun_PanicIsSanitizedInternalError(t *testing.T) {
	addTestCommand(t, &cobra.Command{
		Use:    "test-panic",
		Hidden: true,
		RunE: func(*cobra.Command, []string) error {
			panic("boom \x1b[31mred")
		},
	})

	var stderr bytes.Buffer
	code := run([]string{"test-panic"}, &stderr)

	assert.Equal(t, exitInternalError, code)
	assert.Contains(t, stderr.String(), `glx: internal error: boom \x1b[31mred`)
	assert.NotContains(t, stderr.String(), "\x1b")
}

func TestRemoveTrackedTemps(t *testing.T) {
	dir := t.TempDir()
	tracked := filepath.Join(dir, ".glx-tmp-tracked")
	trackedDir := filepath.Join(dir, ".glx-stage-tracked")
	released := filepath.Join(dir, ".glx-tmp-released")
	require.NoError(t, os.WriteFile(tracked, nil, 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(trackedDir, "nested"), 0o750))
	require.NoError(t, os.WriteFile(released, nil, 0o600))

	defer trackTemp(tracked)()
	defer trackTemp(trackedDir)()
	trackTemp(released)()

	removeTrackedTemps()

	assert.NoFileExists(t, tracked)
	assert.NoDirExists(t, trackedDir)
	assert.FileExists(t, released, "an untracked path must survive")
}
