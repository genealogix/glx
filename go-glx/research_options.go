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
	"errors"
	"fmt"
	"maps"
)

// Research API validation errors support errors.Is.
var (
	ErrNilArchive              = errors.New("archive must not be nil")
	ErrInvalidApproximation    = errors.New("invalid approximation width")
	ErrUnknownResearchQuestion = errors.New("unknown research question")
	ErrUnknownCensusCountry    = errors.New("unknown census country")
	ErrResearchSubjectNotFound = errors.New("research subject not found")
	ErrInvalidResearchSubject  = errors.New("subject must identify exactly one person, event, place, or relationship")
)

// CoverageOptions selects an explicit fallback census country when a person's
// places do not identify one. The zero value makes no geographic assumption.
type CoverageOptions struct{ CensusCountry string }

// ProofOptions controls proof comparison and the coverage checklist. Its zero
// value uses ±2 years and derives census countries solely from archive places.
type ProofOptions struct {
	Comparison ComparisonOptions
	Coverage   CoverageOptions
}

// Validate checks the country without changing options or global state.
func (o CoverageOptions) Validate() error {
	_, err := parseCensusCountry(o.CensusCountry)

	return err
}

// Validate checks all proof options before the archive is inspected.
func (o ProofOptions) Validate() error {
	if err := o.Comparison.Validate(); err != nil {
		return err
	}

	return o.Coverage.Validate()
}

// MergeStandardVocabularies fills missing property definitions and empty type
// vocabularies from embedded standards. Nonempty type vocabularies are archive-owned
// sets; explicit property definitions (including nil entries) override defaults.
// It mutates only vocabulary maps and invalidates cached validation. As with
// LoadStandardVocabulariesIntoGLX, embedded definition pointers are read-only.
func MergeStandardVocabularies(archive *GLXFile) error {
	if archive == nil {
		return ErrNilArchive
	}
	standards := &GLXFile{}
	if err := LoadStandardVocabulariesIntoGLX(standards); err != nil {
		return err
	}
	archive.EventTypes = typeVocabularyDefaults(archive.EventTypes, standards.EventTypes)
	archive.RelationshipTypes = typeVocabularyDefaults(archive.RelationshipTypes, standards.RelationshipTypes)
	archive.PlaceTypes = typeVocabularyDefaults(archive.PlaceTypes, standards.PlaceTypes)
	archive.SourceTypes = typeVocabularyDefaults(archive.SourceTypes, standards.SourceTypes)
	archive.RepositoryTypes = typeVocabularyDefaults(archive.RepositoryTypes, standards.RepositoryTypes)
	archive.ParticipantRoles = typeVocabularyDefaults(archive.ParticipantRoles, standards.ParticipantRoles)
	archive.MediaTypes = typeVocabularyDefaults(archive.MediaTypes, standards.MediaTypes)
	archive.ConfidenceLevels = typeVocabularyDefaults(archive.ConfidenceLevels, standards.ConfidenceLevels)
	archive.SexTypes = typeVocabularyDefaults(archive.SexTypes, standards.SexTypes)
	archive.GenderTypes = typeVocabularyDefaults(archive.GenderTypes, standards.GenderTypes)
	archive.SearchResultTypes = typeVocabularyDefaults(archive.SearchResultTypes, standards.SearchResultTypes)
	archive.ResearchLogStatusTypes = typeVocabularyDefaults(archive.ResearchLogStatusTypes, standards.ResearchLogStatusTypes)
	archive.StudyTypes = typeVocabularyDefaults(archive.StudyTypes, standards.StudyTypes)
	archive.StudyStatuses = typeVocabularyDefaults(archive.StudyStatuses, standards.StudyStatuses)
	archive.LegalStatuses = typeVocabularyDefaults(archive.LegalStatuses, standards.LegalStatuses)
	archive.SourceNatures = typeVocabularyDefaults(archive.SourceNatures, standards.SourceNatures)
	archive.InformationTypes = typeVocabularyDefaults(archive.InformationTypes, standards.InformationTypes)
	archive.PersonProperties = vocabularyDefaults(archive.PersonProperties, standards.PersonProperties)
	archive.EventProperties = vocabularyDefaults(archive.EventProperties, standards.EventProperties)
	archive.RelationshipProperties = vocabularyDefaults(archive.RelationshipProperties, standards.RelationshipProperties)
	archive.PlaceProperties = vocabularyDefaults(archive.PlaceProperties, standards.PlaceProperties)
	archive.MediaProperties = vocabularyDefaults(archive.MediaProperties, standards.MediaProperties)
	archive.RepositoryProperties = vocabularyDefaults(archive.RepositoryProperties, standards.RepositoryProperties)
	archive.CitationProperties = vocabularyDefaults(archive.CitationProperties, standards.CitationProperties)
	archive.SourceProperties = vocabularyDefaults(archive.SourceProperties, standards.SourceProperties)
	archive.validation = nil

	return nil
}

func vocabularyDefaults[T any](custom, standard map[string]*T) map[string]*T {
	merged := maps.Clone(standard)
	maps.Copy(merged, custom)

	return merged
}

func researchArchive(archive *GLXFile) (*GLXFile, error) {
	if archive == nil {
		return nil, ErrNilArchive
	}
	view := *archive
	if err := MergeStandardVocabularies(&view); err != nil {
		return nil, err
	}

	return &view, nil
}

func researchSubjectExists(archive *GLXFile, subject EntityRef) error {
	count := 0
	for _, id := range []string{subject.Person, subject.Event, subject.Relationship, subject.Place} {
		if id != "" {
			count++
		}
	}
	if count != 1 {
		return ErrInvalidResearchSubject
	}
	exists := false
	switch subject.Type() {
	case EntityTypePersons:
		exists = archive.Persons[subject.Person] != nil
	case EntityTypeEvents:
		exists = archive.Events[subject.Event] != nil
	case EntityTypeRelationships:
		exists = archive.Relationships[subject.Relationship] != nil
	case EntityTypePlaces:
		exists = archive.Places[subject.Place] != nil
	}
	if !exists {
		return fmt.Errorf("%w: %s %q", ErrResearchSubjectNotFound, subject.Type(), subject.ID())
	}

	return nil
}

func typeVocabularyDefaults[T any](custom, standard map[string]*T) map[string]*T {
	if len(custom) == 0 {
		return maps.Clone(standard)
	}

	return maps.Clone(custom)
}
