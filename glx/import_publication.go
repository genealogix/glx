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
	"os"
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
