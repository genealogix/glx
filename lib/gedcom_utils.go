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

package lib

import (
	"fmt"
)

// generatePersonID generates an auto-incremented person ID
func generatePersonID(ctx *ConversionContext) string {
	ctx.PersonCounter++
	return fmt.Sprintf("person-%d", ctx.PersonCounter)
}

// generateEventID generates an auto-incremented event ID
func generateEventID(ctx *ConversionContext) string {
	ctx.EventCounter++
	return fmt.Sprintf("event-%d", ctx.EventCounter)
}

// generateRelationshipID generates an auto-incremented relationship ID
func generateRelationshipID(ctx *ConversionContext) string {
	ctx.RelationshipCounter++
	return fmt.Sprintf("relationship-%d", ctx.RelationshipCounter)
}

// generatePlaceID generates an auto-incremented place ID
func generatePlaceID(ctx *ConversionContext) string {
	ctx.PlaceCounter++
	return fmt.Sprintf("place-%d", ctx.PlaceCounter)
}

// generateSourceID generates an auto-incremented source ID
func generateSourceID(ctx *ConversionContext) string {
	ctx.SourceCounter++
	return fmt.Sprintf("source-%d", ctx.SourceCounter)
}

// generateRepositoryID generates an auto-incremented repository ID
func generateRepositoryID(ctx *ConversionContext) string {
	ctx.RepositoryCounter++
	return fmt.Sprintf("repository-%d", ctx.RepositoryCounter)
}

// generateMediaID generates an auto-incremented media ID
func generateMediaID(ctx *ConversionContext) string {
	ctx.MediaCounter++
	return fmt.Sprintf("media-%d", ctx.MediaCounter)
}

// generateCitationID generates an auto-incremented citation ID
func generateCitationID(ctx *ConversionContext) string {
	ctx.CitationCounter++
	return fmt.Sprintf("citation-%d", ctx.CitationCounter)
}

// generateAssertionID generates an auto-incremented assertion ID
func generateAssertionID(ctx *ConversionContext) string {
	ctx.AssertionCounter++
	return fmt.Sprintf("assertion-%d", ctx.AssertionCounter)
}

// generateParticipationID generates an auto-incremented participation ID
func generateParticipationID(ctx *ConversionContext) string {
	ctx.ParticipationCounter++
	return fmt.Sprintf("participation-%d", ctx.ParticipationCounter)
}
