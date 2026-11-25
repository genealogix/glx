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
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"sync/atomic"
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
)

const mcpWatchInterval = time.Second

var (
	errMCPWatchSourceType  = errors.New("archive path must remain a regular file or directory of its original type")
	errMCPWatchFileChanged = errors.New("archive file changed while opening")
	errMCPWatchJournal     = errors.New("archive write journal is present")
)

// mcpArchiveSource always reads source bytes, not a metadata-based binary cache.
// Hashing contents catches atomic replacements and same-size/same-mtime edits.
// Directory membership and symlink containment match the normal archive loader.
type mcpArchiveSource struct {
	path      string
	directory bool
}

type mcpSourceCapture struct {
	files       map[string][]byte
	fingerprint [sha256.Size]byte
}

func (s *mcpArchiveSource) capture(ctx context.Context) (*mcpSourceCapture, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Stat(s.path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() != s.directory || (!s.directory && !info.Mode().IsRegular()) {
		return nil, errMCPWatchSourceType
	}
	var files map[string][]byte
	if s.directory {
		files, err = s.captureDirectory(ctx)
	} else {
		var data []byte
		data, err = readMCPWatchSingleFile(s.path)
		files = map[string][]byte{".": data}
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &mcpSourceCapture{files: files, fingerprint: mcpSourceFingerprint(files)}, nil
}

func (s *mcpArchiveSource) captureDirectory(ctx context.Context) (map[string][]byte, error) {
	if err := checkMCPWatchJournal(s.path); err != nil {
		return nil, err
	}
	files := make(map[string][]byte)
	err := walkGLXFilesUnderReading(s.path, ".", readMCPWatchFile, func(path string, data []byte, readErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", path, readErr)
		}
		files[path] = data

		return nil
	})
	if err != nil {
		return nil, err
	}
	// A GLX in-place write can begin after the first journal check.
	if err := checkMCPWatchJournal(s.path); err != nil {
		return nil, err
	}

	return files, nil
}

// readMCPWatchFile opens within the already-contained archive root. Checking
// the opened descriptor too closes the stat/open race; nonblocking mode prevents
// a FIFO swapped in during that race from hanging the watcher and its shutdown.
func readMCPWatchFile(root *os.Root, path string) ([]byte, error) {
	info, err := root.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errMCPWatchSourceType
	}
	file, err := root.OpenFile(path, os.O_RDONLY|nonblockOpenFlag, 0)
	if err != nil {
		return nil, err
	}

	return readMCPWatchRegularFile(file, nil)
}

func readMCPWatchSingleFile(path string) ([]byte, error) {
	// The leaf open itself must not follow a replacement symlink. Do not
	// open the parent for reading: a known file can legitimately live in a
	// directory that grants search permission without directory read access.
	file, expected, err := openMCPWatchSingleFile(path)
	if err != nil {
		return nil, err
	}

	return readMCPWatchRegularFile(file, expected)
}

func readMCPWatchRegularFile(file *os.File, expected os.FileInfo) ([]byte, error) {
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errMCPWatchSourceType
	}
	// Platforms without a native no-follow open supply the entry's identity
	// from before the open. Reject any substitution before reading bytes.
	if expected != nil && !os.SameFile(expected, info) {
		return nil, errMCPWatchFileChanged
	}

	return io.ReadAll(file)
}

// checkMCPWatchJournal reuses the writers' journal check, so the server and
// in-place writers agree on what marks a write as in progress or interrupted.
func checkMCPWatchJournal(path string) error {
	if err := checkInPlaceJournal(path); err != nil {
		if errors.Is(err, ErrInterruptedInPlaceWrite) {
			return fmt.Errorf("%w: %w", errMCPWatchJournal, err)
		}

		return err
	}

	return nil
}

func mcpSourceFingerprint(files map[string][]byte) [sha256.Size]byte {
	h := sha256.New()
	var size [8]byte
	for _, path := range slices.Sorted(maps.Keys(files)) {
		// Length prefixes separate arbitrary paths and contents unambiguously.
		binary.BigEndian.PutUint64(size[:], uint64(len(path)))
		h.Write(size[:])
		h.Write([]byte(path))
		binary.BigEndian.PutUint64(size[:], uint64(len(files[path])))
		h.Write(size[:])
		h.Write(files[path])
	}

	return [sha256.Size]byte(h.Sum(nil))
}

func (s *mcpArchiveSource) load(capture *mcpSourceCapture) (*mcpArchiveBackend, error) {
	var archive *glxlib.GLXFile
	var duplicates []string
	var err error
	if s.directory {
		archive, duplicates, err = loadArchiveFromFiles(s.path, capture.files, false)
	} else {
		archive, err = createSerializer(false, "").DeserializeSingleFileBytes(capture.files["."])
	}
	if err != nil {
		return nil, err
	}
	if err := mergeStandardVocabularies(archive); err != nil {
		return nil, err
	}

	return &mcpArchiveBackend{archive: archive, duplicates: duplicates}, nil
}

// Only the polling goroutine touches debounce state. The protocol adapter
// serializes tool calls, while the atomic pointer permits nonblocking replacement
// without modifying a snapshot (or its lazy caches) underneath an active call.
type mcpWatchingBackend struct {
	source      mcpArchiveSource
	current     atomic.Pointer[mcpArchiveBackend]
	fingerprint [sha256.Size]byte
	pending     [sha256.Size]byte
	hasPending  bool
	// failed remembers the last stable capture that did not load, so an
	// archive left broken is not reparsed on every debounce cycle.
	failed    [sha256.Size]byte
	failedErr error
}

func newMCPWatchingBackend(ctx context.Context, path string) (*mcpWatchingBackend, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access archive: %w", err)
	}
	b := &mcpWatchingBackend{source: mcpArchiveSource{path: path, directory: info.IsDir()}}
	capture, err := b.source.capture(ctx)
	if err != nil {
		return nil, err
	}
	backend, err := b.source.load(capture)
	if err != nil {
		return nil, err
	}
	b.current.Store(backend)
	b.fingerprint = capture.fingerprint
	for _, duplicate := range backend.duplicates {
		SystemIOStreams().Errorf("Warning: %s\n", duplicate)
	}

	return b, nil
}

func (b *mcpWatchingBackend) CallTool(ctx context.Context, name string, raw json.RawMessage) (any, error) {
	// Load once: all work and any returned entity pointers refer to this snapshot,
	// even when the watcher publishes a replacement before response encoding.
	return b.current.Load().CallTool(ctx, name, raw)
}

// poll publishes a change only after two consecutive equal source captures.
// Parsing uses the captured bytes, so it cannot accidentally reopen newer files.
// This debounces saves; it is not a transaction for arbitrary multi-file writers.
func (b *mcpWatchingBackend) poll(ctx context.Context) (bool, error) {
	capture, err := b.source.capture(ctx)
	if err != nil {
		b.hasPending = false

		return false, err
	}
	if capture.fingerprint == b.fingerprint {
		b.hasPending, b.failedErr = false, nil

		return false, nil
	}
	if b.failedErr != nil && capture.fingerprint == b.failed {
		// Parsing is deterministic in the captured bytes: report the same
		// failure without loading them again until the source changes.
		b.hasPending = false

		return false, b.failedErr
	}
	if !b.hasPending || capture.fingerprint != b.pending {
		b.pending, b.hasPending = capture.fingerprint, true

		return false, nil
	}
	b.hasPending = false
	backend, err := b.source.load(capture)
	if err != nil {
		b.failed, b.failedErr = capture.fingerprint, err

		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	b.current.Store(backend)
	b.fingerprint, b.failedErr = capture.fingerprint, nil

	return true, nil
}

func (b *mcpWatchingBackend) run(ctx context.Context, streams *IOStreams) {
	ticker := time.NewTicker(mcpWatchInterval)
	defer ticker.Stop()
	var lastError string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := b.poll(ctx)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				if message := err.Error(); message != lastError {
					streams.Errorf("MCP watch: keeping previous snapshot: %s\n", message)
					lastError = message
				}

				continue
			}
			// Recovery to the unchanged original source also resets warnings.
			// A pending candidate is not yet a successful load.
			if !b.hasPending {
				lastError = ""
			}
			if changed {
				for _, duplicate := range b.current.Load().duplicates {
					streams.Errorf("Warning: %s\n", duplicate)
				}
				streams.Errorf("MCP watch: reloaded archive snapshot\n")
			}
		}
	}
}
