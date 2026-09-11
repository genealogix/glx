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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountGLXFiles_SkipsDotDirectories(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "a.glx"), []byte("a"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "worktrees", "copy", "persons"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "worktrees", "copy", "persons", "a.glx"), []byte("a"), 0o644))

	assert.Equal(t, 1, countGLXFiles(dir))
}
