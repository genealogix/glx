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
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serveURLPattern = regexp.MustCompile(`Open this URL in your browser: (http://\S+)`)

// lockedBuffer is a bytes.Buffer safe to write from one goroutine while the
// test reads it from another: the scanner goroutine appends serve's stdout,
// and os/exec copies its stderr from a goroutine of its own.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// servedViewer is a running `glx serve` process.
type servedViewer struct {
	url    string
	cmd    *exec.Cmd
	stdout *lockedBuffer
	// done receives cmd.Wait's result once serve has exited and its stdout
	// is fully read. Only the scanner goroutine calls Wait.
	done chan error
	// exited is closed alongside done, so cleanup can tell whether a test
	// already consumed the result.
	exited chan struct{}
}

// startServe launches `glx serve --port 0 args...` in workDir and waits for
// the URL it prints. The process is stopped when the test ends.
func startServe(t *testing.T, workDir string, args ...string) *servedViewer {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), glxBinary, append([]string{"serve", "--port", "0"}, args...)...) //nolint:gosec // test-controlled args
	cmd.Dir = workDir
	cmd.Env = envWithout(os.Environ(), "GLX_CACHE")
	pipe, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stderr := &lockedBuffer{}
	cmd.Stderr = stderr
	require.NoError(t, cmd.Start())

	v := &servedViewer{cmd: cmd, stdout: &lockedBuffer{}, done: make(chan error, 1), exited: make(chan struct{})}
	urls := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text()
			_, _ = v.stdout.Write([]byte(line + "\n"))
			if m := serveURLPattern.FindStringSubmatch(line); m != nil {
				select {
				case urls <- strings.TrimSuffix(m[1], "/"):
				default:
				}
			}
		}
		v.done <- cmd.Wait()
		close(v.exited)
	}()
	t.Cleanup(func() {
		select {
		case <-v.exited:
		default:
			_ = cmd.Process.Kill()
			<-v.exited
		}
	})

	select {
	case v.url = <-urls:
	case err := <-v.done:
		t.Fatalf("glx serve exited before printing a URL: %v\nstderr:\n%s", err, stderr.String())
	case <-time.After(30 * time.Second):
		t.Fatalf("glx serve printed no URL within 30s\nstdout:\n%s\nstderr:\n%s", v.stdout.String(), stderr.String())
	}

	return v
}

// getJSON fetches path from the viewer and decodes the JSON body.
func (v *servedViewer) getJSON(t *testing.T, path string) (int, any) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, v.url+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var decoded any
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.Unmarshal(body, &decoded), "%s returned non-JSON:\n%s", path, body)
	}

	return resp.StatusCode, decoded
}

func TestServe_ServesArchiveAPIAndStops(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	v := startServe(t, archive)

	assert.Contains(t, v.stdout.String(), "4 persons, 1 sources, 5 events")
	for _, path := range []string{
		"/api/overview",
		"/api/persons",
		"/api/persons/person-robert-thompson",
		"/api/persons/person-alice-thompson/tree",
		"/api/sources",
		"/api/sources/source-sangamon-births",
	} {
		status, body := v.getJSON(t, path)
		assert.Equal(t, http.StatusOK, status, path)
		assert.NotEmpty(t, body, path)
	}
	status, _ := v.getJSON(t, "/api/persons/person-nobody")
	assert.Equal(t, http.StatusNotFound, status)
	// Serving is read-only on every platform, checked before the
	// platform-specific shutdown below.
	assertTreeUnchanged(t, before, archive)

	if runtime.GOOS == "windows" {
		// No SIGINT for a child process on Windows; cleanup kills it.
		return
	}
	require.NoError(t, v.cmd.Process.Signal(os.Interrupt))
	select {
	case err := <-v.done:
		require.NoError(t, err, "serve must exit 0 on Ctrl-C")
	case <-time.After(15 * time.Second):
		t.Fatal("glx serve did not stop within 15s of SIGINT")
	}
	assert.Contains(t, v.stdout.String(), "Viewer stopped.")
}

func TestServe_PathArgumentFromOutside(t *testing.T) {
	archive := copyExample(t, "single-file")

	v := startServe(t, t.TempDir(), filepath.Join(archive, "archive.glx"))

	status, _ := v.getJSON(t, "/api/overview")
	assert.Equal(t, http.StatusOK, status)
}

func TestServe_Errors(t *testing.T) {
	work := t.TempDir()

	assertExitWithStderr(t, runGLX(t, work, "serve", "does-not-exist"), "cannot access path")
	assertExitWithStderr(t, runGLX(t, work, "serve", "a", "b"), "accepts at most 1 arg(s), received 2")
}
