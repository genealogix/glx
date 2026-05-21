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
// form). The Source.UnmarshalYAML shim in go-glx already folds that value into
// properties.description when the archive is loaded, so the in-memory archive
// is already correct; this raw re-scan exists to produce an accurate migration
// report and to signal the migrate command to persist the upgraded on-disk
// form. A source whose properties.description is already set is not counted —
// the explicit property wins and the legacy field is ignored, matching the
// shim's no-clobber behavior.
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
// counts sources that declare a top-level `description:` without an explicit
// `properties.description`. Files without a `sources:` block (e.g. vocabulary
// files, which use `source_properties:`) contribute nothing.
func countLegacySourceDescriptions(data []byte) int {
	var doc struct {
		Sources map[string]struct {
			Description string         `yaml:"description"`
			Properties  map[string]any `yaml:"properties"`
		} `yaml:"sources"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return 0
	}
	count := 0
	for _, src := range doc.Sources {
		if src.Description == "" {
			continue
		}
		if _, ok := src.Properties["description"]; ok {
			continue
		}
		count++
	}

	return count
}
