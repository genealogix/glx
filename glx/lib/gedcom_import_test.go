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

package lib

import (
	"testing"
)

// NOTE: GEDCOM import tests are now in gedcom_comprehensive_test.go
// which tests all 35 GEDCOM test files

// TestParseGEDCOMLine tests the GEDCOM line parser
func TestParseGEDCOMLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantLevel   int
		wantXRef    string
		wantTag     string
		wantValue   string
		expectError bool
	}{
		{
			name:      "level 0 with xref",
			line:      "0 @I1@ INDI",
			wantLevel: 0,
			wantXRef:  "@I1@",
			wantTag:   "INDI",
			wantValue: "",
		},
		{
			name:      "level 1 with value",
			line:      "1 NAME John /Smith/",
			wantLevel: 1,
			wantXRef:  "",
			wantTag:   "NAME",
			wantValue: "John /Smith/",
		},
		{
			name:      "level 2 with value",
			line:      "2 GIVN John",
			wantLevel: 2,
			wantXRef:  "",
			wantTag:   "GIVN",
			wantValue: "John",
		},
		{
			name:      "level 0 header",
			line:      "0 HEAD",
			wantLevel: 0,
			wantXRef:  "",
			wantTag:   "HEAD",
			wantValue: "",
		},
		{
			name:        "invalid - no level",
			line:        "INVALID",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGEDCOMLine(tt.line, 1)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if got.Level != tt.wantLevel {
				t.Errorf("Level = %d, want %d", got.Level, tt.wantLevel)
			}
			if got.XRef != tt.wantXRef {
				t.Errorf("XRef = %q, want %q", got.XRef, tt.wantXRef)
			}
			if got.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q", got.Tag, tt.wantTag)
			}
			if got.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", got.Value, tt.wantValue)
			}
		})
	}
}

// TestParseGEDCOMName is now in gedcom_integration_test.go with correct implementation
