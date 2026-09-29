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

// BuildEvidenceReport gathers one subject/property's evidence, ranked within
// overlapping periods. Subject must contain one exact entity ID. An empty
// property selects existence/participation assertions. Property matching prefers
// exact spelling, then a case-insensitive match. Missing standard definitions
// are supplied without changing archive or overriding custom vocabularies.
// Results are deterministic, detached from the archive and safe to modify.
func BuildEvidenceReport(archive *GLXFile, subject EntityRef, property string, opts ComparisonOptions) (EvidenceReport, error) {
	if err := opts.Validate(); err != nil {
		return EvidenceReport{}, err
	}
	prepared, err := researchArchive(archive)
	if err != nil {
		return EvidenceReport{}, err
	}
	if err := researchSubjectExists(prepared, subject); err != nil {
		return EvidenceReport{}, err
	}

	return collectEvidence(prepared, subject, property, opts), nil
}

// BuildProof builds the complete evidence, conflict, search and coverage report
// for an exact person ID and research question (or its documented alias).
// It is read-only, deterministic, and supplies missing standard vocabularies.
// Invalid options, unknown questions, nil archives and missing persons return
// errors without mutating the archive. Returned data is owned by the caller.
func BuildProof(archive *GLXFile, personID, question string, opts ProofOptions) (*ProofResult, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	topic, ok := canonicalQuestion(question)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownResearchQuestion, question)
	}
	prepared, err := researchArchive(archive)
	if err != nil {
		return nil, err
	}
	if err := requirePerson(prepared, personID, "person"); err != nil {
		return nil, err
	}
	opts.Coverage.CensusCountry, err = parseCensusCountry(opts.Coverage.CensusCountry)
	if err != nil {
		return nil, err
	}

	return buildProof(personID, prepared.Persons[personID], topic, prepared, opts), nil
}

// BuildCoverage returns the complete source-coverage checklist for an exact
// person ID, without mutating archive. The result has no references to its maps.
func BuildCoverage(archive *GLXFile, personID string, opts CoverageOptions) (*CoverageResult, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	prepared, err := researchArchive(archive)
	if err != nil {
		return nil, err
	}
	if err := requirePerson(prepared, personID, "person"); err != nil {
		return nil, err
	}
	fallback, err := parseCensusCountry(opts.CensusCountry)
	if err != nil {
		return nil, err
	}

	return buildCoverage(personID, prepared.Persons[personID], prepared, fallback), nil
}

// ProofQuestions returns supported canonical research topics in stable order.
func ProofQuestions() []string { return proofQuestionKeys() }

// CanonicalProofQuestion resolves a canonical topic or a case-insensitive alias.
func CanonicalProofQuestion(question string) (string, bool) { return canonicalQuestion(question) }

// PersonConflicts returns the person's grouped comparisons, including resolved
// disagreements. It shares scope and defaults with AnalyzeConflicts and proof.
func PersonConflicts(archive *GLXFile, personID string, opts ComparisonOptions) ([]ConflictGroup, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	prepared, err := researchArchive(archive)
	if err != nil {
		return nil, err
	}
	if err := requirePerson(prepared, personID, "person"); err != nil {
		return nil, err
	}

	return detectProofConflicts(collectPersonProofAssertions(prepared, personID), prepared, opts), nil
}

// Canonical question keys accepted by BuildProof.
const (
	QuestionParentage = topicParentage
	QuestionBirth     = topicBirth
	QuestionDeath     = topicDeath
	QuestionMarriage  = topicMarriage
	QuestionIdentity  = topicIdentity
)

// Proof conclusion values, also serialized in ProofResult.Conclusion.
const (
	ConclusionProven       = proofConclusionProven
	ConclusionProbable     = proofConclusionProbable
	ConclusionPossible     = proofConclusionPossible
	ConclusionInsufficient = proofConclusionInsufficient
	ConclusionConflicted   = proofConclusionConflicted
)
