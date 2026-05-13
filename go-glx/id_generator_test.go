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
	"errors"
	"fmt"
	"testing"
)

func TestEntityIDToFilename(t *testing.T) {
	tests := []struct {
		entityID string
		want     string
	}{
		{"person-john-smith-1850", "person-john-smith-1850.glx"},
		{"Person-John-Smith", "person-john-smith.glx"},
		{"EVENT-001", "event-001.glx"},
		{"a", "a.glx"},
		{"person-001", "person-001.glx"},
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-abcdefghijklmnopqrstuvwxyz0", "abcdefghijklmnopqrstuvwxyz0123456789-abcdefghijklmnopqrstuvwxyz0.glx"}, // 64 chars (spec max)
	}

	for _, tt := range tests {
		t.Run(tt.entityID, func(t *testing.T) {
			got, err := EntityIDToFilename(tt.entityID)
			if err != nil {
				t.Fatalf("EntityIDToFilename(%q) unexpected error: %v", tt.entityID, err)
			}
			if got != tt.want {
				t.Errorf("EntityIDToFilename(%q) = %q, want %q", tt.entityID, got, tt.want)
			}
		})
	}
}

func TestEntityIDToFilename_Unsafe(t *testing.T) {
	unsafeIDs := []string{
		"",
		".",
		"..",
		"../etc/passwd",
		"person/evil",
		"person\\evil",
		"person:evil",
		".hidden",
		// Windows reserved device names
		"CON",
		"con",
		"PRN",
		"AUX",
		"NUL",
		"COM1",
		"com9",
		"LPT1",
		"lpt9",
	}

	for _, id := range unsafeIDs {
		t.Run(fmt.Sprintf("%q", id), func(t *testing.T) {
			_, err := EntityIDToFilename(id)
			if err == nil {
				t.Errorf("EntityIDToFilename(%q) should return error for unsafe ID", id)
			}
			if !errors.Is(err, ErrUnsafeEntityID) {
				t.Errorf("EntityIDToFilename(%q) error should wrap ErrUnsafeEntityID, got: %v", id, err)
			}
		})
	}
}

