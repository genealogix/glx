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
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exactNamedArgs exists so a wrong argument count names the argument rather
// than only counting it, the way cobra.ExactArgs does (#1274). The exact
// wording is what a user reads, so it is asserted in full.
func TestExactNamedArgs(t *testing.T) {
	validate := exactNamedArgs("<input-directory>", "<output-file>")
	cmd := &cobra.Command{Use: "join <input-directory> <output-file>"}

	tests := []struct {
		name    string
		args    []string
		wantErr error
		wantMsg string
	}{
		{
			name: "exact count is accepted",
			args: []string{"family-archive", "family.glx"},
		},
		{
			name:    "no arguments names both",
			args:    nil,
			wantErr: ErrMissingArguments,
			wantMsg: "join: missing required argument(s): <input-directory> <output-file>",
		},
		{
			name:    "one argument names only the second",
			args:    []string{"family-archive"},
			wantErr: ErrMissingArguments,
			wantMsg: "join: missing required argument(s): <output-file>",
		},
		{
			name:    "too many arguments reports the expected shape",
			args:    []string{"a", "b", "c"},
			wantErr: ErrTooManyArguments,
			wantMsg: "join: too many arguments: accepts 2 (<input-directory> <output-file>), received 3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(cmd, tt.args)
			if tt.wantErr == nil {
				require.NoError(t, err)

				return
			}
			require.Error(t, err)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantMsg, err.Error())
		})
	}
}

// A single-argument command is the degenerate case: with nothing supplied the
// one name is still reported, without the plural reading collapsing into a
// bare count.
func TestExactNamedArgs_SingleArgument(t *testing.T) {
	validate := exactNamedArgs("<person-id>")
	cmd := &cobra.Command{Use: "ancestors <person-id>"}

	require.NoError(t, validate(cmd, []string{"person-1"}))

	err := validate(cmd, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMissingArguments)
	assert.Equal(t, "ancestors: missing required argument(s): <person-id>", err.Error())
}

// join and split are the two commands wired to the helper; a regression that
// swapped either back to cobra.ExactArgs would drop the wording the e2e tests
// assert on.
func TestJoinAndSplitUseNamedArgs(t *testing.T) {
	joinErr := joinCmd.Args(joinCmd, nil)
	require.ErrorIs(t, joinErr, ErrMissingArguments)
	assert.Contains(t, joinErr.Error(), "<input-directory> <output-file>")

	splitErr := splitCmd.Args(splitCmd, nil)
	require.ErrorIs(t, splitErr, ErrMissingArguments)
	assert.Contains(t, splitErr.Error(), "<input-file> <output-directory>")
}

// export must accept a bare invocation (the archive defaults to ".") and must
// still reject a stray second positional.
func TestExportArgsAcceptOptionalArchive(t *testing.T) {
	require.NoError(t, exportCmd.Args(exportCmd, nil))
	require.NoError(t, exportCmd.Args(exportCmd, []string{"family-archive"}))
	require.Error(t, exportCmd.Args(exportCmd, []string{"family-archive", "extra"}))
}
