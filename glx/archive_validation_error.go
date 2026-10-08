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
	"strings"
)

// fileValidationIssues keeps schema diagnostics separate until reporting, so
// validate can limit individual issues rather than entire file blocks.
type fileValidationIssues struct {
	path       string
	issues     []string
	parseError bool
}

// archiveFilesValidationError preserves the loader's full error text and
// sentinel for other commands while allowing validate to cap its report.
type archiveFilesValidationError struct {
	files []fileValidationIssues
}

func (e *archiveFilesValidationError) Error() string {
	return e.format(0)
}

func (e *archiveFilesValidationError) Unwrap() error {
	return ErrMultipleFilesFailed
}

func (e *archiveFilesValidationError) format(showFirstErrors int) string {
	total := 0
	for _, file := range e.files {
		total += len(file.issues)
	}
	remaining := total
	if showFirstErrors > 0 {
		remaining = min(remaining, showFirstErrors)
	}
	hidden := total - remaining
	blocks := make([]string, 0, len(e.files))
	for _, file := range e.files {
		if remaining == 0 {
			break
		}
		shown := min(len(file.issues), remaining)
		block := file.path + ":\n  - " + strings.Join(file.issues[:shown], "\n  - ")
		if file.parseError {
			block = file.path + ": " + file.issues[0]
		}
		blocks = append(blocks, block)
		remaining -= shown
	}
	message := ErrMultipleFilesFailed.Error() + ":\n\n" + strings.Join(blocks, "\n\n")
	if hidden > 0 {
		message += fmt.Sprintf("\n  ... and %d more errors (use --show-first-errors 0 to list all)", hidden)
	}

	return message
}
