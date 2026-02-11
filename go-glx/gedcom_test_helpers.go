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
	"fmt"
	"io"
	"os"
)

// importGEDCOMFromFile is a test helper that imports a GEDCOM file from a file path.
// This function is only used in tests and handles file I/O for convenience.
// Production code should use ImportGEDCOM with an io.Reader instead.
func importGEDCOMFromFile(filepath, logPath string) (*GLXFile, *ImportResult, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Create log writer if logPath is specified
	var logWriter io.Writer
	if logPath != "" {
		logFile, err := os.Create(logPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create log file: %w", err)
		}
		defer func() { _ = logFile.Close() }()
		logWriter = logFile
	}

	return ImportGEDCOM(file, logWriter)
}
