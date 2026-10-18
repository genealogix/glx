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
	"github.com/stretchr/testify/require"
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

func TestFilenameAllocator_CachedSuffixSkipsNaturallyReservedNames(t *testing.T) {
	a := NewFilenameAllocator("Photo.JPG", "photo-2.jpg", "PHOTO-4.JPG")
	assert.Equal(t, "photo-3.jpg", a.Allocate("photo.jpg"))
	assert.Equal(t, "PHOTO-5.JPG", a.Allocate("PHOTO.JPG"))
	assert.True(t, a.Reserve("photo-6.jpg"))
	assert.Equal(t, "Photo-7.jpg", a.Allocate("Photo.jpg"))
}

func TestFilenameAllocator_WindowsAliasesAndReservedNames(t *testing.T) {
	a := NewFilenameAllocator()
	assert.Equal(t, "photo.jpg", a.Allocate("photo.jpg."))
	assert.Equal(t, "photo-2.jpg", a.Allocate("photo.jpg "))
	assert.Equal(t, "CON-2.jpg", a.Allocate("CON.jpg"))
	assert.Equal(t, "CON-2.foo.jpg", a.Allocate("CON.foo.jpg"))
	assert.Equal(t, "NUL-2", a.Allocate("NUL"))
	assert.Equal(t, "COM¹-2.jpg", a.Allocate("COM¹.jpg"))
	assert.Equal(t, "a-b.jpg", a.Allocate("a:b.jpg"))
	assert.Equal(t, FilenameCollisionKey("photo.jpg"), FilenameCollisionKey("PHOTO.JPG. "))
}

func TestFilenameAllocator_AllocateMatchingCanReuseNumberedCandidate(t *testing.T) {
	a := NewFilenameAllocator("photo.jpg", "photo-2.jpg")
	actual, reused, err := a.AllocateMatching("photo.jpg", func(candidate string) (string, error) {
		if candidate == "photo-2.jpg" {
			return candidate, nil
		}

		return "", nil
	})
	require.NoError(t, err)
	assert.True(t, reused)
	assert.Equal(t, "photo-2.jpg", actual)
	actual, reused, err = a.AllocateMatching("photo.jpg", func(candidate string) (string, error) {
		return candidate, nil
	})
	require.NoError(t, err)
	assert.True(t, reused)
	assert.Equal(t, "photo.jpg", actual)
	actual, reused, err = a.AllocateMatching("photo.jpg", func(candidate string) (string, error) {
		return candidate, nil
	})
	require.NoError(t, err)
	assert.False(t, reused)
	assert.Equal(t, "photo-3.jpg", actual)
}

func BenchmarkFilenameAllocator_RepeatedBasename(b *testing.B) {
	a := NewFilenameAllocator()
	b.ResetTimer()
	for range b.N {
		a.Allocate("photo.jpg")
	}
}
