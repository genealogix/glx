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
	"strings"
)

// File extension constants
const (
	FileExtGLX  = ".glx"
	FileExtYAML = ".yaml"
	FileExtYML  = ".yml"
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

// isGLXFile checks if a file has a GLX-related extension
func isGLXFile(filename string) bool {
	ext := filepath.Ext(filename)

	return ext == FileExtGLX || ext == FileExtYAML || ext == FileExtYML
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
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not check directory: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Read exactly one directory entry.
	// Readdirnames returns io.EOF if the directory is empty.
	_, err = f.Readdirnames(1)
	if err != nil {
		// io.EOF means directory is empty (success case)
		if err == io.EOF {
			return nil
		}
		// Other errors (permissions, I/O failures) should be returned
		return fmt.Errorf("error reading directory: %w", err)
	}

	// Successfully read an entry, so directory is not empty
	return ErrNonEmptyDirectory
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

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
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

// createDirectoryStructure creates a list of directories
func createDirectoryStructure(dirs []string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPermissions); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
