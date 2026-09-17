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
	"path"
	"path/filepath"
	"runtime"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// archiveManagedTopLevel is the set of top-level paths written by the multi-file
// serializer. Anything in an archive directory that isn't in this set is foreign
// (user docs, .git, dotfiles, etc.) and must be preserved across a safe-write swap.
// See genealogix/glx#692. Derived from glxlib.AllEntityTypes so new entity types
// are picked up automatically.
// archiveMetadataFile is the top-level file the multi-file serializer writes the
// archive's metadata block to.
const archiveMetadataFile = "metadata.glx"

var archiveManagedTopLevel = func() map[string]bool {
	m := map[string]bool{
		archiveMetadataFile:           true,
		glxlib.ArchiveDirVocabularies: true,
	}
	for _, entityType := range glxlib.AllEntityTypes {
		m[entityType.String()] = true
	}

	return m
}()

// isManagedTopLevel reports whether a top-level archive entry is one the
// serializer owns and may replace: a real directory whose name matches a
// managed directory case-insensitively (PERSONS/ counts, so it is replaced by
// persons/ rather than duplicated — #1246), or the metadata file by its exact
// name. Anything else — a regular file that happens to be named PERSONS, a
// symlink named Persons, a directory named Metadata.glx — is foreign data the
// loader never read and the swap must preserve, not delete.
func isManagedTopLevel(entry fs.DirEntry) bool {
	if entry.Type().IsDir() {
		return archiveManagedTopLevel[strings.ToLower(entry.Name())] && strings.ToLower(entry.Name()) != archiveMetadataFile
	}

	return entry.Type().IsRegular() && entry.Name() == archiveMetadataFile
}

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

	if err := refuseTopLevelGLXFiles(destPath); err != nil {
		return err
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

	// Decide which entries the loader skipped while the backup is still intact.
	// A skipped symlink is recognized through every hop of its chain, and a hop
	// can be a top-level foreign entry (persons/z.glx -> ../alias.glx ->
	// .drafts/x.glx) that restoreForeignEntries moves away, so classification
	// has to precede every move out of the backup. The entries themselves are
	// moved last, once the larger units they may sit inside have gone across.
	skipped, err := classifySkippedEntries(backupDir, destPath)
	if err != nil {
		// Leave backupDir in place so the user can recover. Do not mark success.
		return fmt.Errorf("preserving skipped entries: %w", err)
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
	// This must run before moveSkippedEntries: that step would otherwise
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
	if err := moveSkippedEntries(backupDir, destPath, skipped); err != nil {
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
		if !isManagedTopLevel(entry) {
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
	mediaFilesDirs, err := mediaFilesDirsIn(backupDir)
	if err != nil {
		return fmt.Errorf("inspecting stale backup %s: %w", backupDir, err)
	}
	for _, mediaFilesDir := range mediaFilesDirs {
		mediaEntries, err := os.ReadDir(mediaFilesDir)
		if err != nil {
			return fmt.Errorf("inspecting %s: %w", mediaFilesDir, err)
		}
		if len(mediaEntries) > 0 {
			return fmt.Errorf("%w: %s contains %q", ErrStaleBackupForeignFile, backupDir, mediaFilesDir)
		}
	}
	// A dot-prefixed entry nested inside a managed directory (persons/.drafts/)
	// is skipped by the loader, so it was never re-emitted into the new
	// archive. Like media/files/, it is unrecovered data — refuse to delete it.
	if nested, err := firstSkippedEntry(backupDir); err != nil {
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
		if isManagedTopLevel(entry) {
			continue
		}
		src := filepath.Join(backupDir, name)
		dst := filepath.Join(destPath, name)
		// The fresh output holds only managed names, so a foreign entry that
		// already exists there means the writer produced a file with the same
		// name (metadata.glx from loaded metadata beside a metadata.glx symlink
		// the loader skipped, or persons/ beside a PERSONS file on a
		// case-insensitive filesystem). Renaming over it would silently replace
		// the writer's output; refuse and keep the backup instead.
		if _, err := os.Lstat(dst); err == nil {
			return fmt.Errorf("%w: %s", ErrPreservedEntryCollision, name)
		}
		if err := robustRename(src, dst); err != nil {
			return fmt.Errorf("restoring %s: %w", name, err)
		}
	}

	return nil
}

// firstSkippedEntry returns the path, relative to backupDir, of the first
// entry the loader would skip (loaderSkips: dot-prefixed names, symlinks or
// placeholders into dot-prefixed directories), or "" when there is none. Its
// caller has already rejected top-level foreign entries, so what this reports
// in practice is an entry nested inside a managed directory — unrecovered
// data that a stale backup must not be deleted with.
func firstSkippedEntry(backupDir string) (string, error) {
	var found string
	err := filepath.WalkDir(backupDir, func(srcPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if srcPath == backupDir {
			return nil
		}
		rel, err := filepath.Rel(backupDir, srcPath)
		if err != nil {
			return fmt.Errorf("resolving %s: %w", srcPath, err)
		}
		if !loaderSkips(backupDir, strings.TrimSuffix(backupDir, ".bak"), rel, d) {
			return nil
		}
		found = filepath.ToSlash(rel)

		return filepath.SkipAll
	})
	if err != nil {
		return "", fmt.Errorf("inspecting stale backup %s: %w", backupDir, err)
	}

	return found, nil
}

// refuseTopLevelGLXFiles rejects a destination that holds a .glx file at its
// top level other than metadata.glx. The loader reads such a file as archive
// content, so its entities are re-emitted under the entity directories, but
// the writer has no place for the file itself: carrying it across the swap as
// foreign duplicates every entity in it on the next load, and dropping it
// deletes a file the user laid out by hand. Neither is acceptable without the
// user's say-so — see genealogix/glx#1247 — so the write is refused before
// anything is touched. A missing destination is a fresh archive and passes,
// and so does a dot-prefixed file (._family.glx, .draft.glx): the loader skips
// those, and restoreForeignEntries carries them across untouched.
func refuseTopLevelGLXFiles(destPath string) error {
	entries, err := os.ReadDir(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("inspecting %s: %w", destPath, err)
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && isGLXFile(entry.Name()) && !isDotName(entry.Name()) && entry.Name() != archiveMetadataFile {
			return fmt.Errorf("%w: %s", ErrTopLevelGLXFile, entry.Name())
		}
	}

	return nil
}

// classifySkippedEntries returns the backup-relative paths of every entry the
// loader skipped (loaderSkips): dot-prefixed names, symlinks and placeholders
// into dot-prefixed directories. The serializer never re-emits these, so the
// swap has to carry them across or it deletes them silently with exit 0 and
// no .bak left behind — persons/.drafts/ is the common case, since "persons"
// is managed and the whole subtree goes with the backup.
//
// It must run over the intact backup, before any move: a symlink chain is
// judged through its intermediate hops, and a hop may itself be about to move
// (a top-level alias.glx that restoreForeignEntries carries across, or an
// earlier link in the same directory). Skipped directories are recorded
// without descending; they move as a unit.
func classifySkippedEntries(backupDir, destPath string) ([]string, error) {
	var skipped []string
	err := filepath.WalkDir(backupDir, func(srcPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if srcPath == backupDir {
			return nil
		}
		rel, err := filepath.Rel(backupDir, srcPath)
		if err != nil {
			return fmt.Errorf("resolving %s: %w", srcPath, err)
		}
		if !loaderSkips(backupDir, destPath, rel, d) {
			return nil
		}
		skipped = append(skipped, rel)
		if d.IsDir() {
			return fs.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return skipped, nil
}

// moveSkippedEntries carries the entries classifySkippedEntries found from the
// backup into the freshly written archive. It runs last: an entry that has
// already left the backup inside a larger unit (a top-level dot directory
// restored as foreign, media/files/.keep inside the media/files/ move) is done
// and skipped.
//
// Entries are moved, not copied: backupDir and destPath share a parent, so
// each rename is atomic and the data exists under a predictable name at every
// intermediate state. An entry whose destination already exists is left in the
// backup rather than overwritten — the fresh write wins, and the backup is
// then retained by the caller's error path for the user to inspect.
func moveSkippedEntries(backupDir, destPath string, skipped []string) error {
	for _, rel := range skipped {
		src := filepath.Join(backupDir, rel)
		dst := filepath.Join(destPath, rel)
		if _, err := os.Lstat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return fmt.Errorf("inspecting %s: %w", src, err)
		}
		if _, err := os.Lstat(dst); err == nil {
			// The entry is still in the backup, so nothing earlier carried it
			// across, and the serializer never writes skipped shapes: a match
			// means the writer produced a file at the same path. Moving on
			// would let the backup, and the entry with it, be removed; refuse
			// instead so the user can resolve the clash.
			return fmt.Errorf("%w: %s", ErrPreservedEntryCollision, filepath.ToSlash(rel))
		}
		if err := os.MkdirAll(filepath.Dir(dst), dirPermissions); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(dst), err)
		}
		if err := robustRename(src, dst); err != nil {
			return fmt.Errorf("restoring %s: %w", rel, err)
		}
	}

	return nil
}

// loaderSkips reports whether the archive loader would skip the entry at rel
// (OS-native, relative to root) — the entries the serializer therefore never
// re-emits and the safe-write swap must carry across. It mirrors walkGLXFiles:
//
//   - a dot-prefixed name (isDotName);
//   - a symlink whose chain ends under a dot-prefixed directory
//     (lexicalResolveInArchive, the lexical twin of
//     symlinkTargetIsExcluded);
//   - on Windows, a Git symlink placeholder file whose text points under a
//     dot-prefixed directory (placeholderTarget, as resolveSymlinkPlaceholder).
//
// The symlink and placeholder checks work from link text rather than
// resolving targets, so they give the same answer whether or not the target
// currently exists: by the time the swap inspects the backup, top-level dot
// directories have already been moved to the destination.
func loaderSkips(root, archiveRoot, rel string, d fs.DirEntry) bool {
	return loaderSkipsOn(runtime.GOOS, root, archiveRoot, rel, d)
}

// loaderSkipsOn is loaderSkips with the platform made explicit so the Windows
// placeholder branch can be exercised by tests on any host. root is the tree
// being inspected (the backup); archiveRoot is the archive's own path, which
// absolute symlink targets refer to.
func loaderSkipsOn(goos, root, archiveRoot, rel string, d fs.DirEntry) bool {
	if isDotName(d.Name()) {
		return true
	}
	if d.Type()&fs.ModeSymlink != 0 {
		resolved, ok := lexicalResolveInArchive(root, archiveRoot, filepath.ToSlash(rel))

		return ok && pathHasDotComponent(resolved)
	}
	if goos == goosWindows && d.Type().IsRegular() && isGLXFile(d.Name()) {
		// A placeholder is at most maxSymlinkPlaceholderLength bytes; do not
		// read whole entity files to find that out.
		info, err := d.Info()
		if err != nil || info.Size() > maxSymlinkPlaceholderLength {
			return false
		}
		data, err := os.ReadFile(filepath.Join(root, rel)) // #nosec G304 -- path enumerated from the archive backup
		if err != nil {
			return false
		}
		target, ok := placeholderTarget(rel, data)

		return ok && pathHasDotComponent(target)
	}

	return false
}

// maxSymlinkHops bounds the chain walk in lexicalResolveInArchive so a
// link cycle cannot spin forever; the OS limit for real resolution is similar.
const maxSymlinkHops = 40

// lexicalResolveInArchive expands every symlink on the path rel (slash
// separated, relative to root) component by component, using link text only,
// and returns the fully expanded archive-relative path. It is the
// existence-tolerant twin of the loader's EvalSymlinks: a component that no
// longer exists ends expansion and the remaining components are appended as
// written, so a link into a dot directory that has already been moved away is
// still recognized. Intermediate symlinked directories are expanded too, which
// plain Lstat on the whole path would silently follow. An absolute target is
// accepted when it lies under archiveRoot or root and is folded back into an
// archive-relative path, as the loader accepts an absolute target that resolves
// inside the archive. A target that climbs out, or a cycle, returns false.
func lexicalResolveInArchive(root, archiveRoot, rel string) (string, bool) {
	pending := strings.Split(path.Clean(rel), "/")
	var done []string
	hops := 0
	for len(pending) > 0 {
		done = append(done, pending[0])
		pending = pending[1:]
		cur := path.Join(done...)
		if cur == ".." || strings.HasPrefix(cur, "../") {
			return "", false
		}
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(cur)))
		if err != nil {
			return path.Clean(path.Join(append(done, pending...)...)), true
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		hops++
		if hops > maxSymlinkHops {
			return "", false
		}
		target, err := os.Readlink(filepath.Join(root, filepath.FromSlash(cur)))
		if err != nil {
			return "", false
		}
		var next string
		if filepath.IsAbs(target) {
			inside, ok := relWithinEither(target, archiveRoot, root)
			if !ok {
				return "", false
			}
			next = inside
		} else {
			next = path.Clean(path.Join(path.Join(done[:len(done)-1]...), filepath.ToSlash(target)))
		}
		if next == ".." || strings.HasPrefix(next, "../") {
			return "", false
		}
		pending = append(strings.Split(next, "/"), pending...)
		done = nil
	}

	return path.Clean(path.Join(done...)), true
}

// relWithinEither returns abs relative (slash separated) to the first of the
// given roots that contains it, or false when neither does.
func relWithinEither(abs string, roots ...string) (string, bool) {
	for _, r := range roots {
		if r == "" {
			continue
		}
		if rel, ok := relWithin(abs, r); ok {
			return filepath.ToSlash(rel), true
		}
	}

	return "", false
}

// mediaFilesDirsIn returns every media/files/ subtree under dir, matching each
// path component case-insensitively so an archive laid out as MEDIA/files/ or
// media/FILES/ keeps its binaries across a safe write. On a case-sensitive
// filesystem more than one can exist at once (media/files/ beside
// MEDIA/files/); callers treat that as ambiguous rather than pick one and
// delete the other with the backup. An empty result means there is none.
//
// Only real directories are matched at each level. A symlink such as
// `MEDIA -> ../assets` is not archive content, and renaming a path that runs
// through it would move data from outside the archive. A directory that cannot
// be listed is reported as an error rather than treated as absent, so callers
// that decide whether a backup is safe to delete fail closed.
func mediaFilesDirsIn(dir string) ([]string, error) {
	parts := strings.Split(filepath.ToSlash(glxlib.MediaFilesDir), "/")
	dirs := []string{dir}
	for i, part := range parts {
		var next []string
		for _, d := range dirs {
			names, err := childDirsCaseInsensitive(d, part, i == len(parts)-1)
			if err != nil {
				return nil, err
			}
			for _, name := range names {
				next = append(next, filepath.Join(d, name))
			}
		}
		dirs = next
	}

	return dirs, nil
}

// childDirsCaseInsensitive returns the names of dir's real subdirectories that
// match name case-insensitively, the exact match first when present. With
// allowLink set, a symlink of that name is matched as well: the final
// media/files component may be a link to storage elsewhere, and renaming a
// link moves the link itself, not its target, so preserving it is safe where
// renaming *through* a linked intermediate directory would not be. A missing
// dir yields no names; any other listing failure is returned.
func childDirsCaseInsensitive(dir, name string, allowLink bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("listing %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		isLink := e.Type()&fs.ModeSymlink != 0
		if !e.Type().IsDir() && (!allowLink || !isLink) {
			continue
		}
		switch {
		case e.Name() == name:
			names = append([]string{name}, names...)
		case strings.EqualFold(e.Name(), name):
			names = append(names, e.Name())
		}
	}

	return names, nil
}

// preserveMediaBinaries carries media/files/ from the backup into the freshly
// written destination. The multi-file serializer writes Media entity YAML under
// media/ but never produces anything under media/files/, so the tmpDir that was
// swapped into place is guaranteed to have no binaries — moving the subtree
// from the backup is the only way for the user's media files to survive the
// safe-write swap. See genealogix/glx#593.
func preserveMediaBinaries(backupDir, destPath string) error {
	srcDirs, err := mediaFilesDirsIn(backupDir)
	if err != nil {
		return err
	}
	if len(srcDirs) == 0 {
		return nil
	}
	if len(srcDirs) > 1 {
		// Two case variants cannot both become the canonical media/files/;
		// refuse (the backup is retained) rather than keep one and delete the
		// other.
		return fmt.Errorf("%w: %s and %s", ErrAmbiguousMediaFilesDirs, srcDirs[0], srcDirs[1])
	}
	srcDir := srcDirs[0]
	// Lstat: a media/files symlink is moved as a link, whatever it points at.
	info, err := os.Lstat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("inspecting backup media/files: %w", err)
	}
	if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
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
		if relPath == archiveMetadataFile {
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
