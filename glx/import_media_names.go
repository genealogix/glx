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
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	glxlib "github.com/genealogix/glx/go-glx"
)

func importMediaDir(outputPath, format string) string {
	if format == FormatSingle {
		return filepath.Dir(ensureGLXExtension(outputPath))
	}

	return outputPath
}

// reserveImportMediaDestinations is retained for callers without source bytes.
func reserveImportMediaDestinations(glx *glxlib.GLXFile, files []glxlib.MediaFileSource, archiveDir string) error {
	return allocateImportMediaDestinations(glx, files, archiveDir, "", nil)
}

// allocateImportMediaDestinations reuses identical on-disk media and allocates
// the rest from each source's original unnumbered basename.
func allocateImportMediaDestinations(glx *glxlib.GLXFile, files []glxlib.MediaFileSource, archiveDir, sourceDir string, bundle fs.FS) error {
	if len(files) == 0 {
		return nil
	}
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	existing, err := listMediaFilenames(filesDir)
	if err != nil {
		return err
	}
	byKey := make(map[string][]string, len(existing))
	names := make([]string, 0, len(existing))
	for name := range existing {
		key := glxlib.FilenameCollisionKey(name)
		byKey[key] = append(byKey[key], name)
		names = append(names, name)
	}
	for key := range byKey {
		sort.Strings(byKey[key])
	}
	sort.Strings(names)
	allocator := glxlib.NewFilenameAllocator(names...)
	claimed := make(map[string]bool, len(files))
	destinationHashes := make(map[string]mediaHashCache, len(existing))
	activeURI := make(map[string]string, len(glx.Media))
	for id, media := range glx.Media {
		if media != nil {
			activeURI[id] = media.URI
		}
	}
	for i := range files {
		mf := &files[i]
		oldName := mf.TargetFilename
		requested := mf.RequestedFilename
		if requested == "" {
			requested = oldName
		}
		matcher := &importMediaMatcher{
			source: mf, sourceDir: sourceDir, bundle: bundle,
			archiveDir: archiveDir, filesDir: filesDir,
			byKey: byKey, claimed: claimed, destinationHashes: destinationHashes,
		}
		newName, reused, err := allocator.AllocateMatching(requested, matcher.match)
		if err != nil {
			return fmt.Errorf("allocating media filename %s: %w", requested, err)
		}
		claimed[newName] = true
		if media := glx.Media[mf.MediaID]; media != nil && activeURI[mf.MediaID] == glxlib.MediaFilesDir+"/"+oldName {
			mf.SetMediaURI(media, newName)
		}
		mf.TargetFilename = newName
		mf.ReuseExisting = reused
	}

	return nil
}

type importMediaMatcher struct {
	source            *glxlib.MediaFileSource
	sourceDir         string
	bundle            fs.FS
	archiveDir        string
	filesDir          string
	byKey             map[string][]string
	claimed           map[string]bool
	destinationHashes map[string]mediaHashCache
	checked           bool
	haveSource        bool
	sourceHash        [32]byte
}

type mediaHashCache struct {
	info os.FileInfo
	hash [32]byte
}

func (m *importMediaMatcher) match(candidate string) (string, error) {
	variants := m.byKey[glxlib.FilenameCollisionKey(candidate)]
	if len(variants) == 0 {
		return "", nil
	}
	if !m.checked {
		m.sourceHash, m.haveSource = importSourceHash(m.source, m.sourceDir, m.bundle)
		m.checked = true
	}
	if !m.haveSource {
		return "", nil
	}
	for pass := range 2 {
		for _, name := range variants {
			if m.claimed[name] || (pass == 0) != (name == candidate) {
				continue
			}
			same, err := m.matchesFile(name)
			if err != nil {
				return "", err
			}
			if same {
				return name, nil
			}
		}
	}

	return "", nil
}

func (m *importMediaMatcher) matchesFile(name string) (bool, error) {
	fullPath := filepath.Join(m.filesDir, name)
	info, err := os.Lstat(fullPath)
	if err != nil {
		return false, fmt.Errorf("inspecting media file %s: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}
	if cached, ok := m.destinationHashes[name]; ok && os.SameFile(info, cached.info) &&
		info.Size() == cached.info.Size() && info.ModTime().Equal(cached.info.ModTime()) &&
		cached.hash != m.sourceHash {
		return false, nil
	}
	f, err := openWithin(m.archiveDir, fullPath)
	if err != nil {
		return false, fmt.Errorf("opening media file %s: %w", name, err)
	}
	openedInfo, statErr := f.Stat()
	if statErr != nil {
		_ = f.Close()

		return false, fmt.Errorf("inspecting opened media file %s: %w", name, statErr)
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		_ = f.Close()

		return false, fmt.Errorf("media file %s changed while opening: %w", name, os.ErrInvalid)
	}
	hash, hashErr := hashReader(f)
	closeErr := f.Close()
	if hashErr != nil {
		return false, hashErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	m.destinationHashes[name] = mediaHashCache{info: openedInfo, hash: hash}

	return hash == m.sourceHash, nil
}

func importSourceHash(mf *glxlib.MediaFileSource, sourceDir string, bundle fs.FS) ([32]byte, bool) {
	var reader io.ReadCloser
	switch mf.SourceType {
	case glxlib.MediaSourceFile:
		switch {
		case bundle != nil:
			if mf.MemberPath == "" {
				return [32]byte{}, false
			}
			f, err := bundle.Open(mf.MemberPath)
			if err != nil {
				return [32]byte{}, false
			}
			info, err := f.Stat()
			if err != nil || !info.Mode().IsRegular() {
				_ = f.Close()

				return [32]byte{}, false
			}
			reader = f
		case sourceDir != "":
			f, err := openMediaSourceFile(sourceDir, mf.RelativePath)
			if err != nil {
				return [32]byte{}, false
			}
			reader = f
		default:
			return [32]byte{}, false
		}
	case glxlib.MediaSourceBlob:
		data, err := glxlib.DecodeGEDCOMBlob(mf.BlobData)
		if err != nil {
			return [32]byte{}, false
		}
		reader = io.NopCloser(bytes.NewReader(data))
	default:
		return [32]byte{}, false
	}
	defer func() { _ = reader.Close() }()
	hash, err := hashReader(reader)

	return hash, err == nil
}

func hashReader(r io.Reader) ([32]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return [32]byte{}, err
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))

	return digest, nil
}
