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
	"fmt"
	"io"

	glxlib "github.com/genealogix/glx/go-glx"
)

const (
	confidenceDisputed = "disputed"
	statusDisputed     = "disputed"
)

// migrateConfidenceDisputed moves the legacy `confidence: disputed` signal
// from the confidence vocabulary to the assertion `status` field, per issue
// #516. Confidence captures evidence quality; conclusion state belongs in
// status. Disputed was conflating the two axes.
//
// Behavior per assertion with `confidence: disputed`:
//   - Status empty   → set status: disputed, clear confidence.
//   - Status already disputed → clear confidence (idempotent).
//   - Status set to something else (proven / speculative / disproven / custom)
//     → leave status alone (don't overwrite the user's research state), clear
//     confidence, and emit a warning to warnOut so the user can reconcile by
//     hand.
//
// The archive is mutated in place. Confidence is dropped (not defaulted) — the
// original `confidence: disputed` carried no evidence-quality information, so
// fabricating a replacement value would be dishonest. Users can fill in
// confidence manually post-migration if desired.
func migrateConfidenceDisputed(archive *glxlib.GLXFile, warnOut io.Writer) *MigrateReport {
	if warnOut == nil {
		warnOut = io.Discard
	}
	report := &MigrateReport{}

	if archive == nil || len(archive.Assertions) == 0 {
		return report
	}

	for _, id := range sortedKeys(archive.Assertions) {
		assertion := archive.Assertions[id]
		if assertion == nil || assertion.Confidence != confidenceDisputed {
			continue
		}

		switch assertion.Status {
		case "":
			assertion.Status = statusDisputed
		case statusDisputed:
			// Idempotent: status already records disputed; just drop confidence.
		default:
			_, _ = fmt.Fprintf(warnOut,
				"Warning: assertion %q has both confidence: disputed and status: %q. "+
					"Clearing confidence; preserving existing status. Reconcile by hand if needed.\n",
				id, assertion.Status)
			report.ConfidenceDisputedStatusConflicts++
		}

		assertion.Confidence = ""
		report.ConfidenceDisputedConverted++
	}

	if report.ConfidenceDisputedConverted > 0 {
		archive.InvalidateCache()
	}

	return report
}
