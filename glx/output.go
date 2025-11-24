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
	"fmt"

	"github.com/genealogix/glx/glx/lib"
)

// printArchiveStatistics prints entity counts from a GLX archive
func printArchiveStatistics(glx *lib.GLXFile) {
	fmt.Println("\nImport statistics:")
	fmt.Printf("  Persons:       %d\n", len(glx.Persons))
	fmt.Printf("  Events:        %d\n", len(glx.Events))
	fmt.Printf("  Relationships: %d\n", len(glx.Relationships))
	fmt.Printf("  Places:        %d\n", len(glx.Places))
	fmt.Printf("  Sources:       %d\n", len(glx.Sources))
	fmt.Printf("  Citations:     %d\n", len(glx.Citations))
	fmt.Printf("  Repositories:  %d\n", len(glx.Repositories))
	fmt.Printf("  Media:         %d\n", len(glx.Media))
	fmt.Printf("  Assertions:    %d\n", len(glx.Assertions))
}

// printVerboseArchiveStatistics prints entity counts with a header message
func printVerboseArchiveStatistics(glx *lib.GLXFile, message string) {
	fmt.Println(message)
	fmt.Printf("  Persons:       %d\n", len(glx.Persons))
	fmt.Printf("  Events:        %d\n", len(glx.Events))
	fmt.Printf("  Relationships: %d\n", len(glx.Relationships))
	fmt.Printf("  Places:        %d\n", len(glx.Places))
	fmt.Printf("  Sources:       %d\n", len(glx.Sources))
	fmt.Printf("  Citations:     %d\n", len(glx.Citations))
	fmt.Printf("  Repositories:  %d\n", len(glx.Repositories))
	fmt.Printf("  Media:         %d\n", len(glx.Media))
	fmt.Printf("  Assertions:    %d\n", len(glx.Assertions))
}

// printSuccessSingleFile prints a success message for single-file operations
func printSuccessSingleFile(operation, path string) {
	fmt.Printf("✓ Successfully %s to %s\n", operation, path)
}

// printSuccessMultiFile prints a success message for multi-file operations
func printSuccessMultiFile(operation, path string) {
	fmt.Printf("✓ Successfully %s to %s/\n", operation, path)
}
