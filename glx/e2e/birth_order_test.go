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

package e2e

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Children appear in birth order in descendants and summary, not in ID order,
// with baptisms standing in for missing births (#1504). This is the issue's
// own reproduction, run end to end, once with births and once with baptisms.
func TestBirthOrder_DescendantsAndSummaryAgree(t *testing.T) {
	for _, eventType := range []string{"birth", "baptism"} {
		t.Run(eventType, func(t *testing.T) {
			root := t.TempDir()
			res := runGLX(t, root, "init", "r2")
			require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
			archive := filepath.Join(root, "r2")

			for _, args := range [][]string{
				{"add", "person", "--id", "p-f", "--given", "Hans", "--surname", "Doe", "--sex", "male"},
				{"add", "person", "--id", "p-a", "--given", "Zacharias", "--surname", "Doe", "--sex", "male"},
				{"add", "person", "--id", "p-b", "--given", "Anna", "--surname", "Doe", "--sex", "female"},
				{"add", "person", "--id", "p-c", "--given", "Michel", "--surname", "Doe", "--sex", "male"},
				{"add", "event", "--id", "ev-a", "--type", eventType, "--date", "1711-05-03", "--participant", "p-a:principal"},
				{"add", "event", "--id", "ev-b", "--type", eventType, "--date", "1688-11-04", "--participant", "p-b:principal"},
				{"add", "event", "--id", "ev-c", "--type", eventType, "--date", "1700-10-24", "--participant", "p-c:principal"},
				{"add", "relationship", "--id", "r-c", "--type", "parent_child", "--parent", "p-f", "--child", "p-c"},
				{"add", "relationship", "--id", "r-a", "--type", "parent_child", "--parent", "p-f", "--child", "p-a"},
				{"add", "relationship", "--id", "r-b", "--type", "parent_child", "--parent", "p-f", "--child", "p-b"},
			} {
				res := runGLX(t, archive, args...)
				require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
			}

			desc := runGLX(t, archive, "descendants", "p-f")
			require.Equal(t, 0, desc.exitCode, desc.stdout+desc.stderr)
			anna := strings.Index(desc.stdout, "Anna Doe")
			michel := strings.Index(desc.stdout, "Michel Doe")
			zacharias := strings.Index(desc.stdout, "Zacharias Doe")
			require.True(t, anna >= 0 && michel >= 0 && zacharias >= 0, desc.stdout)
			assert.Less(t, anna, michel, desc.stdout)
			assert.Less(t, michel, zacharias, desc.stdout)

			summary := runGLX(t, archive, "summary", "p-f")
			require.Equal(t, 0, summary.exitCode, summary.stdout+summary.stderr)
			assert.Contains(t, summary.stdout, "three children: Anna, Michel, and Zacharias")

			sibs := runGLX(t, archive, "summary", "p-c")
			require.Equal(t, 0, sibs.exitCode, sibs.stdout+sibs.stderr)
			assert.Contains(t, sibs.stdout, "Anna Doe, Zacharias Doe")
		})
	}
}
