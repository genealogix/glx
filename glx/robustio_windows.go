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

//go:build windows

package main

import (
	"errors"
	"math/rand"
	"os"
	"syscall"
	"time"
)

// errSharingViolation is syscall.Errno(32), equivalent to
// internal/syscall/windows.ERROR_SHARING_VIOLATION which is not importable.
const errSharingViolation syscall.Errno = 32 //nolint:mnd // Windows error code

const retryTimeout = 2000 * time.Millisecond //nolint:mnd // matches Go toolchain robustio

const maxSleep = 500 * time.Millisecond //nolint:mnd // cap per-retry sleep

// robustRename is like os.Rename but retries on transient Windows errors.
//
// Windows Defender, Search Indexer, OneDrive, and other processes can briefly
// hold file handles without FILE_SHARE_DELETE, causing MoveFileEx to fail with
// ERROR_ACCESS_DENIED or ERROR_SHARING_VIOLATION. These locks are transient —
// retrying with exponential backoff resolves them.
//
// Backoff: 1ms, 2ms, 4ms, 8ms, ... up to 500ms per sleep, 2s total timeout.
// Jitter added to each sleep to avoid synchronized retries.
//
// Modeled after Go's cmd/internal/robustio (used by cmd/go, gopls, golangci-lint).
func robustRename(oldpath, newpath string) error {
	var (
		lastErr   error
		start     time.Time
		nextSleep = 1 * time.Millisecond
	)

	for {
		err := os.Rename(oldpath, newpath)
		if err == nil || !isEphemeralError(err) {
			return err
		}

		lastErr = err

		if start.IsZero() {
			start = time.Now()
		} else if time.Since(start)+nextSleep >= retryTimeout {
			break
		}

		time.Sleep(nextSleep)

		// Exponential backoff: double the base, add jitter, cap at maxSleep
		nextSleep = min(nextSleep*2+time.Duration(rand.Int63n(int64(nextSleep))), maxSleep) //nolint:gosec // jitter, not crypto
	}

	return lastErr
}

// isEphemeralError returns true if err is a transient Windows filesystem error
// that may resolve by waiting.
func isEphemeralError(err error) bool {
	if errno, ok := errors.AsType[syscall.Errno](err); ok {
		switch errno {
		case syscall.ERROR_ACCESS_DENIED,
			syscall.ERROR_FILE_NOT_FOUND,
			errSharingViolation:
			return true
		}
	}

	return false
}
