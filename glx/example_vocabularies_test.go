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

	"github.com/stretchr/testify/require"
)

// standardVocabularyDir is the source of truth the example and fixture
// archives' vocabularies/ directories are copied from.
const standardVocabularyDir = "../specification/5-standard-vocabularies"

// customizedVocabularyDirs lists archives whose vocabularies/ files are
// deliberately edited (for example to exercise broken-reference validation)
// and therefore are not expected to match the standard vocabularies.
var customizedVocabularyDirs = map[string]bool{
	"testdata/invalid/comprehensive-broken-references": true,
}

// TestExampleVocabulariesMatchStandard guards the vocabularies/ copies in the
// example and fixture archives against drifting from the standard
// vocabularies in specification/. Those directories used to be symlinks into
// specification/, but archive loading now refuses symlinks that escape the
// archive directory (#1090), so they are plain copies — matching what
// `glx init` scaffolds — and this test is what keeps them in sync.
func TestExampleVocabulariesMatchStandard(t *testing.T) {
	vocabFiles := make([]string, 0, 128)
	for _, pattern := range []string{
		"../docs/examples/*/vocabularies/*.glx",
		"testdata/*/*/vocabularies/*.glx",
	} {
		matches, err := filepath.Glob(pattern)
		require.NoError(t, err)
		vocabFiles = append(vocabFiles, matches...)
	}
	require.NotEmpty(t, vocabFiles, "no example vocabulary files found; glob patterns may be stale")

	checked := 0
	for _, file := range vocabFiles {
		archiveDir := filepath.Dir(filepath.Dir(file))
		if customizedVocabularyDirs[filepath.ToSlash(archiveDir)] {
			continue
		}
		standard := filepath.Join(standardVocabularyDir, filepath.Base(file))
		want, err := os.ReadFile(standard)
		if os.IsNotExist(err) {
			continue // an archive-specific vocabulary with no standard counterpart
		}
		require.NoError(t, err)

		info, err := os.Lstat(file)
		require.NoError(t, err)
		require.Zero(t, info.Mode()&os.ModeSymlink, "%s must be a regular file, not a symlink (archive loads refuse symlinks that escape the archive)", file)

		got, err := os.ReadFile(file)
		require.NoError(t, err)
		if !bytes.Equal(got, want) {
			t.Errorf("%s has drifted from %s; copy the standard vocabulary over it (or list the archive in customizedVocabularyDirs if the edit is intentional)", file, standard)
		}
		checked++
	}
	require.Positive(t, checked)
}
