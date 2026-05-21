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
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	slugNonAlphaNum    = regexp.MustCompile(`[^a-z0-9]+`)
	germanSlugReplacer = strings.NewReplacer(
		"ä", "ae",
		"ö", "oe",
		"ü", "ue",
		"Ä", "Ae",
		"Ö", "Oe",
		"Ü", "Ue",
		"ß", "ss",
		"ẞ", "ss",
	)
)

// SlugifyForID lowercases the input, replaces runs of non-alphanumerics with a
// single hyphen, trims leading/trailing hyphens, and truncates to maxLen.
// Falls back to "unknown" if the input contains no alphanumerics.
func SlugifyForID(s string, maxLen int) string {
	s = germanSlugReplacer.Replace(s)
	s = stripCombiningMarks(norm.NFKD.String(s))
	s = strings.ToLower(s)
	s = slugNonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "unknown"
	}

	return trimToMaxLen(s, maxLen)
}

func stripCombiningMarks(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.Is(unicode.M, r) {
			continue
		}
		b.WriteRune(r)
	}

	return b.String()
}

func trimToMaxLen(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}

	return strings.TrimRight(s[:maxLen], "-")
}
