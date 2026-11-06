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
	"os"
)

var errMergeRenameUncertain = errors.New("merge rename outcome is uncertain; recovery data retained")

// A filesystem can complete a rename but report failure (for example after an
// interrupted network-filesystem reply). Reconcile identities before deciding
// which path rollback owns. Unknown outcomes must retain every recovery copy.
func trackMergeRename(root *os.Root, from, to string, rename func(*os.Root, string, string) error) (bool, error) {
	original, err := root.Lstat(from)
	if err != nil {
		return false, err
	}
	if err := rename(root, from, to); err != nil {
		remaining, fromErr := root.Lstat(from)
		if fromErr == nil && os.SameFile(remaining, original) {
			return false, err
		}
		placed, toErr := root.Lstat(to)
		if os.IsNotExist(fromErr) && toErr == nil && os.SameFile(placed, original) {
			return true, err
		}

		return false, errors.Join(err, errMergeRenameUncertain)
	}

	return true, nil
}
