//go:build !unix && !windows

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

import "os"


func openMCPWatchSingleFile(path string) (*os.File, os.FileInfo, error) {
	expected, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if !expected.Mode().IsRegular() {
		return nil, nil, errMCPWatchSourceType
	}
	// #nosec G304 -- This platform lacks a native no-follow open. The shared
	// descriptor reader verifies the captured identity before reading bytes.
	file, err := os.OpenFile(path, os.O_RDONLY, 0)

	return file, expected, err
}
