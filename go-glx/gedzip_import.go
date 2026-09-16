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
	"testing/fstest"
)

const (
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
			Tag:     "GEDZIP",
			Message: fmt.Sprintf("using non-standard GEDCOM entry %q", gedcomPath),
		})
	}

	// 5. Resolve media references
	resolveBundleMediaFiles(gedcomPath, result, regularFilesSet, caseMap)

	return glxFile, result, nil
}

// inventoryAndValidateBundle collects regular files in the bundle while checking
// entry names for security violations (NUL, backslash, absolute path, volume prefix,
// directory traversal) and case-folded / dot-segment duplicate collisions.
func inventoryAndValidateBundle(bundle fs.FS) ([]string, map[string]struct{}, map[string]string, error) {
	if zr, ok := bundle.(*zip.Reader); ok {
		return collectZipFiles(zr.File)
	}
	if zrc, ok := bundle.(*zip.ReadCloser); ok {
		return collectZipFiles(zrc.File)
	}
	if mfs, ok := bundle.(fstest.MapFS); ok {
		return collectMapFS(mfs)
	}

	return collectGenericFS(bundle)
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
		key := strings.ToLower(path.Clean(f.Name))
		if _, dup := seen[key]; dup {
			return nil, nil, nil, fmt.Errorf("%w: %q", ErrGEDZIPDuplicateEntry, f.Name)
		}
		seen[key] = struct{}{}

		if f.Mode().IsRegular() && !f.FileInfo().IsDir() {
			regularFiles = append(regularFiles, f.Name)
			regularFilesSet[f.Name] = struct{}{}
			caseMap[strings.ToLower(f.Name)] = f.Name
		}
	}

	return regularFiles, regularFilesSet, caseMap, nil
}

func collectMapFS(mfs fstest.MapFS) ([]string, map[string]struct{}, map[string]string, error) {
	var regularFiles []string
	regularFilesSet := make(map[string]struct{})
	caseMap := make(map[string]string)
	seen := make(map[string]struct{}, len(mfs))

	for name, file := range mfs {
		if err := validateEntryName(name); err != nil {
			return nil, nil, nil, err
		}
		key := strings.ToLower(path.Clean(name))
		if _, dup := seen[key]; dup {
			return nil, nil, nil, fmt.Errorf("%w: %q", ErrGEDZIPDuplicateEntry, name)
		}
		seen[key] = struct{}{}

		if file.Mode.IsRegular() {
			regularFiles = append(regularFiles, name)
			regularFilesSet[name] = struct{}{}
			caseMap[strings.ToLower(name)] = name
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

		key := strings.ToLower(path.Clean(p))
		if _, dup := walkSeen[key]; dup {
			return fmt.Errorf("%w: %q", ErrGEDZIPDuplicateEntry, p)
		}
		walkSeen[key] = struct{}{}

		if d.Type().IsRegular() {
			regularFiles = append(regularFiles, p)
			regularFilesSet[p] = struct{}{}
			caseMap[strings.ToLower(p)] = p
		}

		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return regularFiles, regularFilesSet, caseMap, nil
}

// validateEntryName validates an entry name against directory traversal,
// absolute paths, Windows volume prefixes, backslashes, and NUL bytes.
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
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("%w: %q escapes destination", ErrGEDZIPInvalidEntry, name)
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

		dir := path.Dir(p)
		// Depth 0: dir == "."
		// Depth 1 (single wrapper directory): !strings.Contains(dir, "/")
		if dir == "." || !strings.Contains(dir, "/") {
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

		var target string
		if gedDir == "." || gedDir == "" {
			target = path.Clean(relPath)
		} else {
			target = path.Clean(gedDir + "/" + relPath)
		}

		// Ensure target does not escape gedDir
		if gedDir != "." && gedDir != "" {
			if !strings.HasPrefix(target, gedDir+"/") && target != gedDir {
				result.Statistics.Warnings = append(result.Statistics.Warnings, ImportWarning{
					Line:    0,
					Tag:     GedcomTagObje,
					Message: "media file reference rejected (escapes bundle directory): " + mf.RelativePath,
				})

				continue
			}
		} else {
			if target == ".." || strings.HasPrefix(target, "../") {
				result.Statistics.Warnings = append(result.Statistics.Warnings, ImportWarning{
					Line:    0,
					Tag:     GedcomTagObje,
					Message: "media file reference rejected (escapes bundle root): " + mf.RelativePath,
				})

				continue
			}
		}

		resolved, ok := resolveMember(target, filesSet, caseMap)
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

// resolveMember attempts to find a member file in the bundle using:
// 1. Exact match
// 2. GEDCOM 7.0 percent-decoding exact match
// 3. Case-insensitive match
// 4. Case-insensitive percent-decoded match
func resolveMember(target string, filesSet map[string]struct{}, caseMap map[string]string) (string, bool) {
	// 1. Exact match
	if _, ok := filesSet[target]; ok {
		return target, true
	}

	// 2. GEDCOM 7.0 percent-decoding exact match
	decoded, err := url.PathUnescape(target)
	if err == nil && decoded != target && !hasDotDotSegment(decoded) && !strings.HasPrefix(decoded, "/") {
		if _, ok := filesSet[decoded]; ok {
			return decoded, true
		}
	}

	// 3. Case-insensitive match on target
	if actual, ok := caseMap[strings.ToLower(target)]; ok {
		return actual, true
	}

	// 4. Case-insensitive match on percent-decoded target
	if err == nil && decoded != target && !hasDotDotSegment(decoded) && !strings.HasPrefix(decoded, "/") {
		if actual, ok := caseMap[strings.ToLower(decoded)]; ok {
			return actual, true
		}
	}

	return "", false
}
