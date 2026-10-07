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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCommandListsCoverEveryCommand fails when a visible top-level command is
// missing from the hand-written command lists in the root README and the CLI
// reference index, which had fallen behind the CLI (#1339). Subcommands are
// left to each command's own generated page under docs/cli/.
func TestCommandListsCoverEveryCommand(t *testing.T) {
	lists := []string{
		filepath.Join("..", "README.md"),
		filepath.Join("..", "docs", "cli", "index.md"),
	}

	for _, path := range lists {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		text := string(data)

		for _, cmd := range rootCmd.Commands() {
			if cmd.Hidden || !cmd.IsAvailableCommand() {
				continue
			}
			if want := "`glx " + cmd.Name() + "`"; !strings.Contains(text, want) {
				t.Errorf("%s does not list %s; add it to the command list", path, want)
			}
		}
	}
}
