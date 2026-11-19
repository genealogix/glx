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

package main

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

func TestValidateJSONRejectsFIFO(t *testing.T) {
	if path := os.Getenv("GLX_TEST_VALIDATION_FIFO"); path != "" {
		var err error
		if os.Getenv("GLX_TEST_VALIDATION_DIRECT") == "1" {
			_, err = readValidationJSONFile(path)
		} else {
			_, err = validateJSONPath(path, false)
		}
		require.ErrorIs(t, err, errMCPWatchSourceType)

		return
	}
	for _, mode := range []string{"directory", "file", "symlink", "direct-open"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			pipe := filepath.Join(dir, "live.glx")
			require.NoError(t, syscall.Mkfifo(pipe, 0o600))
			path := pipe
			if mode == "directory" {
				path = dir
			}
			if mode == "symlink" {
				path = filepath.Join(dir, "link.glx")
				require.NoError(t, os.Symlink(pipe, path))
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			executable, err := os.Executable()
			require.NoError(t, err)
			// Isolate a blocking regression so it cannot leave a stuck test goroutine.
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestValidateJSONRejectsFIFO$")
			cmd.Env = append(os.Environ(), "GLX_TEST_VALIDATION_FIFO="+path)
			if mode == "direct-open" {
				cmd.Env = append(cmd.Env, "GLX_TEST_VALIDATION_DIRECT=1")
			}
			output, err := cmd.CombinedOutput()
			require.NoError(t, ctx.Err(), "validation blocked on FIFO: %s", output)
			require.NoError(t, err, "%s", output)
		})
	}
}

func TestReadValidationJSONFilePreservesSymlinks(t *testing.T) {
	target := filepath.Join(t.TempDir(), "archive.glx")
	data := []byte("persons: {}\n")
	require.NoError(t, os.WriteFile(target, data, 0o600))
	link := filepath.Join(t.TempDir(), "linked.glx")
	require.NoError(t, os.Symlink(target, link))
	got, err := readValidationJSONFile(link)
	require.NoError(t, err)
	require.Equal(t, data, got)
}
