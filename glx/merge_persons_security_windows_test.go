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
	"strings"
	"testing"

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

func mergeTestDACL(t *testing.T, path string) string {
	t.Helper()
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	require.NoError(t, err)

	return descriptor.String()
}

func assertMergeTestPrivateDACL(t *testing.T, path string) {
	t.Helper()
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	require.NoError(t, err)
	control, _, err := descriptor.Control()
	require.NoError(t, err)
	assert.NotZero(t, control&windows.SE_DACL_PROTECTED, path)
	dacl, _, err := descriptor.DACL()
	require.NoError(t, err)
	require.NotNil(t, dacl, path+" must not have a NULL DACL")
	require.EqualValues(t, 1, dacl.AceCount, path)
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	require.NoError(t, err)
	assert.Contains(t, descriptor.String(), user.User.Sid.String(), path)
	assert.NotContains(t, descriptor.String(), ";;;WD)", path)
}

// Security must exist on the empty handle, before any sensitive byte is written.
func TestMergeWindowsObjectsArePrivateAtCreation(t *testing.T) {
	dir := t.TempDir()
	setMergeTestDACL(t, dir, "D:P(A;OICI;FA;;;WD)")
	// The legacy mode-only API inherits the permissive ACL on Windows.
	// This empty control contains no archive data.
	legacy := filepath.Join(dir, "legacy-empty-control")
	require.NoError(t, os.Mkdir(legacy, 0o700))
	assert.Contains(t, mergeTestDACL(t, legacy), ";;;WD)")
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
				checkedStage, checkedBackup, checkedMarker, checkedOriginal := false, false, false, false
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
					if strings.Contains(name, string(filepath.Separator)+"new"+string(filepath.Separator)) {
						descriptor, securityErr := windows.GetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
						require.NoError(t, securityErr)
						assert.Equal(t, expected, descriptor.String(), "staged replacement DACL at first sync")
						checkedStage = true
					}
					backup := filepath.Join(dir+".bak", path)
					if _, statErr := os.Stat(backup); statErr == nil {
						assert.Equal(t, expected, mergeTestDACL(t, backup))
						checkedBackup = true
						assertMergeTestPrivateDACL(t, dir+".bak")
					}
					marker := filepath.Join(dir+".bak", swapInProgressMarker)
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
				assert.Equal(t, expected, mergeTestDACL(t, live))
				data, readErr := os.ReadFile(live)
				require.NoError(t, readErr)
				if failure == "success" {
					assert.Equal(t, planned[path], data)
				} else {
					assert.Equal(t, original[path], data)
				}
				if failure == "rollback_failure" {
					assert.FileExists(t, filepath.Join(dir+".bak", swapInProgressMarker))
					assert.Equal(t, expected, mergeTestDACL(t, filepath.Join(dir+".bak", path)))
				} else {
					assert.NoDirExists(t, dir+".bak")
				}
			})
		}
	}
}
