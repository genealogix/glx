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

//go:build !windows

package main

import (
	"os"
)

// robustRename is like os.Rename. On non-Windows platforms, rename is atomic
// and does not suffer from transient file locking, so no retry is needed.
func robustRename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
