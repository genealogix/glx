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
	"testing"
)

func TestValidateEntityFieldFormats_EventDate(t *testing.T) {
	glxFile := &GLXFile{
		Events: map[string]*Event{
			"event-valid": {
				Type: "birth",
				Date: "1850-01-15",
				Participants: []Participant{
					{Person: "person-1", Role: "subject"},
				},
			},
			"event-invalid": {
				Type: "birth",
				Date: "not-a-date",
				Participants: []Participant{
					{Person: "person-1", Role: "subject"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateEntityFieldFormats(result)

	warnings := 0
	for _, w := range result.Warnings {
		if w.Field == "date" {
			warnings++
		}
	}
	if warnings != 1 {
		t.Errorf("expected 1 date warning for invalid event date, got %d", warnings)
	}
}

func TestValidateEntityFieldFormats_SourceDate(t *testing.T) {
	glxFile := &GLXFile{
		Sources: map[string]*Source{
			"source-valid": {Title: "Valid", Date: "1900"},
			"source-bad":   {Title: "Bad", Date: "xyz"},
		},
	}

	result := &ValidationResult{}
	glxFile.validateEntityFieldFormats(result)

	warnings := 0
	for _, w := range result.Warnings {
		if w.Field == "date" {
			warnings++
		}
	}
	if warnings != 1 {
		t.Errorf("expected 1 date warning for invalid source date, got %d", warnings)
	}
}

func TestValidateEntityFieldFormats_AssertionDate(t *testing.T) {
	glxFile := &GLXFile{
		Assertions: map[string]*Assertion{
			"assertion-valid": {Date: "1900-01-15"},
			"assertion-range": {Date: "FROM 1900 TO 1910"},
			"assertion-empty": {Date: ""},
			"assertion-bad":   {Date: "not-a-date"},
		},
	}

	result := &ValidationResult{}
	glxFile.validateEntityFieldFormats(result)

	warnings := 0
	for _, w := range result.Warnings {
		if w.Field == "date" {
			warnings++
		}
	}
	if warnings != 1 {
		t.Errorf("expected 1 date warning for invalid assertion date, got %d", warnings)
	}
}

func TestValidateEntityFieldFormats_RepositoryWebsite(t *testing.T) {
	tests := []struct {
		name        string
		website     string
		expectWarns int
	}{
		{"valid https", "https://example.com", 0},
		{"valid http", "http://example.com", 0},
		{"empty", "", 0},
		{"no scheme", "example.com", 1},
		{"ftp scheme", "ftp://example.com", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glxFile := &GLXFile{
				Repositories: map[string]*Repository{
					"repo-1": {Name: "Test", Website: tt.website},
				},
			}

			result := &ValidationResult{}
			glxFile.validateEntityFieldFormats(result)

			warnings := 0
			for _, w := range result.Warnings {
				if w.Field == "website" {
					warnings++
				}
			}
			if warnings != tt.expectWarns {
				t.Errorf("website=%q: expected %d warnings, got %d", tt.website, tt.expectWarns, warnings)
			}
		})
	}
}

func TestValidateEntityFieldFormats_MediaURI(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expectWarns int
	}{
		{"https url", "https://example.com/photo.jpg", 0},
		{"relative path", "media/files/photo.jpg", 0},
		{"empty", "", 0},
		{"whitespace", " leading-space.jpg", 1},
		{"contains newline", "bad\nuri", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glxFile := &GLXFile{
				Media: map[string]*Media{
					"media-1": {URI: tt.uri},
				},
			}

			result := &ValidationResult{}
			glxFile.validateEntityFieldFormats(result)

			warnings := 0
			for _, w := range result.Warnings {
				if w.Field == "uri" {
					warnings++
				}
			}
			if warnings != tt.expectWarns {
				t.Errorf("uri=%q: expected %d warnings, got %d", tt.uri, tt.expectWarns, warnings)
			}
		})
	}
}

func TestValidateEntityFieldFormats_MediaMIMEType(t *testing.T) {
	tests := []struct {
		name        string
		mimeType    string
		expectWarns int
	}{
		{"image/jpeg", "image/jpeg", 0},
		{"application/pdf", "application/pdf", 0},
		{"empty", "", 0},
		{"no slash", "jpeg", 1},
		{"empty type", "/jpeg", 1},
		{"empty subtype", "image/", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glxFile := &GLXFile{
				Media: map[string]*Media{
					"media-1": {URI: "test.jpg", MimeType: tt.mimeType},
				},
			}

			result := &ValidationResult{}
			glxFile.validateEntityFieldFormats(result)

			warnings := 0
			for _, w := range result.Warnings {
				if w.Field == "mime_type" {
					warnings++
				}
			}
			if warnings != tt.expectWarns {
				t.Errorf("mime_type=%q: expected %d warnings, got %d", tt.mimeType, tt.expectWarns, warnings)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://", true},
		{"ftp://example.com", false},
		{"example.com", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isValidURL(tt.input); got != tt.valid {
			t.Errorf("isValidURL(%q) = %v, want %v", tt.input, got, tt.valid)
		}
	}
}

func TestIsValidMediaURI(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"https://example.com/photo.jpg", true},
		{"media/files/photo.jpg", true},
		{"photo.jpg", true},
		{"ftp://example.com/file", true},
		{" leading", false},
		{"has\nnewline", false},
		{"has\ttab", false},
	}
	for _, tt := range tests {
		if got := isValidMediaURI(tt.input); got != tt.valid {
			t.Errorf("isValidMediaURI(%q) = %v, want %v", tt.input, got, tt.valid)
		}
	}
}

func TestIsValidMIMEType(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"image/jpeg", true},
		{"application/pdf", true},
		{"text/plain", true},
		{"jpeg", false},
		{"/jpeg", false},
		{"image/", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isValidMIMEType(tt.input); got != tt.valid {
			t.Errorf("isValidMIMEType(%q) = %v, want %v", tt.input, got, tt.valid)
		}
	}
}
