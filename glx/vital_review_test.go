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
	"testing"

	"github.com/stretchr/testify/assert"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestVitalReview_RoleOverridesUseDeclaredThenNormalizedKeys(t *testing.T) {
	yes, no := true, false
	for _, role := range []string{"Parent", " parent ", " PARENT "} {
		assert.False(t, mentionedRelativeAtEvent("marriage", role, map[string]*glxlib.VocabularyEntry{"parent": {ImpliesPresence: &yes}}), role)
		assert.True(t, mentionedRelativeAtEvent("marriage", role, nil), role)
	}
	assert.True(t, mentionedRelativeAtEvent("marriage", " Parent ", map[string]*glxlib.VocabularyEntry{"Parent": {ImpliesPresence: &no}, "parent": {ImpliesPresence: &yes}}), "a declared key takes precedence over its normalized fallback")
}

func TestVitalReview_BEFDatesSortByTheirDistinctBounds(t *testing.T) {
	assert.Less(t, dateSortKey("BEF 1900"), dateSortKey("BEF 1920"))
	assert.Less(t, dateSortKey("BEF 1900-01"), dateSortKey("BEF 1900-02"))
	assert.Less(t, dateSortKey("BEF 1900-01-01"), dateSortKey("BEF 1900-01-02"))
	assert.Less(t, dateSortKey("BEF 0100 BCE"), dateSortKey("BEF 0050 BCE"))
	assert.Less(t, dateSortKey("JULIAN BEF 1645-10-12"), dateSortKey("1645-10-22"))
}
