package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
				err := os.MkdirAll(schemaDir, 0755)
				require.NoError(t, err)

				// Create a valid schema file
				schemaFile := filepath.Join(schemaDir, "test.schema.json")
				schemaContent := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "test-schema",
  "type": "object"
}`
				err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
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
				err := os.MkdirAll(schemaDir, 0755)
				require.NoError(t, err)

				// Create a schema file without $schema
				schemaFile := filepath.Join(schemaDir, "test.schema.json")
				schemaContent := `{
  "$id": "test-schema",
  "type": "object"
}`
				err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
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
				err := os.MkdirAll(schemaDir, 0755)
				require.NoError(t, err)

				// Create a schema file without $id
				schemaFile := filepath.Join(schemaDir, "test.schema.json")
				schemaContent := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object"
}`
				err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
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
				err := os.MkdirAll(schemaDir, 0755)
				require.NoError(t, err)

				// Create a non-json file
				otherFile := filepath.Join(schemaDir, "test.txt")
				err = os.WriteFile(otherFile, []byte("not a schema"), 0644)
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
			defer os.Chdir(originalDir)

			err = runCheckSchemas()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
