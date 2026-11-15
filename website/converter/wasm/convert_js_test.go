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
	"errors"
	"strings"
	"testing"
)

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
				output, _, err := convert([]byte(input), version, func() {})
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
			output, _, err := convert([]byte(input), version, func() {})
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
		output, _, err := convert(input, destination, func() { prepared = true })
		if !errors.Is(err, errEmbeddedMedia) || len(output) != 0 || prepared {
			t.Fatalf("destination=%s output=%d prepared=%v err=%v", destination, len(output), prepared, err)
		}
	}
}

func TestExportAllowsLegacyDescriptions(t *testing.T) {
	input := []byte("sources:\n  s1:\n    title: Book\n    description: Legacy source\nmedia:\n  m1:\n    uri: image.jpg\n    description: Legacy media\n")
	for _, version := range []string{"5.5.1", "7.0"} {
		output, report, err := convert(input, version, func() {})
		if err != nil || report.Version != version || !strings.Contains(string(output), "Legacy source") {
			t.Fatalf("output=%q err=%v", output, err)
		}
	}
}

func TestExportAllowsCrossSectionAnchors(t *testing.T) {
	input := []byte("persons:\n  p1:\n    properties: &props\n      description: Shared\nsources:\n  s1:\n    title: Book\n    properties: *props\nmedia:\n  m1:\n    uri: image.jpg\n    properties: *props\n")
	for _, version := range []string{"5.5.1", "7.0"} {
		output, report, err := convert(input, version, func() {})
		if err != nil || report.Version != version || !strings.Contains(string(output), "Shared") {
			t.Fatalf("output=%q err=%v", output, err)
		}
	}
}
