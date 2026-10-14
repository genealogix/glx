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
	"path"
	"path/filepath"

	glxlib "github.com/genealogix/glx/go-glx"
)

// reserveImportMediaDestinations keeps a new import from claiming names already
// present beside a single-file archive or inside an existing directory archive.
// It runs before serialization so Media URIs and copied filenames stay aligned.
func reserveImportMediaDestinations(glx *glxlib.GLXFile, files []glxlib.MediaFileSource, archiveDir string) error {
	if len(files) == 0 {
		return nil
	}

	existing, err := listMediaFilenames(filepath.Join(archiveDir, glxlib.MediaFilesDir))
	if err != nil {
		return err
	}
	allocator := glxlib.NewFilenameAllocator()
	for name := range existing {
		allocator.Reserve(name)
	}
	for i := range files {
		oldName := files[i].TargetFilename
		newName := allocator.Allocate(oldName)
		if newName == oldName {
			continue
		}
		files[i].TargetFilename = newName
		if media := glx.Media[files[i].MediaID]; media != nil && media.URI == glxlib.MediaFilesDir+"/"+oldName {
			media.URI = glxlib.MediaFilesDir + "/" + newName
			if files[i].SourceType == glxlib.MediaSourceFile {
				if media.Properties == nil {
					media.Properties = make(map[string]any)
				}
				if _, exists := media.Properties[glxlib.MediaPropertyOriginalFilename]; !exists {
					media.Properties[glxlib.MediaPropertyOriginalFilename] = path.Base(files[i].RelativePath)
				}
			}
		}
	}

	return nil
}
