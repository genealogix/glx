//go:build unix

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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMCPNonWatchNamedPipes(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		relativePipe string
		singleFile   bool
		symlink      bool
		cache        bool
	}{
		{name: "single file", relativePipe: "pipe.glx", singleFile: true},
		{name: "directory member", relativePipe: "pipe.glx"},
		{name: "single symlink", relativePipe: "pipe.glx", singleFile: true, symlink: true},
		{name: "directory symlink", relativePipe: "pipe", symlink: true},
		{name: "binary cache", relativePipe: ".glx/cache.bin", cache: true},
		{name: "binary cache symlink", relativePipe: ".glx/pipe", cache: true, symlink: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "person.glx"), []byte("persons:\n  person-first:\n    properties:\n      name: First\n"), 0o600))
			pipe := filepath.Join(dir, testCase.relativePipe)
			require.NoError(t, os.MkdirAll(filepath.Dir(pipe), 0o700))
			require.NoError(t, syscall.Mkfifo(pipe, 0o600))
			archivePath := dir
			if testCase.singleFile {
				archivePath = pipe
			}
			if testCase.symlink {
				link := filepath.Join(dir, "link.glx")
				if testCase.cache {
					link = filepath.Join(dir, ".glx/cache.bin")
				}
				require.NoError(t, os.Symlink(filepath.Base(pipe), link))
				if testCase.singleFile {
					archivePath = link
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, glxBinary, "mcp", "serve", archivePath)
			cmd.Env = append(envWithout(os.Environ(), "GLX_CACHE"), "GLX_CACHE=auto")
			output, err := cmd.CombinedOutput()
			require.NoError(t, ctx.Err(), "MCP startup blocked opening named pipe: %s", output)
			if testCase.cache {
				require.NoError(t, err, "an unusable cache must fall back to source: %s", output)
			} else {
				require.Error(t, err)
				require.Contains(t, string(output), "regular file")
			}
			info, statErr := os.Stat(pipe)
			require.NoError(t, statErr)
			require.NotZero(t, info.Mode()&os.ModeNamedPipe, "MCP must not overwrite the pipe/cache")
		})
	}
}

func TestMCPNonWatchIgnoresExcludedNamedPipes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "person.glx"), []byte("persons:\n  person-first:\n    properties:\n      name: First\n"), 0o600))
	for _, relative := range []string{"pipe.txt", ".hidden.glx", ".hidden/pipe.glx"} {
		path := filepath.Join(dir, relative)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, syscall.Mkfifo(path, 0o600))
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, glxBinary, "mcp", "serve", dir)
	cmd.Env = append(envWithout(os.Environ(), "GLX_CACHE"), "GLX_CACHE=auto")
	output, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "MCP blocked reading an excluded path: %s", output)
	require.NoError(t, err, "excluded paths must not prevent startup: %s", output)
	_, err = os.Stat(filepath.Join(dir, ".glx", "cache.bin"))
	require.ErrorIs(t, err, os.ErrNotExist, "MCP serving must not create caches")
}
