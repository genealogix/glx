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

package lib

import (
	"fmt"
	"log"
	"os"
)

// ImportLogger handles logging during GEDCOM import
type ImportLogger struct {
	file   *os.File
	logger *log.Logger
}

// NewImportLogger creates a new import logger
func NewImportLogger(logPath string) (*ImportLogger, error) {
	if logPath == "" {
		// No logging if path not specified
		return &ImportLogger{}, nil
	}

	file, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return &ImportLogger{
		file:   file,
		logger: log.New(file, "", log.LstdFlags),
	}, nil
}

// Close closes the log file
func (il *ImportLogger) Close() error {
	if il.file != nil {
		return il.file.Close()
	}

	return nil
}

// LogError logs an error during import
func (il *ImportLogger) LogError(line int, tag string, gedcomXRef string, err error) {
	if il.logger == nil {
		return
	}

	il.logger.Printf("ERROR [Line %d] Tag: %s, XRef: %s - %v", line, tag, gedcomXRef, err)
}

// LogWarning logs a warning during import
func (il *ImportLogger) LogWarning(line int, tag string, gedcomXRef string, message string) {
	if il.logger == nil {
		return
	}

	il.logger.Printf("WARNING [Line %d] Tag: %s, XRef: %s - %s", line, tag, gedcomXRef, message)
}

// LogInfo logs informational messages
func (il *ImportLogger) LogInfo(message string) {
	if il.logger == nil {
		return
	}

	il.logger.Printf("INFO: %s", message)
}

// LogException logs an exception with full context
func (il *ImportLogger) LogException(line int, tag string, gedcomXRef string, operation string, err error, context map[string]any) {
	if il.logger == nil {
		return
	}

	il.logger.Printf("EXCEPTION [Line %d] Tag: %s, XRef: %s", line, tag, gedcomXRef)
	il.logger.Printf("  Operation: %s", operation)
	il.logger.Printf("  Error: %v", err)

	if len(context) > 0 {
		il.logger.Printf("  Context:")
		for key, value := range context {
			il.logger.Printf("    %s: %v", key, value)
		}
	}
}
