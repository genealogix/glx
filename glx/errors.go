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
)

// Command validation errors
var (
	ErrMediaFileNotFound          = errors.New("file not found")
	ErrEmptyBlobData              = errors.New("empty BLOB data")
	ErrInvalidBlobLength          = errors.New("invalid BLOB data length")
	ErrInvalidBlobChar            = errors.New("invalid BLOB character (must be in range '.' to 'm')")
	ErrValidationWithErrors       = errors.New("validation failed with errors")
	ErrInvalidFormat              = errors.New("invalid format (must be 'single' or 'multi')")
	ErrGEDCOMFileNotFound         = errors.New("GEDCOM file not found")
	ErrTargetNotDirectory         = errors.New("target path exists and is not a directory")
	ErrNonEmptyDirectory          = errors.New("cannot run 'glx init' in a non-empty directory. Please create a new directory for your family archive")
	ErrInputDirectoryNotFound     = errors.New("input directory not found")
	ErrOutputFileExists           = errors.New("output file already exists (please remove it first)")
	ErrInputFileNotFound          = errors.New("input file not found")
	ErrOutputDirectoryExists      = errors.New("output directory already exists (please remove it first)")
	ErrStructuralValidationFailed = errors.New("structural validation failed")
	ErrValidationFailed           = errors.New("validation failed")
	ErrYAMLNotObject              = errors.New("YAML document is not an object")
	ErrPathNotFound               = errors.New("path not found")
	ErrInvalidPath                = errors.New("invalid path")
	ErrPointerNotObject           = errors.New("pointer does not reference an object")
	ErrFileValidationFailed       = errors.New("validation of file failed")
	ErrYAMLParseFailed            = errors.New("failed to parse YAML file")
	ErrMultipleFilesFailed        = errors.New("multiple files failed validation")
	ErrInvalidExportFormat        = errors.New("invalid GEDCOM version format")
	ErrInputNotFound              = errors.New("input path not found")
	ErrStaleBackupForeignFile     = errors.New("stale backup contains non-archive file from a previous failed run; move or inspect it before retrying")
	ErrInvalidARK                 = errors.New("not a FamilySearch ARK URL")
	ErrEmptyARK                   = errors.New("empty ARK URL")
	ErrLinkSourceRequired         = errors.New("exactly one of --source or --create-source is required")
	ErrLinkSourceConflict         = errors.New("--source and --create-source are mutually exclusive")
	ErrLinkSourceNotFound         = errors.New("--source not found in archive")
)
