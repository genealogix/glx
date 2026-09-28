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

func TestInferPlaceTypeKeywords(t *testing.T) {
	tests := []struct {
		name     string
		place    string
		level    int
		expected string
	}{
		// Pre-existing keywords
		{"cemetery", "Greenwood Cemetery", 0, PlaceTypeCemetery},
		{"graveyard", "Old Graveyard", 0, PlaceTypeCemetery},
		{"churchyard is a burial ground", "St Mary Churchyard", 0, PlaceTypeCemetery},
		{"church", "St Pauls Church", 0, PlaceTypeChurch},
		{"cathedral", "St Pauls Cathedral", 0, PlaceTypeChurch},
		{"hospital", "General Hospital", 0, PlaceTypeHospital},
		{"county", "Tarrant County", 1, PlaceTypeCounty},
		{"province", "Ontario Province", 2, PlaceTypeState},

		// Institutions added for issue #540
		{"workhouse", "Bakewell Union Workhouse", 0, PlaceTypeWorkhouse},
		{"work house two words", "Union Work House", 0, PlaceTypeWorkhouse},
		{"poorhouse", "County Poorhouse", 0, PlaceTypePoorhouse},
		{"almshouse", "Trinity Almshouse", 0, PlaceTypePoorhouse},
		{"poor farm beats farm", "Union Poor Farm", 0, PlaceTypePoorhouse},
		{"asylum", "Utica State Asylum", 0, PlaceTypeAsylum},
		{"asylum beats county", "Lancaster County Asylum", 0, PlaceTypeAsylum},
		{"lunatic hospital is an asylum", "Devon County Lunatic Hospital", 0, PlaceTypeAsylum},
		{"prison", "Newgate Prison", 0, PlaceTypePrison},
		{"gaol", "Kilmainham Gaol", 0, PlaceTypePrison},
		{"penitentiary", "Eastern State Penitentiary", 0, PlaceTypePrison},
		{"school", "Rugby School", 0, PlaceTypeSchool},
		{"naval base", "Norfolk Naval Base", 0, PlaceTypeMilitaryBase},
		{"air force base", "Wright-Patterson Air Force Base", 0, PlaceTypeMilitaryBase},

		// Geographic and administrative types added for issue #540
		{"plantation", "Boone Hall Plantation", 0, PlaceTypePlantation},
		{"farm", "Bell Farm", 0, PlaceTypeFarm},
		{"estate", "Chatsworth Estate", 0, PlaceTypeEstate},
		{"village", "Village of Eichhorst", 0, PlaceTypeVillage},
		{"dorf", "Dorf Mecklenburg", 0, PlaceTypeVillage},
		{"reservation", "Pine Ridge Reservation", 0, PlaceTypeReservation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, inferPlaceType(tt.place, tt.level))
		})
	}
}

// TestInferPlaceTypeWholeWordOnly pins the keyword matching to whole words.
// Substring matching would type Farmington as a farm and any Estate as a
// state, and settlements named Fort/Port/Hamlet must keep their
// position-derived type because those words name far more towns than
// installations.
func TestInferPlaceTypeWholeWordOnly(t *testing.T) {
	tests := []struct {
		name     string
		place    string
		level    int
		expected string
	}{
		{"united states is a country, not a state", "United States", 3, PlaceTypeCountry},
		{"farmington is not a farm", "Farmington", 0, PlaceTypeCity},
		{"farmville is not a farm", "Farmville", 0, PlaceTypeCity},
		{"estate is not a state", "Estate Whim", 0, PlaceTypeEstate},
		{"schenectady is not a school", "Schenectady", 0, PlaceTypeCity},
		{"fort worth is a city", "Fort Worth", 0, PlaceTypeCity},
		{"port arthur is a city", "Port Arthur", 0, PlaceTypeCity},
		{"hamlet is a city", "Hamlet", 0, PlaceTypeCity},
		{"dusseldorf is not a village", "Düsseldorf", 0, PlaceTypeCity},
		{"prisonville is not a prison", "Prisonville", 0, PlaceTypeCity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, inferPlaceType(tt.place, tt.level))
		})
	}
}

func TestInferPlaceTypeByLevel(t *testing.T) {
	assert.Equal(t, PlaceTypeCity, inferPlaceType("Stratford-upon-Avon", 0))
	assert.Equal(t, PlaceTypeCounty, inferPlaceType("Warwickshire", 1))
	assert.Equal(t, PlaceTypeState, inferPlaceType("England", 2))
	assert.Equal(t, PlaceTypeCountry, inferPlaceType("United Kingdom", 3))
	assert.Equal(t, PlaceTypeLocality, inferPlaceType("Somewhere", 4))
}

// TestInferredPlaceTypesAreStandard guards against inferPlaceType returning a
// key that the standard place-types vocabulary does not define, which would
// make every imported archive fail validation.
func TestInferredPlaceTypesAreStandard(t *testing.T) {
	glx := &GLXFile{}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(glx))

	inferred := []string{
		PlaceTypeCemetery, PlaceTypeChurch, PlaceTypeHospital, PlaceTypeCounty,
		PlaceTypeState, PlaceTypeCity, PlaceTypeCountry, PlaceTypeLocality,
		PlaceTypeVillage, PlaceTypeEstate, PlaceTypeFarm, PlaceTypePlantation,
		PlaceTypeReservation, PlaceTypeWorkhouse, PlaceTypePoorhouse,
		PlaceTypeAsylum, PlaceTypePrison, PlaceTypeMilitaryBase, PlaceTypeSchool,
	}
	for _, placeType := range inferred {
		assert.Contains(t, glx.PlaceTypes, placeType,
			"inferPlaceType can return %q, so place-types.glx must define it", placeType)
	}

	for _, entry := range placeTypeKeywords {
		assert.Contains(t, glx.PlaceTypes, entry.placeType,
			"keyword table maps to %q, so place-types.glx must define it", entry.placeType)
	}
}

func TestContainsPlaceKeyword(t *testing.T) {
	assert.True(t, containsPlaceKeyword("bell farm", "farm"))
	assert.True(t, containsPlaceKeyword("farm", "farm"))
	assert.True(t, containsPlaceKeyword("farm road", "farm"))
	assert.True(t, containsPlaceKeyword("st. mary's-church", "church"))
	assert.False(t, containsPlaceKeyword("farmington", "farm"))
	assert.False(t, containsPlaceKeyword("nonfarm", "farm"))
	assert.False(t, containsPlaceKeyword("", "farm"))
	assert.False(t, containsPlaceKeyword("far", "farm"))
	// Repeated near-misses before a real match must not end the scan early.
	assert.True(t, containsPlaceKeyword("farmington farm", "farm"))
}
