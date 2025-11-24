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
	"strings"
)

// checkSchemaFiles validates that all JSON schema files contain required metadata
func checkSchemaFiles() error {
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
			issues = append(issues, "missing $schema in "+path)
		}
		if !strings.Contains(text, "\"$id\"") {
			issues = append(issues, "missing $id in "+path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(issues) > 0 {
		return fmt.Errorf("%w:\n%s", ErrSchemaValidationFailed, strings.Join(issues, "\n"))
	}

	fmt.Println("All schema files contain $schema and $id")

	return nil
}
