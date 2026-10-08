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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveFilesValidationError_PreservesTextAndCountsIssues(t *testing.T) {
	err := &archiveFilesValidationError{files: []fileValidationIssues{
		{path: "a.glx", issues: []string{"YAML parse error: bad YAML"}, parseError: true},
		{path: "b.glx", issues: []string{"first", "second"}},
		{path: "c.glx", issues: []string{"third"}},
	}}
	full := "multiple files failed validation:\n\na.glx: YAML parse error: bad YAML\n\nb.glx:\n  - first\n  - second\n\nc.glx:\n  - third"
	assert.Equal(t, full, err.Error())
	require.ErrorIs(t, fmt.Errorf("loading: %w", err), ErrMultipleFilesFailed)
	for _, limit := range []int{0, -1, 4, 5} {
		assert.Equal(t, full, err.format(limit))
	}
	assert.Equal(t,
		"multiple files failed validation:\n\na.glx: YAML parse error: bad YAML\n\nb.glx:\n  - first\n  ... and 2 more errors (use --show-first-errors 0 to list all)",
		err.format(2), "the limit counts individual issues across file blocks")

	streams, _, errOut := newTestStreams()
	reportArchiveLoadError(streams, fmt.Errorf("loading: %w", err), 1)
	assert.Equal(t,
		"Error loading archive: multiple files failed validation:\n\na.glx: YAML parse error: bad YAML\n  ... and 3 more errors (use --show-first-errors 0 to list all)\n",
		errOut.String())
}

func TestLoadArchiveFromFiles_SchemaFailureRetainsFullError(t *testing.T) {
	files := map[string][]byte{
		"b.glx": []byte("events:\n  event-b: {}\n"),
		"a.glx": []byte("events:\n  event-a: {}\n"),
	}
	_, _, err := loadArchiveFromFiles("archive", files, true)
	require.ErrorIs(t, err, ErrMultipleFilesFailed)
	assert.Contains(t, err.Error(), "/events/event-a")
	assert.Contains(t, err.Error(), "/events/event-b")
	assert.NotContains(t, err.Error(), "more errors", "loader callers other than validate still see the full list")
}
