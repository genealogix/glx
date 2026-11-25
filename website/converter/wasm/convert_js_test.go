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
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// convertBytes runs convert into a buffer, returning no bytes on error, as
// the worker discards partial output.
func convertBytes(data []byte, destination string, prepare func()) ([]byte, summary, error) {
	var output bytes.Buffer
	report, err := convert(data, destination, prepare, &output)
	if err != nil {
		return nil, report, err
	}

	return output.Bytes(), report, nil
}

// checkCompatibilityFields re-checks only Source and Media, the two structs
// whose UnmarshalYAML decodes through Node.Decode and so ignores KnownFields.
// Any other struct in the archive that gains an UnmarshalYAML would silently
// accept misspelled fields on export until it is added there.
func TestCompatibilityCheckCoversEveryStructUnmarshaller(t *testing.T) {
	unmarshaler := reflect.TypeFor[yaml.Unmarshaler]()
	covered := map[reflect.Type]bool{reflect.TypeFor[glxlib.Source](): true, reflect.TypeFor[glxlib.Media](): true}
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type)
	walk = func(kind reflect.Type) {
		if seen[kind] {
			return
		}
		seen[kind] = true
		switch kind.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			walk(kind.Elem())
		case reflect.Struct:
			if reflect.PointerTo(kind).Implements(unmarshaler) && !covered[kind] {
				t.Errorf("%s has an UnmarshalYAML; add it to checkCompatibilityFields", kind)
			}
			for field := range kind.Fields() {
				if field.IsExported() {
					walk(field.Type)
				}
			}
		default:
		}
	}
	walk(reflect.TypeFor[glxlib.GLXFile]())
}

func TestHasBLOBTag(t *testing.T) {
	for input, want := range map[string]bool{
		"0 HEAD\n0 @M1@ OBJE\n1 BLOB\n2 CONT x\n": true,
		"0 HEAD\r\n  1   BLOB\r\n":                true,
		"0 HEAD\n1 BLOB":                          true,
		"0 HEAD\n1 NOTE 1 BLOB\n":                 false,
		"0 HEAD\n2 CONT 1 BLOB here\n":            false,
		"0 HEAD\n1 BLOBX\n":                       false,
		"0 HEAD\n1BLOB\n":                         false,
		"BLOB\n":                                  false,
	} {
		if got := hasBLOBTag([]byte(input)); got != want {
			t.Errorf("hasBLOBTag(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestImportStreamsTabLedTextThatReadsBack(t *testing.T) {
	input := []byte("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @S1@ SOUR\n1 TITL Register\n1 TEXT \tIndented extract\n2 CONT Next line\n0 TRLR\n")
	glx, _, err := convertBytes(input, singleFileGLX, func() {})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := convertBytes(glx, "7.0", func() {}); err != nil {
		t.Fatalf("the downloaded GLX did not export: %v\n%s", err, glx)
	}
}

func TestExportRejectsLossyStructuralInput(t *testing.T) {
	inputs := []string{
		"persons: {}\npersnos:\n  p1: {}\n",
		"persons:\n  p1:\n    name: Jane\n",
		"persons: {}\n---\npersons:\n  p2: {}\n",
		"persons: {}\n---\n[broken",
		"null\n",
		"sources:\n  s1:\n    titel: Lost\n",
		"media:\n  m1:\n    uri: image.jpg\n    titel: Lost\n",
	}
	for _, version := range []string{"5.5.1", "7.0"} {
		for _, input := range inputs {
			t.Run(version+"/"+input, func(t *testing.T) {
				output, _, err := convertBytes([]byte(input), version, func() {})
				if err == nil || len(output) != 0 {
					t.Fatalf("got output=%q err=%v", output, err)
				}
			})
		}
	}
}

func TestExportAllowsEmptyArchiveAndExtensibleProperties(t *testing.T) {
	for _, input := range []string{"{}\n", "persons:\n  person-jane:\n    properties:\n      custom_property: retained-for-other-tools\n      name:\n        value: Jane\n"} {
		for _, version := range []string{"5.5.1", "7.0"} {
			output, _, err := convertBytes([]byte(input), version, func() {})
			if err != nil || !strings.Contains(string(output), "0 TRLR") {
				t.Fatalf("output=%q err=%v", output, err)
			}
		}
	}
}

func TestImportRejectsEmbeddedMediaBeforeProducingDownload(t *testing.T) {
	input := []byte("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n0 @M1@ OBJE\n1 TITL Flower\n1 FORM PICT\n1 BLOB\n2 CONT .HM.......k.1..F\n0 TRLR\n")
	for _, destination := range []string{singleFileGLX, multiFileGLX} {
		prepared := false
		output, _, err := convertBytes(input, destination, func() { prepared = true })
		if !errors.Is(err, errEmbeddedMedia) || len(output) != 0 || prepared {
			t.Fatalf("destination=%s output=%d prepared=%v err=%v", destination, len(output), prepared, err)
		}
	}
}

func TestExportAllowsLegacyDescriptions(t *testing.T) {
	input := []byte("sources:\n  s1:\n    title: Book\n    description: Legacy source\nmedia:\n  m1:\n    uri: image.jpg\n    description: Legacy media\n")
	for _, version := range []string{"5.5.1", "7.0"} {
		output, report, err := convertBytes(input, version, func() {})
		if err != nil || report.Version != version || !strings.Contains(string(output), "Legacy source") {
			t.Fatalf("output=%q err=%v", output, err)
		}
	}
}

func TestExportAllowsCrossSectionAnchors(t *testing.T) {
	input := []byte("persons:\n  p1:\n    properties: &props\n      description: Shared\nsources:\n  s1:\n    title: Book\n    properties: *props\nmedia:\n  m1:\n    uri: image.jpg\n    properties: *props\n")
	for _, version := range []string{"5.5.1", "7.0"} {
		output, report, err := convertBytes(input, version, func() {})
		if err != nil || report.Version != version || !strings.Contains(string(output), "Shared") {
			t.Fatalf("output=%q err=%v", output, err)
		}
	}
}
