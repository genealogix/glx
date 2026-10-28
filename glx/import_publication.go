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
		if os.IsNotExist(err) {
			snapshot.missing[rel] = true
		} else if err != nil {
			snapshot.valid = false
		} else {
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
