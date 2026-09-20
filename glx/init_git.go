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
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// fallbackInitBranch is the branch a new archive repository starts on when the
// user has no `init.defaultBranch` configured. go-git still defaults to
// "master"; the `git` binary has defaulted to asking for (and most setups now
// configure) "main" for years, so following git's own default here keeps
// `glx init` from handing people a branch name their tooling no longer expects.
const fallbackInitBranch = "main"

// gitInitOutcome says what initArchiveGitRepo did, so the caller can tell the
// user which world they are in: a fresh repository, or an archive nested
// inside one that already exists.
type gitInitOutcome int

const (
	// gitInitCreated: the archive directory is now a Git repository.
	gitInitCreated gitInitOutcome = iota
	// gitInitNested: the archive sits inside an existing repository, which
	// already version-controls it; a nested repository would hide it from the
	// outer one, so none was created.
	gitInitNested
)

// gitInitResult is what initArchiveGitRepo did and, for a repository it
// created, the branch it starts on — so the caller can name it in the output
// instead of resolving the configured default a second time.
type gitInitResult struct {
	outcome gitInitOutcome
	branch  string
}

// initArchiveGitRepo makes dir a Git repository unless it is already inside
// one. It uses the pure-Go go-git library, so no `git` binary on PATH is
// required — the same choice archive_cache.go makes for reading repositories.
//
// Only the repository scaffold is created: no files are staged and no commit
// is made, so the user can set their identity, rename the branch, or edit the
// archive before the first commit records anything.
func initArchiveGitRepo(dir string) (gitInitResult, error) {
	// DetectDotGit walks parent directories, so this is true both when dir is
	// itself a repository and when it is a subdirectory of one.
	if _, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{DetectDotGit: true}); err == nil {
		return gitInitResult{outcome: gitInitNested}, nil
	}

	branch := defaultInitBranch()
	if _, err := git.PlainInitWithOptions(dir, &git.PlainInitOptions{
		DefaultBranch: branch,
	}); err != nil {
		return gitInitResult{}, fmt.Errorf("git init in %s: %w", dir, err)
	}

	return gitInitResult{outcome: gitInitCreated, branch: branch.Short()}, nil
}

// loadGitConfig reads a gitconfig scope. It is a var (not a direct call to
// config.LoadConfig) only so tests can feed defaultInitBranch a config without
// depending on the developer's or CI runner's real gitconfig files.
var loadGitConfig = config.LoadConfig

// defaultInitBranch resolves the branch name a new repository starts on the way
// `git init` does: the user's configured `init.defaultBranch` (global, then
// system scope) if it is set and valid, else fallbackInitBranch. go-git reads
// the same gitconfig files the `git` binary does, so a user who has configured
// a branch name gets it here too.
func defaultInitBranch() plumbing.ReferenceName {
	for _, scope := range []config.Scope{config.GlobalScope, config.SystemScope} {
		cfg, err := loadGitConfig(scope)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(cfg.Init.DefaultBranch)
		if name == "" {
			continue
		}
		// Accept both "main" and a fully qualified "refs/heads/main", since
		// git tolerates the latter in init.defaultBranch.
		ref := plumbing.NewBranchReferenceName(strings.TrimPrefix(name, "refs/heads/"))
		if ref.Validate() != nil {
			continue
		}

		return ref
	}

	return plumbing.NewBranchReferenceName(fallbackInitBranch)
}

// noRepoAdvice closes the loop when the archive is not under version control.
// #1273: `glx init` used to write a .gitignore into a directory it never made a
// repository, so the one artifact hinting at version control was the one that
// looked already handled. Either the repository exists or the output says it
// does not.
const noRepoAdvice = "Not a Git repository — run 'git init' to enable version control and collaboration"

// gitInitLines initializes the archive repository (unless noGit) and returns
// the lines describing what happened, for the caller to print as part of the
// init summary.
//
// A failure to create the repository is reported as a warning on stderr rather
// than an error: the archive itself is already on disk and usable, and
// `git init` is a step the user can repeat by hand.
func gitInitLines(dir string, noGit bool) []string {
	if noGit {
		return []string{noRepoAdvice}
	}

	res, err := initArchiveGitRepo(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not initialize Git repository: %v\n", err)

		return []string{noRepoAdvice}
	}

	if res.outcome == gitInitNested {
		return []string{"Already inside a Git repository — skipped 'git init'"}
	}

	return []string{
		fmt.Sprintf("Initialized empty Git repository on branch '%s'", res.branch),
		`Nothing is committed yet — run: git add . && git commit -m "Initial archive"`,
	}
}
