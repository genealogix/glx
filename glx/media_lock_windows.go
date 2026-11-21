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
	"os"

	"golang.org/x/sys/windows"
)

func lockFileEx(f *os.File, flags uint32) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, new(windows.Overlapped))
}

func tryLockFile(f *os.File) (bool, error) {
	err := lockFileEx(f, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}

	return err == nil, err
}

func lockFile(f *os.File) error { return lockFileEx(f, windows.LOCKFILE_EXCLUSIVE_LOCK) }

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
}
