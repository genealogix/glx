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

	vocabularies "github.com/genealogix/glx/specification/5-standard-vocabularies"
)

func createStandardVocabularies() error {
	for filename, content := range vocabularies.Files {
		outputPath := "vocabularies/" + filename
		if err := os.WriteFile(outputPath, content, filePermissions); err != nil {
			return fmt.Errorf("failed to create %s: %w", outputPath, err)
		}
	}

	fmt.Println("Created standard vocabulary files in vocabularies/")

	return nil
}
