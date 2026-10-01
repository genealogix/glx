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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// writeArchiveWithDeedErrors writes a multi-file archive holding n events that
// each carry two dangling vocabulary references (event type and role), so the
// archive fails validation with 2n errors (#1322).
func writeArchiveWithDeedErrors(t *testing.T, n int) string {
	t.Helper()
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "people.glx"),
		"persons:\n  person-a:\n    properties:\n      name: Ann\n")
	var b strings.Builder
	b.WriteString("events:\n")
	for i := range n {
		fmt.Fprintf(&b, "  ev-deed-%02d:\n    type: land_transaction\n    date: \"1831\"\n"+
			"    participants:\n      - person: person-a\n        role: grantor\n", i)
	}
	writeSkipTestFile(t, filepath.Join(dir, "events", "deeds.glx"), b.String())

	return dir
}

// errorLines returns the "  - " bulleted lines of a validate stderr.
func errorLines(stderr string) []string {
	var lines []string
	for line := range strings.SplitSeq(stderr, "\n") {
		if strings.HasPrefix(line, "  - ") {
			lines = append(lines, line)
		}
	}

	return lines
}

func TestValidatePaths_ShowFirstErrors(t *testing.T) {
	dir := writeArchiveWithDeedErrors(t, 15)

	t.Run("default lists the first ten and names the flag", func(t *testing.T) {
		streams, _, errOut := newTestStreams()
		err := validatePaths(streams, []string{dir})

		require.ErrorIs(t, err, ErrStructuralValidationFailed)
		stderr := errOut.String()
		assert.Contains(t, stderr, "Validation failed: 30 error(s)")
		assert.NotContains(t, stderr, "Error loading archive")
		assert.Len(t, errorLines(stderr), defaultShowFirstErrors)
		assert.Contains(t, stderr, "... and 20 more errors (use --show-first-errors 0 to list all)")
	})

	t.Run("zero lists every error", func(t *testing.T) {
		streams, _, errOut := newTestStreams()
		err := validatePathsShowing(streams, []string{dir}, 0)

		require.Error(t, err)
		assert.Len(t, errorLines(errOut.String()), 30)
		assert.NotContains(t, errOut.String(), "more errors")
	})

	t.Run("a positive limit caps the list", func(t *testing.T) {
		streams, _, errOut := newTestStreams()
		require.Error(t, validatePathsShowing(streams, []string{dir}, 3))

		assert.Len(t, errorLines(errOut.String()), 3)
		assert.Contains(t, errOut.String(), "... and 27 more errors")
	})

	t.Run("the list is sorted and identical run to run", func(t *testing.T) {
		var first string
		for i := range 5 {
			streams, _, errOut := newTestStreams()
			require.Error(t, validatePaths(streams, []string{dir}))
			if i == 0 {
				first = errOut.String()

				continue
			}
			assert.Equal(t, first, errOut.String())
		}
		lines := errorLines(first)
		require.NotEmpty(t, lines)
		assert.Contains(t, lines[0], "events[ev-deed-00]")
		assert.Contains(t, lines[len(lines)-1], "events[ev-deed-04]")
	})
}

// A single-file archive is validated after loading, so its errors go through
// reportArchiveValidation; the flag caps that list the same way.
func TestValidateSelfContainedArchive_ShowFirstErrors(t *testing.T) {
	dir := writeArchiveWithDeedErrors(t, 8)
	joined := filepath.Join(t.TempDir(), "joined.glx")
	require.NoError(t, joinArchive(dir, joined, false, false, 0))

	streams, _, errOut := newTestStreams()
	err := validateSelfContainedArchive(streams, joined, 4)

	require.ErrorIs(t, err, ErrValidationFailed)
	assert.Contains(t, errOut.String(), "Found 16 errors:")
	assert.Equal(t, 4, strings.Count(errOut.String(), "- ❌ "))
	assert.Contains(t, errOut.String(), "... and 12 more errors (use --show-first-errors 0 to list all)")

	streams, _, errOut = newTestStreams()
	require.Error(t, validateSelfContainedArchive(streams, joined, 0))
	assert.Equal(t, 16, strings.Count(errOut.String(), "- ❌ "))
}

func TestReportArchiveLoadError(t *testing.T) {
	t.Run("validation errors are headed as a validation failure", func(t *testing.T) {
		streams, _, errOut := newTestStreams()
		err := fmt.Errorf("validation failed: %w", &glxlib.StructuredValidationError{
			Errors: []glxlib.ValidationError{{Message: "first"}, {Message: "second"}},
		})

		reportArchiveLoadError(streams, err, 1)

		assert.Equal(t,
			"Validation failed: 2 error(s)\n  - first\n  ... and 1 more errors (use --show-first-errors 0 to list all)\n",
			errOut.String())
	})

	t.Run("other errors keep the loading header", func(t *testing.T) {
		streams, _, errOut := newTestStreams()

		reportArchiveLoadError(streams, errGenericTest, defaultShowFirstErrors)

		assert.Equal(t, "Error loading archive: some unrecognized error\n", errOut.String())
	})
}

func TestPrintErrorList(t *testing.T) {
	errs := []string{"a", "b", "c"}
	cases := []struct {
		name  string
		limit int
		want  string
	}{
		{"zero shows all", 0, "- a\n- b\n- c\n"},
		{"limit above count shows all", 5, "- a\n- b\n- c\n"},
		{"limit equal to count shows all", 3, "- a\n- b\n- c\n"},
		{"limit below count truncates", 2, "- a\n- b\n  ... and 1 more errors (use --show-first-errors 0 to list all)\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			streams, _, errOut := newTestStreams()
			printErrorList(streams, "- ", errs, tc.limit)
			assert.Equal(t, tc.want, errOut.String())
		})
	}
}
