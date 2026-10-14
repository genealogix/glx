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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilenameAllocator_CaseAndUnicodeCollisions(t *testing.T) {
	a := NewFilenameAllocator("Photo.JPG", "portrait.jpg", "portrait-2.jpg", "Caf\u00e9.png")

	actual, ok := a.Existing("photo.jpg")
	assert.True(t, ok)
	assert.Equal(t, "Photo.JPG", actual)
	assert.False(t, a.Reserve("PHOTO.jpg"))
	assert.Equal(t, "photo-2.jpg", a.Allocate("photo.jpg"))
	assert.Equal(t, "PORTRAIT-3.jpg", a.Allocate("PORTRAIT.jpg"))
	assert.Equal(t, "Cafe\u0301-2.png", a.Allocate("Cafe\u0301.png"))
	assert.Equal(t, "photo-3.jpg", a.Allocate("photo.jpg"))
}
