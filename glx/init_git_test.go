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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubGitConfig replaces the gitconfig loader for the duration of a test, so
// the branch resolution under test never depends on the gitconfig files of the
// developer or CI runner executing it.
func stubGitConfig(t *testing.T, byScope map[config.Scope]string) {
	t.Helper()
	original := loadGitConfig
	t.Cleanup(func() { loadGitConfig = original })
	loadGitConfig = func(scope config.Scope) (*config.Config, error) {
		cfg := config.NewConfig()
		cfg.Init.DefaultBranch = byScope[scope]

		return cfg, nil
	}
}

func TestDefaultInitBranch(t *testing.T) {
	tests := []struct {
		name     string
		configs  map[config.Scope]string
		expected string
	}{
		{
			name:     "no configured default falls back to main",
			configs:  map[config.Scope]string{},
			expected: "refs/heads/main",
		},
		{
			name:     "global init.defaultBranch is honored",
			configs:  map[config.Scope]string{config.GlobalScope: "trunk"},
			expected: "refs/heads/trunk",
		},
		{
			name:     "system scope is used when global is unset",
			configs:  map[config.Scope]string{config.SystemScope: "master"},
			expected: "refs/heads/master",
		},
		{
			name: "global scope wins over system scope",
			configs: map[config.Scope]string{
				config.GlobalScope: "trunk",
				config.SystemScope: "master",
			},
			expected: "refs/heads/trunk",
		},
		{
			name:     "a fully qualified ref is accepted",
			configs:  map[config.Scope]string{config.GlobalScope: "refs/heads/dev"},
			expected: "refs/heads/dev",
		},
		{
			name:     "surrounding whitespace is trimmed",
			configs:  map[config.Scope]string{config.GlobalScope: " trunk "},
			expected: "refs/heads/trunk",
		},
		{
			name:     "an invalid branch name falls back to main",
			configs:  map[config.Scope]string{config.GlobalScope: "has a space"},
			expected: "refs/heads/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubGitConfig(t, tt.configs)

			assert.Equal(t, tt.expected, defaultInitBranch().String())
		})
	}
}

// The loader indirection exists for the tests above, so at least one case has
// to prove the real thing reads a real gitconfig — otherwise the stub could be
// testing nothing but itself.
func TestDefaultInitBranch_ReadsRealGitconfig(t *testing.T) {
	home := t.TempDir()
	// XDG_CONFIG_HOME is consulted before $HOME/.gitconfig; point it at an
	// empty directory so the file written below is the one that is found.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("HOME", home)
	require.NoError(t, os.WriteFile(
		filepath.Join(home, ".gitconfig"),
		[]byte("[init]\n\tdefaultBranch = heritage\n"),
		0o600,
	))

	assert.Equal(t, "refs/heads/heritage", defaultInitBranch().String())
}

func TestInitArchiveGitRepo_CreatesRepository(t *testing.T) {
	stubGitConfig(t, map[config.Scope]string{})
	dir := t.TempDir()

	res, err := initArchiveGitRepo(dir)
	require.NoError(t, err)

	assert.Equal(t, gitInitCreated, res.outcome)
	assert.Equal(t, "main", res.branch)

	repo, err := git.PlainOpen(dir)
	require.NoError(t, err, "the directory should now be a git repository")

	head, err := repo.Reference("HEAD", false)
	require.NoError(t, err)
	assert.Equal(t, "refs/heads/main", head.Target().String())

	// Scaffolding only: the user still owns the first commit.
	_, err = repo.Head()
	assert.Error(t, err, "a freshly initialized repository has no commits")
}

func TestInitArchiveGitRepo_SkipsWhenAlreadyInsideRepository(t *testing.T) {
	stubGitConfig(t, map[config.Scope]string{})
	outer := t.TempDir()
	_, err := git.PlainInit(outer, false)
	require.NoError(t, err)

	nested := filepath.Join(outer, "archives", "smith-family")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	res, err := initArchiveGitRepo(nested)
	require.NoError(t, err)

	assert.Equal(t, gitInitNested, res.outcome)
	_, err = os.Stat(filepath.Join(nested, ".git"))
	assert.True(t, os.IsNotExist(err), "a nested repository would hide the archive from the outer one")
}
