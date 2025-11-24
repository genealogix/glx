package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/genealogix/glx/glx/lib"
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
	var structuredErr *lib.StructuredValidationError
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

	return errors.New(fullMessage)
}
