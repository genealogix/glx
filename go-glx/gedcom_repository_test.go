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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMapRepositoryType verifies that GEDCOM REPO.TYPE values resolve through
// the vocabulary-driven index, including case-insensitive lookup and the
// fallback to RepositoryTypeOther for unrecognized values.
func TestMapRepositoryType(t *testing.T) {
	glx := &GLXFile{}
	if err := LoadStandardVocabulariesIntoGLX(glx); err != nil {
		t.Fatalf("Failed to load vocabularies: %v", err)
	}
	index := buildGEDCOMIndex(glx)

	cases := []struct {
		name   string
		gedcom string
		want   string
	}{
		{"archive", "archive", RepositoryTypeArchive},
		{"library", "library", RepositoryTypeLibrary},
		{"church", "church", RepositoryTypeChurch},
		{"government -> government_agency", "government", RepositoryTypeGovernmentAgency},
		{"museum", "museum", RepositoryTypeMuseum},
		{"online -> database", "online", RepositoryTypeDatabase},
		{"registry", "registry", RepositoryTypeRegistry},
		{"society -> historical_society", "society", RepositoryTypeHistoricalSociety},
		{"university", "university", RepositoryTypeUniversity},
		{"uppercase normalized", "ARCHIVE", RepositoryTypeArchive},
		{"mixed case normalized", "Library", RepositoryTypeLibrary},
		{"unknown falls back to other", "embassy", RepositoryTypeOther},
		{"empty falls back to other", "", RepositoryTypeOther},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapRepositoryType(tc.gedcom, index); got != tc.want {
				t.Errorf("mapRepositoryType(%q) = %q, want %q", tc.gedcom, got, tc.want)
			}
		})
	}
}

func TestImportRepository_AcceptsLegacyStateTag(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @R1@ REPO
1 NAME State Archives
1 ADDR 123 Main St
2 CITY Springfield
2 STATE Illinois
2 POST 62701
2 CTRY USA
0 TRLR`

	glx, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glx.Repositories, 1)

	for _, repo := range glx.Repositories {
		require.Equal(t, "State Archives", repo.Name)
		require.Equal(t, "123 Main St", repo.Address)
		require.Equal(t, "Springfield", repo.City)
		require.Equal(t, "Illinois", repo.State)
		require.Equal(t, "62701", repo.PostalCode)
		require.Equal(t, "USA", repo.Country)
	}
}
