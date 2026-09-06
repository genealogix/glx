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

// Package e2e exercises the glx CLI as users run it: the real binary, real
// arguments, a real working directory, real exit codes. Runner-level unit
// tests in package main call the function one layer below cobra and cannot
// see flag parsing, argument validation, default flag values, or anything
// that depends on the process's cwd — which is exactly where bugs like
// "rename leaves the shell in a deleted directory" (#1192) live.
//
// TestMain builds the CLI once into a temp dir; each test then runs it via
// runGLX with an explicit working directory and inspects the result.
package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// glxBinary is the path of the CLI built by TestMain.
var glxBinary string

func TestMain(m *testing.M) {
	code, err := buildAndRun(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

// buildAndRun compiles the glx CLI from the parent package into a temp dir,
// runs the test suite against it, and cleans up. Kept separate from TestMain
// so deferred cleanup runs before os.Exit.
func buildAndRun(m *testing.M) (int, error) {
	dir, err := os.MkdirTemp("", "glx-e2e-")
	if err != nil {
		return 0, fmt.Errorf("creating temp dir for binary: %w", err)
	}
	defer os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup

	bin := filepath.Join(dir, "glx")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	// The test's cwd is this package's directory (glx/e2e), so ".." is the
	// CLI's package main.
	build := exec.CommandContext(context.Background(), "go", "build", "-o", bin, "..")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return 0, fmt.Errorf("building glx for e2e tests: %w", err)
	}
	glxBinary = bin

	return m.Run(), nil
}

// result is the observable outcome of one CLI invocation.
type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// runGLX runs the built CLI with the given working directory and arguments.
// It never fails the test on a non-zero exit — callers assert on exitCode —
// but does fail if the process could not be started at all.
func runGLX(t *testing.T, workDir string, args ...string) result {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), glxBinary, args...) //nolint:gosec // args come from the test, not user input
	cmd.Dir = workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := result{stdout: stdout.String(), stderr: stderr.String()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		res.exitCode = 0
	case errors.As(err, &exitErr):
		res.exitCode = exitErr.ExitCode()
	default:
		t.Fatalf("running %s %s: %v", glxBinary, strings.Join(args, " "), err)
	}

	t.Logf("$ glx %s (cwd %s) -> exit %d\n--- stdout ---\n%s--- stderr ---\n%s",
		strings.Join(args, " "), workDir, res.exitCode, res.stdout, res.stderr)

	return res
}

// examplesDir is the repository's docs/examples directory, resolved from
// this package's location so the tests work from any cwd `go test` picks.
func examplesDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)

	return filepath.Join(wd, "..", "..", "docs", "examples")
}

// copyExample copies docs/examples/<name> into a fresh temp dir and returns
// the copy's path. Symlinks are dereferenced: the examples link their
// vocabularies into specification/, and a raw copy would carry dangling
// relative links.
func copyExample(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join(examplesDir(t), name)
	dst := filepath.Join(t.TempDir(), name)
	copyTreeFollowingSymlinks(t, src, dst)

	return dst
}

// copyTreeFollowingSymlinks copies a directory tree, reading through any
// symlinked files so the destination holds real content.
func copyTreeFollowingSymlinks(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := os.Stat(path) // follows symlinks
		if err != nil {
			return err
		}
		if info.IsDir() {
			// A symlink to a directory: WalkDir does not follow it, so walk
			// the resolved target instead. Recursing on the link path itself
			// would hit this branch again forever.
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			copyTreeFollowingSymlinks(t, resolved, target)

			return nil
		}
		data, err := os.ReadFile(path) //nolint:gosec // test fixture path
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o644) //nolint:gosec // test fixture, world-readable is fine
	})
	require.NoError(t, err)
}

// snapshotTree returns every regular file under root (relative slash paths)
// mapped to its content, so a test can assert exactly which files a command
// changed, created, or removed.
func snapshotTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path) //nolint:gosec // test fixture path
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = data

		return nil
	})
	require.NoError(t, err)

	return files
}

// treeDiff classifies the differences between two snapshots.
type treeDiff struct {
	changed []string
	created []string
	removed []string
}

// diffTrees compares two snapshots and returns the sorted sets of paths that
// changed content, appeared, or disappeared.
func diffTrees(before, after map[string][]byte) treeDiff {
	var d treeDiff
	for path, data := range before {
		newData, ok := after[path]
		switch {
		case !ok:
			d.removed = append(d.removed, path)
		case !bytes.Equal(data, newData):
			d.changed = append(d.changed, path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			d.created = append(d.created, path)
		}
	}
	sort.Strings(d.changed)
	sort.Strings(d.created)
	sort.Strings(d.removed)

	return d
}

func TestCopyTreeFollowingSymlinks_DirectorySymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs elevated privileges on Windows")
	}
	// src/linked -> elsewhere/, containing a file. The copy must contain the
	// file under linked/ as a real directory, and must terminate.
	src := t.TempDir()
	elsewhere := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(elsewhere, "inner.glx"), []byte("x\n"), 0o644))
	require.NoError(t, os.Symlink(elsewhere, filepath.Join(src, "linked")))
	require.NoError(t, os.WriteFile(filepath.Join(src, "top.glx"), []byte("y\n"), 0o644))
	dst := filepath.Join(t.TempDir(), "copy")

	copyTreeFollowingSymlinks(t, src, dst)

	info, err := os.Lstat(filepath.Join(dst, "linked"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Zero(t, info.Mode()&os.ModeSymlink, "copied directory must not be a symlink")
	data, err := os.ReadFile(filepath.Join(dst, "linked", "inner.glx"))
	require.NoError(t, err)
	assert.Equal(t, "x\n", string(data))
	assert.FileExists(t, filepath.Join(dst, "top.glx"))
}
