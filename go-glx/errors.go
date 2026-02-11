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
	"errors"
)

// GEDCOM import errors
var (
	ErrSourceNotFound         = errors.New("source not found")
	ErrUnexpectedRecordType   = errors.New("unexpected record type")
	ErrUnknownEventType       = errors.New("unknown event type")
	ErrInvalidGEDCOMLine      = errors.New("invalid GEDCOM line: too few parts")
	ErrInvalidLevel           = errors.New("invalid level")
	ErrMissingTagAfterXRef    = errors.New("invalid GEDCOM line: missing tag after xref")
	ErrUnexpectedMediaRecord  = errors.New("unexpected media record type")
	ErrRepositoryNotFound     = errors.New("repository not found")
	ErrInvalidPlaceHierarchy  = errors.New("invalid place hierarchy")
	ErrCircularPlaceReference = errors.New("circular place reference detected")
	ErrPersonNotFound         = errors.New("person not found")
)

// Serialization errors
var (
	ErrValidationFailed       = errors.New("validation failed")
	ErrNoEntitiesProvided     = errors.New("no entities provided")
	ErrInvalidEntityType      = errors.New("invalid entity type")
	ErrFileWriteFailed        = errors.New("failed to write file")
	ErrUnsupportedEntityType  = errors.New("unsupported entity type")
	ErrUniqueFilenameFailed   = errors.New("failed to generate unique filename")
	ErrVocabularyNotFound     = errors.New("vocabulary not found")
	ErrUnexpectedNoteRecord   = errors.New("unexpected note record type")
	ErrUnexpectedSharedRecord = errors.New("unexpected shared note record type")
	ErrUnexpectedSchemaRecord = errors.New("unexpected schema record type")
	ErrUnexpectedSourceRecord = errors.New("unexpected source record type")
	ErrUnexpectedRepoRecord   = errors.New("unexpected repository record type")
	ErrGLXFileNil             = errors.New("GLX file is nil")
	ErrValidationHasErrors    = errors.New("validation failed with errors")
)

// StructuredValidationError wraps a list of ValidationErrors for structured error handling.
// This allows the CLI layer to format errors according to user preferences.
type StructuredValidationError struct {
	// Errors contains all validation errors from ValidationResult
	Errors []ValidationError
}

// Error implements the error interface.
func (e *StructuredValidationError) Error() string {
	return ErrValidationHasErrors.Error()
}

// Unwrap allows errors.Is to work with StructuredValidationError.
func (e *StructuredValidationError) Unwrap() error {
	return ErrValidationHasErrors
}
