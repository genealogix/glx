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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// archiveManagedTopLevel is the set of top-level paths written by the multi-file
// serializer. Anything in an archive directory that isn't in this set is foreign
// (user docs, .git, dotfiles, etc.) and must be preserved across a safe-write swap.
// See genealogix/glx#692. Derived from glxlib.AllEntityTypes so new entity types
// are picked up automatically.
var archiveManagedTopLevel = func() map[string]bool {
	m := map[string]bool{
		"metadata.glx":                true,
		glxlib.ArchiveDirVocabularies: true,
	}
	for _, entityType := range glxlib.AllEntityTypes {
		m[entityType.String()] = true
	}

	return m
}()

// safeWriteMultiFileArchive writes a multi-file archive to a temporary directory
// first, then swaps it into place. This prevents archive destruction if the write
// fails partway through (e.g., power loss, disk full, signal).
//
// Non-archive top-level entries in destPath (e.g., .git, README.md, CLAUDE.md,
// dotfiles) are preserved across the swap; see archiveManagedTopLevel.
func safeWriteMultiFileArchive(destPath string, archive *glxlib.GLXFile) error {
	// Resolve to absolute path so rename and cwd containment checks work
	// reliably for relative paths like ".". mergeArchives does the same.
	absPath, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("resolving destination path: %w", err)
	}
	destPath = absPath

	// On Windows, a directory cannot be renamed while it is any process's cwd.
	// If our cwd is inside (or equal to) destPath, temporarily move to the
	// parent directory so the rename operations succeed.
	parentDir := filepath.Dir(destPath)
	if cwd, err := os.Getwd(); err == nil {
		if absCwd, err2 := filepath.Abs(cwd); err2 == nil {
			if rel, err3 := filepath.Rel(destPath, absCwd); err3 == nil {
				// cwd is inside destPath if rel is "." or does not start with "..".
				if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
					if err := os.Chdir(parentDir); err != nil {
						return fmt.Errorf("changing to parent directory: %w", err)
					}
					defer os.Chdir(cwd) //nolint:errcheck // best-effort restore
				}
			}
		}
	}

	// Create temp dir next to the destination (same filesystem for rename)
	tmpDir, err := os.MkdirTemp(parentDir, ".glx-tmp-")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}

	// Clean up temp dir on failure
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	// Write the archive to temp
	if err := writeMultiFileArchive(tmpDir, archive, false); err != nil {
		return fmt.Errorf("writing to temp directory: %w", err)
	}

	// Create backup of the original
	backupDir := destPath + ".bak"
	if err := removeStaleBackup(backupDir); err != nil {
		return err
	}
	if err := robustRename(destPath, backupDir); err != nil {
		return fmt.Errorf("backing up original: %w", err)
	}

	// Move temp into place
	if err := robustRename(tmpDir, destPath); err != nil {
		// Restore backup on failure
		_ = robustRename(backupDir, destPath) // best-effort restore

		return fmt.Errorf("moving archive into place: %w", err)
	}

	// Preserve foreign (non-managed) top-level entries from the backup back into
	// the newly-written archive. Without this step the swap silently destroys
	// .git, README.md, and any other user content that sits alongside the
	// managed entity directories — see genealogix/glx#692.
	if err := restoreForeignEntries(backupDir, destPath); err != nil {
		// Leave backupDir in place so the user can recover. Do not mark success.
		return fmt.Errorf("preserving non-archive files: %w", err)
	}

	// The serializer writes Media entity YAML under media/ but never touches
	// media/files/, so the fresh tmpDir has no binaries. Move the dest's
	// pre-existing media/files/ across the swap; without this every safe-write
	// would silently destroy media binaries — see genealogix/glx#593.
	//
	// This must run before preserveSkippedDotEntries: that walk would otherwise
	// recreate media/files/ in destPath to hold a dot entry such as
	// media/files/.keep, and the whole-directory rename below would then fail
	// because its target already exists. Moving media/files/ first carries any
	// dot entries inside it along with the binaries.
	if err := preserveMediaBinaries(backupDir, destPath); err != nil {
		// Leave backupDir in place so the user can recover. Do not mark success.
		return fmt.Errorf("preserving media binaries: %w", err)
	}

	// Entries the loader skips are never re-emitted by the serializer, so the
	// fresh tmpDir cannot contain them. Carry them across the swap for the
	// same reason foreign top-level entries are carried across: otherwise the
	// swap destroys them silently.
	if err := preserveSkippedDotEntries(backupDir, destPath); err != nil {
		// Leave backupDir in place so the user can recover. Do not mark success.
		return fmt.Errorf("preserving skipped entries: %w", err)
	}

	// Clean up backup (now contains only managed entries that have been
	// superseded by the fresh write).
	_ = os.RemoveAll(backupDir)
	success = true

	return nil
}

// removeStaleBackup deletes a leftover .bak directory, but only after verifying
// it contains no foreign (non-managed) entries. If it does, a previous invocation
// failed mid-restore and those entries are the user's unrecovered data — refuse
// to proceed rather than silently destroy them.
func removeStaleBackup(backupDir string) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("inspecting stale backup %s: %w", backupDir, err)
	}
	for _, entry := range entries {
		if !archiveManagedTopLevel[entry.Name()] {
			return fmt.Errorf("%w: %s contains %q", ErrStaleBackupForeignFile, backupDir, entry.Name())
		}
	}
	// media/ is managed (so media/<id>.glx entity files are fair game to drop)
	// but media/files/ holds user binaries that the serializer never produces.
	// A stale backup whose binaries haven't been carried into the new archive
	// is unrecovered data; refuse to delete it. Any error other than "doesn't
	// exist" (e.g., permissions, transient I/O) is also refused — we cannot
	// confirm the backup is empty, so the safe move is to leave it for the
	// user to inspect.
	mediaFilesDir := filepath.Join(backupDir, glxlib.MediaFilesDir)
	switch mediaEntries, err := os.ReadDir(mediaFilesDir); {
	case err == nil:
		if len(mediaEntries) > 0 {
			return fmt.Errorf("%w: %s contains %q", ErrStaleBackupForeignFile, backupDir, glxlib.MediaFilesDir)
		}
	case os.IsNotExist(err):
		// No media/files/ in the backup — safe to proceed.
	default:
		return fmt.Errorf("inspecting %s: %w", mediaFilesDir, err)
	}
	// A dot-prefixed entry nested inside a managed directory (persons/.drafts/)
	// is skipped by the loader, so it was never re-emitted into the new
	// archive. Like media/files/, it is unrecovered data — refuse to delete it.
	if nested, err := firstNestedDotEntry(backupDir); err != nil {
		return err
	} else if nested != "" {
		return fmt.Errorf("%w: %s contains %q", ErrStaleBackupForeignFile, backupDir, nested)
	}
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("removing stale backup %s: %w", backupDir, err)
	}

	return nil
}

// restoreForeignEntries moves every top-level entry from backupDir into destPath
// unless the entry name is in archiveManagedTopLevel. The source and destination
// are on the same filesystem (backupDir and destPath share a parent), so each
// rename is atomic.
//
// Preservation runs after the swap rather than before, so at every intermediate
// state the user's data exists on disk under a predictable name: before the
// rename it lives in backupDir (still named destPath.bak); after it lives in
// destPath. If any rename fails the backup is retained so the user can recover.
func restoreForeignEntries(backupDir, destPath string) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("reading backup directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if archiveManagedTopLevel[name] {
			continue
		}
		src := filepath.Join(backupDir, name)
		dst := filepath.Join(destPath, name)
		if err := robustRename(src, dst); err != nil {
			return fmt.Errorf("restoring %s: %w", name, err)
		}
	}

	return nil
}

// firstNestedDotEntry returns the path, relative to backupDir, of the first
// dot-prefixed entry in the tree, or "" when there is none. Its caller has
// already rejected top-level dot entries (no dot name is in
// archiveManagedTopLevel), so what this reports in practice is an entry nested
// inside a managed directory.
func firstNestedDotEntry(backupDir string) (string, error) {
	var found string
	err := filepath.WalkDir(backupDir, func(srcPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if srcPath == backupDir || !isDotName(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(backupDir, srcPath)
		if err != nil {
			return fmt.Errorf("resolving %s: %w", srcPath, err)
		}
		found = filepath.ToSlash(rel)

		return filepath.SkipAll
	})
	if err != nil {
		return "", fmt.Errorf("inspecting stale backup %s: %w", backupDir, err)
	}

	return found, nil
}

// preserveSkippedDotEntries carries dot-prefixed entries nested *inside* the
// managed entity directories from the backup into the freshly written archive.
//
// restoreForeignEntries covers the top level, where a dot-prefixed name is by
// definition not in archiveManagedTopLevel. It does not reach persons/.drafts/
// or media/.cache/: "persons" and "media" are managed, so the whole subtree —
// dot directory included — is dropped when the backup is removed. Since the
// loader skips those entries (see isDotName) the serializer never re-emits
// them, which made the swap a silent delete with exit 0 and no .bak left
// behind.
//
// Entries are moved, not copied: backupDir and destPath share a parent, so
// each rename is atomic and the data exists under a predictable name at every
// intermediate state. An entry whose destination already exists is left in the
// backup rather than overwritten — the fresh write wins, and the backup is
// then retained by the caller's error path for the user to inspect.
func preserveSkippedDotEntries(backupDir, destPath string) error {
	return filepath.WalkDir(backupDir, func(srcPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if srcPath == backupDir || !isDotName(d.Name()) {
			return nil
		}

		rel, err := filepath.Rel(backupDir, srcPath)
		if err != nil {
			return fmt.Errorf("resolving %s: %w", srcPath, err)
		}
		dst := filepath.Join(destPath, rel)

		if _, err := os.Lstat(dst); err == nil {
			if d.IsDir() {
				return fs.SkipDir
			}

			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), dirPermissions); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(dst), err)
		}
		if err := robustRename(srcPath, dst); err != nil {
			return fmt.Errorf("restoring %s: %w", rel, err)
		}
		if d.IsDir() {
			// The subtree moved with the directory; do not descend into a
			// path that no longer exists.
			return fs.SkipDir
		}

		return nil
	})
}

// preserveMediaBinaries carries media/files/ from the backup into the freshly
// written destination. The multi-file serializer writes Media entity YAML under
// media/ but never produces anything under media/files/, so the tmpDir that was
// swapped into place is guaranteed to have no binaries — moving the subtree
// from the backup is the only way for the user's media files to survive the
// safe-write swap. See genealogix/glx#593.
func preserveMediaBinaries(backupDir, destPath string) error {
	srcDir := filepath.Join(backupDir, glxlib.MediaFilesDir)
	info, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("inspecting backup media/files: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	dstDir := filepath.Join(destPath, glxlib.MediaFilesDir)
	// Parent media/ may not exist if the new archive has no Media entities;
	// create it so the rename target has a valid parent.
	if err := os.MkdirAll(filepath.Dir(dstDir), dirPermissions); err != nil {
		return fmt.Errorf("creating media dir: %w", err)
	}
	// Whole-subdir rename is safe: the safe-write tmpDir never contains
	// media/files/, so dstDir is guaranteed absent.
	if err := robustRename(srcDir, dstDir); err != nil {
		return fmt.Errorf("moving media/files into place: %w", err)
	}

	return nil
}

// LoadArchive loads all GLX files from a directory with schema validation.
// This is the primary entry point for the validate command.
func LoadArchive(rootPath string) (*glxlib.GLXFile, []string, error) {
	return LoadArchiveWithOptions(rootPath, true)
}

// LoadArchiveWithOptions loads all GLX files from a directory into a single GLXFile.
// When schemaValidate is true, each file is validated against the GLX JSON schema
// before deserialization. Deserialization is delegated to DeserializeMultiFileFromMap.
func LoadArchiveWithOptions(rootPath string, schemaValidate bool) (*glxlib.GLXFile, []string, error) {
	files, err := collectGLXFilesFromDir(rootPath)
	if err != nil {
		return nil, nil, err
	}

	return loadArchiveFromFiles(rootPath, files, schemaValidate)
}

// loadArchiveFromFiles builds an archive from an already-collected file map
// (relative path -> contents, as returned by collectGLXFilesFromDir). It is
// the half of LoadArchiveWithOptions that does no I/O of its own, split out so
// a caller that needs to know how many files the archive contains can read it
// off the map instead of walking the tree a second time with its own copy of
// the "what is archive content" rules — two walks that had already drifted
// apart on a symlinked archive root.
func loadArchiveFromFiles(rootPath string, files map[string][]byte, schemaValidate bool) (*glxlib.GLXFile, []string, error) {
	if schemaValidate {
		var allErrors []string
		for relPath, data := range files {
			absPath := filepath.Join(rootPath, relPath)

			doc, parseErr := ParseYAMLFile(data)
			if parseErr != nil {
				allErrors = append(allErrors, fmt.Sprintf("%s: YAML parse error: %v", absPath, parseErr))

				continue
			}

			issues := ValidateGLXFileStructure(doc)
			if len(issues) > 0 {
				allErrors = append(allErrors, fmt.Sprintf("%s:\n  - %s", absPath, strings.Join(issues, "\n  - ")))
			}
		}
		if len(allErrors) > 0 {
			return nil, nil, fmt.Errorf("%w:\n\n%s", ErrMultipleFilesFailed, strings.Join(allErrors, "\n\n"))
		}
	}

	// Pass schemaValidate to serializer to enable referential integrity validation
	serializer := createSerializer(schemaValidate, false, "")
	glx, duplicates, err := serializer.DeserializeMultiFileFromMap(files)
	if err != nil {
		return nil, nil, err
	}
	duplicates = sanitizeDuplicateWarnings(duplicates)

	// Load standard vocabularies as defaults for any vocabulary maps not
	// already defined by the archive. This enables property reference
	// validation (e.g., born_at with reference_type: places) without
	// overwriting user-defined vocabularies.
	if err := mergeStandardVocabularies(glx); err != nil {
		return nil, nil, fmt.Errorf("failed to load standard vocabularies: %w", err)
	}
	// Invalidate cached validation results from deserialization, which ran
	// before vocabularies were loaded and would miss property reference checks.
	glx.InvalidateCache()

	return glx, duplicates, nil
}

// sanitizeDuplicateWarnings makes deserializer duplicate warnings safe to
// print. The warnings quote archive-controlled entity IDs and file names and
// exist only to be shown as diagnostics — two dozen runners write them to
// stderr, several with a bare fmt.Fprintf. Every caller of
// DeserializeMultiFileFromMap must pass its duplicates through here so no
// print site can leak a control sequence (genealogix/glx#925). The slice is
// sanitized in place and returned for convenience.
func sanitizeDuplicateWarnings(duplicates []string) []string {
	for i, d := range duplicates {
		duplicates[i] = sanitizeForTerminal(d)
	}

	return duplicates
}

// createSerializer creates a new serializer with the specified options
func createSerializer(validate, pretty bool, indent string) *glxlib.DefaultSerializer {
	opts := &glxlib.SerializerOptions{
		Validate: validate,
		Pretty:   pretty,
		Indent:   indent,
	}

	return glxlib.NewSerializer(opts)
}

// readSingleFileArchive reads and deserializes a single-file GLX archive
func readSingleFileArchive(path string, validate bool) (*glxlib.GLXFile, error) {
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	serializer := createSerializer(validate, false, "")
	glx, err := serializer.DeserializeSingleFileBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load archive: %w", err)
	}

	return glx, nil
}

// writeSingleFileArchive serializes and writes a single-file GLX archive with
// the default file permissions. Callers rewriting a file that already exists
// should use writeSingleFileArchiveWithMode instead.
func writeSingleFileArchive(path string, glx *glxlib.GLXFile, validate bool) error {
	return writeSingleFileArchiveWithMode(path, glx, validate, filePermissions)
}

// writeSingleFileArchiveWithMode is writeSingleFileArchive with an explicit
// permission mode. In-place rewrites pass the mode the file already has so a
// deliberately private archive (0600, say) does not come back world-readable
// because the write went through a fresh temp file.
func writeSingleFileArchiveWithMode(path string, glx *glxlib.GLXFile, validate bool, perm os.FileMode) error {
	serializer := createSerializer(validate, true, "  ")

	yamlBytes, err := serializer.SerializeSingleFileBytes(glx)
	if err != nil {
		return fmt.Errorf("failed to serialize GLX file: %w", err)
	}

	if err := atomicWriteFile(path, yamlBytes, perm); err != nil {
		return fmt.Errorf("failed to write GLX file: %w", err)
	}

	return nil
}

// serializeMultiFileArchive serializes (and, when validate is set, validates) a
// multi-file GLX archive, returning the files to write without touching the
// filesystem. Split out from writeMultiFileArchive so callers with other side
// effects to commit can fail on serialization or validation *before* any of
// them happen — see the GEDZIP import path, which stages media first.
func serializeMultiFileArchive(glx *glxlib.GLXFile, validate bool) (map[string][]byte, error) {
	serializer := createSerializer(validate, true, "  ")

	files, err := serializer.SerializeMultiFileToMap(glx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize multi-file archive: %w", err)
	}

	return files, nil
}

// writeMultiFileArchive serializes and writes a multi-file GLX archive
func writeMultiFileArchive(dirPath string, glx *glxlib.GLXFile, validate bool) error {
	files, err := serializeMultiFileArchive(glx, validate)
	if err != nil {
		return err
	}

	if err := writeFilesToDir(dirPath, files); err != nil {
		return err
	}

	return nil
}

// writePartialArchive serializes a partial GLXFile (containing only newly
// minted entities) and writes just the entity files into an existing archive
// directory — vocabulary and metadata output from the serializer is dropped so
// the existing archive's definitions are preserved. Used by commands that
// append to an archive rather than rewrite it. Returns the number of entity
// files written.
func writePartialArchive(dirPath string, partial *glxlib.GLXFile) (int, error) {
	serializer := createSerializer(false, true, "  ")

	files, err := serializer.SerializeMultiFileToMap(partial)
	if err != nil {
		return 0, fmt.Errorf("serializing partial archive: %w", err)
	}

	entityFiles := make(map[string][]byte, len(files))
	for relPath, data := range files {
		// SerializeMultiFileToMap emits forward-slash keys on every OS
		// (issue #900) — no need to check for OS-specific separators.
		if strings.HasPrefix(relPath, "vocabularies/") {
			continue
		}
		if relPath == "metadata.glx" {
			continue
		}
		entityFiles[relPath] = data
	}

	if err := writeFilesToDir(dirPath, entityFiles); err != nil {
		return 0, err
	}

	return len(entityFiles), nil
}

// mergeStandardVocabularies loads standard vocabularies into a GLXFile,
// filling only empty maps. User-defined vocabularies are preserved.
//
//nolint:gocyclo // 21 vocab fields with identical empty-map guards; splitting hurts readability
func mergeStandardVocabularies(glx *glxlib.GLXFile) error {
	std := &glxlib.GLXFile{}
	if err := glxlib.LoadStandardVocabulariesIntoGLX(std); err != nil {
		return err
	}

	if len(glx.EventTypes) == 0 {
		glx.EventTypes = std.EventTypes
	}
	if len(glx.RelationshipTypes) == 0 {
		glx.RelationshipTypes = std.RelationshipTypes
	}
	if len(glx.PlaceTypes) == 0 {
		glx.PlaceTypes = std.PlaceTypes
	}
	if len(glx.SourceTypes) == 0 {
		glx.SourceTypes = std.SourceTypes
	}
	if len(glx.RepositoryTypes) == 0 {
		glx.RepositoryTypes = std.RepositoryTypes
	}
	if len(glx.ParticipantRoles) == 0 {
		glx.ParticipantRoles = std.ParticipantRoles
	}
	if len(glx.MediaTypes) == 0 {
		glx.MediaTypes = std.MediaTypes
	}
	if len(glx.ConfidenceLevels) == 0 {
		glx.ConfidenceLevels = std.ConfidenceLevels
	}
	if len(glx.SexTypes) == 0 {
		glx.SexTypes = std.SexTypes
	}
	if len(glx.GenderTypes) == 0 {
		glx.GenderTypes = std.GenderTypes
	}
	if len(glx.StudyTypes) == 0 {
		glx.StudyTypes = std.StudyTypes
	}
	if len(glx.StudyStatuses) == 0 {
		glx.StudyStatuses = std.StudyStatuses
	}
	if len(glx.LegalStatuses) == 0 {
		glx.LegalStatuses = std.LegalStatuses
	}
	if len(glx.SourceNatures) == 0 {
		glx.SourceNatures = std.SourceNatures
	}
	if len(glx.InformationTypes) == 0 {
		glx.InformationTypes = std.InformationTypes
	}
	if len(glx.PersonProperties) == 0 {
		glx.PersonProperties = std.PersonProperties
	}
	if len(glx.EventProperties) == 0 {
		glx.EventProperties = std.EventProperties
	}
	if len(glx.RelationshipProperties) == 0 {
		glx.RelationshipProperties = std.RelationshipProperties
	}
	if len(glx.PlaceProperties) == 0 {
		glx.PlaceProperties = std.PlaceProperties
	}
	if len(glx.MediaProperties) == 0 {
		glx.MediaProperties = std.MediaProperties
	}
	if len(glx.RepositoryProperties) == 0 {
		glx.RepositoryProperties = std.RepositoryProperties
	}
	if len(glx.CitationProperties) == 0 {
		glx.CitationProperties = std.CitationProperties
	}
	if len(glx.SourceProperties) == 0 {
		glx.SourceProperties = std.SourceProperties
	}

	return nil
}
