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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

func importMediaDir(outputPath, format string) string {
	if format == FormatSingle {
		return filepath.Dir(ensureGLXExtension(outputPath))
	}

	return outputPath
}

// allocateImportMediaDestinations reuses identical on-disk media and allocates
// the rest from each source's original unnumbered basename.
//
// Every source takes the same path (#1442). The allocator reports the first
// free <stem>-N<ext> ordinal for the requested name. When that is the name
// itself nothing on disk can match, so nothing is read. Otherwise the source
// is hashed once and looked up in its filename family's hash index, which
// hashes each on-disk member at most once and only up to the first free
// ordinal. The lowest-ordinal identical file still unclaimed is reused, an
// exact spelling winning over a case or Unicode variant. With none, the free
// ordinal is reserved. A claimed file is never handed to a second source.
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
		var reused, keep bool
		var err error
		portable := glxlib.PortableFilename(requested)
		if name := matcher.unverifiableExisting(oldName); name != "" {
			newName, keep = name, true
		} else {
			candidate, firstFree := allocator.NextAvailable(portable)
			if family := families[glxlib.FilenameCollisionKey(portable)]; family != nil && firstFree > 1 {
				newName, err = family.match(portable, firstFree, matcher)
				reused = newName != ""
			}
			if !reused && err == nil {
				allocator.Reserve(candidate)
				newName = candidate
			}
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
		mf.KeepExisting = keep
		if reused {
			mf.ReuseDigest = matcher.sourceHash
		}
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

// match returns the lowest-ordinal on-disk member of the family below
// firstFree that is unclaimed and has the source's bytes, or "" when none
// has. Members are hashed into byHash as firstFree passes them, so each is
// read at most once per import. cursor skips the claimed prefix of each hash's
// list, which keeps a dense refresh linear.
func (family *mediaFamilyIndex) match(requested string, firstFree int, matcher *importMediaMatcher) (string, error) {
	matcher.loadSource()
	if !matcher.haveSource {
		return "", nil
	}
	if family.byHash == nil {
		family.byHash = make(map[[32]byte][]mediaFamilySeed)
		family.cursor = make(map[[32]byte]int)
	}
	for family.indexed < len(family.seeds) && family.seeds[family.indexed].ordinal < firstFree {
		seed := family.seeds[family.indexed]
		if _, ok := matcher.destinationHashes[seed.name]; !ok {
			if _, err := matcher.matchesFile(seed.name); err != nil {
				return "", err
			}
		}
		if cached, ok := matcher.destinationHashes[seed.name]; ok {
			family.byHash[cached.hash] = append(family.byHash[cached.hash], seed)
		}
		family.indexed++
	}
	matching := family.byHash[matcher.sourceHash]
	i := family.cursor[matcher.sourceHash]
	for i < len(matching) && matcher.claimed[matching[i].name] {
		i++
	}
	family.cursor[matcher.sourceHash] = i
	for i < len(matching) && matching[i].ordinal < firstFree {
		ordinal := matching[i].ordinal
		candidate := indexedCandidateName(requested, ordinal)
		var chosen string
		for i < len(matching) && matching[i].ordinal == ordinal {
			name := matching[i].name
			i++
			if matcher.claimed[name] {
				continue
			}
			// Re-check against the stat-keyed cache: a file replaced since
			// it was indexed is hashed again rather than trusted.
			same, err := matcher.matchesFile(name)
			if err != nil {
				return "", err
			}
			if same && (chosen == "" || name == candidate) {
				chosen = name
			}
		}
		if chosen != "" {
			return chosen, nil
		}
	}

	return "", nil
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
// The destination is always read again, because bytes can change without a
// reliable size/mtime signal. The source is not: its allocation digest is what
// this import means by "the source's bytes".
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
		sourceHash := mf.ReuseDigest
		if sourceHash == ([32]byte{}) {
			var hashErr error
			sourceHash, hashErr = importSourceHash(mf, sourceDir, bundle)
			if hashErr != nil {
				return fmt.Errorf("reused media source %s is no longer readable: %w", mf.TargetFilename, errors.Join(os.ErrInvalid, hashErr))
			}
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
// archive, or the canonical path when none exists. The directory is never
// renamed, and URIs follow its spelling (see importMediaURI) so they resolve on
// case-sensitive filesystems.
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
// sources. Only conventional binary references are rewritten. It returns the
// directory those URIs name, which the media commit must still resolve to.
func prepareImportMediaURIs(archive *glxlib.GLXFile, files []glxlib.MediaFileSource, archiveDir string) (string, error) {
	filesDir, err := importMediaFilesDir(archiveDir)
	if err != nil {
		return "", err
	}
	for i := range files {
		mf := &files[i]
		if media := archive.Media[mf.MediaID]; media != nil && mediaURIBasename(media.URI) == mf.TargetFilename {
			media.URI = importMediaURI(archiveDir, filesDir, mf.TargetFilename)
		}
	}

	return filesDir, nil
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
	sourceErr         error
	sourceHash        [32]byte
}

type mediaHashCache struct {
	info os.FileInfo
	hash [32]byte
}

// loadSource hashes the source at most once per allocation.
func (m *importMediaMatcher) loadSource() {
	if !m.checked {
		m.sourceHash, m.sourceErr = importSourceHash(m.source, m.sourceDir, m.bundle)
		m.haveSource = m.sourceErr == nil
		m.checked = true
	}
}

// unverifiableExisting returns the unclaimed on-disk spelling of name when the
// source cannot be read. Such a file cannot be proved identical, but allocating
// a new name would replace a working reference with one to a file this import
// cannot write, so a refresh keeps the existing reference instead.
func (m *importMediaMatcher) unverifiableExisting(name string) string {
	variants := m.byKey[glxlib.FilenameCollisionKey(name)]
	if len(variants) == 0 {
		return ""
	}
	m.loadSource()
	// An over-limit bundle entry is readable and rejected, not unavailable:
	// leave it to staging, which fails the import with the limit error.
	if m.haveSource || errors.Is(m.sourceErr, ErrGEDZIPEntryTooLarge) {
		return ""
	}
	// A previous import must not hide corrupt or unsupported bundle data.
	// Only the members staging downgrades to a warning can keep an
	// unverified existing reference; every other failure fails the import.
	if m.bundle != nil && m.source.SourceType == glxlib.MediaSourceFile &&
		!bundleMemberUnavailable(m.source.MemberPath, m.sourceErr) {
		return ""
	}
	var first string
	for _, variant := range variants {
		if m.claimed[variant] {
			continue
		}
		if variant == name {
			return variant
		}
		if first == "" {
			first = variant
		}
	}

	return first
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
	// Trusting an unchanged file's digest is safe in both directions here: bytes
	// that change in place without touching size or mtime are caught by
	// validateReusedMedia, which rereads the destination with an empty cache.
	if cached, ok := m.destinationHashes[name]; ok && os.SameFile(info, cached.info) &&
		info.Size() == cached.info.Size() && info.ModTime().Equal(cached.info.ModTime()) {
		return cached.hash == m.sourceHash, nil
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

func importSourceHash(mf *glxlib.MediaFileSource, sourceDir string, bundle fs.FS) ([32]byte, error) {
	var reader io.ReadCloser
	switch mf.SourceType {
	case glxlib.MediaSourceFile:
		switch {
		case bundle != nil:
			if mf.MemberPath == "" {
				return [32]byte{}, fmt.Errorf("media %s has no resolved bundle member: %w", mf.MediaID, os.ErrNotExist)
			}
			f, err := bundle.Open(mf.MemberPath)
			if err != nil {
				return [32]byte{}, err
			}
			info, err := f.Stat()
			if err != nil {
				return [32]byte{}, errors.Join(err, f.Close())
			}
			if !info.Mode().IsRegular() {
				return [32]byte{}, errors.Join(errGEDZIPNonRegularEntry, f.Close())
			}
			reader = f
		case sourceDir != "":
			f, err := openMediaSourceFile(sourceDir, mf.RelativePath)
			if err != nil {
				return [32]byte{}, err
			}
			reader = f
		default:
			return [32]byte{}, fmt.Errorf("media %s has no source location: %w", mf.MediaID, os.ErrNotExist)
		}
	case glxlib.MediaSourceBlob:
		data, err := glxlib.DecodeGEDCOMBlob(mf.BlobData)
		if err != nil {
			return [32]byte{}, err
		}
		reader = io.NopCloser(bytes.NewReader(data))
	default:
		return [32]byte{}, fmt.Errorf("media %s has an unknown source type: %w", mf.MediaID, os.ErrInvalid)
	}
	defer func() { _ = reader.Close() }()

	return hashReader(reader)
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
