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
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"slices"
	"strings"
	"unicode"
)

const (
	// WarningTagGEDZIP is the tag used for GEDZIP-specific import warnings.
	WarningTagGEDZIP  = "GEDZIP"
	gedzipGedcomEntry = "gedcom.ged"
)

// ImportGEDZIP imports a GEDZIP (.gdz) bundle from an fs.FS abstraction without
// performing direct filesystem I/O.
//
// The GEDZIP specification mandates a root entry named gedcom.ged. ImportGEDZIP
// prefers gedcom.ged at root, but falls back to case-insensitive .ged or .gedcom
// discovery (requiring exactly one candidate) at the archive root or within a single
// wrapper directory. When fallback discovery is used, a warning is recorded on
// ImportResult.
//
// Referenced media files are resolved against the bundle relative to the GEDCOM file's
// directory. Exact matches are tried first, followed by GEDCOM 7.0 percent-decoding
// (url.PathUnescape) and case-insensitive matching. Unresolved media references emit
// warnings on ImportResult rather than failing the import.
func ImportGEDZIP(bundle fs.FS, logW io.Writer) (*GLXFile, *ImportResult, error) {
	if bundle == nil {
		return nil, nil, ErrGEDZIPNilBundle
	}

	// 1. Inventory entries and validate entry names and duplicates
	regularFiles, regularFilesSet, caseMap, err := inventoryAndValidateBundle(bundle)
	if err != nil {
		return nil, nil, err
	}

	// 2. Discover root GEDCOM
	gedcomPath, usedFallback, err := discoverRootGEDCOM(regularFiles, regularFilesSet)
	if err != nil {
		return nil, nil, err
	}

	// 3. Open and import the GEDCOM file
	f, err := bundle.Open(gedcomPath)
	if err != nil {
		if errors.Is(err, zip.ErrAlgorithm) {
			return nil, nil, fmt.Errorf("%w: %q: %w", ErrGEDZIPUnsupportedAlgorithm, gedcomPath, err)
		}

		return nil, nil, fmt.Errorf("opening gedcom entry %q: %w", gedcomPath, err)
	}
	defer f.Close()

	glxFile, result, err := ImportGEDCOM(f, logW)
	if err != nil {
		return nil, nil, err
	}

	// 4. Record warning if non-standard GEDCOM entry fallback was used
	if usedFallback {
		result.Statistics.Warnings = append(result.Statistics.Warnings, ImportWarning{
			Line:    0,
			Tag:     WarningTagGEDZIP,
			Message: fmt.Sprintf("using non-standard GEDCOM entry %q", gedcomPath),
		})
	}

	// 5. Resolve media references
	resolveBundleMediaFiles(gedcomPath, result, regularFilesSet, caseMap)

	return glxFile, result, nil
}

type fsUnwrapper interface {
	Unwrap() fs.FS
}

// maxFSUnwrapDepth bounds unwrapFS so a wrapper whose Unwrap returns itself (or
// a cycle of wrappers) cannot spin forever. Real wrapper chains are one or two
// deep; the interfaces involved are not guaranteed comparable, so a depth cap is
// the only cycle guard that cannot itself panic on `==`.
const maxFSUnwrapDepth = 16

func unwrapFS(f fs.FS) fs.FS {
	for range maxFSUnwrapDepth {
		u, ok := f.(fsUnwrapper)
		if !ok {
			return f
		}
		next := u.Unwrap()
		if next == nil {
			return f
		}
		f = next
	}

	return f
}

// inventoryAndValidateBundle collects regular files in the bundle while checking
// entry names for security violations (NUL, backslash, absolute path, volume prefix,
// directory traversal) and Unicode case-folded duplicate collisions.
func inventoryAndValidateBundle(bundle fs.FS) ([]string, map[string]struct{}, map[string]string, error) {
	underlying := unwrapFS(bundle)
	if zr, ok := underlying.(*zip.Reader); ok {
		return collectZipFiles(zr.File)
	}
	if zrc, ok := underlying.(*zip.ReadCloser); ok {
		return collectZipFiles(zrc.File)
	}

	return collectGenericFS(bundle)
}

// foldRune returns the canonical minimum rune in r's Unicode SimpleFold cycle.
func foldRune(r rune) rune {
	minR := r
	for curr := unicode.SimpleFold(r); curr != r; curr = unicode.SimpleFold(curr) {
		if curr < minR {
			minR = curr
		}
	}

	return minR
}

// foldKey returns a canonical Unicode case-folded string suitable for collision detection
// and case-insensitive lookup.
func foldKey(s string) string {
	return strings.Map(foldRune, s)
}

func collectZipFiles(files []*zip.File) ([]string, map[string]struct{}, map[string]string, error) {
	var regularFiles []string
	regularFilesSet := make(map[string]struct{})
	caseMap := make(map[string]string)
	seen := make(map[string]struct{}, len(files))

	for _, f := range files {
		if err := validateEntryName(f.Name); err != nil {
			return nil, nil, nil, err
		}
		key := foldKey(path.Clean(f.Name))
		if _, dup := seen[key]; dup {
			return nil, nil, nil, fmt.Errorf("%w: %q", ErrGEDZIPDuplicateEntry, f.Name)
		}
		seen[key] = struct{}{}

		if f.Mode().IsRegular() && !f.FileInfo().IsDir() {
			regularFiles = append(regularFiles, f.Name)
			regularFilesSet[f.Name] = struct{}{}
			caseMap[foldKey(f.Name)] = f.Name
		}
	}

	return regularFiles, regularFilesSet, caseMap, nil
}

func collectGenericFS(bundle fs.FS) ([]string, map[string]struct{}, map[string]string, error) {
	var regularFiles []string
	regularFilesSet := make(map[string]struct{})
	caseMap := make(map[string]string)
	walkSeen := make(map[string]struct{})

	err := fs.WalkDir(bundle, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if strings.Contains(walkErr.Error(), "duplicate entries in zip file") {
				return fmt.Errorf("%w: %w", ErrGEDZIPDuplicateEntry, walkErr)
			}

			return walkErr
		}
		if p == "." {
			return nil
		}

		if err := validateEntryName(p); err != nil {
			return err
		}

		key := foldKey(path.Clean(p))
		if _, dup := walkSeen[key]; dup {
			return fmt.Errorf("%w: %q", ErrGEDZIPDuplicateEntry, p)
		}
		walkSeen[key] = struct{}{}

		if d.Type().IsRegular() {
			regularFiles = append(regularFiles, p)
			regularFilesSet[p] = struct{}{}
			caseMap[foldKey(p)] = p
		}

		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return regularFiles, regularFilesSet, caseMap, nil
}

// validateEntryName validates an entry name against directory traversal,
// absolute paths, Windows volume prefixes, backslashes, NUL bytes, and non-canonical dot segments.
func validateEntryName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty entry name", ErrGEDZIPInvalidEntry)
	}
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("%w: NUL byte in entry name %q", ErrGEDZIPInvalidEntry, name)
	}
	if strings.Contains(name, `\`) {
		return fmt.Errorf("%w: backslash in entry name %q", ErrGEDZIPInvalidEntry, name)
	}
	if strings.HasPrefix(name, "/") || path.IsAbs(name) {
		return fmt.Errorf("%w: absolute path %q", ErrGEDZIPInvalidEntry, name)
	}
	if hasVolumePrefix(name) {
		return fmt.Errorf("%w: volume-prefixed path %q", ErrGEDZIPInvalidEntry, name)
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("%w: %q escapes destination", ErrGEDZIPInvalidEntry, name)
	}
	if cleaned != strings.TrimSuffix(name, "/") {
		return fmt.Errorf("%w: %q is not a canonical path", ErrGEDZIPInvalidEntry, name)
	}

	return nil
}

func hasVolumePrefix(name string) bool {
	return len(name) >= 2 && ((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) && name[1] == ':'
}

func hasDotDotSegment(name string) bool {
	return slices.Contains(strings.Split(name, "/"), "..")
}

// discoverRootGEDCOM finds the root GEDCOM file according to Requirement 1:
// Prefer root gedcom.ged; otherwise fall back to case-insensitive .ged / .gedcom
// discovery (exactly one required — multiple is an error), tolerating a single wrapper directory.
func discoverRootGEDCOM(regularFiles []string, regularFilesSet map[string]struct{}) (string, bool, error) {
	// 1. Prefer gedcom.ged at root
	if _, ok := regularFilesSet[gedzipGedcomEntry]; ok {
		return gedzipGedcomEntry, false, nil
	}

	// 2. Fall back to case-insensitive .ged / .gedcom discovery
	var candidates []string
	for _, p := range regularFiles {
		ext := strings.ToLower(path.Ext(p))
		if ext != ".ged" && ext != ".gedcom" {
			continue
		}

		// path.Dir returns "." at depth 0 and a single segment (no "/") inside a
		// single wrapper directory; anything deeper is not a candidate.
		if !strings.Contains(path.Dir(p), "/") {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return "", false, ErrGEDZIPMissingGedcom
	}
	if len(candidates) > 1 {
		return "", false, fmt.Errorf("%w: found %d candidate files (%s)", ErrGEDZIPMultipleGedcom, len(candidates), strings.Join(candidates, ", "))
	}

	return candidates[0], true, nil
}

// resolveBundleMediaFiles resolves each MediaFiles[].RelativePath against the bundle,
// relative to the .ged's directory. Exact match first, then GEDCOM 7.0 percent-decoding,
// then case-insensitive match. Unresolved refs emit a warning on ImportResult.
func resolveBundleMediaFiles(gedcomPath string, result *ImportResult, filesSet map[string]struct{}, caseMap map[string]string) {
	gedDir := path.Dir(gedcomPath)

	for i := range result.MediaFiles {
		mf := &result.MediaFiles[i]
		if mf.SourceType != MediaSourceFile {
			continue
		}

		relPath := strings.ReplaceAll(mf.RelativePath, `\`, "/")

		// Reject raw ".." segments and absolute paths before path.Clean
		if relPath == "" || strings.HasPrefix(relPath, "/") || path.IsAbs(relPath) || hasVolumePrefix(relPath) || hasDotDotSegment(relPath) {
			result.Statistics.Warnings = append(result.Statistics.Warnings, ImportWarning{
				Line:    0,
				Tag:     GedcomTagObje,
				Message: "media file reference rejected (traversal or absolute path): " + mf.RelativePath,
			})

			continue
		}

		resolved, ok := resolveMember(gedDir, relPath, filesSet, caseMap)
		if ok {
			mf.MemberPath = resolved
		} else {
			result.Statistics.Warnings = append(result.Statistics.Warnings, ImportWarning{
				Line:    0,
				Tag:     GedcomTagObje,
				Message: "unresolved media file reference: " + mf.RelativePath,
			})
		}
	}
}

func joinBundlePath(dir, file string) string {
	if dir == "." || dir == "" {
		return path.Clean(file)
	}

	return path.Clean(dir + "/" + file)
}

func isContained(gedDir, target string) bool {
	if gedDir == "." || gedDir == "" {
		return target != ".." && !strings.HasPrefix(target, "../")
	}

	return strings.HasPrefix(target, gedDir+"/") || target == gedDir
}

// resolveMember attempts to find a member file in the bundle using:
// 1. Exact match
// 2. GEDCOM 7.0 percent-decoding exact match (decoding ONLY relPath, preserving gedDir)
// 3. Case-insensitive match on target
// 4. Case-insensitive percent-decoded match
func resolveMember(gedDir, relPath string, filesSet map[string]struct{}, caseMap map[string]string) (string, bool) {
	// Candidate 1: exact target
	target := joinBundlePath(gedDir, relPath)
	if isContained(gedDir, target) {
		if _, ok := filesSet[target]; ok {
			return target, true
		}
	}

	// Candidate 2: percent-decoded relative reference (decode ONLY relPath, preserving gedDir)
	decodedRel, err := url.PathUnescape(relPath)
	var decodedTarget string
	hasDecoded := err == nil && decodedRel != relPath && !hasDotDotSegment(decodedRel) && !strings.HasPrefix(decodedRel, "/")
	if hasDecoded {
		decodedTarget = joinBundlePath(gedDir, decodedRel)
		if isContained(gedDir, decodedTarget) {
			if _, ok := filesSet[decodedTarget]; ok {
				return decodedTarget, true
			}
		}
	}

	// Candidate 3: case-insensitive match on target
	if isContained(gedDir, target) {
		if actual, ok := caseMap[foldKey(target)]; ok {
			return actual, true
		}
	}

	// Candidate 4: case-insensitive match on percent-decoded target
	if hasDecoded && isContained(gedDir, decodedTarget) {
		if actual, ok := caseMap[foldKey(decodedTarget)]; ok {
			return actual, true
		}
	}

	return "", false
}
