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
	"fmt"
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

func TestFilenameAllocator_NextAvailableDoesNotReserve(t *testing.T) {
	a := NewFilenameAllocator("photo.jpg", "photo-2.jpg")
	name, ordinal := a.NextAvailable("photo.jpg")
	assert.Equal(t, "photo-3.jpg", name)
	assert.Equal(t, 3, ordinal)
	name, ordinal = a.NextAvailable("photo.jpg")
	assert.Equal(t, "photo-3.jpg", name)
	assert.Equal(t, 3, ordinal)
	assert.Equal(t, "photo-3.jpg", a.Allocate("photo.jpg"))
	name, ordinal = a.NextAvailable("photo.jpg")
	assert.Equal(t, "photo-4.jpg", name)
	assert.Equal(t, 4, ordinal)
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

func TestFilenameAllocator_AllocateMatchingSkipsGeneratedPrefixWithSparseHighSeed(t *testing.T) {
	a := NewFilenameAllocator("photo.jpg", "photo-1000000000.jpg")
	matchCalls := 0
	for n := 2; n < 1002; n++ {
		actual, reused, err := a.AllocateMatching("photo.jpg", func(string) (string, error) {
			matchCalls++

			return "", nil
		})
		require.NoError(t, err)
		assert.False(t, reused)
		assert.Equal(t, fmt.Sprintf("photo-%d.jpg", n), actual)
	}
	assert.LessOrEqual(t, matchCalls, 1000, "generated candidates should not be rescanned")
}

func TestFilenameAllocator_AllocateMatchingStillChecksSeedWithinGeneratedPrefix(t *testing.T) {
	a := NewFilenameAllocator("photo.jpg", "photo-3.jpg")
	noMatch := func(string) (string, error) { return "", nil }
	first, reused, err := a.AllocateMatching("photo.jpg", noMatch)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.Equal(t, "photo-2.jpg", first)
	second, reused, err := a.AllocateMatching("photo.jpg", noMatch)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.Equal(t, "photo-4.jpg", second)

	third, reused, err := a.AllocateMatching("photo.jpg", func(candidate string) (string, error) {
		if candidate == "photo-3.jpg" {
			return candidate, nil
		}

		return "", nil
	})
	require.NoError(t, err)
	assert.True(t, reused)
	assert.Equal(t, "photo-3.jpg", third)
}

func TestFilenameAllocator_AllocateMatchingChecksTrailingDotSeed(t *testing.T) {
	a := NewFilenameAllocator("photo.jpg", "photo-2.jpg.")
	noMatch := func(string) (string, error) { return "", nil }
	first, reused, err := a.AllocateMatching("photo.jpg", noMatch)
	require.NoError(t, err)
	assert.False(t, reused)
	assert.Equal(t, "photo-3.jpg", first)

	second, reused, err := a.AllocateMatching("photo.jpg", func(candidate string) (string, error) {
		if candidate == "photo-2.jpg" {
			return "photo-2.jpg.", nil
		}

		return "", nil
	})
	require.NoError(t, err)
	assert.True(t, reused)
	assert.Equal(t, "photo-2.jpg.", second)
}

func TestFilenameAllocator_AllocateMatchingSkipsFullyClaimedReusePrefix(t *testing.T) {
	const count = 1000
	names := make([]string, 0, count)
	names = append(names, "photo.jpg")
	for n := 2; n <= count; n++ {
		names = append(names, fmt.Sprintf("photo-%d.jpg", n))
	}
	a := NewFilenameAllocator(names...)
	matchCalls := 0
	for _, want := range names {
		actual, reused, err := a.AllocateMatching("photo.jpg", func(candidate string) (string, error) {
			matchCalls++

			return candidate, nil
		})
		require.NoError(t, err)
		assert.True(t, reused)
		assert.Equal(t, want, actual)
	}
	assert.Equal(t, count, matchCalls, "claimed destinations should be skipped on later refreshes")
	actual, reused, err := a.AllocateMatching("photo.jpg", func(string) (string, error) {
		matchCalls++

		return "", nil
	})
	require.NoError(t, err)
	assert.False(t, reused)
	assert.Equal(t, "photo-1001.jpg", actual)
	assert.Equal(t, count, matchCalls)
}

func TestFilenameAllocator_AllocateMatchingKeepsUnclaimedCaseVariants(t *testing.T) {
	a := NewFilenameAllocator("Photo.JPG", "photo.jpg")
	for _, want := range []string{"Photo.JPG", "photo.jpg"} {
		actual, reused, err := a.AllocateMatching("photo.jpg", func(string) (string, error) {
			return want, nil
		})
		require.NoError(t, err)
		assert.True(t, reused)
		assert.Equal(t, want, actual)
	}
	actual, reused, err := a.AllocateMatching("photo.jpg", func(string) (string, error) {
		t.Fatal("all case variants were claimed")

		return "", nil
	})
	require.NoError(t, err)
	assert.False(t, reused)
	assert.Equal(t, "photo-2.jpg", actual)
}

func BenchmarkFilenameAllocator_RepeatedBasename(b *testing.B) {
	a := NewFilenameAllocator()
	b.ResetTimer()
	for range b.N {
		a.Allocate("photo.jpg")
	}
}

func TestFilenameAllocator_DenseReuseCachesFreeSuffix(t *testing.T) {
	names := make([]string, 1000)
	names[0] = "photo.jpg"
	for i := 1; i < len(names); i++ {
		names[i] = fmt.Sprintf("photo-%d.jpg", i+1)
	}
	a := NewFilenameAllocator(names...)
	for range names {
		name, ordinal := a.NextAvailable("photo.jpg")
		require.Equal(t, "photo-1001.jpg", name)
		require.Equal(t, 1001, ordinal)
		require.Equal(t, 1001, a.nextSuffix[FilenameCollisionKey("photo.jpg")])
	}
	require.Equal(t, "photo-1001.jpg", a.Allocate("photo.jpg"))
	require.Equal(t, "photo-1002.jpg", a.Allocate("photo.jpg"))
}

func TestPortableFilename_ControlBytesAndZeroDevices(t *testing.T) {
	require.Equal(t, "COM0-2.jpg", PortableFilename("COM0.jpg"))
	require.Equal(t, "LPT0-2.jpg", PortableFilename("LPT0.jpg"))
	require.Equal(t, "a-b.jpg", PortableFilename("a\x01b.jpg"))
}

func TestFilenameAllocator_IndexedClaimCannotBeReused(t *testing.T) {
	a := NewFilenameAllocator("photo.jpg", "photo-2.jpg")
	require.True(t, a.ClaimExisting("photo.jpg", "photo.jpg"))
	require.False(t, a.ClaimExisting("photo.jpg", "photo.jpg"))
	name, reused, err := a.AllocateMatching("photo.jpg", func(candidate string) (string, error) { return candidate, nil })
	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, "photo-2.jpg", name)
	require.Equal(t, "photo-3.jpg", a.Allocate("photo.jpg"))
}

func TestMediaFileSource_ReallocationClearsOnlyGeneratedCanonicalProvenance(t *testing.T) {
	source := MediaFileSource{SourceType: MediaSourceFile, RequestedFilename: "photo.jpg", OriginalFilename: "photo.jpg", RelativePath: "media/files/photo.jpg"}
	media := &Media{}
	source.SetMediaURI(media, "photo-2.jpg")
	require.Equal(t, "photo.jpg", media.Properties[MediaPropertyOriginalFilename])
	source.SetMediaURI(media, "photo.jpg")
	require.NotContains(t, media.Properties, MediaPropertyOriginalFilename)
	media.Properties[MediaPropertyOriginalFilename] = "supplied.jpg"
	source.SetMediaURI(media, "photo-2.jpg")
	source.SetMediaURI(media, "photo.jpg")
	require.Equal(t, "supplied.jpg", media.Properties[MediaPropertyOriginalFilename])
}
