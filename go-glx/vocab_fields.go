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
)

// vocabField is one vocabulary map on a GLXFile: its top-level YAML key and a
// pointer to the field, so code that does the same thing to every vocabulary
// loops over vocabFields or propertyFields instead of naming each one (#726).
type vocabField[T any] struct {
	Key string
	Map *map[string]*T
}

// Filename is the vocabulary's file under vocabularies/, such as
// "event-types.glx" for event_types.
func (f vocabField[T]) Filename() string {
	return strings.ReplaceAll(f.Key, "_", "-") + ".glx"
}

// vocabFields lists every type vocabulary on g. Adding a vocabulary means
// adding its GLXFile field and one line here; TestVocabFieldsCoverGLXFile
// fails until both are done.
func (g *GLXFile) vocabFields() []vocabField[VocabularyEntry] {
	return []vocabField[VocabularyEntry]{
		{VocabEventTypes, &g.EventTypes},
		{VocabRelationshipTypes, &g.RelationshipTypes},
		{VocabPlaceTypes, &g.PlaceTypes},
		{VocabSourceTypes, &g.SourceTypes},
		{VocabRepositoryTypes, &g.RepositoryTypes},
		{VocabMediaTypes, &g.MediaTypes},
		{VocabSexTypes, &g.SexTypes},
		{VocabGenderTypes, &g.GenderTypes},
		{VocabLegalStatuses, &g.LegalStatuses},
		{VocabSourceNatures, &g.SourceNatures},
		{VocabInformationTypes, &g.InformationTypes},
		{VocabParticipantRoles, &g.ParticipantRoles},
		{VocabConfidenceLevels, &g.ConfidenceLevels},
		{VocabSearchResultTypes, &g.SearchResultTypes},
		{VocabResearchLogStatusTypes, &g.ResearchLogStatusTypes},
		{VocabLeadStatuses, &g.LeadStatuses},
		{VocabStudyTypes, &g.StudyTypes},
		{VocabStudyStatuses, &g.StudyStatuses},
	}
}

// propertyFields lists every property vocabulary on g, the PropertyDefinition
// counterpart of vocabFields.
func (g *GLXFile) propertyFields() []vocabField[PropertyDefinition] {
	return []vocabField[PropertyDefinition]{
		{PropPersonProperties, &g.PersonProperties},
		{PropEventProperties, &g.EventProperties},
		{PropRelationshipProperties, &g.RelationshipProperties},
		{PropPlaceProperties, &g.PlaceProperties},
		{PropMediaProperties, &g.MediaProperties},
		{PropRepositoryProperties, &g.RepositoryProperties},
		{PropCitationProperties, &g.CitationProperties},
		{PropSourceProperties, &g.SourceProperties},
	}
}
