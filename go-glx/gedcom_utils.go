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
)

// generatePersonID generates an auto-incremented person ID
func generatePersonID(conv *ConversionContext) string {
	conv.PersonCounter++

	return fmt.Sprintf("person-%d", conv.PersonCounter)
}

// generateEventID generates an auto-incremented event ID
func generateEventID(conv *ConversionContext) string {
	conv.EventCounter++

	return fmt.Sprintf("event-%d", conv.EventCounter)
}

// generateRelationshipID generates an auto-incremented relationship ID
func generateRelationshipID(conv *ConversionContext) string {
	conv.RelationshipCounter++

	return fmt.Sprintf("relationship-%d", conv.RelationshipCounter)
}

// generatePlaceID generates an auto-incremented place ID
func generatePlaceID(conv *ConversionContext) string {
	conv.PlaceCounter++

	return fmt.Sprintf("place-%d", conv.PlaceCounter)
}

// generateSourceID generates an auto-incremented source ID
func generateSourceID(conv *ConversionContext) string {
	conv.SourceCounter++

	return fmt.Sprintf("source-%d", conv.SourceCounter)
}

// generateRepositoryID generates an auto-incremented repository ID
func generateRepositoryID(conv *ConversionContext) string {
	conv.RepositoryCounter++

	return fmt.Sprintf("repository-%d", conv.RepositoryCounter)
}

// generateMediaID generates an auto-incremented media ID
func generateMediaID(conv *ConversionContext) string {
	conv.MediaCounter++

	return fmt.Sprintf("media-%d", conv.MediaCounter)
}

// generateCitationID generates an auto-incremented citation ID
func generateCitationID(conv *ConversionContext) string {
	conv.CitationCounter++

	return fmt.Sprintf("citation-%d", conv.CitationCounter)
}

// generateAssertionID generates an auto-incremented assertion ID
func generateAssertionID(conv *ConversionContext) string {
	conv.AssertionCounter++

	return fmt.Sprintf("assertion-%d", conv.AssertionCounter)
}
