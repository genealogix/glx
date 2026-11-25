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

package glx

import (
	"archive/zip"
	"fmt"
	"io"
	"runtime"
)

// WriteMultiFileZIP writes a GLX archive to a ZIP stream using the same layout
// as SerializeMultiFileToMap. The caller supplies the destination and discards
// any partial output on error.
func (s *DefaultSerializer) WriteMultiFileZIP(archive *GLXFile, output io.Writer) error {
	writer := zip.NewWriter(output)
	filesWritten := 0
	if err := s.serializeMultiFile(archive, func(filename string, data []byte) error {
		entry, err := writer.Create(filename)
		if err != nil {
			return fmt.Errorf("create ZIP entry %q: %w", filename, err)
		}
		if _, err := entry.Write(data); err != nil {
			return fmt.Errorf("write ZIP entry %q: %w", filename, err)
		}
		filesWritten++
		// Reclaim temporary YAML buffers during long synchronous WASM calls.
		if runtime.GOARCH == "wasm" && filesWritten%wasmGCInterval == 0 {
			runtime.GC()
		}

		return nil
	}); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish GLX ZIP: %w", err)
	}

	return nil
}
