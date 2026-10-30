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
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type importMetadataSnapshot struct {
	contents map[string][32]byte
	missing  map[string]bool
	valid    bool
}

func snapshotImportMetadata(root string, files map[string][]byte) importMetadataSnapshot {
	snapshot := importMetadataSnapshot{contents: make(map[string][32]byte), missing: make(map[string]bool), valid: true}
	handle, err := os.OpenRoot(root)
	if err != nil {
		snapshot.valid = false

		return snapshot
	}
	defer func() { _ = handle.Close() }()
	for rel := range files {
		content, err := hashMetadataFile(handle, rel)
		switch {
		case os.IsNotExist(err):
			snapshot.missing[rel] = true
		case err != nil:
			snapshot.valid = false
		default:
			snapshot.contents[rel] = content
		}
	}

	return snapshot
}

// unchanged confirms that a failed writer published no changed references.
// Unknown filesystem outcomes retain binaries for manual recovery.
func (snapshot importMetadataSnapshot) unchanged(root string) bool {
	if !snapshot.valid {
		return false
	}
	handle, err := os.OpenRoot(root)
	if err != nil {
		return false
	}
	defer func() { _ = handle.Close() }()
	for rel := range snapshot.missing {
		if _, err := handle.Lstat(rel); !os.IsNotExist(err) {
			return false
		}
	}
	for rel, original := range snapshot.contents {
		content, err := hashMetadataFile(handle, rel)
		if err != nil || original != content {
			return false
		}
	}

	return true
}

// mergeMetadataUnchanged confirms that the active archive still consists of the
// metadata loaded before the merge. A partial write or unreadable state retains
// media because active references may already use it.
func mergeMetadataUnchanged(root string, loaded map[string][]byte) bool {
	current, err := collectGLXFilesFromDir(root)
	if err != nil || len(current) != len(loaded) {
		return false
	}
	for rel, old := range loaded {
		if !bytes.Equal(old, current[rel]) {
			return false
		}
	}

	return true
}

// hashMetadataFile streams existing metadata so a successful import does not
// keep a second archive's worth of bytes alive only for a failure check.
func hashMetadataFile(root *os.Root, rel string) ([32]byte, error) {
	f, err := root.Open(rel)
	if err != nil {
		return [32]byte{}, err
	}
	info, err := f.Stat()
	if err != nil {
		return [32]byte{}, errors.Join(err, f.Close())
	}
	if !info.Mode().IsRegular() {
		return [32]byte{}, errors.Join(os.ErrInvalid, f.Close())
	}
	digest, hashErr := hashReader(f)

	return digest, errors.Join(hashErr, f.Close())
}

// publishSingleFileImport reconciles the filesystem after a rename error. NFS
// can complete a rename and still report an error, so the error alone cannot
// justify deleting media that the installed metadata may now reference.
// mayHavePublished is false on failure only when the owned staged file remains
// at its source and the destination is demonstrably a different file or absent.
func publishSingleFileImport(staged, target string, rename func(string, string) error) (mayHavePublished bool, err error) {
	entry, err := os.Lstat(staged)
	if err != nil {
		return true, fmt.Errorf("cannot verify staged archive %s before publication; media and staging retained: %w", staged, err)
	}
	if !entry.Mode().IsRegular() {
		return true, fmt.Errorf("staged archive %s is not a regular file; media and staging retained: %w", staged, os.ErrInvalid)
	}
	opened, err := openWithin(filepath.Dir(staged), staged)
	if err != nil {
		return true, fmt.Errorf("cannot open staged archive %s before publication; media and staging retained: %w", staged, err)
	}
	// Windows path-based FileInfo loads identity lazily in SameFile. File.Stat
	// captures it from the open handle now, before this path moves or is reused.
	// Close the handle before rename so Windows can replace the target.
	owned, statErr := opened.Stat()
	closeErr := opened.Close()
	if err := errors.Join(statErr, closeErr); err != nil {
		return true, fmt.Errorf("cannot capture staged archive identity for %s; media and staging retained: %w", staged, err)
	}
	if !owned.Mode().IsRegular() || !os.SameFile(entry, owned) {
		return true, fmt.Errorf("staged archive %s changed while opening; media and staging retained: %w", staged, os.ErrInvalid)
	}
	renameErr := rename(staged, target)
	if renameErr == nil {
		return true, nil
	}

	// Stat follows a destination symlink: it may expose our staged metadata
	// without moving it, in which case deleting that source is unsafe too.
	destination, destinationErr := os.Stat(target)
	if destinationErr == nil && os.SameFile(owned, destination) {
		return true, fmt.Errorf("archive %s was published despite a rename error; media retained: %w", target, renameErr)
	}
	source, sourceErr := os.Lstat(staged)
	if sourceErr == nil && os.SameFile(owned, source) && (destinationErr == nil || os.IsNotExist(destinationErr)) {
		return false, fmt.Errorf("publishing archive %s: %w", target, renameErr)
	}

	return true, fmt.Errorf("archive publication outcome for %s is uncertain; media retained and any staged metadata is at %s: %w", target, staged, errors.Join(renameErr, sourceErr, destinationErr))
}

// hasImportMediaMetadata distinguishes Media YAML that needs a media directory
// from an unrelated sibling media path in a truly media-free import.
func hasImportMediaMetadata(files map[string][]byte) bool {
	for rel := range files {
		if strings.HasPrefix(rel, "media/") {
			return true
		}
	}

	return false
}

func checkImportMediaMetadata(rootDir string, files map[string][]byte) error {
	if !hasImportMediaMetadata(files) {
		return nil
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return fmt.Errorf("opening archive for media metadata: %w", err)
	}
	defer func() { _ = root.Close() }()

	return inspectImportMediaMetadata(root, files)
}

// inspectImportMediaMetadata ignores media/files: only the directory and files
// the serializer will write are relevant, including URL-only Media entities.
func inspectImportMediaMetadata(root *os.Root, files map[string][]byte) error {
	info, err := root.Lstat("media")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspecting media metadata directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("media metadata directory must be a plain directory: %w", os.ErrInvalid)
	}
	for rel := range files {
		if !strings.HasPrefix(rel, "media/") {
			continue
		}
		info, err := root.Lstat(rel)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspecting media metadata %s: %w", rel, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("media metadata %s must be a regular file: %w", rel, os.ErrInvalid)
		}
	}

	return nil
}

// writeImportFilesToDir confines Media metadata writes to the archive, including
// after the preflight check. All other metadata uses the existing writer policy.
func writeImportFilesToDir(rootDir string, files map[string][]byte) error {
	if !hasImportMediaMetadata(files) {
		return writeFilesToDir(rootDir, files)
	}
	if err := os.MkdirAll(rootDir, dirPermissions); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return fmt.Errorf("opening archive for media metadata: %w", err)
	}
	defer func() { _ = root.Close() }()
	if err := inspectImportMediaMetadata(root, files); err != nil {
		return err
	}
	// Keep the established writer for non-Media paths without cloning any
	// content bytes. Media paths alone use the confined root below.
	otherFiles := make(map[string][]byte, len(files))
	for rel, data := range files {
		if !strings.HasPrefix(rel, "media/") {
			otherFiles[rel] = data
		}
	}
	if err := writeFilesToDir(rootDir, otherFiles); err != nil {
		return err
	}
	if err := root.MkdirAll("media", dirPermissions); err != nil {
		return fmt.Errorf("creating media metadata directory: %w", err)
	}
	for rel, data := range files {
		if !strings.HasPrefix(rel, "media/") {
			continue
		}
		if err := root.WriteFile(rel, data, filePermissions); err != nil {
			return fmt.Errorf("writing media metadata %s: %w", rel, err)
		}
	}

	return nil
}
