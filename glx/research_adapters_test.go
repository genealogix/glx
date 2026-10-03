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

package main

import (
	glxlib "github.com/genealogix/glx/go-glx"
)

func mustBuildProof(personID string, person *glxlib.Person, topic string, archive *glxlib.GLXFile) *proofResult {
	if archive.Persons[personID] != person {
		panic("inconsistent proof test fixture")
	}
	result, err := glxlib.BuildProof(archive, personID, topic, glxlib.ProofOptions{Comparison: glxlib.ComparisonOptions{}, Coverage: glxlib.CoverageOptions{CensusCountry: censusCountryFallback}})
	if err != nil {
		panic(err)
	}

	return result
}

func mustCollectEvidence(archive *glxlib.GLXFile, subject glxlib.EntityRef, property string, options ...glxlib.ComparisonOptions) EvidenceReport {
	result, err := glxlib.BuildEvidenceReport(archive, subject, property, comparisonOptions(options))
	if err != nil {
		panic(err)
	}

	return result
}

func mustAnalyzeConflicts(archive *glxlib.GLXFile) []AnalysisIssue {
	return mustAnalyzeConflictsWithOptions(archive, glxlib.ComparisonOptions{})
}

func mustAnalyzeConflictsWithOptions(archive *glxlib.GLXFile, opts glxlib.ComparisonOptions) []AnalysisIssue {
	result, err := analyzeConflictsWithOptions(archive, opts)
	if err != nil {
		panic(err)
	}

	return result
}

func mustBuildCoverage(personID string, person *glxlib.Person, archive *glxlib.GLXFile) *coverageResult {
	if archive.Persons[personID] != person {
		panic("inconsistent coverage test fixture")
	}
	result, err := glxlib.BuildCoverage(archive, personID, glxlib.CoverageOptions{CensusCountry: censusCountryFallback})
	if err != nil {
		panic(err)
	}

	return result
}

// Proof conclusion levels, ordered from strongest to weakest. These summarize
// where the evidence stands relative to the BCG Genealogical Proof Standard.
const (
	proofConclusionProven       = "PROVEN"
	proofConclusionProbable     = "PROBABLE"
	proofConclusionPossible     = "POSSIBLE"
	proofConclusionInsufficient = "INSUFFICIENT EVIDENCE"
	proofConclusionConflicted   = "CONFLICTED"
)

// Canonical research-question topic keys.
const (
	topicParentage = "parentage"
	topicBirth     = "birth"
	topicDeath     = "death"
	topicMarriage  = "marriage"
	topicIdentity  = "identity"
)

// Assertion status values that drive conflict resolution. These are free-text
// in the spec but the standard vocabulary uses these tokens. (statusDisputed is
// declared in migrate_confidence_runner.go and reused here.)
const (
	statusProven    = "proven"
	statusDisproven = "disproven"
)
