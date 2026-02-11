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
	"fmt"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

const defaultShowFirstErrors = 10

// formatValidationError formats a StructuredValidationError with optional truncation.
// If showFirstErrors is 0, all errors are shown.
// If showFirstErrors > 0, only the first N errors are shown with a count of remaining errors.
func formatValidationError(err error, showFirstErrors int) error {
	if err == nil {
		return nil
	}

	// Check if this is a structured validation error
	var structuredErr *glxlib.StructuredValidationError
	if !errors.As(err, &structuredErr) {
		// Not a structured validation error, return as-is
		return err
	}

	totalErrors := len(structuredErr.Errors)
	if totalErrors == 0 {
		return err
	}

	// Determine how many errors to show
	maxErrors := showFirstErrors
	if maxErrors == 0 {
		maxErrors = totalErrors // Show all
	}

	// Build error message
	errorLines := make([]string, 0, maxErrors+2)

	// Add each error message
	for i, validationErr := range structuredErr.Errors {
		if i >= maxErrors {
			break
		}
		errorLines = append(errorLines, "  - "+validationErr.Message)
	}

	// Add truncation message if needed
	if totalErrors > maxErrors {
		errorLines = append(errorLines, fmt.Sprintf("  ... and %d more errors", totalErrors-maxErrors))
	}

	// Build final error message
	header := fmt.Sprintf("validation failed with errors (%d error(s))", totalErrors)
	fullMessage := header
	var fullMessageSb57 strings.Builder
	for _, line := range errorLines {
		fullMessageSb57.WriteString("\n" + line)
	}
	fullMessage += fullMessageSb57.String()

	return fmt.Errorf("%s: %w", fullMessage, ErrValidationWithErrors)
}
