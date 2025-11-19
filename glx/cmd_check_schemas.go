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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var checkSchemasCmd = &cobra.Command{
	Use:   "check-schemas",
	Short: "Validate JSON schema files for required metadata",
	Long: `Validate that JSON schema files contain required metadata fields.

Checks all .json files in the schema/ directory to ensure they have:
- $schema field (JSON Schema version)
- $id field (unique identifier)

This command is primarily used for GENEALOGIX specification development
to ensure all schema files are properly formatted.`,
	Example: `  # Check schemas in current directory
  glx check-schemas

  # Run from specification directory
  cd specification/
  glx check-schemas`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckSchemas()
	},
}

func init() {
	rootCmd.AddCommand(checkSchemasCmd)
}

func runCheckSchemas() error {
	var issues []string

	err := filepath.Walk("schema", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		text := string(content)
		if !strings.Contains(text, "\"$schema\"") {
			issues = append(issues, fmt.Sprintf("missing $schema in %s", path))
		}
		if !strings.Contains(text, "\"$id\"") {
			issues = append(issues, fmt.Sprintf("missing $id in %s", path))
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "\n"))
	}

	fmt.Println("All schema files contain $schema and $id")
	return nil
}
