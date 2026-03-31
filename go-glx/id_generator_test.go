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

package glx

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestGenerateRandomID(t *testing.T) {
	// Generate IDs to test format and basic uniqueness.
	// Using 1,000 iterations keeps P(collision) at ~0.01% in a 32-bit space,
	// making collisions extremely unlikely (but not impossible) in CI.
	ids := make(map[string]bool)
	iterations := 1000
	hexPattern := regexp.MustCompile("^[a-f0-9]{8}$")

	for i := range iterations {
		id, err := GenerateRandomID()
		if err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}

		// Check format: 8-character lowercase hex
		if len(id) != 8 {
			t.Errorf("Expected 8 character ID, got %d: %s", len(id), id)
		}

		// Check hex format
		if !hexPattern.MatchString(id) {
			t.Errorf("ID doesn't match hex pattern: %s", id)
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s (iteration %d)", id, i)
		}
		ids[id] = true
	}

	t.Logf("Generated %d unique IDs successfully", len(ids))
}

func TestGenerateEntityFilename(t *testing.T) {
	tests := []struct {
		entityType string
		wantPrefix string
	}{
		{"person", "person-"},
		{"event", "event-"},
		{"relationship", "relationship-"},
		{"place", "place-"},
		{"source", "source-"},
		{"repository", "repository-"},
		{"media", "media-"},
		{"citation", "citation-"},
		{"assertion", "assertion-"},
	}

	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			filename, err := GenerateEntityFilename(tt.entityType)
			if err != nil {
				t.Fatalf("Failed to generate filename: %v", err)
			}

			// Check prefix
			if !strings.HasPrefix(filename, tt.wantPrefix) {
				t.Errorf("Expected prefix %s, got %s", tt.wantPrefix, filename)
			}

			// Check suffix
			if !strings.HasSuffix(filename, ".glx") {
				t.Errorf("Expected .glx suffix, got %s", filename)
			}

			// Check total format: {entity-type}-{8-hex}.glx
			pattern := fmt.Sprintf("^%s[a-f0-9]{8}\\.glx$", regexp.QuoteMeta(tt.entityType+"-"))
			matched, _ := regexp.MatchString(pattern, filename)
			if !matched {
				t.Errorf("Filename %s doesn't match pattern %s", filename, pattern)
			}

			// Check expected length
			expectedLen := len(tt.entityType) + 1 + 8 + 4 // type + "-" + id + ".glx"
			if len(filename) != expectedLen {
				t.Errorf("Expected length %d, got %d: %s", expectedLen, len(filename), filename)
			}
		})
	}
}

func TestGenerateUniqueFilename(t *testing.T) {
	usedFilenames := make(map[string]bool)

	// Generate 100 unique filenames
	for range 100 {
		filename, err := GenerateUniqueFilename("person", usedFilenames, 10)
		if err != nil {
			t.Fatalf("Failed to generate unique filename: %v", err)
		}

		// Should be marked as used
		if !usedFilenames[filename] {
			t.Errorf("Filename not marked as used: %s", filename)
		}
	}

	// Should have 100 unique filenames
	if len(usedFilenames) != 100 {
		t.Errorf("Expected 100 unique filenames, got %d", len(usedFilenames))
	}
}

func TestGenerateUniqueFilenameCollisionRetry(t *testing.T) {
	usedFilenames := make(map[string]bool)

	// Pre-fill with many filenames to increase collision probability
	// (This is a synthetic test - real collisions are extremely rare)
	for range 1000 {
		filename, _ := GenerateEntityFilename("test")
		usedFilenames[filename] = true
	}

	// Should still be able to generate unique filename with retries
	filename, err := GenerateUniqueFilename("test", usedFilenames, 100)
	if err != nil {
		t.Fatalf("Failed to generate unique filename with retries: %v", err)
	}

	if !usedFilenames[filename] {
		t.Errorf("Generated filename not marked as used: %s", filename)
	}
}

func BenchmarkGenerateRandomID(b *testing.B) {
	for b.Loop() {
		_, err := GenerateRandomID()
		if err != nil {
			b.Fatalf("Failed to generate ID: %v", err)
		}
	}
}

func BenchmarkGenerateEntityFilename(b *testing.B) {
	for b.Loop() {
		_, err := GenerateEntityFilename("person")
		if err != nil {
			b.Fatalf("Failed to generate filename: %v", err)
		}
	}
}

func BenchmarkGenerateUniqueFilename(b *testing.B) {
	usedFilenames := make(map[string]bool)

	for b.Loop() {
		_, err := GenerateUniqueFilename("person", usedFilenames, 10)
		if err != nil {
			b.Fatalf("Failed to generate unique filename: %v", err)
		}
	}
}
