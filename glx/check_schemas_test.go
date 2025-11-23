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

// Package main provides the GLX command-line tool for GENEALOGIX archives.
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCheckSchemas(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() string
		wantError bool
	}{
		{
			name: "valid schemas",
			setup: func() string {
				tmpDir := t.TempDir()
				schemaDir := filepath.Join(tmpDir, "schema")
				err := os.MkdirAll(schemaDir, 0o755)
				require.NoError(t, err)

				// Create a valid schema file
				schemaFile := filepath.Join(schemaDir, "test.schema.json")
				schemaContent := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "test-schema",
  "type": "object"
}`
				err = os.WriteFile(schemaFile, []byte(schemaContent), 0o644)
				require.NoError(t, err)

				return tmpDir
			},
			wantError: false,
		},
		{
			name: "missing $schema",
			setup: func() string {
				tmpDir := t.TempDir()
				schemaDir := filepath.Join(tmpDir, "schema")
				err := os.MkdirAll(schemaDir, 0o755)
				require.NoError(t, err)

				// Create a schema file without $schema
				schemaFile := filepath.Join(schemaDir, "test.schema.json")
				schemaContent := `{
  "$id": "test-schema",
  "type": "object"
}`
				err = os.WriteFile(schemaFile, []byte(schemaContent), 0o644)
				require.NoError(t, err)

				return tmpDir
			},
			wantError: true,
		},
		{
			name: "missing $id",
			setup: func() string {
				tmpDir := t.TempDir()
				schemaDir := filepath.Join(tmpDir, "schema")
				err := os.MkdirAll(schemaDir, 0o755)
				require.NoError(t, err)

				// Create a schema file without $id
				schemaFile := filepath.Join(schemaDir, "test.schema.json")
				schemaContent := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object"
}`
				err = os.WriteFile(schemaFile, []byte(schemaContent), 0o644)
				require.NoError(t, err)

				return tmpDir
			},
			wantError: true,
		},
		{
			name: "non-json files ignored",
			setup: func() string {
				tmpDir := t.TempDir()
				schemaDir := filepath.Join(tmpDir, "schema")
				err := os.MkdirAll(schemaDir, 0o755)
				require.NoError(t, err)

				// Create a non-json file
				otherFile := filepath.Join(schemaDir, "test.txt")
				err = os.WriteFile(otherFile, []byte("not a schema"), 0o644)
				require.NoError(t, err)

				return tmpDir
			},
			wantError: false,
		},
		{
			name: "no schema directory",
			setup: func() string {
				tmpDir := t.TempDir()
				// Don't create schema directory - filepath.Walk will return an error
				// but we expect the function to handle it gracefully
				return tmpDir
			},
			wantError: true, // filepath.Walk returns error for non-existent directory
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalDir, err := os.Getwd()
			require.NoError(t, err)

			testDir := tt.setup()
			err = os.Chdir(testDir)
			require.NoError(t, err)
			defer func() { _ = os.Chdir(originalDir) }()

			err = checkSchemaFiles()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
