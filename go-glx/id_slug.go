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

const slugFallback = "unknown"

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

// SlugOption configures Slugify. Options are applied in order and the last
// call wins per field, so callers can compose defaults with overrides.
type SlugOption func(*slugConfig)

type slugConfig struct {
	prefix    string
	suffix    string
	maxLength int
	fallback  string
}

// WithSlugPrefix prepends prefix verbatim to the returned slug — it is NOT
// itself slugified, so the caller is responsible for passing a slug-safe
// value. The intended use is a constant entity-ID prefix such as
// [EntityIDPrefixPerson] ("person-"); passing a value containing spaces,
// uppercase letters, or non-ASCII characters will leak those into the output
// and violate the [a-z0-9-] charset Slugify otherwise guarantees. The prefix
// counts against any length budget set by [WithSlugMaxLength].
func WithSlugPrefix(prefix string) SlugOption {
	return func(c *slugConfig) { c.prefix = prefix }
}

// WithSlugSuffix appends suffix verbatim to the returned slug — it is NOT
// itself slugified, so it can carry a hash or numeric disambiguator (e.g.
// "-abc12345" or "-2"). For the same reason, the caller is responsible for
// passing a slug-safe value; passing a suffix containing spaces, uppercase
// letters, or non-ASCII characters will leak those into the output and
// violate the [a-z0-9-] charset Slugify otherwise guarantees. The suffix
// counts against any length budget set by [WithSlugMaxLength].
func WithSlugSuffix(suffix string) SlugOption {
	return func(c *slugConfig) { c.suffix = suffix }
}

// WithSlugMaxLength caps the returned string at n characters by trimming the
// slug body; the prefix and suffix are always preserved verbatim (the suffix
// often carries a hash or disambiguator that must stay intact). When
// len(prefix)+len(suffix) <= n the body is trimmed so prefix+body+suffix fits
// within n. When len(prefix)+len(suffix) > n the body is dropped entirely and
// the result is prefix+suffix, which can exceed n. 0 (the default) leaves the
// slug uncapped.
func WithSlugMaxLength(n int) SlugOption {
	return func(c *slugConfig) { c.maxLength = n }
}

// WithSlugFallback replaces the default "unknown" placeholder used when the
// input has no alphanumeric characters. Useful for deterministic hash-based
// fallbacks (e.g., shortHash of a related field). The fallback is itself run
// through the slug pipeline, so Slugify's output stays within [a-z0-9-]
// regardless of the fallback's contents; a fallback that slugifies to empty
// yields "unknown".
func WithSlugFallback(fallback string) SlugOption {
	return func(c *slugConfig) { c.fallback = fallback }
}

// Slugify normalizes s into a URL/ID-safe identifier:
//
//   - composes to NFC so decomposed input (base letter + combining diaeresis,
//     etc.) reaches the digraph layer in precomposed form
//   - applies German digraph transliteration (ä→ae, ß→ss, …) before NFKD
//     normalization so umlauts produce the canonical German digraph form
//     rather than bare ASCII
//   - normalizes via NFKD and strips combining marks (Mn/Mc/Me)
//   - lowercases and collapses non-alphanumeric runs into single hyphens
//   - trims leading and trailing hyphens
//   - falls back to "unknown" (or [WithSlugFallback]) when no alphanumerics
//     survive
//
// Options apply in order; last write wins per field.
func Slugify(s string, opts ...SlugOption) string {
	cfg := slugConfig{fallback: slugFallback}
	for _, opt := range opts {
		opt(&cfg)
	}

	body := slugifyBody(s, cfg.fallback)
	if cfg.maxLength > 0 {
		body = capSlugBody(body, cfg.maxLength-len(cfg.prefix)-len(cfg.suffix))
	}

	return cfg.prefix + body + cfg.suffix
}

// EntityID is an ergonomic wrapper for the standard case of minting a GLX
// entity ID: it slugifies text and prepends prefix, capping the result at
// [MaxEntityIDLength]. Entity-type prefixes are far shorter than
// [MaxEntityIDLength], so the cap always holds in practice; see
// [WithSlugMaxLength] for the over-budget edge case.
func EntityID(prefix, text string) string {
	return Slugify(text, WithSlugPrefix(prefix), WithSlugMaxLength(MaxEntityIDLength))
}

func slugifyBody(s, fallback string) string {
	if body := slugifyOnce(s); body != "" {
		return body
	}
	// The input produced no slug-safe content. Normalize the fallback through
	// the same pipeline so Slugify's output always stays within [a-z0-9-]; if
	// the fallback is itself empty after slugifying, use the guaranteed-safe
	// package default.
	if fb := slugifyOnce(fallback); fb != "" {
		return fb
	}

	return slugFallback
}

// slugifyOnce runs the slug normalization pipeline — NFC normalization,
// German digraph transliteration, NFKD, combining-mark stripping,
// lowercasing, collapsing non-alphanumeric runs to hyphens, and trimming
// hyphens — returning "" when nothing survives.
func slugifyOnce(s string) string {
	// Compose to NFC first so decomposed input (e.g. "u" + U+0308 combining
	// diaeresis, as produced by some macOS/filesystem sources) matches the
	// precomposed umlauts in germanSlugReplacer and expands to the conventional
	// digraph (ue) — otherwise the digraph layer would miss the umlaut and the
	// subsequent NFKD + mark-strip would silently reduce it to a bare vowel (u).
	s = norm.NFC.String(s)
	s = germanSlugReplacer.Replace(s)
	s = stripCombiningMarks(norm.NFKD.String(s))
	s = strings.ToLower(s)
	s = slugNonAlphaNum.ReplaceAllString(s, "-")

	return strings.Trim(s, "-")
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

// capSlugBody trims body to fit budget characters, dropping trailing hyphens.
// budget <= 0 means prefix + suffix already meet or exceed the cap, so there
// is no room for any body characters.
func capSlugBody(body string, budget int) string {
	if budget <= 0 {
		return ""
	}
	if len(body) <= budget {
		return body
	}

	return strings.TrimRight(body[:budget], "-")
}
