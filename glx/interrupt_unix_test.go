//go:build !windows

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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWatchInterruptsSubprocess exercises actual signals and process exit rather
// than calling the cleanup helper directly. Windows cannot send these signals.
func TestWatchInterruptsSubprocess(t *testing.T) {
	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			dir := t.TempDir()
			tracked := filepath.Join(dir, ".glx-tmp-tracked")
			kept := filepath.Join(dir, "keep")
			require.NoError(t, os.WriteFile(tracked, nil, 0o600))
			require.NoError(t, os.WriteFile(kept, nil, 0o600))
			executable, err := os.Executable()
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestInterruptProcessHelper$")
			cmd.Env = append(os.Environ(), "GLX_INTERRUPT_HELPER="+tracked)
			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err)
			stdin, err := cmd.StdinPipe()
			require.NoError(t, err)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			require.NoError(t, cmd.Start())
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			line, err := bufio.NewReader(stdout).ReadString('\n')
			require.NoError(t, err)
			require.Equal(t, "ready\n", line)
			require.NoError(t, cmd.Process.Signal(sig))
			exited := make(chan error, 1)
			go func() { exited <- cmd.Wait() }()
			select {
			case err := <-exited:
				t.Fatalf("exited while critical section held: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
			assert.FileExists(t, tracked, "cleanup must wait for the critical section")
			_, err = fmt.Fprintln(stdin, "release")
			require.NoError(t, err)
			require.NoError(t, stdin.Close())
			err = <-exited
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, exitInterrupted, exitErr.ExitCode(), stderr.String())
			assert.Contains(t, stderr.String(), "glx: interrupted")
			assert.NoFileExists(t, tracked)
			assert.FileExists(t, kept)
		})
	}
}

func TestInterruptProcessHelper(t *testing.T) {
	tracked := os.Getenv("GLX_INTERRUPT_HELPER")
	if tracked == "" {
		return
	}
	release := holdInterrupts()
	defer trackTemp(tracked)()
	defer watchInterrupts(os.Stderr)()
	_, err := fmt.Fprintln(os.Stdout, "ready")
	require.NoError(t, err)
	_, err = bufio.NewReader(os.Stdin).ReadString('\n')
	require.NoError(t, err)
	release()
	select {}
}
