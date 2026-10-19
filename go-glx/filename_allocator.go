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
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// FilenameAllocator reserves basenames across case-insensitive filesystems and
// Unicode normalization forms. It does not access the filesystem: callers seed
// it with existing names before allocating new ones.
type FilenameAllocator struct {
	used       map[string]string // normalized key -> first reserved spelling
	nextSuffix map[string]int    // normalized basename -> next suffix worth trying
	claimed    map[string]bool   // on-disk spellings already used by this import
	seeded     map[string][]int  // numbered on-disk candidates, grouped by basename
	seededSort map[string]bool   // whether a basename's seeded suffixes are sorted
	seededName map[string]string // original spelling -> normalized key
	remaining  map[string]int    // unclaimed on-disk variants per normalized key
	exhausted  map[string]bool   // candidates with no unclaimed on-disk variant
	firstOpen  map[string]int    // first candidate not yet exhausted per basename
}

// NewFilenameAllocator creates an allocator seeded with existing basenames.
func NewFilenameAllocator(existing ...string) *FilenameAllocator {
	a := &FilenameAllocator{
		used:       make(map[string]string, len(existing)),
		nextSuffix: make(map[string]int),
		claimed:    make(map[string]bool),
		seeded:     make(map[string][]int),
		seededSort: make(map[string]bool),
		seededName: make(map[string]string, len(existing)),
		remaining:  make(map[string]int),
		exhausted:  make(map[string]bool),
		firstOpen:  make(map[string]int),
	}
	for _, name := range existing {
		key := FilenameCollisionKey(name)
		a.reserveKey(name, key)
		a.recordExisting(name, key)
	}

	return a
}

// FilenameCollisionKey folds case and canonical Unicode equivalents so names
// that can address the same file on common Windows and macOS filesystems share
// a reservation. Conservative extra collisions only cause a numbered suffix.
func FilenameCollisionKey(name string) string {
	return cases.Fold().String(norm.NFC.String(strings.TrimRight(name, ". ")))
}

// Existing returns the first reserved spelling equivalent to name, if any.
func (a *FilenameAllocator) Existing(name string) (string, bool) {
	actual, ok := a.used[FilenameCollisionKey(name)]

	return actual, ok
}

// Reserve claims name if no equivalent spelling has been reserved.
func (a *FilenameAllocator) Reserve(name string) bool {
	key := FilenameCollisionKey(name)
	if !a.reserveKey(name, key) {
		return false
	}
	a.recordExisting(name, key)

	return true
}

func (a *FilenameAllocator) reserveKey(name, key string) bool {
	if _, exists := a.used[key]; exists {
		return false
	}
	a.used[key] = name

	return true
}

func (a *FilenameAllocator) recordExisting(name, key string) {
	if _, seen := a.seededName[name]; seen {
		return
	}
	a.seededName[name] = key
	a.remaining[key]++
	a.recordSeed(name)
}

// Allocate reserves basename unchanged when available, or the first free
// <stem>-N<ext> spelling starting at -2. The supplied spelling is preserved.
func (a *FilenameAllocator) Allocate(basename string) string {
	basename = portableFilename(basename)
	key := FilenameCollisionKey(basename)
	if a.reserveKey(basename, key) {
		return basename
	}

	ext := filepath.Ext(basename)
	stem := strings.TrimSuffix(basename, ext)
	next := max(a.nextSuffix[key], 2)
	for n := next; ; n++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, n, ext)
		if a.reserveKey(candidate, FilenameCollisionKey(candidate)) {
			a.nextSuffix[key] = n + 1

			return candidate
		}
	}
}

// AllocateMatching reuses an occupied candidate when match finds identical
// content on disk. match receives candidate names in allocation order and may
// return an existing spelling (including a case or Unicode variant). The
// allocator itself remains independent of filesystem I/O.
func (a *FilenameAllocator) AllocateMatching(basename string, match func(string) (string, error)) (string, bool, error) {
	basename = portableFilename(basename)
	baseKey := FilenameCollisionKey(basename)
	ext := filepath.Ext(basename)
	stem := strings.TrimSuffix(basename, ext)
	for n := max(a.firstOpen[baseKey], 1); ; {
		candidate := basename
		if n > 1 {
			candidate = fmt.Sprintf("%s-%d%s", stem, n, ext)
		}
		key := FilenameCollisionKey(candidate)
		if _, occupied := a.used[key]; !occupied {
			a.used[key] = candidate
			a.claimed[candidate] = true
			a.exhausted[key] = true
			a.nextSuffix[baseKey] = n + 1
			a.advanceFirstOpen(baseKey, basename, stem, ext)

			return candidate, false, nil
		}
		if !a.exhausted[key] {
			actual, err := match(candidate)
			if err != nil {
				return "", false, err
			}
			if actual != "" && !a.claimed[actual] {
				a.claimed[actual] = true
				if a.seededName[actual] == key {
					a.remaining[key]--
				}
				if a.remaining[key] == 0 {
					a.exhausted[key] = true
					a.advanceFirstOpen(baseKey, basename, stem, ext)
				}

				return actual, true, nil
			}
		}
		n++
		if n < a.nextSuffix[baseKey] {
			seed := a.nextSeededSuffix(baseKey, n)
			if seed != 0 && seed < a.nextSuffix[baseKey] {
				n = seed
			} else {
				n = a.nextSuffix[baseKey]
			}
		}
	}
}

func (a *FilenameAllocator) advanceFirstOpen(baseKey, basename, stem, ext string) {
	n := max(a.firstOpen[baseKey], 1)
	for {
		candidate := basename
		if n > 1 {
			candidate = fmt.Sprintf("%s-%d%s", stem, n, ext)
		}
		if !a.exhausted[FilenameCollisionKey(candidate)] {
			break
		}
		n++
	}
	a.firstOpen[baseKey] = n
}

func (a *FilenameAllocator) recordSeed(name string) {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	base, suffix, found := strings.CutLast(stem, "-")
	if !found {
		return
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n < 2 {
		return
	}
	key := FilenameCollisionKey(base + ext)
	a.seeded[key] = append(a.seeded[key], n)
	a.seededSort[key] = false
}

func (a *FilenameAllocator) nextSeededSuffix(key string, atLeast int) int {
	seeds := a.seeded[key]
	if !a.seededSort[key] {
		sort.Ints(seeds)
		a.seededSort[key] = true
	}
	i := sort.SearchInts(seeds, atLeast)
	if i < len(seeds) {
		return seeds[i]
	}

	return 0
}

// portableFilename avoids Win32's trailing-dot/space aliases and reserved
// device basenames. The original source spelling remains available as provenance.
func portableFilename(name string) string {
	name = strings.TrimRight(name, ". ")
	if name == "" {
		return "unnamed"
	}
	if strings.ContainsAny(name, "<>:\"/\\|?*") {
		name = strings.Map(func(r rune) rune {
			if strings.ContainsRune("<>:\"/\\|?*", r) {
				return '-'
			}

			return r
		}, name)
	}
	first, rest, hasDot := strings.Cut(name, ".")
	stem := strings.ToUpper(strings.TrimRight(first, " ."))
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "COM¹", "COM²", "COM³", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9", "LPT¹", "LPT²", "LPT³":
		if hasDot {
			return first + "-2." + rest
		}

		return first + "-2"
	}

	return name
}
