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
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

const mediaFamilyIndexThreshold = 32

func importMediaDir(outputPath, format string) string {
	if format == FormatSingle {
		return filepath.Dir(ensureGLXExtension(outputPath))
	}

	return outputPath
}

// allocateImportMediaDestinations reuses identical on-disk media and allocates
// the rest from each source's original unnumbered basename.
func allocateImportMediaDestinations(glx *glxlib.GLXFile, files []glxlib.MediaFileSource, archiveDir, sourceDir string, bundle fs.FS) error {
	if len(files) == 0 {
		return nil
	}
	filesDir, err := importMediaFilesDir(archiveDir)
	if err != nil {
		return err
	}
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
	families := indexMediaFamilies(names)
	familyUses := make(map[string]int)
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
		var newName string
		var reused bool
		var err error
		portable := glxlib.PortableFilename(requested)
		familyKey := glxlib.FilenameCollisionKey(portable)
		family := families[familyKey]
		familyUses[familyKey]++
		// Keep the first source's short-circuit match cheap; repeated imports
		// through a dense occupied family use the content index.
		if family != nil && len(family.seeds) >= mediaFamilyIndexThreshold && familyUses[familyKey] > 1 {
			_, firstFree := allocator.NextAvailable(portable)
			newName, reused, err = family.allocate(portable, firstFree, allocator, matcher)
		} else {
			newName, reused, err = allocator.AllocateMatching(requested, matcher.match)
		}
		if err != nil {
			return fmt.Errorf("allocating media filename %s: %w", requested, err)
		}
		claimed[newName] = true
		if media := glx.Media[mf.MediaID]; media != nil && activeURI[mf.MediaID] == glxlib.MediaFilesDir+"/"+oldName {
			mf.SetMediaURI(media, newName)
			media.URI = importMediaURI(archiveDir, filesDir, newName)
		}
		mf.TargetFilename = newName
		mf.ReuseExisting = reused
	}

	return nil
}

type mediaFamilySeed struct {
	name    string
	ordinal int
}

// mediaFamilyIndex groups the on-disk numbered names for one requested
// basename. Hashes are loaded only for the occupied prefix before the first
// free spelling, then reused across imports with different source bytes.
type mediaFamilyIndex struct {
	seeds   []mediaFamilySeed
	indexed int
	byHash  map[[32]byte][]mediaFamilySeed
	cursor  map[[32]byte]int
}

func indexMediaFamilies(names []string) map[string]*mediaFamilyIndex {
	families := make(map[string]*mediaFamilyIndex, len(names))
	add := func(key, name string, ordinal int) {
		family := families[key]
		if family == nil {
			family = &mediaFamilyIndex{}
			families[key] = family
		}
		family.seeds = append(family.seeds, mediaFamilySeed{name, ordinal})
	}
	for _, name := range names {
		add(glxlib.FilenameCollisionKey(name), name, 1)
		if baseKey, ordinal, ok := glxlib.NumberedFilenameSeed(name); ok {
			add(baseKey, name, ordinal)
		}
	}
	for _, family := range families {
		sort.Slice(family.seeds, func(i, j int) bool {
			if family.seeds[i].ordinal != family.seeds[j].ordinal {
				return family.seeds[i].ordinal < family.seeds[j].ordinal
			}

			return family.seeds[i].name < family.seeds[j].name
		})
	}

	return families
}

func (family *mediaFamilyIndex) allocate(requested string, firstFree int, allocator *glxlib.FilenameAllocator, matcher *importMediaMatcher) (string, bool, error) {
	matcher.sourceHash, matcher.haveSource = importSourceHash(matcher.source, matcher.sourceDir, matcher.bundle)
	matcher.checked = true
	if !matcher.haveSource {
		return allocator.Allocate(requested), false, nil
	}
	if family.byHash == nil {
		family.byHash = make(map[[32]byte][]mediaFamilySeed)
		family.cursor = make(map[[32]byte]int)
	}
	for family.indexed < len(family.seeds) && family.seeds[family.indexed].ordinal < firstFree {
		seed := family.seeds[family.indexed]
		if _, ok := matcher.destinationHashes[seed.name]; !ok {
			if _, err := matcher.matchesFile(seed.name); err != nil {
				return "", false, err
			}
		}
		if cached, ok := matcher.destinationHashes[seed.name]; ok {
			family.byHash[cached.hash] = append(family.byHash[cached.hash], seed)
		}
		family.indexed++
	}
	matching := family.byHash[matcher.sourceHash]
	for i := family.cursor[matcher.sourceHash]; i < len(matching) && matching[i].ordinal < firstFree; {
		ordinal := matching[i].ordinal
		candidate := indexedCandidateName(requested, ordinal)
		var chosen string
		for i < len(matching) && matching[i].ordinal == ordinal {
			name := matching[i].name
			i++
			if matcher.claimed[name] {
				continue
			}
			same, err := matcher.matchesFile(name)
			if err != nil {
				return "", false, err
			}
			if same && (chosen == "" || name == candidate) {
				chosen = name
			}
		}
		if chosen != "" && allocator.ClaimExisting(requested, chosen) {
			return chosen, true, nil
		}
		family.cursor[matcher.sourceHash] = i
	}

	return allocator.Allocate(requested), false, nil
}

func indexedCandidateName(base string, ordinal int) string {
	if ordinal == 1 {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	return fmt.Sprintf("%s-%d%s", stem, ordinal, ext)
}

// validateReusedMedia checks that a reused destination still has the source's
// bytes just before the archive is published. Allocation and publication are
// separate filesystem operations, so a sync process may change the file in
// between; publishing its URI without this check would silently misattribute it.
// Both sides are read again: ordinary files and generic fs.FS members can change
// without a reliable size/mtime signal, so allocation digests are only hints.
func validateReusedMedia(archiveDir string, files []glxlib.MediaFileSource, sourceDir string, bundle fs.FS) error {
	var hasReuse bool
	for i := range files {
		hasReuse = hasReuse || files[i].ReuseExisting
	}
	if !hasReuse {
		return nil
	}
	filesDir, err := importMediaFilesDir(archiveDir)
	if err != nil {
		return err
	}
	if err := checkMediaDirectory(filesDir); err != nil {
		return err
	}
	for i := range files {
		mf := &files[i]
		if !mf.ReuseExisting {
			continue
		}
		sourceHash, available := importSourceHash(mf, sourceDir, bundle)
		if !available {
			return fmt.Errorf("reused media source %s is no longer readable: %w", mf.TargetFilename, os.ErrInvalid)
		}
		matcher := &importMediaMatcher{
			archiveDir: archiveDir, filesDir: filesDir, sourceHash: sourceHash,
			destinationHashes: make(map[string]mediaHashCache),
		}
		same, err := matcher.matchesFile(mf.TargetFilename)
		if err != nil {
			return fmt.Errorf("rechecking reused media %s: %w", mf.TargetFilename, err)
		}
		if !same {
			return fmt.Errorf("reused media %s changed before archive publication: %w", mf.TargetFilename, os.ErrInvalid)
		}
	}

	return nil
}

// importMediaFilesDir resolves the single on-disk case variant used by an
// archive. The URI remains canonical; publication normalizes the directory
// before publishing metadata.
func importMediaFilesDir(archiveDir string) (string, error) {
	filesDir, err := archiveMediaFilesDir(archiveDir)
	if err != nil {
		return "", err
	}
	if filesDir == "" {
		filesDir = filepath.Join(archiveDir, glxlib.MediaFilesDir)
	}

	return filesDir, nil
}

// importMediaURI keeps a shared binary directory at its original spelling:
// renaming it would break references in previously imported sibling archives.
func importMediaURI(archiveDir, filesDir, name string) string {
	rel, err := filepath.Rel(archiveDir, filesDir)
	if err != nil {
		return glxlib.MediaFilesDir + "/" + name
	}

	return filepath.ToSlash(rel) + "/" + name
}

// prepareImportMediaURIs also serves writer callers that already have allocated
// sources. Only conventional binary references are rewritten.
func prepareImportMediaURIs(archive *glxlib.GLXFile, files []glxlib.MediaFileSource, archiveDir string) error {
	filesDir, err := importMediaFilesDir(archiveDir)
	if err != nil {
		return err
	}
	for i := range files {
		mf := &files[i]
		if media := archive.Media[mf.MediaID]; media != nil && mediaURIBasename(media.URI) == mf.TargetFilename {
			media.URI = importMediaURI(archiveDir, filesDir, mf.TargetFilename)
		}
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
	// A positive digest is never sufficient evidence for reuse: bytes may
	// change in place while size and mtime stay the same. Cache only rejects.
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
