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
	"os"
	"path/filepath"
)

type importMetadataSnapshot struct {
	contents map[string][]byte
	missing  map[string]bool
	valid    bool
}

func snapshotImportMetadata(root string, files map[string][]byte) importMetadataSnapshot {
	snapshot := importMetadataSnapshot{contents: make(map[string][]byte), missing: make(map[string]bool), valid: true}
	for rel := range files {
		content, err := os.ReadFile(filepath.Join(root, rel))
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
	for rel := range snapshot.missing {
		if _, err := os.Lstat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			return false
		}
	}
	for rel, original := range snapshot.contents {
		content, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil || !bytes.Equal(original, content) {
			return false
		}
	}

	return true
}
