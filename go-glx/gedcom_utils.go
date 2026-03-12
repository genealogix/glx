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
	"strconv"
)

// generatePersonID generates an auto-incremented person ID
func generatePersonID(conv *ConversionContext) string {
	conv.PersonCounter++

	return "person-" + strconv.Itoa(conv.PersonCounter)
}

// generateEventID generates an auto-incremented event ID
func generateEventID(conv *ConversionContext) string {
	conv.EventCounter++

	return "event-" + strconv.Itoa(conv.EventCounter)
}

// generateRelationshipID generates an auto-incremented relationship ID
func generateRelationshipID(conv *ConversionContext) string {
	conv.RelationshipCounter++

	return "relationship-" + strconv.Itoa(conv.RelationshipCounter)
}

// generatePlaceID generates an auto-incremented place ID
func generatePlaceID(conv *ConversionContext) string {
	conv.PlaceCounter++

	return "place-" + strconv.Itoa(conv.PlaceCounter)
}

// generateSourceID generates an auto-incremented source ID
func generateSourceID(conv *ConversionContext) string {
	conv.SourceCounter++

	return "source-" + strconv.Itoa(conv.SourceCounter)
}

// generateRepositoryID generates an auto-incremented repository ID
func generateRepositoryID(conv *ConversionContext) string {
	conv.RepositoryCounter++

	return "repository-" + strconv.Itoa(conv.RepositoryCounter)
}

// generateMediaID generates an auto-incremented media ID
func generateMediaID(conv *ConversionContext) string {
	conv.MediaCounter++

	return "media-" + strconv.Itoa(conv.MediaCounter)
}

// generateCitationID generates an auto-incremented citation ID
func generateCitationID(conv *ConversionContext) string {
	conv.CitationCounter++

	return "citation-" + strconv.Itoa(conv.CitationCounter)
}

// generateAssertionID generates an auto-incremented assertion ID
func generateAssertionID(conv *ConversionContext) string {
	conv.AssertionCounter++

	return "assertion-" + strconv.Itoa(conv.AssertionCounter)
}
