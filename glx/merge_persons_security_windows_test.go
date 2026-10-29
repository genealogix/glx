//go:build windows

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
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func setMergeTestDACL(t *testing.T, path, sddl string) {
	t.Helper()
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	require.NoError(t, err)
	dacl, _, err := descriptor.DACL()
	require.NoError(t, err)
	require.NoError(t, windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil))
}

type mergeTestAccessEntry struct {
	Kind  uint8
	Flags uint8
	Mask  windows.ACCESS_MASK
	SID   string
}

type mergeTestAccess struct {
	Protected bool
	Null      bool
	Entries   []mergeTestAccessEntry
}

func mergeTestDescriptorAccess(t *testing.T, descriptor *windows.SECURITY_DESCRIPTOR) mergeTestAccess {
	t.Helper()
	control, _, err := descriptor.Control()
	require.NoError(t, err)
	dacl, _, err := descriptor.DACL()
	require.NoError(t, err)
	access := mergeTestAccess{Protected: control&windows.SE_DACL_PROTECTED != 0, Null: dacl == nil}
	if dacl != nil {
		for i := range uint32(dacl.AceCount) {
			var ace *windows.ACCESS_ALLOWED_ACE
			require.NoError(t, windows.GetAce(dacl, i, &ace))
			// These fixtures use standard allow/deny ACEs, whose SID follows Mask.
			sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
			access.Entries = append(access.Entries, mergeTestAccessEntry{Kind: ace.Header.AceType, Flags: ace.Header.AceFlags, Mask: ace.Mask, SID: sid.String()})
		}
	}
	runtime.KeepAlive(descriptor)

	return access
}

func mergeTestDACL(t *testing.T, path string) mergeTestAccess {
	t.Helper()
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	require.NoError(t, err)

	return mergeTestDescriptorAccess(t, descriptor)
}

func assertMergeTestPrivateDACL(t *testing.T, path string) {
	t.Helper()
	access := mergeTestDACL(t, path)
	assert.True(t, access.Protected, path)
	assert.False(t, access.Null, path)
	require.Len(t, access.Entries, 1, path)
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	require.NoError(t, err)
	assert.Equal(t, user.User.Sid.String(), access.Entries[0].SID, path)
	assert.EqualValues(t, windows.ACCESS_ALLOWED_ACE_TYPE, access.Entries[0].Kind, path)
	const fullFileAccess = 0x1f01ff // FILE_ALL_ACCESS (WinNT.h), represented as FA in SDDL.
	assert.EqualValues(t, fullFileAccess, access.Entries[0].Mask, path)
}

// Security must exist on the empty handle, before any sensitive byte is written.
func TestMergeWindowsObjectsArePrivateAtCreation(t *testing.T) {
	dir := t.TempDir()
	setMergeTestDACL(t, dir, "D:P(A;OICI;FA;;;WD)")
	// The legacy mode-only API inherits the permissive ACL on Windows.
	// This empty control contains no archive data.
	legacy := filepath.Join(dir, "legacy-empty-control")
	require.NoError(t, os.Mkdir(legacy, 0o700))
	legacyAccess := mergeTestDACL(t, legacy)
	assert.False(t, legacyAccess.Protected)
	require.Len(t, legacyAccess.Entries, 1)
	assert.Equal(t, "S-1-1-0", legacyAccess.Entries[0].SID)
	assert.NotZero(t, legacyAccess.Entries[0].Flags&windows.INHERITED_ACE)
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	defer func() { require.NoError(t, root.Close()) }()
	require.NoError(t, makeMergePrivateDirectory(root, "transaction"))
	assertMergeTestPrivateDACL(t, filepath.Join(dir, "transaction"))
	file, err := createMergeFile(root, "marker", 0o600, nil, "")
	require.NoError(t, err)
	defer func() { require.NoError(t, file.Close()) }()
	info, err := file.Stat()
	require.NoError(t, err)
	assert.Zero(t, info.Size())
	assertMergeTestPrivateDACL(t, filepath.Join(dir, "marker"))
	_, err = file.WriteString("sensitive recovery metadata")
	require.NoError(t, err)

	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	require.NoError(t, err)
	source := filepath.Join(dir, "source")
	require.NoError(t, os.WriteFile(source, nil, 0o600))
	setMergeTestDACL(t, source, "D:P(A;;FA;;;"+user.User.Sid.String()+")")
	copied, err := createMergeFile(root, "copy", 0o600, root, "source")
	require.NoError(t, err)
	defer func() { require.NoError(t, copied.Close()) }()
	info, err = copied.Stat()
	require.NoError(t, err)
	assert.Zero(t, info.Size())
	assert.Equal(t, mergeTestDACL(t, source), mergeTestDACL(t, filepath.Join(dir, "copy")))
	_, err = copied.WriteString("sensitive archive data")
	require.NoError(t, err)
}

func TestMergeWindowsRecoveryPreservesOriginalDACL(t *testing.T) {
	for _, privateArchive := range []bool{false, true} {
		for _, failure := range []string{"success", "install_failure", "rollback_failure"} {
			name := "broad_archive/" + failure
			if privateArchive {
				name = "private_archive/" + failure
			}
			t.Run(name, func(t *testing.T) {
				shared := t.TempDir()
				setMergeTestDACL(t, shared, "D:P(A;OICI;FA;;;WD)")
				dir := filepath.Join(shared, "archive")
				require.NoError(t, os.Mkdir(dir, 0o700))
				user, err := windows.GetCurrentProcessToken().GetTokenUser()
				require.NoError(t, err)
				private := "D:P(A;;FA;;;" + user.User.Sid.String() + ")"
				if privateArchive {
					setMergeTestDACL(t, dir, "D:P(A;OICI;FA;;;"+user.User.Sid.String()+")")
				}
				path := "person.glx"
				live := filepath.Join(dir, path)
				original := map[string][]byte{path: []byte("persons: {}\n# private original\n")}
				planned := map[string][]byte{path: []byte("persons: {}\n# private replacement\n")}
				require.NoError(t, os.WriteFile(live, original[path], 0o600))
				setMergeTestDACL(t, live, private)
				expected := mergeTestDACL(t, live)
				checkedStage, checkedBackup, checkedMarker, checkedOriginal, checkedJournal := false, false, false, false, false
				failed := false
				err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
					if strings.Contains(from, string(filepath.Separator)+"new"+string(filepath.Separator)) {
						transaction, _, _ := strings.Cut(from, string(filepath.Separator))
						assert.Equal(t, expected, mergeTestDACL(t, filepath.Join(dir, transaction, "original", path)))
						assert.Equal(t, expected, mergeTestDACL(t, filepath.Join(dir, from)))
						checkedOriginal = true
					}
					if strings.Contains(from, string(filepath.Separator)+"new"+string(filepath.Separator)) && failure != "success" {
						failed = true

						return fs.ErrPermission
					}

					return robustRenameIn(root, from, to)
				}, func(file *os.File) error {
					name := file.Name()
					journal := filepath.Join(dir, inPlaceJournalFile)
					if _, statErr := os.Stat(journal); statErr == nil {
						assertMergeTestPrivateDACL(t, journal)
						checkedJournal = true
					}
					if strings.Contains(name, string(filepath.Separator)+"new"+string(filepath.Separator)) {
						descriptor, securityErr := windows.GetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
						require.NoError(t, securityErr)
						assert.Equal(t, expected, mergeTestDescriptorAccess(t, descriptor), "staged replacement DACL at first sync")
						checkedStage = true
					}
					backup := filepath.Join(dir+".bak", path)
					if _, statErr := os.Stat(backup); statErr == nil {
						assert.Equal(t, expected, mergeTestDACL(t, backup))
						checkedBackup = true
						assertMergeTestPrivateDACL(t, dir+".bak")
					}
					marker := filepath.Join(dir+".bak", mergeBackupMarker)
					if _, statErr := os.Stat(marker); statErr == nil {
						assertMergeTestPrivateDACL(t, marker)
						checkedMarker = true
					}
					transactions, globErr := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
					require.NoError(t, globErr)
					for _, transaction := range transactions {
						assertMergeTestPrivateDACL(t, transaction)
					}
					if failed && failure == "rollback_failure" {
						return fs.ErrInvalid
					}

					return syncMergeHandle(file)
				})
				if failure == "success" {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, fs.ErrPermission)
				}
				assert.True(t, checkedStage)
				assert.True(t, checkedOriginal)
				assert.True(t, checkedBackup)
				assert.True(t, checkedMarker)
				assert.True(t, checkedJournal)
				assert.Equal(t, expected, mergeTestDACL(t, live))
				data, readErr := os.ReadFile(live)
				require.NoError(t, readErr)
				if failure == "success" {
					assert.Equal(t, planned[path], data)
				} else {
					assert.Equal(t, original[path], data)
				}
				if failure == "rollback_failure" {
					assertMergeTestPrivateDACL(t, filepath.Join(dir, inPlaceJournalFile))
					assert.FileExists(t, filepath.Join(dir+".bak", mergeBackupMarker))
					assert.Equal(t, expected, mergeTestDACL(t, filepath.Join(dir+".bak", path)))
				} else {
					assert.NoDirExists(t, dir+".bak")
				}
			})
		}
	}
}
