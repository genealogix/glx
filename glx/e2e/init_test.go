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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workspace returns a temp dir to run `glx init` in, skipping the test if that
// dir happens to sit inside a git repository — `glx init` deliberately does not
// nest a repository inside one, so the assertions below would be measuring the
// runner's TMPDIR rather than the CLI.
func workspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for probe := dir; ; {
		if _, err := os.Stat(filepath.Join(probe, ".git")); err == nil {
			t.Skipf("temp dir %s is inside a git repository", dir)
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}

	return dir
}

// headRef reads .git/HEAD without going through a git library, so the test
// asserts on what is actually on disk.
func headRef(t *testing.T, repo string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, ".git", "HEAD"))
	require.NoError(t, err)

	return strings.TrimSpace(string(data))
}

// #1273: `glx init` wrote a .gitignore into a directory it never made a
// repository, so the archive's one hint of version control was inert until the
// user ran `git init` by hand.
func TestInit_MakesTheArchiveAGitRepository(t *testing.T) {
	work := workspace(t)

	res := runGLX(t, work, "init", "my-archive")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	archive := filepath.Join(work, "my-archive")
	info, err := os.Stat(filepath.Join(archive, ".git"))
	require.NoError(t, err, "init should leave a git repository behind")
	assert.True(t, info.IsDir())

	// The .gitignore init writes is now inside a repository that can honor it.
	_, err = os.Stat(filepath.Join(archive, ".gitignore"))
	require.NoError(t, err)

	assert.Contains(t, res.stdout, "Initialized empty Git repository on branch")

	// Whatever branch was chosen (the user's init.defaultBranch, else main),
	// the output and the repository must agree.
	head := headRef(t, archive)
	require.True(t, strings.HasPrefix(head, "ref: refs/heads/"), "HEAD: %q", head)
	branch := strings.TrimPrefix(head, "ref: refs/heads/")
	assert.Contains(t, res.stdout, "on branch '"+branch+"'")

	// Scaffolding only: no commit is made, so the user can set their identity
	// or edit the archive before the first commit records anything.
	assert.Contains(t, res.stdout, "Nothing is committed yet")
	refs, err := os.ReadDir(filepath.Join(archive, ".git", "refs", "heads"))
	require.NoError(t, err)
	assert.Empty(t, refs, "a scaffolded repository has no branches yet")
}

// The repository is created in-process by go-git (the CLI never shells out to
// git), so it is worth proving the real git binary accepts the result.
func TestInit_RepositoryIsUsableByTheGitBinary(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	work := workspace(t)

	res := runGLX(t, work, "init", "my-archive")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	archive := filepath.Join(work, "my-archive")
	out, err := exec.CommandContext(context.Background(), "git", "-C", archive, "status", "--porcelain").CombinedOutput()
	require.NoErrorf(t, err, "git status: %s", out)
	// The archive's own files are untracked, and the ignored cache dir is not
	// listed — proof the .gitignore is in force.
	assert.Contains(t, string(out), "README.md")
	assert.NotContains(t, string(out), ".glx/")
}

// --no-git is the opt-out, and it has to say plainly that no repository exists
// rather than leaving the .gitignore as a false signal.
func TestInit_NoGitSkipsTheRepositoryAndSaysSo(t *testing.T) {
	work := workspace(t)

	res := runGLX(t, work, "init", "my-archive", "--no-git")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	archive := filepath.Join(work, "my-archive")
	_, err := os.Stat(filepath.Join(archive, ".git"))
	assert.True(t, os.IsNotExist(err), "--no-git must not create a repository")
	assert.Contains(t, res.stdout, "Not a Git repository")
	assert.Contains(t, res.stdout, "git init")
}

// An archive created inside an existing repository is already version
// controlled; a nested repository would hide it from the outer one.
func TestInit_InsideExistingRepositoryDoesNotNest(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	work := workspace(t)
	out, err := exec.CommandContext(context.Background(), "git", "-C", work, "init").CombinedOutput()
	require.NoErrorf(t, err, "git init: %s", out)

	res := runGLX(t, work, "init", "my-archive")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	_, err = os.Stat(filepath.Join(work, "my-archive", ".git"))
	assert.True(t, os.IsNotExist(err), "init must not nest a repository inside one")
	assert.Contains(t, res.stdout, "Already inside a Git repository")
}

// The single-file layout gets the same treatment: one directory, one archive
// file, under version control.
func TestInit_SingleFileArchiveAlsoGetsARepository(t *testing.T) {
	work := workspace(t)

	res := runGLX(t, work, "init", "my-archive", "--single-file")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	archive := filepath.Join(work, "my-archive")
	_, err := os.Stat(filepath.Join(archive, "archive.glx"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(archive, ".git"))
	assert.NoError(t, err, "a single-file archive directory is version controlled too")
}
