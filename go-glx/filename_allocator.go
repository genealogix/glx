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

package glx

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// FilenameAllocator reserves basenames across case-insensitive filesystems and
// Unicode normalization forms. It does not access the filesystem: callers seed
// it with existing names before allocating new ones.
type FilenameAllocator struct {
	used map[string]string // normalized key -> first reserved spelling
}

// NewFilenameAllocator creates an allocator seeded with existing basenames.
func NewFilenameAllocator(existing ...string) *FilenameAllocator {
	a := &FilenameAllocator{used: make(map[string]string, len(existing))}
	for _, name := range existing {
		a.Reserve(name)
	}

	return a
}

// FilenameCollisionKey folds case and canonical Unicode equivalents so names
// that can address the same file on common Windows and macOS filesystems share
// a reservation. Conservative extra collisions only cause a numbered suffix.
func FilenameCollisionKey(name string) string {
	return norm.NFC.String(cases.Fold().String(norm.NFC.String(name)))
}

// Existing returns the first reserved spelling equivalent to name, if any.
func (a *FilenameAllocator) Existing(name string) (string, bool) {
	actual, ok := a.used[FilenameCollisionKey(name)]

	return actual, ok
}

// Reserve claims name if no equivalent spelling has been reserved.
func (a *FilenameAllocator) Reserve(name string) bool {
	key := FilenameCollisionKey(name)
	if _, exists := a.used[key]; exists {
		return false
	}
	a.used[key] = name

	return true
}

// Allocate reserves basename unchanged when available, or the first free
// <stem>-N<ext> spelling starting at -2. The supplied spelling is preserved.
func (a *FilenameAllocator) Allocate(basename string) string {
	if a.Reserve(basename) {
		return basename
	}

	ext := filepath.Ext(basename)
	stem := strings.TrimSuffix(basename, ext)
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, n, ext)
		if a.Reserve(candidate) {
			return candidate
		}
	}
}
