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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// File extension constants
const (
	FileExtGLX = ".glx"
)

// File permission constants
const (
	dirPermissions  = 0o755
	filePermissions = 0o644
)

// ensureGLXExtension adds .glx extension if not present
func ensureGLXExtension(path string) string {
	if !strings.HasSuffix(path, FileExtGLX) {
		return path + FileExtGLX
	}

	return path
}

// isGLXFile checks if a file has the .glx extension.
func isGLXFile(filename string) bool {
	return filepath.Ext(filename) == FileExtGLX
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}

// isDirectoryEmpty checks if a directory is empty
func isDirectoryEmpty(path string) error {
	path = filepath.Clean(path)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not check directory: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Read up to 6 entries: display at most 5, use the 6th to detect truncation.
	// Readdirnames may return a non-empty slice AND io.EOF when the directory
	// has fewer than n entries, so we must check len(names) before treating
	// io.EOF as "empty directory".
	const displayLimit = 5
	names, err := f.Readdirnames(displayLimit + 1)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error reading directory: %w", err)
	}

	if len(names) == 0 {
		return nil
	}

	// Show what files were found to help diagnose unexpected blockers
	// (e.g., .DS_Store, OneDrive sync files, hidden files)
	truncated := len(names) > displayLimit
	display := names
	if truncated {
		display = names[:displayLimit]
	}
	listing := strings.Join(display, ", ")
	if truncated {
		listing += ", ..."
	}
	return fmt.Errorf("%w (found: %s)", ErrNonEmptyDirectory, listing)
}

// collectGLXFilesFromDir recursively collects all GLX/YAML files from a directory
// into a map with relative paths as keys and file contents as values.
// Only files with .glx, .yaml, or .yml extensions are included.
func collectGLXFilesFromDir(rootDir string) (map[string][]byte, error) {
	files := make(map[string][]byte)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isGLXFile(d.Name()) {
			return nil
		}

		path = filepath.Clean(path)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// On Windows, Git stores symlinks as text files containing the
		// target path. Detect these and read the actual target file.
		if runtime.GOOS == "windows" {
			data = resolveSymlinkPlaceholder(path, data)
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		files[relPath] = data

		return nil
	})

	return files, err
}

// resolveSymlinkPlaceholder detects Git symlink placeholders on Windows.
// Git stores symlinks as small text files containing the target path when
// core.symlinks is false (the default on Windows). This function detects
// such files and reads the actual target content.
func resolveSymlinkPlaceholder(filePath string, data []byte) []byte {
	content := strings.TrimSpace(string(data))
	// Symlink placeholders are short, single-line, and look like relative paths
	if len(content) > 200 || strings.ContainsAny(content, "\n\r{[") {
		return data
	}
	if !strings.Contains(content, "/") && !strings.Contains(content, "\\") {
		return data
	}
	// Git symlink targets are always relative; reject absolute paths
	// to prevent reading arbitrary files.
	if filepath.IsAbs(filepath.FromSlash(content)) || filepath.VolumeName(filepath.FromSlash(content)) != "" {
		return data
	}

	// Resolve the target path relative to the file's directory
	dir := filepath.Dir(filePath)
	targetPath := filepath.Join(dir, filepath.FromSlash(content))
	targetPath = filepath.Clean(targetPath)
	targetData, err := os.ReadFile(targetPath) //nolint:gosec // path is relative to archive, not user input
	if err != nil {
		return data // Not a valid symlink placeholder; return original content
	}

	return targetData
}

// writeFilesToDir writes a map of files (relative path -> content) to a directory
func writeFilesToDir(rootDir string, files map[string][]byte) error {
	// Create root directory
	if err := os.MkdirAll(rootDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write all files
	for relPath, content := range files {
		absPath := filepath.Join(rootDir, relPath)

		// Create parent directory
		parentDir := filepath.Dir(absPath)
		if err := os.MkdirAll(parentDir, dirPermissions); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
		}

		// Write file
		if err := os.WriteFile(absPath, content, filePermissions); err != nil {
			return fmt.Errorf("failed to write file %s: %w", absPath, err)
		}
	}

	return nil
}

// atomicWriteFile writes data to a file atomically using a temp file + rename.
// The target file either has the old content or the new content, never a partial
// write. The temp file is created in the same directory to ensure same-filesystem
// rename.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".glx-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp file on any failure
	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("setting file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replacing target file: %w", err)
	}

	success = true
	return nil
}

// createDirectoryStructure creates a list of directories
func createDirectoryStructure(dirs []string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPermissions); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
