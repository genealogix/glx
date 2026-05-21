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
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		opts []SlugOption
		want string
	}{
		{name: "plain", in: "Hello, World!", want: "hello-world"},
		{name: "german umlauts", in: "ausgewählte Kirchenbücher", want: "ausgewaehlte-kirchenbuecher"},
		{name: "eszett", in: "Straße", want: "strasse"},
		{name: "mixed diacritics", in: "Crème brûlée y piñata", want: "creme-brulee-y-pinata"},
		{name: "unknown default", in: "   ", want: "unknown"},
		{
			name: "with prefix",
			in:   "Mary Smith",
			opts: []SlugOption{WithSlugPrefix("person-")},
			want: "person-mary-smith",
		},
		{
			name: "with suffix verbatim",
			in:   "foo bar",
			opts: []SlugOption{WithSlugSuffix("-abcd1234")},
			want: "foo-bar-abcd1234",
		},
		{
			name: "max length trims body",
			in:   "A Very Long Title That Will Get Trimmed",
			opts: []SlugOption{WithSlugMaxLength(10)},
			want: "a-very-lon",
		},
		{
			name: "prefix counts toward max length",
			in:   "very-long-title-needs-truncation-here",
			opts: []SlugOption{WithSlugPrefix("p-"), WithSlugMaxLength(10)},
			want: "p-very-lon",
		},
		{
			name: "suffix counts toward max length",
			in:   "very long title",
			opts: []SlugOption{WithSlugSuffix("-x"), WithSlugMaxLength(10)},
			want: "very-lon-x",
		},
		{
			name: "prefix and suffix together count toward max length",
			in:   "very long title here",
			opts: []SlugOption{WithSlugPrefix("p-"), WithSlugSuffix("-x"), WithSlugMaxLength(10)},
			want: "p-very-l-x",
		},
		{
			name: "fallback overrides unknown",
			in:   "   ",
			opts: []SlugOption{WithSlugFallback("h4sh")},
			want: "h4sh",
		},
		{
			name: "fallback with prefix",
			in:   "   ",
			opts: []SlugOption{WithSlugPrefix("person-"), WithSlugFallback("abc12345")},
			want: "person-abc12345",
		},
		{
			name: "last option wins (prefix)",
			in:   "x",
			opts: []SlugOption{WithSlugPrefix("a-"), WithSlugPrefix("b-")},
			want: "b-x",
		},
		{
			name: "last option wins (max length)",
			in:   "abcdefghij",
			opts: []SlugOption{WithSlugMaxLength(3), WithSlugMaxLength(5)},
			want: "abcde",
		},
		{
			name: "tight budget yields empty body",
			in:   "title",
			opts: []SlugOption{WithSlugPrefix("citation-"), WithSlugSuffix("-hash"), WithSlugMaxLength(len("citation-") + len("-hash"))},
			want: "citation--hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.in, tt.opts...); got != tt.want {
				t.Errorf("Slugify(%q, ...) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEntityID(t *testing.T) {
	if got := EntityID("person-", "Mary Smith"); got != "person-mary-smith" {
		t.Errorf("EntityID = %q, want %q", got, "person-mary-smith")
	}

	long := strings.Repeat("a", MaxEntityIDLength*2)
	if got := EntityID("person-", long); len(got) > MaxEntityIDLength {
		t.Errorf("EntityID exceeded MaxEntityIDLength: %d > %d", len(got), MaxEntityIDLength)
	}

	if got := EntityID("person-", "   "); got != "person-unknown" {
		t.Errorf("EntityID empty fallback = %q, want %q", got, "person-unknown")
	}
}

func TestStripCombiningMarks_RemovesAllMarkCategories(t *testing.T) {
	const (
		nonSpacingMark = "́" // Mn COMBINING ACUTE ACCENT
		spacingMark    = "ः" // Mc DEVANAGARI SIGN VISARGA
		enclosingMark  = "⃝" // Me COMBINING ENCLOSING CIRCLE
	)

	got := stripCombiningMarks("a" + nonSpacingMark + "b" + spacingMark + "c" + enclosingMark + "d")
	if got != "abcd" {
		t.Fatalf("stripCombiningMarks should remove Mn/Mc/Me marks: got %q, want %q", got, "abcd")
	}
}
