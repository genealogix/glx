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
	"os"
	"path/filepath"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// ErrSingleFileArchiveUnsupported is returned by commands that write entities
// in the multi-file layout when they are pointed at a single-file archive.
var ErrSingleFileArchiveUnsupported = errors.New("does not support single-file archives yet")

// loadArchiveForRead loads an archive for a read-only command. It is the one
// loading path every read-only runner shares (#1464):
//
//   - A directory is loaded through LoadArchiveCached, so a fresh binary cache
//     is used when present.
//   - A single file is read with readSingleFileArchive, and the standard
//     vocabularies are merged into it. A directory archive gets them from the
//     loader already; without this step, a reference-type property such as
//     residence resolved to a place name for a directory archive but not for
//     the same archive in one file. mergeStandardVocabularies only fills
//     vocabularies the archive leaves empty, so archive-defined ones are kept.
//   - Duplicate-ID warnings are sorted and written to streams.ErrOut, one
//     "Warning:" line each, sanitized for the terminal.
func loadArchiveForRead(streams *IOStreams, path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, err := LoadArchiveCached(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}
		sorted := append([]string(nil), duplicates...)
		sort.Strings(sorted)
		for _, d := range sorted {
			streams.Errorf("Warning: %s\n", d)
		}

		return archive, nil
	}

	archive, err := readSingleFileArchive(path, false)
	if err != nil {
		return nil, err
	}
	if err := mergeStandardVocabularies(archive); err != nil {
		return nil, fmt.Errorf("failed to load standard vocabularies: %w", err)
	}

	return archive, nil
}

// refuseSingleFileArchive returns an error when path is a single-file archive
// and the named command can only write the multi-file layout (#1356). Both
// ways of pointing at one are caught before anything is read or written:
//
//   - path is the .glx file itself, which previously failed with an OS
//     "not a directory" error;
//   - path is the directory `glx init --single-file` creates, which previously
//     succeeded and wrote a multi-file entity beside the single file, leaving
//     a mixed layout whose single file silently lacked the new entity.
//
// A directory counts as a single-file archive when its root holds exactly one
// .glx file and none of the multi-file layout's directories exist. Both tests
// follow the loader's rules (walkGLXFiles), so the refusal and the load agree
// on what the archive is:
//
//   - layout directory names match case-insensitively (PERSONS/ is persons/);
//   - a media/ directory holding only the binary store media/files/ is not
//     multi-file layout: `glx import --format single` puts it beside the file;
//   - dot-prefixed names (._archive.glx, .git/) are not archive content;
//   - a root .glx entry that is a relative symlink to a regular file inside
//     the archive counts, as the loader reads it; one that escapes does not.
func refuseSingleFileArchive(command, path string) error {
	// A missing or unreadable path is left for the loader to report.
	singleFile := ""
	if info, err := os.Stat(path); err == nil {
		switch {
		case info.IsDir():
			singleFile = singleFileArchiveInDir(path)
		case strings.EqualFold(filepath.Ext(path), ".glx"):
			singleFile = path
		}
	}
	if singleFile == "" {
		return nil
	}

	return fmt.Errorf("glx %s %w (%s); convert it first with: glx split %s <output-directory>",
		command, ErrSingleFileArchiveUnsupported, singleFile, singleFile)
}

// singleFileArchiveInDir returns the path of the one .glx file at dir's root
// when dir has the single-file layout, and "" otherwise.
func singleFileArchiveInDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	resolvedRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return ""
	}

	var glxFiles []string
	for _, e := range entries {
		name := e.Name()
		if isDotName(name) {
			continue
		}
		full := filepath.Join(dir, name)
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.IsDir() {
			if isMultiFileLayoutDir(full, name) {
				return ""
			}

			continue
		}
		if isGLXFile(name) && info.Mode().IsRegular() && rootEntryIsContained(resolvedRoot, full, e) {
			glxFiles = append(glxFiles, full)
		}
	}
	if len(glxFiles) != 1 {
		return ""
	}

	return glxFiles[0]
}

// isMultiFileLayoutDir reports whether the root directory name is one the
// multi-file layout creates, compared case-insensitively. media/ counts only
// when it holds more than the binary store media/files/, which a single-file
// archive with media also has.
func isMultiFileLayoutDir(path, name string) bool {
	if strings.EqualFold(name, mediaDirName) {
		entries, err := os.ReadDir(path)
		if err != nil {
			return true
		}
		for _, e := range entries {
			if isDotName(e.Name()) {
				continue
			}
			if !e.IsDir() || !strings.EqualFold(e.Name(), mediaFilesSubdir) {
				return true
			}
		}

		return false
	}
	if strings.EqualFold(name, glxlib.ArchiveDirVocabularies) {
		return true
	}
	for _, et := range glxlib.AllEntityTypes {
		if strings.EqualFold(name, et.String()) {
			return true
		}
	}

	return false
}

// rootEntryIsContained reports whether a root .glx entry is one the loader
// reads: a regular file, or a relative symlink whose target resolves inside
// the archive and outside any dot-prefixed directory. walkGLXFiles refuses an
// absolute link and one that escapes the root.
func rootEntryIsContained(resolvedRoot, full string, e os.DirEntry) bool {
	if e.Type()&os.ModeSymlink == 0 {
		return true
	}
	target, err := os.Readlink(full)
	if err != nil || filepath.IsAbs(target) {
		return false
	}
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}

	return !pathHasDotComponent(filepath.ToSlash(rel))
}

// mediaFilesSubdir is the binary store's name inside media/ (glxlib.MediaFilesDir).
const mediaFilesSubdir = "files"

// archivePathHint returns a hint to append to a "not found" error when a
// command's positional person argument names an existing archive (a directory
// or a .glx file) rather than a person, so the user is pointed at --archive
// instead of left debugging a missing person (#719). It returns "" otherwise.
func archivePathHint(command, arg string) string {
	info, err := os.Stat(arg)
	if err != nil {
		return ""
	}
	if !info.IsDir() && !strings.EqualFold(filepath.Ext(arg), ".glx") {
		return ""
	}

	return fmt.Sprintf("\nHint: the positional argument is a person filter, not an archive path.\n"+
		"Did you mean: glx %s --archive %s", command, arg)
}
