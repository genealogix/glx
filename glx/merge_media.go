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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// plannedMediaCopy describes one binary file to copy from the source archive
// into the destination archive once the merge has been written. Several Media
// entities may reference the same binary, so the plan is keyed by source
// file: MediaIDs lists every entity whose URI points at this binary.
type plannedMediaCopy struct {
	SrcPath    string   // absolute path of the file under srcPath/media/files/
	TargetName string   // basename to write into destPath/media/files/
	MediaIDs   []string // Media entities in src whose URI references this file
}

// planSourceMediaCopies inspects srcPath/media/files/ and builds a plan for
// carrying the binaries referenced by src.Media entities into destPath, with
// name dedup against the destination's existing media/files/. Source files
// whose contents match an identically-named file already in the destination
// are skipped (no copy, no rename). Source files whose name collides with a
// different-content destination file are scheduled with a "<stem>-N<ext>"
// rename, and the corresponding src.Media URIs are rewritten in memory so the
// merged archive will reference the new name.
//
// srcPath and destPath must be absolute. A single-file archive (or any
// archive with no media/files/ directory) produces an empty plan.
func planSourceMediaCopies(srcPath, destPath string, src *glxlib.GLXFile) ([]plannedMediaCopy, error) {
	srcFiles := filepath.Join(srcPath, glxlib.MediaFilesDir)
	info, err := os.Stat(srcFiles)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("inspecting source media/files: %w", err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	mediaByBasename := indexMediaByBasename(src.Media)

	taken, err := listMediaFilenames(filepath.Join(destPath, glxlib.MediaFilesDir))
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(srcFiles)
	if err != nil {
		return nil, fmt.Errorf("reading source media/files: %w", err)
	}

	var plan []plannedMediaCopy
	for _, entry := range entries {
		if entry.IsDir() {
			// Nested subdirectories under media/files/ aren't part of the GLX
			// convention; skip them rather than guess at their semantics.
			continue
		}
		name := entry.Name()
		srcFile := filepath.Join(srcFiles, name)

		mediaIDs := mediaByBasename[name]
		if len(mediaIDs) == 0 {
			// Binary isn't referenced by any src Media entity. Don't copy
			// orphan binaries — only carry what the merged archive needs.
			continue
		}

		target := name
		if _, exists := taken[name]; exists {
			same, err := filesIdentical(srcFile, filepath.Join(destPath, glxlib.MediaFilesDir, name))
			if err != nil {
				return nil, fmt.Errorf("comparing media file %s: %w", name, err)
			}
			if same {
				// Identical content — reuse the existing binary, no rename.
				continue
			}
			target = nextAvailableMediaName(name, taken)
			for _, id := range mediaIDs {
				if m := src.Media[id]; m != nil {
					m.URI = glxlib.MediaFilesDir + "/" + target
				}
			}
		}
		taken[target] = struct{}{}
		plan = append(plan, plannedMediaCopy{
			SrcPath:    srcFile,
			TargetName: target,
			MediaIDs:   mediaIDs,
		})
	}

	return plan, nil
}

// executeMediaCopies copies each planned source binary into the merged
// archive's media/files/ directory. A planned binary is skipped (and counted)
// when none of its referencing Media entities survived the merge with a URI
// that still points at the planned target — copying it would leave the binary
// orphaned on disk.
func executeMediaCopies(plan []plannedMediaCopy, destPath string, merged *glxlib.GLXFile) (copied, skipped int, err error) {
	if len(plan) == 0 {
		return 0, 0, nil
	}
	dstDir := filepath.Join(destPath, glxlib.MediaFilesDir)
	if err := os.MkdirAll(dstDir, dirPermissions); err != nil {
		return 0, 0, fmt.Errorf("creating media/files dir: %w", err)
	}
	expectedURI := func(target string) string { return glxlib.MediaFilesDir + "/" + target }
	for _, p := range plan {
		// Copy the binary if at least one referencing entity survived with a
		// URI that still points at this target. If every reference was
		// dropped or rewritten elsewhere by a merge conflict, skip.
		var referenced bool
		for _, id := range p.MediaIDs {
			if m, ok := merged.Media[id]; ok && m != nil && m.URI == expectedURI(p.TargetName) {
				referenced = true

				break
			}
		}
		if !referenced {
			skipped++

			continue
		}
		dst := filepath.Join(dstDir, p.TargetName)
		if err := copyFile(p.SrcPath, dst); err != nil {
			return copied, skipped, fmt.Errorf("copying %s: %w", p.SrcPath, err)
		}
		copied++
	}

	return copied, skipped, nil
}

// indexMediaByBasename maps each Media entity's URI basename to the list of
// entity IDs that reference it. Entities whose URI isn't a flat
// media/files/<name> reference (URLs, absolute paths, nested paths, empty
// URIs) are excluded.
func indexMediaByBasename(media map[string]*glxlib.Media) map[string][]string {
	out := make(map[string][]string, len(media))
	for id, m := range media {
		if m == nil {
			continue
		}
		base := mediaURIBasename(m.URI)
		if base == "" {
			continue
		}
		out[base] = append(out[base], id)
	}

	return out
}

// mediaURIBasename returns the filename portion of a URI of the form
// "media/files/<name>", or "" for URIs that don't point at a local flat-path
// binary (URLs, absolute paths, nested subpaths).
func mediaURIBasename(uri string) string {
	prefix := glxlib.MediaFilesDir + "/"
	if !strings.HasPrefix(uri, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(uri, prefix)
	if rest == "" || strings.ContainsRune(rest, '/') {
		return ""
	}

	return rest
}

// listMediaFilenames returns the set of regular-file names in dir. A missing
// directory is treated as empty.
func listMediaFilenames(dir string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]struct{}{}, nil
		}

		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	out := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out[e.Name()] = struct{}{}
	}

	return out, nil
}

// nextAvailableMediaName returns the first "<stem>-N<ext>" form not already
// present in taken, starting at -2 to match the GEDCOM importer's convention
// (see go-glx/gedcom_media.go::deduplicateFilename).
func nextAvailableMediaName(basename string, taken map[string]struct{}) string {
	ext := filepath.Ext(basename)
	stem := strings.TrimSuffix(basename, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if _, exists := taken[candidate]; !exists {
			return candidate
		}
	}
}

// filesIdentical reports whether two files have identical content. Size is
// used as a fast reject and SHA-256 for the byte-level comparison.
func filesIdentical(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", a, err)
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", b, err)
	}
	if infoA.Size() != infoB.Size() {
		return false, nil
	}
	hashA, err := sha256File(a)
	if err != nil {
		return false, err
	}
	hashB, err := sha256File(b)
	if err != nil {
		return false, err
	}

	return hashA == hashB, nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
