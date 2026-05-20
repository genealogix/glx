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

import "testing"

func TestSlugifyForID(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		maxLen int
		want   string
	}{
		{name: "german umlauts", in: "ausgewählte Kirchenbücher", maxLen: 64, want: "ausgewaehlte-kirchenbuecher"},
		{name: "eszett", in: "Straße", maxLen: 64, want: "strasse"},
		{name: "mixed diacritics", in: "Crème brûlée y piñata", maxLen: 64, want: "creme-brulee-y-pinata"},
		{name: "punctuation", in: "Hello, World!", maxLen: 64, want: "hello-world"},
		{name: "unknown fallback", in: "   ", maxLen: 64, want: "unknown"},
		{name: "trim", in: "A Very Long Title That Exceeds Sixty Characters To Test Trimming Behavior", maxLen: 10, want: "a-very-lon"},
		{name: "no max trim", in: "Hello, World!", maxLen: 0, want: "hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SlugifyForID(tt.in, tt.maxLen); got != tt.want {
				t.Errorf("SlugifyForID(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
			}
		})
	}
}
