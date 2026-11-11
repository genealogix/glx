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
	"crypto/md5"  //nolint:gosec // only its digest size is read
	"crypto/sha1" //nolint:gosec // only its digest size is read
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// validateAdvisories adds warnings for values that are valid but probably not
// what the author meant. None of them fails validation.
func (glx *GLXFile) validateAdvisories(result *ValidationResult) {
	glx.validateCensusSourceTypes(result)
	glx.validateMediaHashes(result)
}

// validateCensusSourceTypes warns when a source's title mentions a census but
// its type is not census (#497). `glx census add` finds an existing source by
// title and type census, so a census source typed anything else is not found
// and a second source is created beside it.
func (glx *GLXFile) validateCensusSourceTypes(result *ValidationResult) {
	for id, source := range glx.Sources {
		if source.Type == SourceTypeCensus || !strings.Contains(strings.ToLower(source.Title), SourceTypeCensus) {
			continue
		}
		typeDesc := "has no type"
		if source.Type != "" {
			typeDesc = fmt.Sprintf("has type %q", source.Type)
		}
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: EntityTypeSources,
			SourceID:   id,
			Field:      "type",
			Message: fmt.Sprintf("%s[%s]: title %q mentions a census but the source %s; "+
				"if it is a census record, set type: census so `glx census add` finds it instead of creating a duplicate",
				EntityTypeSources, id, source.Title, typeDesc),
		})
	}
}

// mediaHashPattern is the `algorithm:hexstring` form the media specification
// gives for `hash`.
var mediaHashPattern = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9-]*):([0-9A-Fa-f]+)$`)

// hexDigitsPerByte is how many hex digits spell one byte of a digest.
const hexDigitsPerByte = 2

// mediaHashHexDigits is the digest length, in hex digits, of the algorithms
// whose length is fixed. Other algorithm names are accepted at any length.
var mediaHashHexDigits = map[string]int{
	"md5":    md5.Size * hexDigitsPerByte,
	"sha1":   sha1.Size * hexDigitsPerByte,
	"sha224": sha256.Size224 * hexDigitsPerByte,
	"sha256": sha256.Size * hexDigitsPerByte,
	"sha384": sha512.Size384 * hexDigitsPerByte,
	"sha512": sha512.Size * hexDigitsPerByte,
}

// validateMediaHashes warns when a media `hash` is not `algorithm:hexstring`,
// or when a well-known algorithm's digest has the wrong length (#510).
func (glx *GLXFile) validateMediaHashes(result *ValidationResult) {
	for id, media := range glx.Media {
		if media.Hash == "" {
			continue
		}
		warn := func(message string) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypeMedia,
				SourceID:   id,
				Field:      "hash",
				Message:    fmt.Sprintf("%s[%s]: hash %q %s", EntityTypeMedia, id, media.Hash, message),
			})
		}
		match := mediaHashPattern.FindStringSubmatch(media.Hash)
		if match == nil {
			warn("should be algorithm:hexstring, for example sha256:<64 hex digits>")

			continue
		}
		algorithm, digest := strings.ToLower(match[1]), match[2]
		if want, known := mediaHashHexDigits[algorithm]; known && len(digest) != want {
			warn(fmt.Sprintf("has %d hex digits; a %s digest has %d", len(digest), algorithm, want))
		}
	}
}

// topLevelFieldNames returns the YAML names of the fields of t (a struct, or a
// pointer to one) other than `properties`: the keys an author may have meant
// to write beside `properties` rather than inside it (#648).
func topLevelFieldNames(t reflect.Type) map[string]bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	names := make(map[string]bool)
	if t.Kind() == reflect.Struct {
		for f := range t.Fields() {
			name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
			if name != "" && name != "-" && name != "properties" {
				names[name] = true
			}
		}
	}

	return names
}
