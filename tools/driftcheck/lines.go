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

	"github.com/genealogix/glx/tools/internal/structdump"
)

// attachLines enriches findings with the source line of the drifting field in
// the Go types file, using the AST extractor (structdump, #795). The rest of
// the checker compares via reflection, which cannot report source positions;
// the AST can. Best-effort: a finding whose field can't be located keeps
// Line == 0, and a read/parse failure leaves every Line at 0 without failing
// the check (line numbers are a reporting nicety, not a correctness signal).
func attachLines(findings []finding, typesPath string) error {
	src, err := os.ReadFile(typesPath) // #nosec G304 -- repo types file, not user input
	if err != nil {
		return err
	}
	dump, err := structdump.Extract(typesPath, src)
	if err != nil {
		return err
	}

	// (Go type, yaml tag) -> line, and (Go type, Go field name) -> line.
	byTag := map[string]map[string]int{}
	byName := map[string]map[string]int{}
	for typeName, ti := range dump.Types {
		byTag[typeName] = map[string]int{}
		byName[typeName] = map[string]int{}
		for _, f := range ti.Fields {
			if f.YAMLTag != "" {
				byTag[typeName][f.YAMLTag] = f.Line
			}
			byName[typeName][f.Name] = f.Line
		}
	}

	for i := range findings {
		f := &findings[i]
		if line, ok := byTag[f.Entity][f.YamlTag]; ok && line > 0 {
			f.Line = line

			continue
		}
		if line, ok := byName[f.Entity][f.Field]; ok && line > 0 {
			f.Line = line
		}
	}

	return nil
}
