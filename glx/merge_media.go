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
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// plannedMediaCopy describes one binary file to copy from the source archive
// into the destination archive before the merge is published. Several Media
// entities may reference the same binary, so the plan is keyed by source
// file: SrcMedia captures the src.Media entries by ID at planning time so
// execution can use pointer identity — rather than a URI coincidence — to
// detect entities that survived the merge.
type plannedMediaCopy struct {
	SrcPath    string                   // absolute path of the source binary
	SrcRoot    string                   // source archive root for confined reads
	TargetDir  string                   // actual destination media/files spelling, relative to destPath
	TargetName string                   // basename to write under TargetDir
	SrcMedia   map[string]*glxlib.Media // src.Media entries by ID at planning time
}

// planSourceMediaCopies inspects the source's case-equivalent media/files/
// directory and builds a plan for carrying referenced binaries into destPath, with
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
	srcFiles, err := mergeMediaFilesDir(srcPath)
	if err != nil {
		return nil, fmt.Errorf("inspecting source media/files: %w", err)
	}
	if srcFiles == "" {
		// Keep rejecting a canonical directory reached through a symlink,
		// while treating a missing or non-directory path as no media to copy.
		canonical := filepath.Join(srcPath, glxlib.MediaFilesDir)
		info, statErr := os.Stat(canonical)
		if statErr != nil && !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("inspecting source media/files: %w", statErr)
		}
		if statErr == nil && info.IsDir() {
			if _, err := listMediaFilenames(canonical); err != nil {
				return nil, fmt.Errorf("inspecting source media/files: %w", err)
			}
		}

		return nil, nil
	}
	if _, err := listMediaFilenames(srcFiles); err != nil {
		return nil, fmt.Errorf("inspecting source media/files: %w", err)
	}

	mediaByBasename := indexMediaByBasename(src.Media)

	destFiles, err := mergeMediaFilesDir(destPath)
	if err != nil {
		return nil, fmt.Errorf("inspecting destination media/files: %w", err)
	}
	if destFiles == "" {
		destFiles = filepath.Join(destPath, glxlib.MediaFilesDir)
	}
	targetDir, err := filepath.Rel(destPath, destFiles)
	if err != nil {
		return nil, fmt.Errorf("resolving destination media/files: %w", err)
	}
	taken, err := listMediaFilenames(destFiles)
	if err != nil {
		return nil, err
	}
	existingNames := make([]string, 0, len(taken))
	for name := range taken {
		existingNames = append(existingNames, name)
	}
	sort.Strings(existingNames)
	allocator := glxlib.NewFilenameAllocator(existingNames...)

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
		existingName, exists := allocator.Existing(name)
		if _, exact := taken[name]; exact {
			existingName, exists = name, true
		}
		if exists {
			if _, onDisk := taken[existingName]; onDisk {
				existingPath := filepath.Join(destFiles, existingName)
				info, statErr := os.Lstat(existingPath)
				if statErr != nil {
					return nil, fmt.Errorf("inspecting media file %s: %w", existingName, statErr)
				}
				if info.Mode().IsRegular() {
					same, compareErr := filesIdenticalWithin(srcPath, srcFile, destPath, existingPath)
					if compareErr != nil {
						return nil, fmt.Errorf("comparing media file %s: %w", name, compareErr)
					}
					if same {
						// Reuse the actual on-disk spelling on case-sensitive filesystems too.
						for _, id := range mediaIDs {
							if m := src.Media[id]; m != nil {
								m.URI = glxlib.MediaFilesDir + "/" + existingName
							}
						}

						continue
					}
				}
			}
			target = allocator.Allocate(name)
			for _, id := range mediaIDs {
				if m := src.Media[id]; m != nil {
					m.URI = glxlib.MediaFilesDir + "/" + target
				}
			}
		} else {
			allocator.Reserve(name)
		}
		srcMedia := make(map[string]*glxlib.Media, len(mediaIDs))
		for _, id := range mediaIDs {
			srcMedia[id] = src.Media[id]
		}
		plan = append(plan, plannedMediaCopy{
			SrcPath:    srcFile,
			SrcRoot:    srcPath,
			TargetDir:  filepath.ToSlash(targetDir),
			TargetName: target,
			SrcMedia:   srcMedia,
		})
	}

	return plan, nil
}

// mergeMediaFilesDir finds the one real media/files directory, preserving its
// on-disk spelling. A case-ambiguous archive cannot be safely merged.
func mergeMediaFilesDir(archivePath string) (string, error) {
	dirs, err := mediaFilesDirsIn(archivePath)
	if err != nil {
		return "", err
	}
	if len(dirs) > 1 {
		return "", fmt.Errorf("%w: %s and %s", ErrAmbiguousMediaFilesDirs, dirs[0], dirs[1])
	}
	if len(dirs) == 0 {
		return "", nil
	}

	return dirs[0], nil
}

// executeMediaCopies copies each planned source binary into the merged
// archive's media/files/ directory. A planned binary is skipped (and counted)
// when none of its referencing src Media entities survived the merge —
// detected by pointer identity against merged.Media. URI coincidence with a
// destination entity that was kept by a conflict does NOT cause a copy:
// importing src's binary content under dest's preserved metadata would
// misattribute the binary.
func executeMediaCopies(plan []plannedMediaCopy, destPath string, merged *glxlib.GLXFile) (copied, skipped int, err error) {
	copied, skipped, _, err = executeMediaCopiesWithRollback(plan, destPath, merged)

	return copied, skipped, err
}

// executeMediaCopiesWithRollback leaves metadata publication to its caller and
// returns a rollback for only the new binaries it successfully created.
func executeMediaCopiesWithRollback(plan []plannedMediaCopy, destPath string, merged *glxlib.GLXFile) (copied, skipped int, rollback func(), err error) {
	if len(plan) == 0 {
		return 0, 0, func() {}, nil
	}
	root, err := os.OpenRoot(destPath)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("opening destination archive: %w", err)
	}
	defer func() { _ = root.Close() }()
	targetDir := plannedMediaTargetDir(plan[0])
	if err := root.MkdirAll(targetDir, dirPermissions); err != nil {
		return 0, 0, nil, fmt.Errorf("creating media/files dir: %w", err)
	}
	committed := make([]committedMediaFile, 0, len(plan))
	defer func() {
		if err != nil {
			rollbackCommittedMedia(root, committed)
		}
	}()
	for _, p := range plan {
		// Copy the binary if at least one src Media entity for this plan
		// entry made it into the merged archive (same pointer, since Merge()
		// preserves the src entity by reference when there's no conflict).
		var referenced bool
		for id, srcM := range p.SrcMedia {
			if mergedM, ok := merged.Media[id]; ok && mergedM != nil && mergedM == srcM {
				referenced = true

				break
			}
		}
		if !referenced {
			skipped++

			continue
		}
		file, err := copyPlannedMedia(root, p)
		if err != nil {
			return copied, skipped, nil, err
		}
		committed = append(committed, file)
		copied++
	}

	return copied, skipped, func() {
		root, openErr := os.OpenRoot(destPath)
		if openErr != nil {
			return
		}
		defer func() { _ = root.Close() }()
		rollbackCommittedMedia(root, committed)
	}, nil
}

func copyPlannedMedia(root *os.Root, p plannedMediaCopy) (committedMediaFile, error) {
	dst := path.Join(plannedMediaTargetDir(p), p.TargetName)
	srcRoot := p.SrcRoot
	if srcRoot == "" {
		srcRoot = filepath.Dir(p.SrcPath)
	}
	src, err := openWithin(srcRoot, p.SrcPath)
	if err != nil {
		return committedMediaFile{}, fmt.Errorf("opening source media %s: %w", p.SrcPath, err)
	}
	defer func() { _ = src.Close() }()
	out, err := root.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePermissions)
	if err != nil {
		return committedMediaFile{}, fmt.Errorf("creating destination media %s: %w", dst, err)
	}
	createdInfo, statErr := out.Stat()
	if statErr != nil {
		_ = out.Close()

		return committedMediaFile{}, fmt.Errorf("inspecting destination media %s: %w", dst, statErr)
	}
	_, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		current, err := root.Lstat(dst)
		if err == nil && os.SameFile(current, createdInfo) {
			_ = root.Remove(dst)
		}

		return committedMediaFile{}, fmt.Errorf("copying %s: %w", p.SrcPath, errors.Join(copyErr, closeErr))
	}

	return committedMediaFile{dst, createdInfo}, nil
}

func plannedMediaTargetDir(p plannedMediaCopy) string {
	if p.TargetDir != "" {
		return p.TargetDir
	}

	return glxlib.MediaFilesDir
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

// listMediaFilenames returns the set of names in dir. Directories and symlinks
// also occupy a destination name and must never be replaced. A missing
// directory is treated as empty; an occupied non-directory path is an error.
// Archive safe-write can preserve a media/files symlink by moving the link
// itself, but merge copies through this path and must not write to its target.
func listMediaFilenames(dir string) (map[string]struct{}, error) {
	base := filepath.Dir(dir)
	rel := filepath.Base(dir)
	parts := []string{rel}
	// For archive media, anchor at the archive root so a swapped media
	// directory cannot redirect even the listing outside the archive. Keep
	// the actual spelling for case-sensitive MEDIA/files variants.
	if strings.EqualFold(filepath.Base(base), "media") && strings.EqualFold(rel, "files") {
		mediaName := filepath.Base(base)
		base = filepath.Dir(base)
		rel = path.Join(mediaName, rel)
		parts = []string{mediaName, rel}
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]struct{}{}, nil
		}

		return nil, fmt.Errorf("opening media parent %s: %w", base, err)
	}
	defer func() { _ = root.Close() }()
	var checked os.FileInfo
	for _, current := range parts {
		info, err := root.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return map[string]struct{}{}, nil
			}

			return nil, fmt.Errorf("inspecting %s: %w", filepath.Join(base, current), err)
		}
		if !info.IsDir() {
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("media directory %s is a symlink; imports and merges cannot write through media-directory symlinks: %w", filepath.Join(base, current), os.ErrInvalid)
			}

			return nil, fmt.Errorf("media path %s is not a directory: %w", filepath.Join(base, current), os.ErrInvalid)
		}
		checked = info
	}
	opened, err := root.Open(rel)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	openedInfo, statErr := opened.Stat()
	if statErr != nil || !os.SameFile(checked, openedInfo) {
		_ = opened.Close()

		return nil, fmt.Errorf("media directory %s changed while listing: %w", dir, os.ErrInvalid)
	}
	names, readErr := opened.Readdirnames(-1)
	closeErr := opened.Close()
	if readErr != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing %s: %w", dir, closeErr)
	}
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[name] = struct{}{}
	}

	return out, nil
}

// filesIdenticalWithin compares two archive media files through confined roots.
func filesIdenticalWithin(rootA, a, rootB, b string) (bool, error) {
	fa, err := openWithin(rootA, a)
	if err != nil {
		return false, err
	}
	defer func() { _ = fa.Close() }()
	fb, err := openWithin(rootB, b)
	if err != nil {
		return false, err
	}
	defer func() { _ = fb.Close() }()
	sa, err := fa.Stat()
	if err != nil {
		return false, err
	}
	sb, err := fb.Stat()
	if err != nil {
		return false, err
	}
	if !sa.Mode().IsRegular() || !sb.Mode().IsRegular() || sa.Size() != sb.Size() {
		return false, nil
	}
	ha, err := hashReader(fa)
	if err != nil {
		return false, err
	}
	hb, err := hashReader(fb)
	if err != nil {
		return false, err
	}

	return ha == hb, nil
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
