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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

func mergePrivateSecurityDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}

	return windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")")
}

func makeMergePrivateDirectory(root *os.Root, path string) error {
	security, err := mergePrivateSecurityDescriptor()
	if err != nil {
		return err
	}
	file, err := createMergeWindowsObject(root, path, security, true)
	if err != nil {
		return err
	}

	return file.Close()
}

func createMergeFile(root *os.Root, path string, mode os.FileMode, source *os.Root, sourcePath string) (*os.File, error) {
	var security *windows.SECURITY_DESCRIPTOR
	var err error
	if source == nil {
		security, err = mergePrivateSecurityDescriptor()
	} else {
		var original *os.File
		original, err = source.Open(sourcePath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = original.Close() }()
		security, err = windows.GetSecurityInfo(windows.Handle(original.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err == nil {
			// Freeze the original access list; do not append destination inheritance.
			err = security.SetControl(windows.SE_DACL_PROTECTED, windows.SE_DACL_PROTECTED)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("reading merge file security: %w", err)
	}

	return createMergeWindowsObject(root, path, security, false)
}

// Create with the DACL already attached: applying it after creation permits an
// unauthorized opener to retain a handle while sensitive bytes are written.
// Resolve only the parent via os.Root and create a single leaf relative to its
// handle, so security APIs cannot escape confinement through renamed paths.
func createMergeWindowsObject(root *os.Root, path string, security *windows.SECURITY_DESCRIPTOR, directory bool) (*os.File, error) {
	parent, err := root.Open(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = parent.Close() }()
	name, err := windows.NewNTUnicodeString(filepath.Base(path))
	if err != nil {
		return nil, err
	}
	attributes := windows.OBJECT_ATTRIBUTES{
		RootDirectory: windows.Handle(parent.Fd()), ObjectName: name,
		Attributes:         windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE,
		SecurityDescriptor: security,
	}
	attributes.Length = uint32(unsafe.Sizeof(attributes))
	options := uint32(windows.FILE_NON_DIRECTORY_FILE | windows.FILE_SYNCHRONOUS_IO_NONALERT)
	if directory {
		options = windows.FILE_DIRECTORY_FILE | windows.FILE_SYNCHRONOUS_IO_NONALERT
	}
	var handle windows.Handle
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(&handle, windows.GENERIC_READ|windows.GENERIC_WRITE|windows.SYNCHRONIZE,
		&attributes, &status, nil, windows.FILE_ATTRIBUTE_NORMAL,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		windows.FILE_CREATE, options, 0, 0)
	runtime.KeepAlive(security)
	if err != nil {
		return nil, fmt.Errorf("creating protected merge object %s: %w", path, err)
	}

	return os.NewFile(uintptr(handle), path), nil
}
