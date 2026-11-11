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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// lockImportMedia takes an exclusive, cross-process lock on the media
// directory an import writes to, and returns its release. It is held from
// destination allocation through publication or rollback (#1443): without
// it, import A could commit photo.jpg, import B could reuse those bytes and
// publish, and A's rollback after a failed publication would then delete the
// file B's archive references. Single-file archives that sit side by side
// share one media/files directory, which is how two imports meet.
//
// The lock file lives in the system temporary directory, named after the
// resolved archive directory, so no archive gains a stray file. A second
// import waits, saying so on out, rather than failing.
func lockImportMedia(archiveDir string, out io.Writer) (func(), error) {
	path, err := importMediaLockPath(archiveDir)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, filePermissions)
	if err != nil {
		return nil, fmt.Errorf("opening media lock %s: %w", path, err)
	}
	locked, err := tryLockFile(f)
	if err == nil && !locked {
		_, _ = fmt.Fprintf(out, "Waiting for another import writing media under %s...\n", archiveDir)
		err = lockFile(f)
	}
	if err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("locking media directory %s: %w", archiveDir, err)
	}

	return func() {
		_ = unlockFile(f)
		_ = f.Close()
	}, nil
}

// importMediaLockPath names the lock for archiveDir. The directory is
// resolved through symlinks, and folded to lower case where the filesystem
// usually ignores case, so two spellings of one directory share a lock.
func importMediaLockPath(archiveDir string) (string, error) {
	abs, err := filepath.Abs(archiveDir)
	if err != nil {
		return "", fmt.Errorf("resolving media directory %s: %w", archiveDir, err)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	key := filepath.Clean(abs)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		key = strings.ToLower(key)
	}
	sum := sha256.Sum256([]byte(key))

	return filepath.Join(os.TempDir(), "glx-media-"+hex.EncodeToString(sum[:8])+".lock"), nil
}
