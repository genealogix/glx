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
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
)

// gitCommandTimeout bounds each git invocation so a hung or pathological repo
// can never stall a glx command — git is only ever an optimization here.
const gitCommandTimeout = 5 * time.Second

// Binary archive cache (genealogix/glx#197).
//
// Every command that loads a multi-file archive pays the full YAML parse cost
// on each invocation. For large archives (tens of thousands of entities spread
// across hundreds of thousands of .glx files) that is tens of seconds and
// gigabytes of peak memory. The binary cache persists the fully parsed
// *glxlib.GLXFile to disk with `encoding/gob`; subsequent loads deserialize the
// blob instead of re-parsing YAML.
//
// The cache is strictly disposable and best-effort: any detected problem
// (missing, stale, corrupt, version mismatch, decode failure) silently falls
// back to the authoritative YAML parse. Staleness is detected from the git
// commit and a (path, size, mtime) fingerprint, so — like any mtime-based
// cache — a content edit that preserves a file's size and mtime is the one
// change it cannot see; a git-tracked workflow, or `glx cache build`/`clean`,
// forces freshness in that corner case. Outside it, a cached load mirrors the
// YAML parse exactly.

const (
	// cacheDirName is the per-archive directory that holds derived, disposable
	// data. It sits at the archive root and is git-ignored by `glx init`.
	cacheDirName = ".glx"
	// cacheFileName is the binary cache file inside cacheDirName.
	cacheFileName = "cache.bin"
	// cacheFormatVersion is bumped whenever the on-disk layout changes in a way
	// that makes older caches unreadable. A mismatch triggers a rebuild.
	cacheFormatVersion uint32 = 1
)

// cacheMagic is a fixed prefix written before the gob stream so a foreign or
// truncated file is rejected cheaply, without attempting a gob decode.
var cacheMagic = [8]byte{'G', 'L', 'X', 'C', 'A', 'C', 'H', 'E'}

func init() {
	// Property maps (map[string]any) carry values whose concrete types come from
	// the YAML decoder: nested maps, sequences, and !!timestamp scalars. gob
	// transmits interface values by concrete type name, so each composite type
	// must be registered. Scalars (string, int, uint64, float64, bool, nil) are
	// handled natively and need no registration.
	gob.Register(map[string]any{})
	gob.Register([]any{})
	gob.Register(time.Time{})
}

// EntityCounts is a per-type summary of an archive, stored in the cache header
// so `glx cache status` can report contents without deserializing the body.
type EntityCounts struct {
	Persons       int
	Relationships int
	Events        int
	Places        int
	Sources       int
	Citations     int
	Repositories  int
	Assertions    int
	Media         int
	ResearchLogs  int
	Studies       int
}

// Total returns the sum across all entity types.
func (c *EntityCounts) Total() int {
	return c.Persons + c.Relationships + c.Events + c.Places + c.Sources +
		c.Citations + c.Repositories + c.Assertions + c.Media + c.ResearchLogs +
		c.Studies
}

// countEntities tallies each entity type in an archive.
func countEntities(a *glxlib.GLXFile) EntityCounts {
	return EntityCounts{
		Persons:       len(a.Persons),
		Relationships: len(a.Relationships),
		Events:        len(a.Events),
		Places:        len(a.Places),
		Sources:       len(a.Sources),
		Citations:     len(a.Citations),
		Repositories:  len(a.Repositories),
		Assertions:    len(a.Assertions),
		Media:         len(a.Media),
		ResearchLogs:  len(a.ResearchLogs),
		Studies:       len(a.Studies),
	}
}

// CacheHeader is the small, quickly decodable preamble of a cache file. It is
// gob-encoded ahead of the archive body so staleness and summary information
// can be read without deserializing the (potentially huge) body.
type CacheHeader struct {
	Version       uint32       // cache format version (cacheFormatVersion)
	GLXVersion    string       // glx binary version that wrote the cache
	CreatedAt     time.Time    // when the cache was built
	GitCommitSHA  string       // HEAD at build time ("" if not a git repo)
	GitClean      bool         // whether the work tree was clean at build time
	FSFingerprint string       // hash of (path, size, mtime) over all .glx files
	Counts        EntityCounts // entity summary for `cache status`
	Duplicates    []string     // duplicate-ID warnings captured at build time
}

// cacheMode selects how transparent caching behaves, controlled by GLX_CACHE.
type cacheMode int

const (
	// cacheModeReadOnly (default) uses a fresh cache when present but never
	// writes one as a side effect of a read command.
	cacheModeReadOnly cacheMode = iota
	// cacheModeAuto additionally builds the cache on a miss.
	cacheModeAuto
	// cacheModeOff bypasses the cache entirely (no read, no write).
	cacheModeOff
)

// cacheEnvMode reads the GLX_CACHE environment variable.
//
//	off|none|disable|disabled|no -> cacheModeOff
//	auto|write|on|true|yes|1      -> cacheModeAuto
//	(anything else / unset)       -> cacheModeReadOnly
func cacheEnvMode() cacheMode {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GLX_CACHE"))) {
	case "off", "none", "disable", "disabled", "no":
		return cacheModeOff
	case "auto", "write", "on", "true", "yes", "1":
		return cacheModeAuto
	default:
		return cacheModeReadOnly
	}
}

// cacheDir returns the .glx directory inside an archive root.
func cacheDir(root string) string { return filepath.Join(root, cacheDirName) }

// cachePath returns the binary cache file path for an archive root.
func cachePath(root string) string { return filepath.Join(root, cacheDirName, cacheFileName) }

// computeFSFingerprint walks every .glx entity file under root and returns a
// SHA-256 hash of the sorted (relative-path, size, mtime) tuples. It performs
// stat calls only — no file reads — so it stays cheap even on large archives.
// The .glx (cache) and .git directories are skipped: neither holds entity
// files, and skipping them keeps the fingerprint independent of cache writes.
func computeFSFingerprint(root string) (string, error) {
	type fileMeta struct {
		rel  string
		size int64
		mod  int64
	}
	var metas []fileMeta

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (d.Name() == cacheDirName || d.Name() == ".git") {
				return filepath.SkipDir
			}

			return nil
		}
		if !isGLXFile(d.Name()) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		metas = append(metas, fileMeta{rel: filepath.ToSlash(rel), size: info.Size(), mod: info.ModTime().UnixNano()})

		return nil
	})
	if walkErr != nil {
		return "", walkErr
	}

	sort.Slice(metas, func(i, j int) bool { return metas[i].rel < metas[j].rel })

	h := sha256.New()
	var buf []byte
	for _, m := range metas {
		buf = buf[:0]
		buf = append(buf, m.rel...)
		buf = append(buf, 0)
		buf = strconv.AppendInt(buf, m.size, 10)
		buf = append(buf, 0)
		buf = strconv.AppendInt(buf, m.mod, 10)
		buf = append(buf, '\n')
		h.Write(buf)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// runGit runs a git subcommand rooted at root and returns its stdout. Every
// caller treats a non-nil error as "git unavailable" — git is only ever used to
// skip the filesystem walk, never to validate freshness on its own.
func runGit(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// #nosec G204 -- git is invoked with fixed subcommands; only the archive
	// root (a local path the user already passed to the command) is variable,
	// and it is an argument to `-C`, never shell-interpreted.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return stdout.String(), nil
}

// gitHeadSHA returns the current HEAD commit, or "" if root is not a git repo
// (or git is unavailable).
func gitHeadSHA(root string) string {
	out, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(out)
}

// gitWorkingTreeClean reports whether the work tree has no changes. The second
// return is false when git is unavailable or root is not a repo, in which case
// the boolean result is meaningless and callers must fall back.
func gitWorkingTreeClean(root string) (clean, ok bool) {
	out, err := runGit(root, "status", "--porcelain")
	if err != nil {
		return false, false
	}

	return strings.TrimSpace(out) == "", true
}

// writeCache builds the cache header and gob-encodes the header plus archive to
// {root}/.glx/cache.bin. The write is atomic. root must be a directory.
func writeCache(root string, archive *glxlib.GLXFile, duplicates []string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrCacheNotDirectory
	}

	fingerprint, err := computeFSFingerprint(root)
	if err != nil {
		return fmt.Errorf("computing fingerprint: %w", err)
	}

	sha := gitHeadSHA(root)
	clean := false
	if sha != "" {
		if c, ok := gitWorkingTreeClean(root); ok {
			clean = c
		}
	}

	header := CacheHeader{
		Version:       cacheFormatVersion,
		GLXVersion:    version,
		CreatedAt:     time.Now().UTC(),
		GitCommitSHA:  sha,
		GitClean:      clean,
		FSFingerprint: fingerprint,
		Counts:        countEntities(archive),
		Duplicates:    duplicates,
	}

	if err := os.MkdirAll(cacheDir(root), dirPermissions); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	// Stream the magic prefix and gob payload straight to the temp file rather
	// than buffering the whole serialized archive in RAM first — on a large
	// archive the gob blob is hundreds of MB, and that buffer would land on top
	// of the already-resident parsed archive at peak.
	if err := atomicWriteStream(cachePath(root), filePermissions, func(w io.Writer) error {
		if _, err := w.Write(cacheMagic[:]); err != nil {
			return err
		}
		enc := gob.NewEncoder(w)
		if err := enc.Encode(&header); err != nil {
			return fmt.Errorf("encoding cache header: %w", err)
		}
		if err := enc.Encode(archive); err != nil {
			return fmt.Errorf("encoding cache body: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	return nil
}

// openCache opens the cache file, validates the magic prefix and format
// version, and decodes the header. It returns the still-open file and a decoder
// positioned to read the archive body. The caller owns the file and must close
// it.
func openCache(root string) (*os.File, *gob.Decoder, *CacheHeader, error) {
	f, err := os.Open(cachePath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil, ErrCacheNotFound
		}

		return nil, nil, nil, err
	}

	var magic [8]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil || magic != cacheMagic {
		_ = f.Close()

		return nil, nil, nil, ErrNotCacheFile
	}

	dec := gob.NewDecoder(f)
	var header CacheHeader
	if err := dec.Decode(&header); err != nil {
		_ = f.Close()

		return nil, nil, nil, fmt.Errorf("%w: %w", ErrNotCacheFile, err)
	}
	if header.Version != cacheFormatVersion {
		_ = f.Close()

		return nil, nil, nil, ErrCacheVersion
	}

	return f, dec, &header, nil
}

// readCacheHeader decodes only the header (cheap — no body deserialization).
func readCacheHeader(root string) (*CacheHeader, error) {
	f, _, header, err := openCache(root)
	if err != nil {
		return nil, err
	}
	_ = f.Close()

	return header, nil
}

// loadCache decodes the full archive from the cache file.
func loadCache(root string) (*glxlib.GLXFile, *CacheHeader, error) {
	f, dec, header, err := openCache(root)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close() //nolint:errcheck // read-only handle

	var archive glxlib.GLXFile
	if err := dec.Decode(&archive); err != nil {
		return nil, nil, fmt.Errorf("decoding cache body: %w", err)
	}

	return &archive, header, nil
}

// cacheIsFresh reports whether a cache with the given header still matches the
// on-disk archive. The filesystem fingerprint is authoritative; the git check
// is only an optimization that can skip the fingerprint walk when it can prove
// freshness (built clean at a commit we are still on, with a still-clean tree).
func cacheIsFresh(root string, header *CacheHeader) bool {
	// A cache written by a different glx version may not gob-decode into the
	// current struct layout; treat it as stale up front.
	if header.GLXVersion != version {
		return false
	}

	if header.GitClean && header.GitCommitSHA != "" && gitHeadSHA(root) == header.GitCommitSHA {
		if clean, ok := gitWorkingTreeClean(root); ok && clean {
			return true
		}
	}

	fingerprint, err := computeFSFingerprint(root)
	if err != nil {
		return false
	}

	return fingerprint == header.FSFingerprint
}

// tryLoadFreshCache returns the cached archive and its duplicate warnings when a
// valid, fresh cache exists for root. The final bool reports a usable hit; on
// any miss (no cache, stale, corrupt, wrong version) it returns false so the
// caller can fall back to the YAML parse.
func tryLoadFreshCache(root string) (*glxlib.GLXFile, []string, bool) {
	header, err := readCacheHeader(root)
	if err != nil {
		return nil, nil, false
	}
	if !cacheIsFresh(root, header) {
		return nil, nil, false
	}

	archive, _, err := loadCache(root)
	if err != nil {
		// Header looked fine but the body is unreadable (e.g. a struct change in
		// a dev build). Ignore and fall back to the authoritative YAML parse.
		return nil, nil, false
	}

	return archive, header.Duplicates, true
}

// LoadArchiveCached loads a multi-file archive, transparently using a fresh
// binary cache when one is present. It is a drop-in replacement for
// LoadArchiveWithOptions(path, false) for read-only commands and returns the
// same (archive, duplicate-warnings, error) tuple.
//
// Behavior is governed by GLX_CACHE (see cacheEnvMode):
//   - default: read a fresh cache if present; otherwise parse YAML.
//   - auto:    additionally build the cache after a miss.
//   - off:     ignore the cache entirely.
//
// Caching only applies to directory archives; single-file paths and any error
// reading the cache fall through to LoadArchiveWithOptions, so behavior is
// always identical to the uncached path apart from speed.
func LoadArchiveCached(path string) (*glxlib.GLXFile, []string, error) {
	mode := cacheEnvMode()

	info, statErr := os.Stat(path)
	cacheable := mode != cacheModeOff && statErr == nil && info.IsDir()

	if cacheable {
		if archive, duplicates, ok := tryLoadFreshCache(path); ok {
			return archive, duplicates, nil
		}
	}

	archive, duplicates, err := LoadArchiveWithOptions(path, false)
	if err != nil {
		return nil, nil, err
	}

	if cacheable && mode == cacheModeAuto {
		if werr := writeCache(path, archive, duplicates); werr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write archive cache: %v\n", werr)
		}
	}

	return archive, duplicates, nil
}
