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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrPathEscapesDir is returned when a path that must stay inside a base
// directory resolves outside it, either lexically (absolute or "../"
// traversal) or through a symbolic link.
var ErrPathEscapesDir = errors.New("path escapes base directory")

// goosWindows is the runtime.GOOS value for Windows, where Git checkouts
// without core.symlinks store symlinks as placeholder text files.
const goosWindows = "windows"

// relWithin returns child's path relative to parent when child is lexically
// contained in parent (after cleaning and making both absolute). The parent
// directory itself is not considered contained. This is a lexical check only:
// it does not resolve symbolic links, so callers that read the file must go
// through openWithin / readFileWithin, which enforce containment at open time.
func relWithin(child, parent string) (string, bool) {
	absChild, err := filepath.Abs(filepath.Clean(child))
	if err != nil {
		return "", false
	}
	absParent, err := filepath.Abs(filepath.Clean(parent))
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(absParent, absChild)
	if err != nil {
		return "", false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}

	return rel, true
}

// isPathWithin checks whether child is contained within the parent directory.
// Uses filepath.Rel to handle edge cases like parent being "." or "/".
func isPathWithin(child, parent string) bool {
	_, ok := relWithin(child, parent)

	return ok
}

// openWithin opens path for reading, refusing any path that does not stay
// inside baseDir. Containment is enforced both lexically (via relWithin) and
// at the filesystem level: the file is opened through an os.Root scoped to
// baseDir, so a symbolic link inside baseDir whose target lies outside it is
// rejected rather than followed. Relative symlinks that stay inside baseDir
// are followed. Absolute symlinks are always refused, even when their target
// resolves inside baseDir: os.Root rejects any absolute link target, and
// resolving the target ourselves to re-check containment would reopen the
// TOCTOU window this helper closes. Such links must be rewritten as relative.
// baseDir itself may be a symlink to a directory; only escapes from the
// resolved base are rejected. Lexical escapes return ErrPathEscapesDir.
func openWithin(baseDir, path string) (*os.File, error) {
	rel, ok := relWithin(path, baseDir)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPathEscapesDir, path)
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	return root.Open(rel)
}

// readFileWithin reads the file at path with the same containment guarantees
// as openWithin.
func readFileWithin(baseDir, path string) ([]byte, error) {
	rel, ok := relWithin(path, baseDir)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPathEscapesDir, path)
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	return root.ReadFile(rel)
}

// walkGLXFiles calls visit for every .glx file under rootDir with the file's
// path relative to rootDir (OS separators) and its contents. Dot-prefixed
// directories below rootDir are not entered. Reads go through
// an os.Root scoped to rootDir, so a symlink in the tree whose target lies
// outside rootDir yields a read error instead of the target's contents, and a
// path swapped for a symlink between the directory listing and the read
// (CWE-367) cannot redirect the read outside the root either. As with
// openWithin, an absolute symlink is refused even when it resolves inside
// rootDir; only relative in-root links are followed. A read failure
// is passed to visit as err (with nil data); visit decides whether it aborts
// the walk. On Windows, Git symlink placeholders are resolved within the same
// root before visit is called.
func walkGLXFiles(rootDir string, visit func(relPath string, data []byte, err error) error) error {
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	return fs.WalkDir(root.FS(), ".", func(entryPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if entryPath != "." && isDotDir(d.Name()) {
				return fs.SkipDir
			}

			return nil
		}
		if !isGLXFile(d.Name()) {
			return nil
		}

		data, readErr := root.ReadFile(entryPath)
		if readErr == nil && runtime.GOOS == goosWindows {
			// On Windows, Git stores symlinks as text files containing the
			// target path. Detect these and read the actual target file.
			data = resolveSymlinkPlaceholder(root, entryPath, data)
		}

		return visit(filepath.FromSlash(entryPath), data, readErr)
	})
}
