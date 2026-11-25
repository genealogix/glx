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
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginRootFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this integration fixture is a POSIX executable script")
	}
	dir := t.TempDir()
	path := []string{"PATH=" + dir}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "glx-example"), []byte("#!/bin/sh\nprintf '<%s>\\n' \"$@\"\nexit 42\n"), 0o700))

	t.Run("passes the suffix unchanged", func(t *testing.T) {
		got := runGLXWithEnv(t, path, dir, "-qq", "example", "--help", "--plugins", "--", "", "two words")
		assert.Equal(t, 42, got.exitCode)
		assert.Equal(t, "<--help>\n<--plugins>\n<-->\n<>\n<two words>\n", got.stdout)
		assert.Empty(t, got.stderr)
	})

	t.Run("GLX actions stay with GLX", func(t *testing.T) {
		for _, tc := range []struct {
			flag string
			want string
		}{
			{"-qh", "Usage:"},
			{"--version", "glx version "},
		} {
			got := runGLXWithEnv(t, path, dir, tc.flag, "example", "--plugin-only")
			assert.Zero(t, got.exitCode)
			assert.Contains(t, got.stdout, tc.want)
			assert.Empty(t, got.stderr)
		}
	})

	t.Run("invalid prefixes are not dispatched", func(t *testing.T) {
		for _, flag := range []string{"--bogus", "--quiet=bogus", "--plugins"} {
			got := runGLXWithEnv(t, path, dir, flag, "example")
			assert.Equal(t, 2, got.exitCode, flag)
			assert.Empty(t, got.stdout, flag)
			assert.NotEmpty(t, got.stderr, flag)
		}
	})

	t.Run("listing accepts normal shorthand syntax", func(t *testing.T) {
		got := runGLXWithEnv(t, path, dir, "-qq", "--plugins")
		assert.Zero(t, got.exitCode)
		assert.Contains(t, got.stdout, filepath.Join(dir, "glx-example"))
		assert.Empty(t, got.stderr)
	})
}
