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
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// FilenameAllocator reserves basenames across case-insensitive filesystems and
// Unicode normalization forms. It does not access the filesystem: callers seed
// it with existing names before allocating new ones. Reservations are never
// released, so the first free suffix of a basename only moves forward and is
// cached. Reusing identical content already on disk is the caller's decision
// (glx import makes it with NextAvailable and Reserve, #1442).
type FilenameAllocator struct {
	used       map[string]string // normalized key -> first reserved spelling
	nextSuffix map[string]int    // normalized basename -> lowest suffix that may be free
}

// NewFilenameAllocator creates an allocator seeded with existing basenames.
func NewFilenameAllocator(existing ...string) *FilenameAllocator {
	a := &FilenameAllocator{
		used:       make(map[string]string, len(existing)),
		nextSuffix: make(map[string]int),
	}
	for _, name := range existing {
		a.reserveKey(name, FilenameCollisionKey(name))
	}

	return a
}

var filenameCaseFolders = sync.Pool{New: func() any {
	folder := cases.Fold()

	return &folder
}}

// FilenameCollisionKey folds case and canonical Unicode equivalents so names
// that address the same file on common filesystems share a reservation.
func FilenameCollisionKey(name string) string {
	name = strings.TrimRight(name, ". ")
	// ASCII is invariant under NFD and folds to its lowercase, so the common
	// case skips both normalization passes and the pooled caser.
	if isASCII(name) {
		return strings.ToLower(name)
	}
	folder := filenameCaseFolders.Get().(*cases.Caser) //nolint:errcheck // the private pool only stores *cases.Caser
	// Canonical caseless matching requires decomposition on both sides of
	// folding: folding itself can introduce a different normalization form.
	key := norm.NFD.String(folder.String(norm.NFD.String(name)))
	filenameCaseFolders.Put(folder)

	return key
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}

	return true
}

// Existing returns the first reserved spelling equivalent to name, if any.
func (a *FilenameAllocator) Existing(name string) (string, bool) {
	actual, ok := a.used[FilenameCollisionKey(name)]

	return actual, ok
}

// Reserve claims name if no equivalent spelling has been reserved.
func (a *FilenameAllocator) Reserve(name string) bool {
	return a.reserveKey(name, FilenameCollisionKey(name))
}

func (a *FilenameAllocator) reserveKey(name, key string) bool {
	if _, exists := a.used[key]; exists {
		return false
	}
	a.used[key] = name

	return true
}

// Allocate reserves basename unchanged when available, or the first free
// <stem>-N<ext> spelling starting at -2. The supplied spelling is preserved.
func (a *FilenameAllocator) Allocate(basename string) string {
	candidate, _ := a.NextAvailable(basename)
	a.reserveKey(candidate, FilenameCollisionKey(candidate))

	return candidate
}

// NextAvailable reports the spelling and ordinal Allocate would reserve,
// without reserving a name. It caches occupied suffixes. Ordinal 1 is the unsuffixed basename.
func (a *FilenameAllocator) NextAvailable(basename string) (string, int) {
	basename = PortableFilename(basename)
	key := FilenameCollisionKey(basename)
	if _, occupied := a.used[key]; !occupied {
		return basename, 1
	}

	ext := filepath.Ext(basename)
	stem := strings.TrimSuffix(basename, ext)
	for n := max(a.nextSuffix[key], 2); ; n++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, n, ext)
		if _, occupied := a.used[FilenameCollisionKey(candidate)]; !occupied {
			a.nextSuffix[key] = n

			return candidate, n
		}
	}
}

// NumberedFilenameSeed returns the collision key and ordinal of a canonical
// <base>-N<ext> name. Noncanonical numbers such as -02 are separate basenames.
func NumberedFilenameSeed(name string) (string, int, bool) {
	trimmed := strings.TrimRight(name, ". ")
	ext := filepath.Ext(trimmed)
	stem := strings.TrimSuffix(trimmed, ext)
	base, suffix, found := strings.CutLast(stem, "-")
	if !found {
		return "", 0, false
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n < 2 || FilenameCollisionKey(name) != FilenameCollisionKey(fmt.Sprintf("%s-%d%s", base, n, ext)) {
		return "", 0, false
	}

	return FilenameCollisionKey(base + ext), n, true
}

// PortableFilename avoids Win32's trailing-dot/space aliases and reserved
// device basenames. The original source spelling remains available as provenance.
func PortableFilename(name string) string {
	name = strings.TrimRight(name, ". ")
	if name == "" {
		return "unnamed"
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune("<>:\"/\\|?*", r) {
			return '-'
		}

		return r
	}, name)
	first, rest, hasDot := strings.Cut(name, ".")
	stem := strings.ToUpper(strings.TrimRight(first, " ."))
	switch stem {
	case "CON", "CONIN$", "CONOUT$", "PRN", "AUX", "NUL", "COM0", "LPT0", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "COM¹", "COM²", "COM³", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9", "LPT¹", "LPT²", "LPT³":
		if hasDot {
			return first + "-2." + rest
		}

		return first + "-2"
	}

	return name
}
