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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateGLXFileStructure_VocabularyEntryRejectsMissingLabel verifies that
// the fragment-style $ref ("vocabularies/X.schema.json#/definitions/Y") in
// glx-file.schema.json resolves correctly through resolveFileRef, so each
// vocabulary entry is validated against the inner type-definition schema (which
// requires `label`) rather than the outer wrapper schema.
func TestValidateGLXFileStructure_VocabularyEntryRejectsMissingLabel(t *testing.T) {
	cases := []struct {
		name   string
		vocab  string
		field  string
		needle string
	}{
		{"event_types", "event_types", "label", "label"},
		{"search_result_types", "search_result_types", "label", "label"},
		{"research_log_status_types", "research_log_status_types", "label", "label"},
		{"person_properties", "person_properties", "label", "label"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := map[string]any{
				tc.vocab: map[string]any{
					"foo": map[string]any{"description": "no label"},
				},
			}
			issues := ValidateGLXFileStructure(doc)
			matched := false
			for _, i := range issues {
				if strings.Contains(i, tc.needle) {
					matched = true

					break
				}
			}
			assert.True(t, matched, "expected validation error mentioning %q, got: %v", tc.needle, issues)
		})
	}
}

// TestValidateGLXFileStructure_VocabularyEntryAcceptsWellFormed verifies the
// positive path: a well-formed vocabulary entry passes structural validation.
func TestValidateGLXFileStructure_VocabularyEntryAcceptsWellFormed(t *testing.T) {
	doc := map[string]any{
		"event_types": map[string]any{
			"birth": map[string]any{"label": "Birth", "category": "lifecycle"},
		},
		"search_result_types": map[string]any{
			"found": map[string]any{"label": "Found"},
		},
		"research_log_status_types": map[string]any{
			"open": map[string]any{"label": "Open"},
		},
	}
	issues := ValidateGLXFileStructure(doc)
	assert.Empty(t, issues, "expected no validation issues, got: %v", issues)
}
