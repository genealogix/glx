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

/*
Package glx provides a Go library for working with GLX (Genealogix) archives,
a modern YAML-based format for genealogical data.

# Overview

GLX archives store genealogical data as interconnected entities: persons, events,
relationships, places, sources, citations, repositories, media, and assertions.
Each entity type uses controlled vocabularies for type-safe property definitions.

This package provides:

  - GEDCOM import (5.5.1 and 7.0)
  - Serialization to single-file and multi-file GLX formats
  - Archive validation with reference integrity checking
  - Standard vocabulary management

# Quick Start

Import a GEDCOM file:

	glxFile, result, err := glx.ImportGEDCOM(reader, nil)
	if err != nil {
	    log.Fatal(err)
	}
	fmt.Printf("Imported %d persons\n", len(glxFile.Persons))

Serialize to a single YAML file:

	s := glx.NewSerializer(nil)
	bytes, err := s.SerializeSingleFileBytes(glxFile)

Serialize to a multi-file archive (one file per entity):

	s := glx.NewSerializer(nil)
	files, err := s.SerializeMultiFileToMap(glxFile)
	// files maps relative paths ("persons/person-abc123.glx") to YAML content

Validate an archive:

	result := glxFile.Validate()
	if len(result.Errors) > 0 {
	    for _, e := range result.Errors {
	        fmt.Println(e.Message)
	    }
	}

# Entity Types

The core entity types map to genealogical concepts:

  - [Person] — an individual, with vocabulary-defined properties (name, gender, etc.)
  - [Event] — a life event (birth, death, marriage) linked to participants and places
  - [Relationship] — a connection between people (parent-child, marriage, etc.)
  - [Place] — a geographic location with optional hierarchical parent
  - [Source] — an information source (book, record, website)
  - [Citation] — a specific reference within a source
  - [Repository] — where sources are held (archive, library)
  - [Media] — a photo, document, or other file
  - [Assertion] — a researcher's conclusion backed by evidence

All entities are stored in [GLXFile], keyed by unique string IDs.

# Vocabularies

GLX uses controlled vocabularies to define valid entity types, participant roles,
and property schemas. Standard vocabularies are embedded in the binary and loaded
automatically during GEDCOM import.

Use [StandardVocabularies] to access them programmatically, or
[LoadStandardVocabulariesIntoGLX] to populate a [GLXFile].

# Serialization

[DefaultSerializer] supports two formats:

Single-file: all entities in one YAML document, suitable for small archives
or programmatic exchange. See [DefaultSerializer.SerializeSingleFileBytes].

Multi-file: one YAML file per entity, organized in directories by type.
Designed for version control (git) and human editing.
See [DefaultSerializer.SerializeMultiFileToMap].

Use [NewSerializer] to create a serializer with default or custom
[SerializerOptions].

# GEDCOM Import

[ImportGEDCOM] converts GEDCOM 5.5.1 and 7.0 files into GLX format. The import
handles name parsing, place hierarchy construction, source/citation linking, and
media file tracking. It accumulates errors rather than failing fast, enabling
partial conversion of malformed files.

The returned [ImportResult] contains statistics and a list of [MediaFileSource]
entries describing media files to be copied into the archive. The caller is
responsible for performing the actual file I/O.

# Validation

[GLXFile.Validate] checks reference integrity (all entity cross-references
resolve), vocabulary compliance (types and roles exist in vocabularies), and
structural rules (required fields, participant counts). Results are cached and
invalidated by [GLXFile.InvalidateCache].

# I/O Boundary

This package is a pure library and never performs filesystem I/O. All methods
accept and return in-memory types ([]byte, io.Reader, io.Writer, map[string][]byte).
The calling application is responsible for reading from and writing to disk.
*/
package glx
