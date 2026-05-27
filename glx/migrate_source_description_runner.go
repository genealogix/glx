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
	"os"

	"gopkg.in/yaml.v3"
)

// migrateSourceDescriptions reports how many Source entities in the archive at
// archivePath carry a legacy top-level `description:` field (the pre-#667
// form). The Source.UnmarshalYAML shim in go-glx already handles the value on
// load — folding it into properties.description, or, when an explicit
// properties.description is already present, dropping the legacy duplicate
// (no-clobber). Either way the schema-invalid top-level key must be removed
// from disk, so this raw re-scan counts every source that still carries it:
// the count produces an accurate report and signals the migrate command to
// re-save the archive in the upgraded on-disk form.
func migrateSourceDescriptions(archivePath string, isDir bool) int {
	if isDir {
		files, err := collectGLXFilesFromDir(archivePath)
		if err != nil {
			return 0
		}
		count := 0
		for _, data := range files {
			count += countLegacySourceDescriptions(data)
		}

		return count
	}

	// archivePath is the user-supplied path to their own archive — the same
	// trust model as the archive loader's own read of this file moments earlier.
	data, err := os.ReadFile(archivePath) // #nosec G304
	if err != nil {
		return 0
	}

	return countLegacySourceDescriptions(data)
}

// countLegacySourceDescriptions parses a single .glx file's raw bytes and
// counts sources that declare a legacy top-level `description:` field —
// regardless of whether an explicit `properties.description` also exists, since
// in both cases the top-level key is schema-invalid and must be removed on
// save. Files without a `sources:` block (e.g. vocabulary files, which use
// `source_properties:`) contribute nothing.
func countLegacySourceDescriptions(data []byte) int {
	var doc struct {
		Sources map[string]struct {
			Description string `yaml:"description"`
		} `yaml:"sources"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return 0
	}
	count := 0
	for _, src := range doc.Sources {
		if src.Description != "" {
			count++
		}
	}

	return count
}
