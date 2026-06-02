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
	"path/filepath"
	"strings"
	"testing"
)

func TestDataURI(t *testing.T) {
	if got := dataURI("image/png", "x.png", []byte("data")); !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("explicit mime: %q", got)
	}
	if got := dataURI("", "photo.png", []byte("data")); !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("mime from extension: %q", got)
	}
	if got := dataURI("", "noext", []byte("data")); !strings.HasPrefix(got, "data:application/octet-stream;base64,") {
		t.Errorf("octet-stream fallback: %q", got)
	}
}

func TestIsImageMedia(t *testing.T) {
	cases := []struct {
		mime, uri string
		want      bool
	}{
		{"image/jpeg", "weird.bin", true}, // explicit mime wins
		{"application/pdf", "x.jpg", false},
		{"", "scan.PNG", true}, // extension, case-insensitive
		{"", "doc.pdf", false},
		{"", "noext", false},
	}
	for _, tc := range cases {
		if got := isImageMedia(tc.mime, tc.uri); got != tc.want {
			t.Errorf("isImageMedia(%q,%q) = %v, want %v", tc.mime, tc.uri, got, tc.want)
		}
	}
}

func TestSourcePath(t *testing.T) {
	r := &mediaResolver{baseDir: filepath.Join("base", "dir")}

	rel := r.sourcePath("sub/photo.jpg")
	if want := filepath.Join("base", "dir", "sub", "photo.jpg"); rel != want {
		t.Errorf("relative sourcePath = %q, want %q", rel, want)
	}

	abs := filepath.Join(t.TempDir(), "photo.jpg")
	if got := r.sourcePath(abs); got != filepath.Clean(abs) {
		t.Errorf("absolute sourcePath = %q, want %q", got, filepath.Clean(abs))
	}
}

func TestPersonFileName(t *testing.T) {
	cases := map[string]string{
		"person-john-smith": "person-john-smith.html",
		"Weird ID!":         "weird-id.html",
		"---":               "person.html", // sanitizes to empty -> fallback
	}
	for in, want := range cases {
		if got := personFileName(in); got != want {
			t.Errorf("personFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOutputName_CollisionSuffix(t *testing.T) {
	r := &mediaResolver{assigned: map[string]string{}, usedNames: map[string]bool{}}

	first := r.outputName("/a/photo.jpg", "photo.jpg")
	second := r.outputName("/b/photo.jpg", "photo.jpg") // same basename, different source
	reuse := r.outputName("/a/photo.jpg", "photo.jpg")  // same source path -> same name

	if first != "photo.jpg" {
		t.Errorf("first = %q, want photo.jpg", first)
	}
	if second != "photo-2.jpg" {
		t.Errorf("second = %q, want photo-2.jpg", second)
	}
	if reuse != first {
		t.Errorf("reuse = %q, want %q", reuse, first)
	}
}

func TestAppendUnique(t *testing.T) {
	out := appendUnique(nil, "a")
	out = appendUnique(out, "a") // duplicate -> ignored
	out = appendUnique(out, "b")
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Errorf("appendUnique result = %v, want [a b]", out)
	}
}
