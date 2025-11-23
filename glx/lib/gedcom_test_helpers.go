package lib

import (
	"fmt"
	"os"
)

// importGEDCOMFromFile is a test helper that imports a GEDCOM file from a file path.
// This function is only used in tests and handles file I/O for convenience.
// Production code should use ImportGEDCOM with an io.Reader instead.
func importGEDCOMFromFile(filepath string, logPath string) (*GLXFile, *ImportResult, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return ImportGEDCOM(file, logPath)
}
