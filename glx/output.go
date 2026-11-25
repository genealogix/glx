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
	glxlib "github.com/genealogix/glx/go-glx"
)

// printArchiveStatistics prints entity counts from a GLX archive
func printArchiveStatistics(glx *glxlib.GLXFile) {
	termPrintln("\nImport statistics:")
	termPrintf("  Persons:       %d\n", len(glx.Persons))
	termPrintf("  Events:        %d\n", len(glx.Events))
	termPrintf("  Relationships: %d\n", len(glx.Relationships))
	termPrintf("  Places:        %d\n", len(glx.Places))
	termPrintf("  Sources:       %d\n", len(glx.Sources))
	termPrintf("  Citations:     %d\n", len(glx.Citations))
	termPrintf("  Repositories:  %d\n", len(glx.Repositories))
	termPrintf("  Media:         %d\n", len(glx.Media))
	termPrintf("  Assertions:    %d\n", len(glx.Assertions))
}

// printVerboseArchiveStatistics prints entity counts with a header message
func printVerboseArchiveStatistics(glx *glxlib.GLXFile, message string) {
	termPrintln(message)
	termPrintf("  Persons:       %d\n", len(glx.Persons))
	termPrintf("  Events:        %d\n", len(glx.Events))
	termPrintf("  Relationships: %d\n", len(glx.Relationships))
	termPrintf("  Places:        %d\n", len(glx.Places))
	termPrintf("  Sources:       %d\n", len(glx.Sources))
	termPrintf("  Citations:     %d\n", len(glx.Citations))
	termPrintf("  Repositories:  %d\n", len(glx.Repositories))
	termPrintf("  Media:         %d\n", len(glx.Media))
	termPrintf("  Assertions:    %d\n", len(glx.Assertions))
}

// printSuccessSingleFile prints a success message for single-file operations
func printSuccessSingleFile(operation, path string) {
	termPrintf("✓ Successfully %s to %s\n", operation, path)
}

// printSuccessMultiFile prints a success message for multi-file operations
func printSuccessMultiFile(operation, path string) {
	termPrintf("✓ Successfully %s to %s/\n", operation, path)
}
